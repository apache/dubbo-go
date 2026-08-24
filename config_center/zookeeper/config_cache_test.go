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
	"github.com/go-zookeeper/zk"

	"github.com/stretchr/testify/require"
)

func newTestWatchRegistration(sessionID int64) watchRegistration {
	return watchRegistration{
		events:          make(chan zk.Event, 1),
		beforeSessionID: sessionID,
		afterSessionID:  sessionID,
	}
}

func TestConfigCacheLoadAndExpiry(t *testing.T) {
	cache := newConfigCache(20 * time.Millisecond)
	var loads atomic.Int32
	loader := func(registerWatch bool) (configCacheEntry, watchRegistration, error) {
		count := loads.Add(1)
		if registerWatch {
			return configCacheEntry{content: string(rune('0' + count)), exists: true},
				newTestWatchRegistration(1), nil
		}
		return configCacheEntry{content: string(rune('0' + count)), exists: true}, watchRegistration{}, nil
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

	for i := range 4096 {
		locks[cache.pathLock(fmt.Sprintf("/path/%d", i))] = struct{}{}
	}

	require.Same(t, pathLock, cache.pathLock("/path"))
	require.LessOrEqual(t, len(locks), pathLockShardCount)
}

func TestConfigCacheBoundsEntriesUnderKeyChurn(t *testing.T) {
	cache := newConfigCache(time.Minute)

	for i := range 4096 {
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
	for i := range maxCacheEntries {
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

	for i := range 4096 {
		path := fmt.Sprintf("/path/%d", i)
		wg.Go(func() {
			_, err := cache.load(path, func(registerWatch bool) (configCacheEntry, watchRegistration, error) {
				if registerWatch {
					return configCacheEntry{exists: true}, newTestWatchRegistration(1), nil
				}
				return configCacheEntry{exists: true}, watchRegistration{}, nil
			})
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	require.Equal(t, maxCacheEntries, cache.entries.Len())
}

func TestConfigCacheConcurrentAutoWatchRegistrationsRemainBounded(t *testing.T) {
	cache := newConfigCache(time.Minute)
	var registrations atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan error, 4096)

	for i := range 4096 {
		path := fmt.Sprintf("/watch/%d", i)
		wg.Go(func() {
			_, err := cache.load(path, func(registerWatch bool) (configCacheEntry, watchRegistration, error) {
				if registerWatch {
					registrations.Add(1)
					return configCacheEntry{exists: true}, newTestWatchRegistration(1), nil
				}
				return configCacheEntry{exists: true}, watchRegistration{}, nil
			})
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	require.Equal(t, int32(maxAutoWatches), registrations.Load())
	require.Equal(t, maxAutoWatches, cache.autoWatchCount)
	require.Zero(t, cache.autoWatchReservations)
}

func TestConfigCacheWatchUpdateWinsOverLoad(t *testing.T) {
	cache := newConfigCache(time.Minute)
	loadStarted := make(chan struct{})
	releaseLoad := make(chan struct{})
	loadDone := make(chan struct{})
	go func() {
		defer close(loadDone)
		_, _ = cache.load("/path", func(registerWatch bool) (configCacheEntry, watchRegistration, error) {
			close(loadStarted)
			<-releaseLoad
			if registerWatch {
				return configCacheEntry{content: "old", exists: true}, newTestWatchRegistration(1), nil
			}
			return configCacheEntry{content: "old", exists: true}, watchRegistration{}, nil
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
	loadStarted := make(chan struct{})
	releaseLoad := make(chan struct{})
	type loadResult struct {
		entry configCacheEntry
		err   error
	}
	result := make(chan loadResult, 1)
	var loads atomic.Int32

	go func() {
		entry, err := cache.load("/path", func(bool) (configCacheEntry, watchRegistration, error) {
			if loads.Add(1) == 1 {
				close(loadStarted)
				<-releaseLoad
				return configCacheEntry{content: "old", exists: true}, newTestWatchRegistration(1), nil
			}
			return configCacheEntry{content: "new", exists: true}, newTestWatchRegistration(2), nil
		})
		result <- loadResult{entry: entry, err: err}
	}()

	<-loadStarted
	cache.reset(2)
	_, ok := cache.getFresh("/path")
	require.False(t, ok)
	_, watchState := cache.snapshot("/path")
	require.True(t, watchState.pending)
	require.Equal(t, 1, cache.autoWatchReservations)
	close(releaseLoad)

	load := <-result
	require.NoError(t, load.err)
	require.Equal(t, "new", load.entry.content)
	require.Equal(t, int32(2), loads.Load())
	entry, ok := cache.getFresh("/path")
	require.True(t, ok)
	require.Equal(t, "new", entry.content)
	_, watchState = cache.snapshot("/path")
	require.True(t, watchState.registered)
	require.Equal(t, int64(2), watchState.sessionID)
}

func TestConfigCacheKeepsInFlightWatchInSameSession(t *testing.T) {
	cache := newConfigCache(time.Minute)
	loadStarted := make(chan struct{})
	releaseLoad := make(chan struct{})
	result := make(chan error, 1)
	decisions := make(chan bool, 2)
	var loads atomic.Int32
	var registrations atomic.Int32

	go func() {
		_, err := cache.load("/path", func(registerWatch bool) (configCacheEntry, watchRegistration, error) {
			decisions <- registerWatch
			if loads.Add(1) == 1 {
				registrations.Add(1)
				close(loadStarted)
				<-releaseLoad
				return configCacheEntry{content: "old", exists: true}, newTestWatchRegistration(1), nil
			}
			return configCacheEntry{content: "new", exists: true}, watchRegistration{}, nil
		})
		result <- err
	}()

	<-loadStarted
	cache.reset(1)
	close(releaseLoad)

	require.NoError(t, <-result)
	require.Equal(t, int32(2), loads.Load())
	require.Equal(t, int32(1), registrations.Load())
	require.True(t, <-decisions)
	require.False(t, <-decisions)
	entry, ok := cache.getFresh("/path")
	require.True(t, ok)
	require.Equal(t, "new", entry.content)
	_, watchState := cache.snapshot("/path")
	require.True(t, watchState.registered)
	require.Equal(t, int64(1), watchState.sessionID)
}

func TestConfigCacheResetPreservesOnlyCurrentSessionWatches(t *testing.T) {
	cache := newConfigCache(time.Minute)
	cache.store("/entry", configCacheEntry{content: "value", exists: true})
	require.True(t, cache.setWatch("/active", configWatchState{
		registered: true,
		auto:       true,
		sessionID:  2,
	}))
	require.True(t, cache.setWatch("/retired", configWatchState{
		registered: true,
		retired:    true,
		sessionID:  2,
	}))
	require.True(t, cache.setWatch("/stale", configWatchState{
		registered: true,
		auto:       true,
		sessionID:  1,
	}))
	require.True(t, cache.setWatch("/pending", configWatchState{
		pending:   true,
		auto:      true,
		sessionID: 1,
	}))

	cache.reset(2)

	_, ok := cache.getFresh("/entry")
	require.False(t, ok)
	_, active := cache.snapshot("/active")
	require.True(t, active.registered)
	_, retired := cache.snapshot("/retired")
	require.True(t, retired.registered)
	require.True(t, retired.retired)
	_, stale := cache.snapshot("/stale")
	require.False(t, stale.tracked())
	_, pending := cache.snapshot("/pending")
	require.True(t, pending.pending)
	require.Equal(t, 1, cache.autoWatchCount)
	require.Equal(t, 1, cache.autoWatchReservations)
}

func TestWatchRegistrationResolveAcrossSession(t *testing.T) {
	t.Run("current session watch remains active", func(t *testing.T) {
		registration := watchRegistration{
			events:          make(chan zk.Event, 1),
			beforeSessionID: 1,
			afterSessionID:  2,
		}

		sessionID, active := registration.resolve()
		require.True(t, active)
		require.Equal(t, int64(2), sessionID)
	})

	t.Run("invalidated watch is discarded", func(t *testing.T) {
		events := make(chan zk.Event, 1)
		events <- zk.Event{Type: zk.EventNotWatching}
		close(events)
		registration := watchRegistration{
			events:          events,
			beforeSessionID: 1,
			afterSessionID:  2,
		}

		sessionID, active := registration.resolve()
		require.False(t, active)
		require.Zero(t, sessionID)
	})
}

func TestConfigCacheResetDiscardsInFlightBusinessWatch(t *testing.T) {
	cache := newConfigCache(time.Minute)
	registerStarted := make(chan struct{})
	releaseRegister := make(chan struct{})
	result := make(chan error, 1)
	var registrations atomic.Int32

	go func() {
		result <- cache.ensureBusinessWatch("/path", func() (watchRegistration, error) {
			if registrations.Add(1) == 1 {
				close(registerStarted)
				<-releaseRegister
				return newTestWatchRegistration(1), nil
			}
			return newTestWatchRegistration(2), nil
		})
	}()

	<-registerStarted
	cache.reset(2)
	_, pendingState := cache.snapshot("/path")
	require.True(t, pendingState.pending)
	close(releaseRegister)

	require.NoError(t, <-result)
	require.Equal(t, int32(2), registrations.Load())
	_, watchState := cache.snapshot("/path")
	require.True(t, watchState.registered)
	require.False(t, watchState.auto)
	require.Equal(t, int64(2), watchState.sessionID)
}

func TestConfigCacheBusinessWatchRegistrationRetryIsBounded(t *testing.T) {
	cache := newConfigCache(time.Minute)
	var registrations atomic.Int32
	invalidRegistration := func() (watchRegistration, error) {
		registrations.Add(1)
		events := make(chan zk.Event, 1)
		events <- zk.Event{Type: zk.EventNotWatching}
		close(events)
		return watchRegistration{
			events:          events,
			beforeSessionID: 1,
			afterSessionID:  2,
		}, nil
	}

	err := cache.ensureBusinessWatchWithRetry("/path", invalidRegistration, 1)
	require.ErrorIs(t, err, errWatchRegistrationStale)
	require.Equal(t, int32(2), registrations.Load())
	_, watchState := cache.snapshot("/path")
	require.False(t, watchState.tracked())
}

func TestConfigCacheLoadRejectsStaleRegistrationBeforeStoringEntry(t *testing.T) {
	cache := newConfigCache(time.Minute)
	var registrations atomic.Int32

	loader := func(registerWatch bool) (configCacheEntry, watchRegistration, error) {
		if !registerWatch {
			return configCacheEntry{content: "fallback", exists: true}, watchRegistration{}, nil
		}
		if registrations.Add(1) == 1 {
			events := make(chan zk.Event, 1)
			events <- zk.Event{Type: zk.EventNotWatching}
			close(events)
			return configCacheEntry{content: "stale", exists: true}, watchRegistration{
				events:          events,
				beforeSessionID: 1,
				afterSessionID:  2,
			}, nil
		}
		return configCacheEntry{content: "current", exists: true}, newTestWatchRegistration(2), nil
	}

	entry, err := cache.load("/path", loader)
	require.NoError(t, err)
	require.Equal(t, "current", entry.content)
	require.Equal(t, int32(2), registrations.Load())
	cached, ok := cache.getFresh("/path")
	require.True(t, ok)
	require.Equal(t, "current", cached.content)
}

func TestConfigCacheLoadFallsBackAfterRepeatedStaleRegistrations(t *testing.T) {
	cache := newConfigCache(time.Minute)
	var registrations atomic.Int32
	var fallbacks atomic.Int32

	loader := func(registerWatch bool) (configCacheEntry, watchRegistration, error) {
		if !registerWatch {
			fallbacks.Add(1)
			return configCacheEntry{content: "fallback", exists: true}, watchRegistration{}, nil
		}
		registrations.Add(1)
		events := make(chan zk.Event, 1)
		events <- zk.Event{Type: zk.EventNotWatching}
		close(events)
		return configCacheEntry{content: "stale", exists: true}, watchRegistration{
			events:          events,
			beforeSessionID: 1,
			afterSessionID:  2,
		}, nil
	}

	entry, err := cache.load("/path", loader)
	require.NoError(t, err)
	require.Equal(t, "fallback", entry.content)
	require.Equal(t, int32(2), registrations.Load())
	require.Equal(t, int32(1), fallbacks.Load())
}
