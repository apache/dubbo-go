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
	"errors"
	"sync"
	"time"
)

import (
	"github.com/go-zookeeper/zk"

	"github.com/hashicorp/golang-lru"
)

const (
	pathLockShardCount  = 128
	maxCacheEntries     = 1024
	maxAutoWatches      = 1024
	maxCacheLoadRetries = 3
)

var errWatchRegistrationStale = errors.New("zookeeper watch registration became stale")
var errBusinessWatchCanceled = errors.New("zookeeper business watch registration canceled")

type watchOperation struct {
	token     uint64
	done      chan struct{}
	err       error
	completed bool
}

type configCacheEntry struct {
	content   string
	exists    bool
	expiresAt time.Time
}

type watchRegistration struct {
	events              <-chan zk.Event
	beforeSessionID     int64
	afterSessionID      int64
	readBeforeSessionID int64
	readAfterSessionID  int64
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
	return r.resultSessionID(), true
}

func (r watchRegistration) sessionStable() bool {
	if !stableSessionPair(r.beforeSessionID, r.afterSessionID) ||
		!stableSessionPair(r.readBeforeSessionID, r.readAfterSessionID) {
		return false
	}
	var sessionID int64
	for _, id := range []int64{
		r.beforeSessionID,
		r.afterSessionID,
		r.readBeforeSessionID,
		r.readAfterSessionID,
	} {
		if id == 0 {
			continue
		}
		if sessionID == 0 {
			sessionID = id
			continue
		}
		if sessionID != id {
			return false
		}
	}
	return true
}

func (r watchRegistration) resultSessionID() int64 {
	if r.readAfterSessionID != 0 {
		return r.readAfterSessionID
	}
	return r.afterSessionID
}

func stableSessionPair(beforeSessionID, afterSessionID int64) bool {
	return (beforeSessionID == 0 && afterSessionID == 0) ||
		(beforeSessionID != 0 && beforeSessionID == afterSessionID)
}

type configWatchState struct {
	registered bool
	pending    bool
	auto       bool
	retired    bool
	sessionID  int64
	pendingOp  *watchOperation
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

func (s configWatchState) pendingOpToken() uint64 {
	if s.pendingOp == nil {
		return 0
	}
	return s.pendingOp.token
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
	nextWatchToken        uint64

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
		generation, _ := c.snapshot(path)
		entry, registration, err := loader(false)
		if err == nil && !c.loadResultCurrent(generation, registration) {
			return configCacheEntry{}, errWatchRegistrationStale
		}
		return entry, err
	}
	if entry, ok := c.getFresh(path); ok {
		return entry, nil
	}

	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	watchRetries := 0
	for attempt := 0; attempt <= maxCacheLoadRetries; attempt++ {
		if entry, ok := c.getFresh(path); ok {
			return entry, nil
		}

		generation, registerWatch, token := c.prepareLoad(path)
		entry, registration, err := loader(registerWatch)
		if registerWatch {
			stored := c.finishWatchRegistrationLocked(path, generation, token, registration)
			if !stored || !c.loadResultCurrent(generation, registration) {
				if watchRetries == 0 && attempt < maxCacheLoadRetries {
					watchRetries++
					continue
				}
				fallbackEntry, fallbackRegistration, fallbackErr := loader(false)
				if fallbackErr != nil {
					return configCacheEntry{}, fallbackErr
				}
				if !c.loadResultCurrent(generation, fallbackRegistration) {
					if attempt == maxCacheLoadRetries {
						return configCacheEntry{}, errWatchRegistrationStale
					}
					continue
				}
				if c.storeLoadEntry(path, generation, fallbackEntry) {
					return fallbackEntry, nil
				}
				if attempt == maxCacheLoadRetries {
					return configCacheEntry{}, errWatchRegistrationStale
				}
				continue
			}
		} else if !c.loadResultCurrent(generation, registration) {
			if attempt == maxCacheLoadRetries {
				return configCacheEntry{}, errWatchRegistrationStale
			}
			continue
		}

		if err != nil {
			if attempt == maxCacheLoadRetries && !c.isCurrentGeneration(generation) {
				return configCacheEntry{}, errWatchRegistrationStale
			}
			if !c.isCurrentGeneration(generation) {
				continue
			}
			return configCacheEntry{}, err
		}

		if c.storeLoadEntry(path, generation, entry) {
			return entry, nil
		}
		if attempt == maxCacheLoadRetries {
			return configCacheEntry{}, errWatchRegistrationStale
		}
	}
	return configCacheEntry{}, errWatchRegistrationStale
}

func (c *configCache) prepareLoad(path string) (uint64, bool, uint64) {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	generation := c.generation
	if c.watches[path].tracked() {
		return generation, false, 0
	}

	op := c.newWatchOperationLocked()
	pendingState := configWatchState{auto: true, pending: true, sessionID: c.sessionID, pendingOp: op}
	if !c.setWatchStateLocked(path, pendingState) {
		return generation, false, 0
	}
	return generation, true, op.token
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

func (c *configCache) storeAtGenerationLocked(path string, generation uint64, entry configCacheEntry) {
	if !c.enabled() {
		return
	}
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	if c.generation == generation {
		c.storeEntryLocked(path, entry)
	}
}

func (c *configCache) storeLocked(path string, entry configCacheEntry) {
	if !c.enabled() {
		return
	}
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	c.storeEntryLocked(path, entry)
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

func (c *configCache) loadResultCurrent(generation uint64, registration watchRegistration) bool {
	if !registration.sessionStable() {
		return false
	}
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	if c.generation != generation {
		return false
	}
	resultSessionID := registration.resultSessionID()
	return c.sessionID == 0 ||
		(resultSessionID != 0 && resultSessionID == c.sessionID)
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

func (c *configCache) newWatchOperationLocked() *watchOperation {
	c.nextWatchToken++
	return &watchOperation{
		token: c.nextWatchToken,
		done:  make(chan struct{}),
	}
}

func (c *configCache) completeWatchOperationLocked(op *watchOperation, err error) {
	if op == nil || op.completed {
		return
	}
	op.err = err
	op.completed = true
	close(op.done)
}

func (c *configCache) clearWatchStateLocked(path string, err error) {
	current := c.watches[path]
	if current.pending {
		c.completeWatchOperationLocked(current.pendingOp, err)
	}
	c.setWatchStateLocked(path, configWatchState{})
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
	if !watchState.pending {
		watchState.pendingOp = nil
	}
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

func (c *configCache) finishWatchRegistration(path string, generation, token uint64, registration watchRegistration) bool {
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()
	return c.finishWatchRegistrationLocked(path, generation, token, registration)
}

func (c *configCache) finishWatchRegistrationLocked(path string, generation, token uint64, registration watchRegistration) bool {
	sessionID, active := registration.resolve()
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	watchState := c.watches[path]
	if !watchState.pending {
		return false
	}
	if (watchState.pendingOp != nil && watchState.pendingOp.token != token) ||
		(watchState.pendingOp == nil && token != 0) {
		return false
	}
	generationCurrent := c.generation == generation
	sessionCurrent := c.sessionID == 0 ||
		(sessionID != 0 && sessionID == c.sessionID)
	if !active || !generationCurrent || !registration.sessionStable() || !sessionCurrent {
		c.clearWatchStateLocked(path, errWatchRegistrationStale)
		return false
	}
	op := watchState.pendingOp
	watchState.registered = true
	watchState.pending = false
	watchState.sessionID = sessionID
	watchState.pendingOp = nil
	stored := c.setWatchStateLocked(path, watchState)
	if stored {
		c.completeWatchOperationLocked(op, nil)
	}
	return stored
}

func (c *configCache) ensureBusinessWatch(
	path string,
	register func() (watchRegistration, error),
) error {
	return c.ensureBusinessWatchWithRetry(path, register, 1)
}

func (c *configCache) ensureBusinessWatchWithRetry(
	path string,
	register func() (watchRegistration, error),
	maxRetries int,
) error {
	return c.ensureBusinessWatchWithRetryIf(path, register, maxRetries, nil)
}

func (c *configCache) ensureBusinessWatchWithRetryIf(
	path string,
	register func() (watchRegistration, error),
	maxRetries int,
	stillNeeded func() bool,
) error {
	for attempt := 0; ; attempt++ {
		if stillNeeded != nil && !stillNeeded() {
			return errBusinessWatchCanceled
		}
		err := c.ensureBusinessWatchOnce(path, register)
		if err == nil && stillNeeded != nil && !stillNeeded() {
			return errBusinessWatchCanceled
		}
		if !errors.Is(err, errWatchRegistrationStale) || attempt >= maxRetries {
			return err
		}
	}
}

func (c *configCache) ensureBusinessWatchOnce(
	path string,
	register func() (watchRegistration, error),
) error {
	pathLock := c.pathLock(path)
	pathLock.Lock()
	c.stateLock.Lock()
	watchState := c.watches[path]
	if watchState.registered {
		watchState.auto = false
		watchState.retired = false
		c.setWatchStateLocked(path, watchState)
		c.stateLock.Unlock()
		pathLock.Unlock()
		return nil
	}
	if watchState.pending {
		watchState.auto = false
		watchState.retired = false
		c.setWatchStateLocked(path, watchState)
		op := watchState.pendingOp
		c.stateLock.Unlock()
		pathLock.Unlock()
		if op == nil {
			return nil
		}
		<-op.done
		return op.err
	}
	generation := c.generation
	sessionID := c.sessionID
	op := c.newWatchOperationLocked()
	c.setWatchStateLocked(path, configWatchState{
		pending:   true,
		sessionID: sessionID,
		pendingOp: op,
	})
	c.stateLock.Unlock()
	pathLock.Unlock()

	registration, err := register()
	pathLock.Lock()
	defer pathLock.Unlock()
	if err != nil {
		c.stateLock.Lock()
		current := c.watches[path]
		if current.pendingOp == op {
			c.clearWatchStateLocked(path, err)
		} else {
			c.completeWatchOperationLocked(op, err)
		}
		c.stateLock.Unlock()
		return err
	}
	if c.finishWatchRegistrationLocked(path, generation, op.token, registration) {
		return nil
	}
	c.stateLock.Lock()
	c.completeWatchOperationLocked(op, errWatchRegistrationStale)
	c.stateLock.Unlock()
	return errWatchRegistrationStale
}

func (c *configCache) releaseBusinessWatchLocked(path string) {
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

func (c *configCache) beginWatchRenewalLocked(path string, hasListeners bool) (uint64, configWatchState, bool) {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	generation := c.generation
	watchState, ok := c.watches[path]
	if !ok {
		if !hasListeners {
			return generation, watchState, false
		}
		c.setWatchStateLocked(path, configWatchState{
			pending:   true,
			sessionID: c.sessionID,
			pendingOp: c.newWatchOperationLocked(),
		})
		return generation, c.watches[path], true
	}
	if watchState.pending {
		if hasListeners {
			watchState.auto = false
			watchState.retired = false
			c.setWatchStateLocked(path, watchState)
		}
		return generation, c.watches[path], false
	}
	if watchState.retired && !hasListeners {
		c.setWatchStateLocked(path, configWatchState{})
		return generation, configWatchState{}, false
	}
	if hasListeners {
		watchState.auto = false
		watchState.retired = false
	} else if !watchState.auto {
		c.setWatchStateLocked(path, configWatchState{})
		return generation, configWatchState{}, false
	}
	watchState.registered = false
	watchState.pending = true
	watchState.pendingOp = c.newWatchOperationLocked()
	c.setWatchStateLocked(path, watchState)
	return generation, c.watches[path], true
}

func (c *configCache) cancelPendingLocked(path string) {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	if c.watches[path].pending {
		c.clearWatchStateLocked(path, errWatchRegistrationStale)
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
