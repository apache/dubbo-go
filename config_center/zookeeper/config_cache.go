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
	"github.com/dubbogo/go-zookeeper/zk"

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

type configWatchState struct {
	watcher *zk.Watcher
	auto    bool
	pending bool
}

func (s configWatchState) tracked() bool {
	return s.watcher != nil || s.pending
}

func (s configWatchState) holdsAutoSlot() bool {
	return s.auto && s.tracked()
}

func (s configWatchState) holdsAutoWatch() bool {
	return s.auto && s.watcher != nil
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
	loader func(*zk.Watcher, bool) (configCacheEntry, *zk.Watcher, error),
	removeWatcher func(*zk.Watcher),
) (configCacheEntry, error) {
	if !c.enabled() {
		entry, _, err := loader(nil, false)
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

		generation, watchState, registerWatch := c.prepareLoad(path)
		entry, watcher, err := loader(watchState.watcher, registerWatch)
		nextWatchState := watchState
		if registerWatch {
			nextWatchState = configWatchState{}
			if watcher != nil {
				nextWatchState = configWatchState{watcher: watcher, auto: true}
			}
		}
		if err != nil {
			if !c.storeWatchState(path, generation, nextWatchState) {
				if registerWatch {
					removeRegisteredWatcher(removeWatcher, watcher)
				}
				continue
			}
			return configCacheEntry{}, err
		}
		if !c.storeLoad(path, generation, entry, nextWatchState) {
			if registerWatch {
				removeRegisteredWatcher(removeWatcher, watcher)
			}
			continue
		}
		return entry, nil
	}
}

func (c *configCache) prepareLoad(path string) (uint64, configWatchState, bool) {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	generation := c.generation
	watchState := c.watches[path]
	if watchState.tracked() {
		return generation, watchState, false
	}

	pendingState := configWatchState{auto: true, pending: true}
	if !c.setWatchStateLocked(path, pendingState) {
		return generation, configWatchState{}, false
	}
	return generation, pendingState, true
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

func (c *configCache) storeLoad(path string, generation uint64, entry configCacheEntry, watchState configWatchState) bool {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	if c.generation != generation {
		return false
	}

	c.storeEntryLocked(path, entry)
	c.setWatchStateLocked(path, watchState)
	return true
}

func (c *configCache) storeWatchState(path string, generation uint64, watchState configWatchState) bool {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	if c.generation != generation {
		return false
	}
	return c.setWatchStateLocked(path, watchState)
}

func (c *configCache) setWatch(path string, watchState configWatchState) bool {
	if !c.enabled() {
		return false
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	return c.setWatchStateLocked(path, watchState)
}

func (c *configCache) setWatchAtGeneration(path string, generation uint64, watchState configWatchState) bool {
	if !c.enabled() {
		return false
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()
	return c.storeWatchState(path, generation, watchState)
}

func (c *configCache) setWatchStateLocked(path string, watchState configWatchState) bool {
	current := c.watches[path]
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

func (c *configCache) ensureBusinessWatch(
	path string,
	register func() (*zk.Watcher, error),
	removeWatcher func(*zk.Watcher),
) error {
	if !c.enabled() {
		_, err := register()
		return err
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	for {
		generation, watchState := c.snapshot(path)
		if watchState.tracked() {
			if watchState.auto {
				watchState.auto = false
				c.storeWatchState(path, generation, watchState)
			}
			return nil
		}
		watcher, err := register()
		if err != nil {
			if !c.isCurrentGeneration(generation) {
				continue
			}
			return err
		}

		c.stateLock.Lock()
		if c.generation == generation {
			c.setWatchStateLocked(path, configWatchState{watcher: watcher})
			c.stateLock.Unlock()
			return nil
		}
		c.stateLock.Unlock()
		removeRegisteredWatcher(removeWatcher, watcher)
	}
}

func (c *configCache) promoteWatch(path string) {
	if !c.enabled() {
		return
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	watchState, ok := c.watches[path]
	if !ok || !watchState.auto {
		return
	}
	watchState.auto = false
	c.setWatchStateLocked(path, watchState)
}

func (c *configCache) releaseBusinessWatch(path string) *zk.Watcher {
	if !c.enabled() {
		return nil
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	watchState, ok := c.watches[path]
	if !ok || watchState.auto {
		return nil
	}

	autoWatchState := watchState
	autoWatchState.auto = true
	if c.setWatchStateLocked(path, autoWatchState) {
		return nil
	}
	c.setWatchStateLocked(path, configWatchState{})
	return watchState.watcher
}

func (c *configCache) beginWatchRenewal(path string, generation uint64, auto bool) bool {
	if !c.enabled() {
		return !auto
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	return c.storeWatchState(path, generation, configWatchState{auto: auto, pending: true})
}

func (c *configCache) cancelWatchRenewal(path string, generation uint64) {
	if !c.enabled() {
		return
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	watchState := c.watches[path]
	if c.generation == generation && watchState.pending {
		c.setWatchStateLocked(path, configWatchState{})
	}
}

func (c *configCache) reset() []*zk.Watcher {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	watchers := make([]*zk.Watcher, 0, len(c.watches))
	for _, watchState := range c.watches {
		if watchState.watcher != nil {
			watchers = append(watchers, watchState.watcher)
		}
	}
	c.generation++
	c.entries.Purge()
	c.watches = make(map[string]configWatchState)
	c.autoWatchCount = 0
	c.autoWatchReservations = 0
	return watchers
}

func removeRegisteredWatcher(removeWatcher func(*zk.Watcher), watcher *zk.Watcher) {
	if watcher != nil && removeWatcher != nil {
		removeWatcher(watcher)
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
