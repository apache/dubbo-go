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
)

type configCacheEntry struct {
	content   string
	exists    bool
	expiresAt time.Time
}

type configWatchState struct {
	watcher *zk.Watcher
	auto    bool
}

type configCache struct {
	ttl time.Duration

	stateLock      sync.RWMutex
	entries        *lru.Cache
	watches        map[string]configWatchState
	autoWatchCount int
	generation     uint64

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

func (c *configCache) load(path string, loader func(*zk.Watcher) (configCacheEntry, *zk.Watcher, error)) (configCacheEntry, error) {
	if !c.enabled() {
		entry, _, err := loader(nil)
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

		generation, watchState := c.snapshot(path)
		entry, watcher, err := loader(watchState.watcher)
		nextWatchState := configWatchState{watcher: watcher, auto: true}
		if watcher == watchState.watcher {
			nextWatchState.auto = watchState.auto
		}
		if err != nil {
			if !c.storeWatchState(path, generation, nextWatchState) {
				continue
			}
			return configCacheEntry{}, err
		}
		if !c.storeLoad(path, generation, entry, nextWatchState) {
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

	value, ok := c.entries.Get(path)
	if !ok {
		return configCacheEntry{}, false
	}
	entry := value.(configCacheEntry)
	if !entry.expiresAt.After(time.Now()) {
		c.entries.Remove(path)
		return configCacheEntry{}, false
	}
	return entry, true
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
	c.setWatchStateLocked(path, watchState)
	return true
}

func (c *configCache) setWatch(path string, watchState configWatchState) {
	if !c.enabled() {
		return
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	c.setWatchStateLocked(path, watchState)
}

func (c *configCache) setWatchAtGeneration(path string, generation uint64, watchState configWatchState) {
	if !c.enabled() {
		return
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()
	c.storeWatchState(path, generation, watchState)
}

func (c *configCache) setWatchStateLocked(path string, watchState configWatchState) {
	if current, ok := c.watches[path]; ok && current.auto {
		c.autoWatchCount--
	}
	if watchState.watcher == nil {
		delete(c.watches, path)
		return
	}
	c.watches[path] = watchState
	if watchState.auto {
		c.autoWatchCount++
	}
}

func (c *configCache) ensureBusinessWatch(path string, register func() (*zk.Watcher, error)) error {
	if !c.enabled() {
		_, err := register()
		return err
	}
	pathLock := c.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()

	for {
		generation, watchState := c.snapshot(path)
		if watchState.watcher != nil {
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

func (c *configCache) reset() {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	c.generation++
	c.entries.Purge()
	c.watches = make(map[string]configWatchState)
	c.autoWatchCount = 0
}

func (c *configCache) pathLock(path string) *sync.Mutex {
	var hash uint32 = 2166136261
	for i := 0; i < len(path); i++ {
		hash ^= uint32(path[i])
		hash *= 16777619
	}
	return &c.pathLocks[hash%pathLockShardCount]
}
