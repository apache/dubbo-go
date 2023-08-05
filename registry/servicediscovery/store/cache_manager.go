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
	"reflect"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

type CacheManager struct {
	name         string        // The name of the cache manager
	cacheFile    string        // The file path where the cache is stored
	cacheExpired time.Duration // The duration after which the cache expires
	stop         chan struct{} // Channel used to stop the cache expiration routine
	lruCache     *LruCache     // The LRU cache implementation
}

// NewCacheManager creates a new CacheManager instance.
// It initializes the cache manager with the provided parameters and starts a routine for cache expiration.
func NewCacheManager(name, cacheFile string, cacheExpired time.Duration, maxCacheSize int) (*CacheManager, error) {
	cm := &CacheManager{
		name:         name,
		cacheFile:    cacheFile,
		cacheExpired: cacheExpired,
		stop:         make(chan struct{}, 1),
		lruCache:     NewLruCache(maxCacheSize),
	}

	// Check if the cache file exists and load the cache if it does
	if _, err := os.Stat(cacheFile); err == nil {
		if err := cm.loadCache(); err != nil {
			logger.Warnf("Failed to load the cache file:[%s].", cm.cacheFile)
			os.Remove(cm.cacheFile)
			return nil, err
		}
	}

	go func(stop chan struct{}) {
		ticker := time.NewTicker(cm.cacheExpired)
		for {
			select {
			case <-ticker.C:
				// Dump the cache to the file
				if err := cm.dumpCache(); err != nil {
					// Handle error
					logger.Warnf("Failed to dump cache,the err is %v", err)
				} else {
					logger.Infof("Dumping [%s] caches, latest entries %d", cm.name, len(cm.lruCache.cache))
				}
			case <-stop:
				return
			}
		}

	}(cm.stop)

	return cm, nil
}

// Get retrieves the value associated with the given key from the cache.
func (cm *CacheManager) Get(key string) (interface{}, error) {
	return cm.lruCache.Get(key)
}

// Set sets the value associated with the given key in the cache.
func (cm *CacheManager) Set(key string, value interface{}) {
	cm.lruCache.Set(key, value)
}

// Remove removes the value associated with the given key from the cache.
func (cm *CacheManager) Remove(key string) {
	cm.lruCache.Remove(key)
}

// GetAll returns all the key-value pairs in the cache.
func (cm *CacheManager) GetAll() map[string]interface{} {
	return cm.lruCache.GetAll()
}

// loadCache loads the cache from the cache file.
func (cm *CacheManager) loadCache() error {
	file, err := os.Open(cm.cacheFile)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	for {
		var item CacheItem
		err := decoder.Decode(&item)
		if err != nil {
			if err.Error() == "EOF" {
				break // Reached end of file
			}
			return err
		}
		// Add the loaded keys to the front of the LRU list
		cm.Set(item.Key, item.Value)
	}

	return nil
}

// dumpCache dumps the cache to the cache file.
func (cm *CacheManager) dumpCache() error {
	items := make([]*CacheItem, 0, cm.lruCache.maxCacheSize)
	for _, v := range cm.lruCache.Export() {
		items = append(items, v)
	}

	file, err := os.Create(cm.cacheFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	for _, item := range items {
		err := encoder.Encode(item)
		if err != nil {
			return err
		}
	}

	return nil
}

// destroy stops the cache expiration routine, clears the cache and removes the cache file.
func (cm *CacheManager) destroy() {
	cm.stop <- struct{}{} // Stop the cache expiration routine
	cm.lruCache.Clear()   // Clear the cache
	cm.dumpCache()        // Dump the cache to the file

	// Remove the cache file if it exists
	if _, err := os.Stat(cm.cacheFile); err == nil {
		if err := os.Remove(cm.cacheFile); err == nil {
			logger.Infof("The cacheFile [%s] was cleared", cm.cacheFile)
		}
	}
}
