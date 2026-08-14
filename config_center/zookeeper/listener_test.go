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
	"github.com/dubbogo/go-zookeeper/zk"

	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

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
	cache.setWatch(path, configWatchState{watcher: &zk.Watcher{}, auto: true})
	pathLock := cache.pathLock(path)
	pathLock.Lock()
	watchStateDone := make(chan struct{})
	go func() {
		l.WatchStateChanged(path, nil)
		close(watchStateDone)
	}()
	require.Eventually(t, func() bool {
		_, ok := l.eventGeneration.Load(path)
		return ok
	}, time.Second, time.Millisecond)

	cache.reset()
	pathLock.Unlock()
	<-watchStateDone
	l.WatchStateChanged(path, &zk.Watcher{})
	l.DataChange(remoting.Event{Path: path, Action: remoting.EventTypeUpdate, Content: "old"})

	_, ok := cache.getFresh(path)
	if ok {
		t.Fatal("event started before reset should not repopulate cache")
	}
	_, watchState := cache.snapshot(path)
	if watchState.watcher != nil {
		t.Fatal("event started before reset should not reactivate watch state")
	}
}

func TestAddListenerPromotesAutoWatch(t *testing.T) {
	cache := newConfigCache(time.Minute)
	path := "/dubbo/config/group/app"
	watcher := &zk.Watcher{}
	cache.setWatch(path, configWatchState{watcher: watcher, auto: true})
	l := &CacheListener{cache: &cache}
	rec := &recListener{}

	require.NotPanics(t, func() {
		l.AddListener(path, rec)
	})

	_, watchState := cache.snapshot(path)
	require.Same(t, watcher, watchState.watcher)
	require.False(t, watchState.auto)
	require.Zero(t, cache.autoWatchCount)
	listeners, ok := l.keyListeners.Load(path)
	require.True(t, ok)
	_, ok = listeners.(map[config_center.ConfigurationListener]struct{})[rec]
	require.True(t, ok)
}

func TestAddListenerRegistersBusinessWatchAtAutoWatchLimit(t *testing.T) {
	cluster, client, _, err := gxzookeeper.NewMockZookeeperClient("business-watch-limit", 5*time.Second)
	if err != nil {
		t.Skipf("skip mock zk setup: %v", err)
	}
	defer cluster.Stop()

	cache := newConfigCache(time.Minute)
	for i := range maxAutoWatches {
		require.True(t, cache.setWatch(fmt.Sprintf("/auto/%d", i), configWatchState{
			watcher: &zk.Watcher{},
			auto:    true,
		}))
	}
	zkListener := remotingzookeeper.NewZkEventListener(client)
	defer zkListener.Close()
	l := newCacheListener("/dubbo/config", zkListener, &cache)
	path := "/dubbo/config/group/app"
	rec := &recListener{}

	l.AddListener(path, rec)

	_, watchState := cache.snapshot(path)
	require.NotNil(t, watchState.watcher)
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
	cache.setWatch(path, configWatchState{watcher: &zk.Watcher{}, auto: true})
	l := &CacheListener{cache: &cache}

	require.True(t, l.WatchStateChanged(path, nil))
	_, watchState := cache.snapshot(path)
	require.Nil(t, watchState.watcher)
	require.True(t, watchState.pending)
	require.True(t, watchState.auto)
	require.Zero(t, cache.autoWatchCount)
	require.Equal(t, 1, cache.autoWatchReservations)

	replacement := &zk.Watcher{}
	require.True(t, l.WatchStateChanged(path, replacement))
	_, watchState = cache.snapshot(path)
	require.Same(t, replacement, watchState.watcher)
	require.True(t, watchState.auto)
	require.False(t, watchState.pending)
	require.Equal(t, 1, cache.autoWatchCount)
	require.Zero(t, cache.autoWatchReservations)
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
	watcher := &zk.Watcher{}
	cache.setWatch(key, configWatchState{watcher: watcher})
	l.keyListeners.Store(key, map[config_center.ConfigurationListener]struct{}{rec: {}})

	l.RemoveListener(key, rec)

	_, ok := l.keyListeners.Load(key)
	require.False(t, ok)
	_, watchState := cache.snapshot(key)
	require.Same(t, watcher, watchState.watcher)
	require.True(t, watchState.auto)
	require.Equal(t, 1, cache.autoWatchCount)
}

func TestCacheListenerRemoveListenerDropsWatchAtAutoLimit(t *testing.T) {
	cluster, client, _, err := gxzookeeper.NewMockZookeeperClient("remove-listener-watch-limit", 5*time.Second)
	if err != nil {
		t.Skipf("skip mock zk setup: %v", err)
	}
	defer cluster.Stop()

	cache := newConfigCache(time.Minute)
	for i := range maxAutoWatches {
		require.True(t, cache.setWatch(fmt.Sprintf("/auto/%d", i), configWatchState{
			watcher: &zk.Watcher{},
			auto:    true,
		}))
	}
	path := "/dubbo/config/group/app"
	_, _, watcher, err := client.Conn.ExistsW(path)
	require.NoError(t, err)
	require.True(t, cache.setWatch(path, configWatchState{watcher: watcher}))
	zkListener := remotingzookeeper.NewZkEventListener(client)
	defer zkListener.Close()
	l := newCacheListener("/dubbo/config", zkListener, &cache)
	rec := &recListener{}
	l.keyListeners.Store(path, map[config_center.ConfigurationListener]struct{}{rec: {}})

	l.RemoveListener(path, rec)

	_, ok := l.keyListeners.Load(path)
	require.False(t, ok)
	_, watchState := cache.snapshot(path)
	require.False(t, watchState.tracked())
	require.Equal(t, maxAutoWatches, cache.autoWatchCount)
	select {
	case event := <-watcher.EvtCh:
		require.ErrorIs(t, event.Err, zk.ErrWatcherRemoved)
	case <-time.After(time.Second):
		t.Fatal("watcher was not removed")
	}
}

func TestCacheListenerResidualEventDoesNotRenewWatch(t *testing.T) {
	cache := newConfigCache(time.Minute)
	path := "/dubbo/config/group/app"
	l := &CacheListener{rootPath: "/dubbo/config", cache: &cache}

	require.False(t, l.WatchStateChanged(path, nil))
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
