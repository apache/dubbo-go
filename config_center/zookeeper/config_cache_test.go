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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/stretchr/testify/require"
)

func TestConfigCacheLoadAndExpiry(t *testing.T) {
	cache := newConfigCache(20 * time.Millisecond)
	var loads atomic.Int32
	loader := func(*zk.Watcher) (configCacheEntry, *zk.Watcher, error) {
		count := loads.Add(1)
		return configCacheEntry{content: string(rune('0' + count)), exists: true}, nil, nil
	}

	first, err := cache.load("/path", loader)
	require.NoError(t, err)
	second, err := cache.load("/path", loader)
	require.NoError(t, err)
	require.Equal(t, first.content, second.content)
	require.Equal(t, int32(1), loads.Load())

	require.Eventually(t, func() bool {
		entry, loadErr := cache.load("/path", loader)
		return loadErr == nil && entry.content == "2"
	}, time.Second, 5*time.Millisecond)
	require.Equal(t, int32(2), loads.Load())
}

func TestConfigCacheUsesFixedPathLockShards(t *testing.T) {
	cache := newConfigCache(time.Minute)
	locks := make(map[*sync.Mutex]struct{})
	pathLock := cache.pathLock("/path")

	for i := 0; i < 4096; i++ {
		locks[cache.pathLock(fmt.Sprintf("/path/%d", i))] = struct{}{}
	}

	require.Equal(t, pathLockShardCount, len(cache.pathLocks))
	require.Same(t, pathLock, cache.pathLock("/path"))
	require.LessOrEqual(t, len(locks), pathLockShardCount)
}

func TestConfigCacheBoundsEntriesUnderKeyChurn(t *testing.T) {
	cache := newConfigCache(time.Minute)

	for i := 0; i < 4096; i++ {
		cache.store(fmt.Sprintf("/path/%d", i), configCacheEntry{
			content: fmt.Sprintf("value-%d", i),
			exists:  true,
		})
	}

	require.Equal(t, maxCacheEntries, cache.entries.Len())
}

func TestConfigCacheStoresMissingEntry(t *testing.T) {
	cache := newConfigCache(time.Minute)
	cache.store("/missing", configCacheEntry{exists: false})

	require.Equal(t, 1, cache.entries.Len())
	entry, ok := cache.getFresh("/missing")
	require.True(t, ok)
	require.False(t, entry.exists)
}

func TestConfigCacheEvictsLeastRecentlyUsed(t *testing.T) {
	cache := newConfigCache(time.Minute)
	for i := 0; i < maxCacheEntries; i++ {
		cache.store(fmt.Sprintf("/path/%d", i), configCacheEntry{exists: true})
	}

	_, ok := cache.getFresh("/path/0")
	require.True(t, ok)
	cache.store("/new", configCacheEntry{exists: true})

	_, ok = cache.getFresh("/path/1")
	require.False(t, ok)
	_, ok = cache.getFresh("/path/0")
	require.True(t, ok)
}

func TestConfigCacheRemovesExpiredEntryOnAccess(t *testing.T) {
	cache := newConfigCache(10 * time.Millisecond)
	cache.store("/path", configCacheEntry{exists: true})
	require.Equal(t, 1, cache.entries.Len())
	time.Sleep(20 * time.Millisecond)
	require.Equal(t, 1, cache.entries.Len())

	_, ok := cache.getFresh("/path")
	require.False(t, ok)
	require.Zero(t, cache.entries.Len())
}

func TestConfigCacheConcurrentLoadsRemainBounded(t *testing.T) {
	cache := newConfigCache(time.Minute)
	var wg sync.WaitGroup
	errs := make(chan error, 4096)

	for i := 0; i < 4096; i++ {
		path := fmt.Sprintf("/path/%d", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.load(path, func(*zk.Watcher) (configCacheEntry, *zk.Watcher, error) {
				return configCacheEntry{exists: true}, nil, nil
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	require.Equal(t, maxCacheEntries, cache.entries.Len())
}

func TestConfigCacheWatchUpdateWinsOverLoad(t *testing.T) {
	cache := newConfigCache(time.Minute)
	loadStarted := make(chan struct{})
	releaseLoad := make(chan struct{})
	loadDone := make(chan struct{})
	go func() {
		defer close(loadDone)
		_, _ = cache.load("/path", func(watcher *zk.Watcher) (configCacheEntry, *zk.Watcher, error) {
			close(loadStarted)
			<-releaseLoad
			return configCacheEntry{content: "old", exists: true}, watcher, nil
		})
	}()

	<-loadStarted
	updateDone := make(chan struct{})
	go func() {
		defer close(updateDone)
		cache.store("/path", configCacheEntry{content: "new", exists: true})
	}()
	close(releaseLoad)
	<-loadDone
	<-updateDone

	entry, ok := cache.getFresh("/path")
	require.True(t, ok)
	require.Equal(t, "new", entry.content)
}

func TestConfigCacheResetDiscardsInFlightLoad(t *testing.T) {
	cache := newConfigCache(time.Minute)
	cache.setWatch("/path", configWatchState{watcher: &zk.Watcher{}, auto: true})
	loadStarted := make(chan struct{})
	releaseLoad := make(chan struct{})
	result := make(chan configCacheEntry, 1)
	var loads atomic.Int32

	go func() {
		entry, _ := cache.load("/path", func(watcher *zk.Watcher) (configCacheEntry, *zk.Watcher, error) {
			if loads.Add(1) == 1 {
				close(loadStarted)
				<-releaseLoad
				return configCacheEntry{content: "old", exists: true}, watcher, nil
			}
			return configCacheEntry{content: "new", exists: true}, watcher, nil
		})
		result <- entry
	}()

	<-loadStarted
	cache.reset()
	_, ok := cache.getFresh("/path")
	require.False(t, ok)
	_, watchState := cache.snapshot("/path")
	require.Nil(t, watchState.watcher)
	close(releaseLoad)

	require.Equal(t, "new", (<-result).content)
	require.Equal(t, int32(2), loads.Load())
	entry, ok := cache.getFresh("/path")
	require.True(t, ok)
	require.Equal(t, "new", entry.content)
}
