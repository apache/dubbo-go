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
	"testing"
	"time"
)

import (
	"github.com/go-zookeeper/zk"

	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	remotingzookeeper "dubbo.apache.org/dubbo-go/v3/remoting/zookeeper"
)

type recListener struct {
	events []*config_center.ConfigChangeEvent
}

func (r *recListener) Process(e *config_center.ConfigChangeEvent) {
	r.events = append(r.events, e)
}

func TestCacheListenerDataChange(t *testing.T) {
	l := &CacheListener{rootPath: "/dubbo/config"}
	path := "/dubbo/config/group/app"
	rec := &recListener{}
	l.keyListeners.Store(path, map[config_center.ConfigurationListener]struct{}{rec: {}})

	ok := l.DataChange(remoting.Event{Path: path, Action: remoting.EventTypeUpdate, Content: "val"})
	if !ok {
		t.Fatalf("expected listeners to be notified")
	}
	if len(rec.events) != 1 || rec.events[0].Value != "val" || rec.events[0].ConfigType != remoting.EventTypeUpdate {
		t.Fatalf("unexpected events %+v", rec.events)
	}
}

func TestCacheListenerDataChangeEmptyContent(t *testing.T) {
	cache := newConfigCache(time.Minute)
	l := &CacheListener{rootPath: "/dubbo/config", cache: &cache}
	path := "/dubbo/config/group/app"
	rec := &recListener{}
	l.keyListeners.Store(path, map[config_center.ConfigurationListener]struct{}{rec: {}})

	ok := l.DataChange(remoting.Event{Path: path, Action: remoting.EventTypeAdd})
	if !ok {
		t.Fatalf("expected listeners to be notified")
	}
	if len(rec.events) != 1 || rec.events[0].ConfigType != remoting.EventTypeAdd {
		t.Fatalf("unexpected events %+v", rec.events)
	}
	entry, ok := cache.getFresh(path)
	if !ok || !entry.exists || entry.content != "" {
		t.Fatalf("empty configuration should be cached as existing: %+v", entry)
	}

	l.DataChange(remoting.Event{Path: path, Action: remoting.EventTypeDel})
	if len(rec.events) != 2 || rec.events[1].ConfigType != remoting.EventTypeDel {
		t.Fatalf("unexpected events %+v", rec.events)
	}
	entry, ok = cache.getFresh(path)
	if !ok || entry.exists {
		t.Fatalf("deleted configuration should be cached as missing: %+v", entry)
	}
}

func TestCacheListenerIgnoresEventAcrossReset(t *testing.T) {
	cache := newConfigCache(time.Minute)
	l := &CacheListener{rootPath: "/dubbo/config", cache: &cache}
	path := "/dubbo/config/group/app"
	cache.setWatch(path, configWatchState{registered: true, auto: true, sessionID: 1})
	require.True(t, l.WatchStateChanged(path))
	cache.reset(2)
	l.WatchRegistered(path, make(chan zk.Event, 1), 1, 1)
	l.DataChange(remoting.Event{Path: path, Action: remoting.EventTypeUpdate, Content: "old"})

	_, ok := cache.getFresh(path)
	if ok {
		t.Fatal("event started before reset should not repopulate cache")
	}
	_, watchState := cache.snapshot(path)
	if watchState.tracked() {
		t.Fatal("event started before reset should not reactivate watch state")
	}
}

func TestAddListenerPromotesAutoWatch(t *testing.T) {
	cache := newConfigCache(time.Minute)
	path := "/dubbo/config/group/app"
	cache.setWatch(path, configWatchState{registered: true, auto: true, sessionID: 1})
	l := &CacheListener{cache: &cache}
	rec := &recListener{}

	require.NotPanics(t, func() {
		l.AddListener(path, rec)
	})

	_, watchState := cache.snapshot(path)
	require.True(t, watchState.registered)
	require.False(t, watchState.auto)
	require.Zero(t, cache.autoWatchCount)
	listeners, ok := l.keyListeners.Load(path)
	require.True(t, ok)
	_, ok = listeners.(map[config_center.ConfigurationListener]struct{})[rec]
	require.True(t, ok)
}

func TestAddListenerReactivatesRetiredWatch(t *testing.T) {
	cache := newConfigCache(time.Minute)
	path := "/dubbo/config/group/app"
	cache.setWatch(path, configWatchState{
		registered: true,
		retired:    true,
		sessionID:  1,
	})
	l := &CacheListener{cache: &cache}
	rec := &recListener{}

	l.AddListener(path, rec)

	_, watchState := cache.snapshot(path)
	require.True(t, watchState.registered)
	require.False(t, watchState.retired)
	require.False(t, watchState.auto)
	_, ok := l.keyListeners.Load(path)
	require.True(t, ok)
}

func TestWatchStateChangedUsesCurrentBusinessOwnership(t *testing.T) {
	cache := newConfigCache(time.Minute)
	path := "/dubbo/config/group/app"
	require.True(t, cache.setWatch(path, configWatchState{
		registered: true,
		auto:       true,
		sessionID:  1,
	}))
	l := &CacheListener{cache: &cache}
	rec := &recListener{}

	l.AddListener(path, rec)
	require.True(t, l.WatchStateChanged(path))

	_, watchState := cache.snapshot(path)
	require.True(t, watchState.pending)
	require.False(t, watchState.auto)
	require.Zero(t, cache.autoWatchReservations)
}

func TestAddListenerRegistersBusinessWatchAtAutoWatchLimit(t *testing.T) {
	client, _ := newZookeeperTestClient(t, "business-watch-limit")
	root := newTestRoot(t, client)

	cache := newConfigCache(time.Minute)
	for i := range maxAutoWatches {
		require.True(t, cache.setWatch(fmt.Sprintf("/auto/%d", i), configWatchState{
			registered: true,
			auto:       true,
			sessionID:  1,
		}))
	}
	zkListener := remotingzookeeper.NewZkEventListener(client)
	defer zkListener.Close()
	l := newCacheListener(root, zkListener, &cache)
	path := root + "/group/app"
	rec := &recListener{}

	l.AddListener(path, rec)

	_, watchState := cache.snapshot(path)
	require.True(t, watchState.registered)
	require.False(t, watchState.auto)
	require.Equal(t, maxAutoWatches, cache.autoWatchCount)
	require.Zero(t, cache.autoWatchReservations)
	listeners, ok := l.keyListeners.Load(path)
	require.True(t, ok)
	_, ok = listeners.(map[config_center.ConfigurationListener]struct{})[rec]
	require.True(t, ok)
}

func TestCacheListenerPreservesAutoWatchOwnershipOnRenewal(t *testing.T) {
	cache := newConfigCache(time.Minute)
	path := "/dubbo/config/group/app"
	cache.setWatch(path, configWatchState{registered: true, auto: true, sessionID: 1})
	l := &CacheListener{cache: &cache}

	require.True(t, l.WatchStateChanged(path))
	_, watchState := cache.snapshot(path)
	require.False(t, watchState.registered)
	require.True(t, watchState.pending)
	require.True(t, watchState.auto)
	require.Zero(t, cache.autoWatchCount)
	require.Equal(t, 1, cache.autoWatchReservations)

	require.True(t, l.WatchRegistered(path, make(chan zk.Event, 1), 1, 1))
	_, watchState = cache.snapshot(path)
	require.True(t, watchState.registered)
	require.True(t, watchState.auto)
	require.False(t, watchState.pending)
	require.Equal(t, 1, cache.autoWatchCount)
	require.Zero(t, cache.autoWatchReservations)
}

func TestCacheListenerRetriesInvalidatedWatchInNewSession(t *testing.T) {
	client, _ := newZookeeperTestClient(t, "retry-invalidated-business-watch")
	root := newTestRoot(t, client)
	zkListener := remotingzookeeper.NewZkEventListener(client)
	defer zkListener.Close()

	cache := newConfigCache(time.Minute)
	currentSessionID := client.Conn.SessionID()
	cache.reset(currentSessionID)
	path := root + "/group/app"
	previousSessionID := currentSessionID - 1
	require.True(t, cache.setWatch(path, configWatchState{
		pending:   true,
		sessionID: previousSessionID,
	}))
	l := newCacheListener(root, zkListener, &cache)
	l.keyListeners.Store(path, map[config_center.ConfigurationListener]struct{}{&recListener{}: {}})
	l.eventGeneration.Store(path, watchEventState{
		generation: cache.generation,
		sessionID:  previousSessionID,
	})
	events := make(chan zk.Event, 1)
	events <- zk.Event{Type: zk.EventNotWatching}
	close(events)

	require.False(t, l.WatchRegistered(path, events, previousSessionID, currentSessionID))
	_, watchState := cache.snapshot(path)
	require.True(t, watchState.registered)
	require.False(t, watchState.pending)
	require.False(t, watchState.auto)
	require.Equal(t, currentSessionID, watchState.sessionID)
}

func TestCacheListenerPathToKeyGroup(t *testing.T) {
	l := &CacheListener{rootPath: "/dubbo/config"}
	key, group := l.pathToKeyGroup("/dubbo/config/g/app")
	if key != "app" || group != "g" {
		t.Fatalf("unexpected key/group %s %s", key, group)
	}
}

func TestCacheListenerRemoveListener(t *testing.T) {
	cache := newConfigCache(time.Minute)
	l := &CacheListener{cache: &cache}
	key := "k"
	rec := &recListener{}
	cache.setWatch(key, configWatchState{registered: true, sessionID: 1})
	l.keyListeners.Store(key, map[config_center.ConfigurationListener]struct{}{rec: {}})

	l.RemoveListener(key, rec)

	_, ok := l.keyListeners.Load(key)
	require.False(t, ok)
	_, watchState := cache.snapshot(key)
	require.True(t, watchState.registered)
	require.True(t, watchState.auto)
	require.Equal(t, 1, cache.autoWatchCount)
}

func TestCacheListenerRemoveListenerRetiresWatchAtAutoLimit(t *testing.T) {
	cache := newConfigCache(time.Minute)
	for i := range maxAutoWatches {
		require.True(t, cache.setWatch(fmt.Sprintf("/auto/%d", i), configWatchState{
			registered: true,
			auto:       true,
			sessionID:  1,
		}))
	}
	path := "/dubbo/config/group/app"
	require.True(t, cache.setWatch(path, configWatchState{
		registered: true,
		sessionID:  1,
	}))
	l := newCacheListener("/dubbo/config", nil, &cache)
	rec := &recListener{}
	l.keyListeners.Store(path, map[config_center.ConfigurationListener]struct{}{rec: {}})

	l.RemoveListener(path, rec)

	_, ok := l.keyListeners.Load(path)
	require.False(t, ok)
	_, watchState := cache.snapshot(path)
	require.True(t, watchState.registered)
	require.True(t, watchState.retired)
	require.False(t, watchState.auto)
	require.Equal(t, maxAutoWatches, cache.autoWatchCount)

	require.False(t, l.WatchStateChanged(path))
	_, watchState = cache.snapshot(path)
	require.False(t, watchState.tracked())
	require.False(t, l.DataChange(remoting.Event{Path: path, Action: remoting.EventTypeUpdate, Content: "value"}))
	entry, ok := cache.getFresh(path)
	require.True(t, ok)
	require.Equal(t, "value", entry.content)
}

func TestCacheListenerResidualEventDoesNotRenewWatch(t *testing.T) {
	cache := newConfigCache(time.Minute)
	path := "/dubbo/config/group/app"
	l := &CacheListener{rootPath: "/dubbo/config", cache: &cache}

	require.False(t, l.WatchStateChanged(path))
	_, watchState := cache.snapshot(path)
	require.False(t, watchState.tracked())
	require.False(t, l.DataChange(remoting.Event{Path: path, Action: remoting.EventTypeUpdate, Content: "value"}))

	entry, ok := cache.getFresh(path)
	require.True(t, ok)
	require.True(t, entry.exists)
	require.Equal(t, "value", entry.content)
	_, ok = l.eventGeneration.Load(path)
	require.False(t, ok)
}
