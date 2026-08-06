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

package store

import (
	"encoding/gob"
	"os"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/hashicorp/golang-lru"
)

type CacheManager struct {
	name         string        // The name of the cache manager
	cacheFile    string        // The file path where the cache is stored
	dumpInterval time.Duration // The duration after which the cache dump
	stop         chan struct{} // Channel used to stop the cache expiration routine
	done         chan struct{} // Channel closed after the cache expiration routine stops
	cache        *lru.Cache    // The LRU cache implementation
	lock         sync.Mutex
	stopOnce     sync.Once
}

type Item struct {
	Key   string
	Value any
}

// NewCacheManager creates a new CacheManager instance.
// It initializes the cache manager with the provided parameters and starts a routine for cache dumping.
func NewCacheManager(name, cacheFile string, dumpInterval time.Duration, maxCacheSize int, enableDump bool) (*CacheManager, error) {
	cm := &CacheManager{
		name:         name,
		cacheFile:    cacheFile,
		dumpInterval: dumpInterval,
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
	}
	cache, err := lru.New(maxCacheSize)
	if err != nil {
		return nil, err
	}
	cm.cache = cache

	// Check if the cache file exists and load the cache if it does
	if _, err := os.Stat(cacheFile); err == nil {
		if err = cm.loadCache(); err != nil {
			logger.Warnf("[Registry][ServiceDiscovery] failed to load the cache file=%s, err=%v", cm.cacheFile, err)
		}
	}

	if enableDump {
		cm.runDumpTask()
	} else {
		close(cm.done)
	}

	return cm, nil
}

// Get retrieves the value associated with the given key from the cache.
func (cm *CacheManager) Get(key string) (any, bool) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	return cm.cache.Get(key)
}

// Set sets the value associated with the given key in the cache.
func (cm *CacheManager) Set(key string, value any) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	cm.cache.Add(key, value)
}

// Delete removes the value associated with the given key from the cache.
func (cm *CacheManager) Delete(key string) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	cm.cache.Remove(key)
}

// GetAll returns all the key-value pairs in the cache.
func (cm *CacheManager) GetAll() map[string]any {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	return cm.getAllLocked()
}

func (cm *CacheManager) getAllLocked() map[string]any {
	keys := cm.cache.Keys()

	result := make(map[string]any, len(keys))
	for _, k := range keys {
		key, ok := k.(string)
		if !ok {
			continue
		}
		if value, ok := cm.cache.Get(k); ok {
			result[key] = value
		}
	}

	return result
}

func (cm *CacheManager) len() int {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	return cm.cache.Len()
}

// loadCache loads the cache from the cache file.
func (cm *CacheManager) loadCache() error {
	cf, err := os.Open(cm.cacheFile)
	if err != nil {
		return err
	}

	decoder := gob.NewDecoder(cf)
	for {
		var it Item
		err = decoder.Decode(&it)
		if err != nil {
			if err.Error() == "EOF" {
				break // Reached end of file
			}
			return err
		}
		// Add the loaded keys to the front of the LRU list
		cm.Set(it.Key, it.Value)
	}

	return cf.Close()
}

// dumpCache dumps the cache to the cache file.
func (cm *CacheManager) dumpCache() error {
	items := cm.GetAll()

	file, err := os.Create(cm.cacheFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	for k, v := range items {
		gob.Register(v)
		err = encoder.Encode(&Item{
			Key:   k,
			Value: v,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (cm *CacheManager) runDumpTask() {
	go func() {
		ticker := time.NewTicker(cm.dumpInterval)
		defer func() {
			ticker.Stop()
			close(cm.done)
		}()
		for {
			select {
			case <-ticker.C:
				// Dump the cache to the file
				if err := cm.dumpCache(); err != nil {
					// Handle error
					logger.Warnf("[Registry][ServiceDiscovery] failed to dump cache, err=%v", err)
				} else {
					logger.Infof("[Registry][ServiceDiscovery] dumping [%s] caches, latest entries=%d", cm.name, cm.len())
				}
			case <-cm.stop:
				return
			}
		}
	}()
}

func (cm *CacheManager) StopDump() {
	cm.stopOnce.Do(func() {
		close(cm.stop)
	})
	<-cm.done
}

// destroy stops the cache dump routine, clears the cache and removes the cache file.
func (cm *CacheManager) destroy() {
	cm.StopDump() // Stop the cache dump routine

	cm.lock.Lock()
	cm.cache.Purge() // Clear the cache
	cm.lock.Unlock()

	// Delete the cache file if it exists
	if _, err := os.Stat(cm.cacheFile); err == nil {
		if err := os.Remove(cm.cacheFile); err == nil {
			logger.Infof("[Registry][ServiceDiscovery] the cacheFile [%s] was cleared", cm.cacheFile)
		}
	}
}
