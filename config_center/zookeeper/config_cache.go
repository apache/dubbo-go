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

type configCacheEntry struct {
	content   string
	exists    bool
	expiresAt time.Time
}

type configCache struct {
	ttl time.Duration

	stateLock  sync.RWMutex
	entries    map[string]configCacheEntry
	watches    map[string]bool
	generation uint64

	pathLocks sync.Map
}

func newConfigCache(ttl time.Duration) configCache {
	return configCache{
		ttl:     ttl,
		entries: make(map[string]configCacheEntry),
		watches: make(map[string]bool),
	}
}

func (c *configCache) enabled() bool {
	return c.ttl > 0
}

func (c *configCache) load(path string, loader func(bool) (configCacheEntry, bool, error)) (configCacheEntry, error) {
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

		generation, watchActive := c.snapshot(path)
		entry, nextWatchActive, err := loader(watchActive)
		if err != nil {
			if !c.storeWatchState(path, generation, nextWatchActive) {
				continue
			}
			return configCacheEntry{}, err
		}
		if !c.storeLoad(path, generation, entry, nextWatchActive) {
			continue
		}
		return entry, nil
	}
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
	defer c.stateLock.RUnlock()

	entry, ok := c.entries[path]
	if !ok {
		return configCacheEntry{}, false
	}
	return entry, entry.expiresAt.After(time.Now())
}

func (c *configCache) storeEntryLocked(path string, entry configCacheEntry) {
	entry.expiresAt = time.Now().Add(c.ttl)
	c.entries[path] = entry
}

func (c *configCache) snapshot(path string) (uint64, bool) {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	return c.generation, c.watches[path]
}

func (c *configCache) isCurrentGeneration(generation uint64) bool {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	return c.generation == generation
}

func (c *configCache) storeLoad(path string, generation uint64, entry configCacheEntry, watchActive bool) bool {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	if c.generation != generation {
		return false
	}

	c.storeEntryLocked(path, entry)
	c.setWatchActiveLocked(path, watchActive)
	return true
}

func (c *configCache) storeWatchState(path string, generation uint64, watchActive bool) bool {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	if c.generation != generation {
		return false
	}
	c.setWatchActiveLocked(path, watchActive)
	return true
}

func (c *configCache) setWatchActive(path string, active bool) {
	if !c.enabled() {
		return
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	c.setWatchActiveLocked(path, active)
}

func (c *configCache) setWatchActiveAtGeneration(path string, generation uint64, active bool) {
	if !c.enabled() {
		return
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()
	c.storeWatchState(path, generation, active)
}

func (c *configCache) setWatchActiveLocked(path string, active bool) {
	if active {
		c.watches[path] = true
		return
	}
	delete(c.watches, path)
}

func (c *configCache) ensureWatch(path string, register func() error) error {
	if !c.enabled() {
		return register()
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	for {
		generation, active := c.snapshot(path)
		if active {
			return nil
		}
		if err := register(); err != nil {
			if !c.isCurrentGeneration(generation) {
				continue
			}
			return err
		}

		c.stateLock.Lock()
		if c.generation == generation {
			c.watches[path] = true
			c.stateLock.Unlock()
			return nil
		}
		c.stateLock.Unlock()
	}
}

func (c *configCache) reset() {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	c.generation++
	c.entries = make(map[string]configCacheEntry)
	c.watches = make(map[string]bool)
}

func (c *configCache) pathLock(path string) *sync.Mutex {
	lock, _ := c.pathLocks.LoadOrStore(path, &sync.Mutex{})
	return lock.(*sync.Mutex)
}
