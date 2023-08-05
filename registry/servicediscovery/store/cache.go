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
	"container/list"
	"sync"
)

import (
	"github.com/pkg/errors"
)

type LruCache struct {
	cache        map[string]*list.Element
	cacheMutex   sync.RWMutex
	lruList      *list.List
	maxCacheSize int
}

type CacheItem struct {
	Key   string
	Value interface{}
}

func NewLruCache(maxCacheSize int) *LruCache {
	return &LruCache{
		cache:        make(map[string]*list.Element),
		lruList:      list.New(),
		maxCacheSize: maxCacheSize,
	}
}

func (lc *LruCache) Get(key string) (interface{}, error) {
	lc.cacheMutex.RLock()
	defer lc.cacheMutex.RUnlock()

	if val, ok := lc.cache[key]; ok {
		// Move the accessed Key to the front of the list
		lc.lruList.MoveToFront(val)
		return val.Value.(*CacheItem).Value, nil
	}
	return nil, errors.New("Key not found")
}

func (lc *LruCache) Set(key string, value interface{}) {
	lc.cacheMutex.Lock()
	defer lc.cacheMutex.Unlock()

	if val, ok := lc.cache[key]; ok {
		// If the Key exists, move it to the front of the list and update the Value
		lc.lruList.MoveToFront(val)
		val.Value.(*CacheItem).Value = value
	} else {
		// If the Key doesn't exist, insert a new Key at the front of the list and store it in the cache
		element := lc.lruList.PushFront(key)

		item := &CacheItem{
			Key:   key,
			Value: value,
		}

		element.Value = item
		lc.cache[key] = element
	}

	// Check if the cache size exceeds the maximum limit
	if lc.lruList.Len() > lc.maxCacheSize {
		// If it exceeds the limit, remove the least recently used Key from the back of the list and delete it from the cache
		element := lc.lruList.Back()
		if element != nil {
			keyToRemove := element.Value.(*CacheItem).Key
			delete(lc.cache, keyToRemove)
			lc.lruList.Remove(element)
		}
	}
}

func (lc *LruCache) Remove(key string) {
	lc.cacheMutex.Lock()
	defer lc.cacheMutex.Unlock()

	if val, ok := lc.cache[key]; ok {
		// If the Key exists, remove it from the list and cache
		lc.lruList.Remove(val)
		delete(lc.cache, key)
	}
}

func (lc *LruCache) GetAll() map[string]interface{} {
	lc.cacheMutex.RLock()
	defer lc.cacheMutex.RUnlock()

	m := make(map[string]interface{}, len(lc.cache))
	for k, v := range lc.cache {
		m[k] = v.Value.(*CacheItem).Value
	}
	return m
}

func (lc *LruCache) Export() []*CacheItem {
	lc.cacheMutex.RLock()
	defer lc.cacheMutex.RUnlock()

	items := make([]*CacheItem, 0, lc.maxCacheSize)
	for _, element := range lc.cache {
		items = append(items, element.Value.(*CacheItem))
	}
	return items
}

func (lc *LruCache) Clear() {
	lc.cacheMutex.Lock()
	defer lc.cacheMutex.Unlock()

	lc.lruList.Init()
	lc.cache = make(map[string]*list.Element)
}
