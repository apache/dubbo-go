/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package zookeeper

import (
	"sync"
	"time"
)

import (
	"github.com/go-zookeeper/zk"

	"github.com/hashicorp/golang-lru"
)

const (
	pathLockShardCount = 128
	maxCacheEntries    = 1024
	maxAutoWatches     = 1024
)

type configCacheEntry struct {
	content   string
	exists    bool
	expiresAt time.Time
}

type watchRegistration struct {
	events          <-chan zk.Event
	beforeSessionID int64
	afterSessionID  int64
}

func (r watchRegistration) resolve() (int64, bool) {
	if r.events == nil || r.afterSessionID == 0 {
		return 0, false
	}
	if r.beforeSessionID != r.afterSessionID {
		select {
		case <-r.events:
			return 0, false
		default:
		}
	}
	return r.afterSessionID, true
}

type configWatchState struct {
	registered bool
	pending    bool
	auto       bool
	retired    bool
	sessionID  int64
}

func (s configWatchState) tracked() bool {
	return s.registered || s.pending
}

func (s configWatchState) holdsAutoSlot() bool {
	return s.auto && s.tracked()
}

func (s configWatchState) holdsAutoWatch() bool {
	return s.auto && s.registered
}

func (s configWatchState) holdsAutoReservation() bool {
	return s.auto && s.pending
}

type configCache struct {
	ttl time.Duration

	stateLock             sync.RWMutex
	entries               *lru.Cache
	watches               map[string]configWatchState
	autoWatchCount        int
	autoWatchReservations int
	generation            uint64
	sessionID             int64

	pathLocks [pathLockShardCount]sync.Mutex
}

func newConfigCache(ttl time.Duration) configCache {
	entries, err := lru.New(maxCacheEntries)
	if err != nil {
		panic(err)
	}
	return configCache{
		ttl:     ttl,
		entries: entries,
		watches: make(map[string]configWatchState),
	}
}

func (c *configCache) enabled() bool {
	return c.ttl > 0
}

func (c *configCache) load(
	path string,
	loader func(bool) (configCacheEntry, watchRegistration, error),
) (configCacheEntry, error) {
	if !c.enabled() {
		entry, _, err := loader(false)
		return entry, err
	}
	if entry, ok := c.getFresh(path); ok {
		return entry, nil
	}

	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	for {
		if entry, ok := c.getFresh(path); ok {
			return entry, nil
		}

		generation, registerWatch := c.prepareLoad(path)
		entry, registration, err := loader(registerWatch)
		if registerWatch {
			c.finishWatchRegistrationLocked(path, generation, registration)
		}
		if err != nil {
			if !c.isCurrentGeneration(generation) {
				continue
			}
			return configCacheEntry{}, err
		}
		if !c.storeLoadEntry(path, generation, entry) {
			continue
		}
		return entry, nil
	}
}

func (c *configCache) prepareLoad(path string) (uint64, bool) {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	generation := c.generation
	if c.watches[path].tracked() {
		return generation, false
	}

	pendingState := configWatchState{auto: true, pending: true, sessionID: c.sessionID}
	if !c.setWatchStateLocked(path, pendingState) {
		return generation, false
	}
	return generation, true
}

func (c *configCache) store(path string, entry configCacheEntry) {
	if !c.enabled() {
		return
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	c.storeEntryLocked(path, entry)
}

func (c *configCache) storeAtGeneration(path string, generation uint64, entry configCacheEntry) {
	if !c.enabled() {
		return
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	if c.generation == generation {
		c.storeEntryLocked(path, entry)
	}
}

func (c *configCache) getFresh(path string) (configCacheEntry, bool) {
	c.stateLock.RLock()
	value, ok := c.entries.Get(path)
	if !ok {
		c.stateLock.RUnlock()
		return configCacheEntry{}, false
	}
	entry := value.(configCacheEntry)
	if entry.expiresAt.After(time.Now()) {
		c.stateLock.RUnlock()
		return entry, true
	}
	c.stateLock.RUnlock()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	value, ok = c.entries.Get(path)
	if !ok {
		return configCacheEntry{}, false
	}
	entry = value.(configCacheEntry)
	if entry.expiresAt.After(time.Now()) {
		return entry, true
	}
	c.entries.Remove(path)
	return configCacheEntry{}, false
}

func (c *configCache) storeEntryLocked(path string, entry configCacheEntry) {
	entry.expiresAt = time.Now().Add(c.ttl)
	c.entries.Add(path, entry)
}

func (c *configCache) snapshot(path string) (uint64, configWatchState) {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	return c.generation, c.watches[path]
}

func (c *configCache) isCurrentGeneration(generation uint64) bool {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	return c.generation == generation
}

func (c *configCache) storeLoadEntry(path string, generation uint64, entry configCacheEntry) bool {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	if c.generation != generation {
		return false
	}
	c.storeEntryLocked(path, entry)
	return true
}

func (c *configCache) setWatch(path string, watchState configWatchState) bool {
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	return c.setWatchStateLocked(path, watchState)
}

func (c *configCache) setWatchStateLocked(path string, watchState configWatchState) bool {
	current := c.watches[path]
	if watchState.auto && !c.enabled() {
		return false
	}
	if watchState.holdsAutoSlot() && !current.holdsAutoSlot() &&
		c.autoWatchCount+c.autoWatchReservations >= maxAutoWatches {
		return false
	}
	if current.holdsAutoWatch() {
		c.autoWatchCount--
	}
	if current.holdsAutoReservation() {
		c.autoWatchReservations--
	}
	if !watchState.tracked() {
		delete(c.watches, path)
		return true
	}
	c.watches[path] = watchState
	if watchState.holdsAutoWatch() {
		c.autoWatchCount++
	}
	if watchState.holdsAutoReservation() {
		c.autoWatchReservations++
	}
	return true
}

func (c *configCache) finishWatchRegistration(path string, generation uint64, registration watchRegistration) bool {
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()
	return c.finishWatchRegistrationLocked(path, generation, registration)
}

func (c *configCache) finishWatchRegistrationLocked(path string, generation uint64, registration watchRegistration) bool {
	sessionID, active := registration.resolve()
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	watchState := c.watches[path]
	if !watchState.pending {
		return false
	}
	if !active || c.generation != generation && c.sessionID != 0 && sessionID != c.sessionID {
		c.setWatchStateLocked(path, configWatchState{})
		return false
	}
	watchState.registered = true
	watchState.pending = false
	watchState.sessionID = sessionID
	return c.setWatchStateLocked(path, watchState)
}

func (c *configCache) ensureBusinessWatch(
	path string,
	register func() (watchRegistration, error),
) error {
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	for {
		c.stateLock.Lock()
		watchState := c.watches[path]
		if watchState.tracked() {
			watchState.auto = false
			watchState.retired = false
			c.setWatchStateLocked(path, watchState)
			c.stateLock.Unlock()
			return nil
		}
		generation := c.generation
		c.setWatchStateLocked(path, configWatchState{pending: true, sessionID: c.sessionID})
		c.stateLock.Unlock()

		registration, err := register()
		if err != nil {
			c.cancelPendingLocked(path)
			return err
		}
		if c.finishWatchRegistrationLocked(path, generation, registration) {
			return nil
		}
	}
}

func (c *configCache) releaseBusinessWatch(path string) {
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	watchState, ok := c.watches[path]
	if !ok || watchState.auto {
		return
	}

	autoWatchState := watchState
	autoWatchState.auto = true
	autoWatchState.retired = false
	if c.setWatchStateLocked(path, autoWatchState) {
		return
	}
	watchState.retired = true
	c.setWatchStateLocked(path, watchState)
}

func (c *configCache) beginWatchRenewal(path string, hasListeners bool) (uint64, bool) {
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	generation := c.generation
	watchState, ok := c.watches[path]
	if !ok {
		if !hasListeners {
			return generation, false
		}
		c.setWatchStateLocked(path, configWatchState{pending: true, sessionID: c.sessionID})
		return generation, true
	}
	if watchState.pending {
		if hasListeners {
			watchState.auto = false
			watchState.retired = false
			c.setWatchStateLocked(path, watchState)
		}
		return generation, false
	}
	if watchState.retired && !hasListeners {
		c.setWatchStateLocked(path, configWatchState{})
		return generation, false
	}
	if hasListeners {
		watchState.auto = false
		watchState.retired = false
	} else if !watchState.auto {
		c.setWatchStateLocked(path, configWatchState{})
		return generation, false
	}
	watchState.registered = false
	watchState.pending = true
	c.setWatchStateLocked(path, watchState)
	return generation, true
}

func (c *configCache) cancelWatchRenewal(path string) {
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()
	c.cancelPendingLocked(path)
}

func (c *configCache) cancelPendingLocked(path string) {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	if c.watches[path].pending {
		c.setWatchStateLocked(path, configWatchState{})
	}
}

func (c *configCache) reset(sessionID int64) {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	c.generation++
	c.sessionID = sessionID
	c.entries.Purge()
	for path, watchState := range c.watches {
		if watchState.pending || watchState.sessionID == sessionID {
			continue
		}
		c.setWatchStateLocked(path, configWatchState{})
	}
}

func (c *configCache) pathLock(path string) *sync.Mutex {
	var hash uint32 = 2166136261
	for i := 0; i < len(path); i++ {
		hash ^= uint32(path[i])
		hash *= 16777619
	}
	return &c.pathLocks[hash%pathLockShardCount]
}
