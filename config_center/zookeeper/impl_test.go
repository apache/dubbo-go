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
	"encoding/base64"
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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	remotingzookeeper "dubbo.apache.org/dubbo-go/v3/remoting/zookeeper"
)

type channelConfigListener struct {
	events chan *config_center.ConfigChangeEvent
}

func (l *channelConfigListener) Process(event *config_center.ConfigChangeEvent) {
	l.events <- event
}

func TestBuildPath(t *testing.T) {
	tests := []struct {
		root     string
		sub      string
		expected string
	}{
		{root: "/dubbo/config", sub: "group/key", expected: "/dubbo/config/group/key"},
		{root: "/dubbo/config/", sub: "/group/key/", expected: "/dubbo/config/group/key"},
		{root: "dubbo/config", sub: "group/key", expected: "/dubbo/config/group/key"},
	}
	for _, tt := range tests {
		if got := buildPath(tt.root, tt.sub); got != tt.expected {
			t.Fatalf("buildPath(%q,%q) = %q, want %q", tt.root, tt.sub, got, tt.expected)
		}
	}
}

func TestGetPath(t *testing.T) {
	cfg := &zookeeperDynamicConfiguration{rootPath: "/root"}

	if got := cfg.getPath("k", "g"); got != "/root/g/k" {
		t.Fatalf("getPath with group returned %q", got)
	}
	if got := cfg.getPath("", "g"); got != "/root/g" {
		t.Fatalf("getPath empty key returned %q", got)
	}
	if got := cfg.getPath("k", ""); got != "/root/"+config_center.DefaultGroup+"/k" {
		t.Fatalf("getPath default group returned %q", got)
	}
}

func TestPublishAndRemoveConfigWithMockZk(t *testing.T) {
	cluster, client, _, err := gxzookeeper.NewMockZookeeperClient("test", 5e9)
	if err != nil {
		t.Skipf("skip mock zk setup: %v", err)
	}
	defer cluster.Stop()

	cfg := &zookeeperDynamicConfiguration{
		rootPath: "/dubbo/config",
		client:   client,
		done:     make(chan struct{}),
		url:      mustURL(t, "registry://127.0.0.1:2181"),
	}

	err = cfg.PublishConfig("k", "g", "v1")
	require.NoError(t, err)

	content, _, err := client.GetContent("/dubbo/config/g/k")
	require.NoError(t, err)
	require.Equal(t, "v1", string(content))

	// update existing node path
	err = cfg.PublishConfig("k", "g", "v2")
	require.NoError(t, err)
	content, _, err = client.GetContent("/dubbo/config/g/k")
	require.NoError(t, err)
	require.Equal(t, "v2", string(content))

	// remove
	err = cfg.RemoveConfig("k", "g")
	require.NoError(t, err)
	_, _, err = client.GetContent("/dubbo/config/g/k")
	require.ErrorIs(t, err, zk.ErrNoNode)
}

func TestGetPropertiesWithMockZk(t *testing.T) {
	cluster, client, _, err := gxzookeeper.NewMockZookeeperClient("test2", 5e9)
	if err != nil {
		t.Skipf("skip mock zk setup: %v", err)
	}
	defer cluster.Stop()

	cfg := &zookeeperDynamicConfiguration{
		rootPath: "/dubbo/config",
		client:   client,
		done:     make(chan struct{}),
		url:      mustURL(t, "registry://127.0.0.1:2181"),
	}

	require.NoError(t, cfg.PublishConfig("file.properties", "grp", "val"))

	val, err := cfg.GetProperties("file.properties", config_center.WithGroup("grp"))
	require.NoError(t, err)
	require.Equal(t, "val", val)

	// non-existing returns empty string and nil error
	empty, err := cfg.GetProperties("missing", config_center.WithGroup("grp"))
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestLoadPropertiesRegistersWatchOnlyWhenInactive(t *testing.T) {
	cluster, client, events, err := gxzookeeper.NewMockZookeeperClient("watch-selection", 5*time.Second)
	if err != nil {
		t.Skipf("skip mock zk setup: %v", err)
	}
	defer cluster.Stop()

	cfg := &zookeeperDynamicConfiguration{
		rootPath: "/dubbo/config",
		client:   client,
		url:      mustURL(t, "registry://127.0.0.1:2181"),
		cache:    newConfigCache(time.Minute),
	}
	activePath := cfg.getPath("active", "group")
	inactivePath := cfg.getPath("inactive", "group")
	require.NoError(t, cfg.PublishConfig("active", "group", "v1"))
	require.NoError(t, cfg.PublishConfig("inactive", "group", "v1"))

	waitForEvent := func(path string, timeout time.Duration) bool {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		for {
			select {
			case event := <-events:
				if event.Path == path && event.Type == zk.EventNodeDataChanged {
					return true
				}
			case <-timer.C:
				return false
			}
		}
	}

	activeWatcher := &zk.Watcher{}
	_, watcher, err := cfg.loadProperties(activePath, activeWatcher, false)
	require.NoError(t, err)
	require.Same(t, activeWatcher, watcher)
	_, stat, err := client.GetContent(activePath)
	require.NoError(t, err)
	_, err = client.SetContent(activePath, []byte("v2"), stat.Version)
	require.NoError(t, err)
	require.False(t, waitForEvent(activePath, time.Second))

	_, watcher, err = cfg.loadProperties(inactivePath, nil, true)
	require.NoError(t, err)
	require.NotNil(t, watcher)
	_, stat, err = client.GetContent(inactivePath)
	require.NoError(t, err)
	_, err = client.SetContent(inactivePath, []byte("v2"), stat.Version)
	require.NoError(t, err)
	require.True(t, waitForEvent(inactivePath, time.Second))
}

func TestListenerUsesGroupOption(t *testing.T) {
	cluster, client, _, err := gxzookeeper.NewMockZookeeperClient("listener-group", 5*time.Second)
	if err != nil {
		t.Skipf("skip mock zk setup: %v", err)
	}
	defer cluster.Stop()

	zkListener := remotingzookeeper.NewZkEventListener(client)
	defer zkListener.Close()
	cfg := &zookeeperDynamicConfiguration{
		rootPath: "/dubbo/config",
		client:   client,
		url:      mustURL(t, "registry://127.0.0.1:2181"),
		cache:    newConfigCache(time.Minute),
		listener: zkListener,
	}
	cfg.cacheListener = newCacheListener(cfg.rootPath, zkListener, &cfg.cache)
	key := "app.properties"
	group := "custom"
	path := cfg.getPropertiesPath(key, config_center.WithGroup(group))
	rec := &recListener{}

	cfg.AddListener(key, rec, config_center.WithGroup(group))
	_, ok := cfg.cacheListener.keyListeners.Load(path)
	require.True(t, ok)

	cfg.RemoveListener(key, rec, config_center.WithGroup(group))
	_, ok = cfg.cacheListener.keyListeners.Load(path)
	require.False(t, ok)
}

func TestGetPropertiesFallsBackToTTLAtAutoWatchLimit(t *testing.T) {
	cluster, client, events, err := gxzookeeper.NewMockZookeeperClient("watch-limit", 5*time.Second)
	if err != nil {
		t.Skipf("skip mock zk setup: %v", err)
	}
	defer cluster.Stop()

	cfg := &zookeeperDynamicConfiguration{
		rootPath: "/dubbo/config",
		client:   client,
		url:      mustURL(t, "registry://127.0.0.1:2181"),
		cache:    newConfigCache(time.Minute),
	}
	for i := range maxAutoWatches {
		require.True(t, cfg.cache.setWatch(fmt.Sprintf("/watch/%d", i), configWatchState{
			watcher: &zk.Watcher{},
			auto:    true,
		}))
	}

	require.NoError(t, cfg.PublishConfig("fallback", "group", "v1"))
	value, err := cfg.GetProperties("fallback", config_center.WithGroup("group"))
	require.NoError(t, err)
	require.Equal(t, "v1", value)

	path := cfg.getPath("fallback", "group")
	_, watchState := cfg.cache.snapshot(path)
	require.False(t, watchState.tracked())
	require.Equal(t, maxAutoWatches, cfg.cache.autoWatchCount)
	require.Zero(t, cfg.cache.autoWatchReservations)
	_, stat, err := client.GetContent(path)
	require.NoError(t, err)
	_, err = client.SetContent(path, []byte("v2"), stat.Version)
	require.NoError(t, err)

	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case event := <-events:
			if event.Path == path && event.Type == zk.EventNodeDataChanged {
				t.Fatal("TTL fallback should not register an auto watch")
			}
		case <-timer.C:
			value, err = cfg.GetProperties("fallback", config_center.WithGroup("group"))
			require.NoError(t, err)
			require.Equal(t, "v1", value)
			return
		}
	}
}

func TestGetPropertiesCacheUpdatedByWatch(t *testing.T) {
	cluster, client, _, err := gxzookeeper.NewMockZookeeperClient("cache-watch", 5*time.Second)
	if err != nil {
		t.Skipf("skip mock zk setup: %v", err)
	}
	defer cluster.Stop()
	go (&gxzookeeper.DefaultHandler{}).HandleZkEvent(client)

	cfg := &zookeeperDynamicConfiguration{
		rootPath: "/dubbo/config",
		client:   client,
		done:     make(chan struct{}),
		url:      mustURL(t, "registry://127.0.0.1:2181"),
		cache:    newConfigCache(time.Minute),
	}
	cfg.listener = remotingzookeeper.NewZkEventListener(client)
	cfg.cacheListener = newCacheListener(cfg.rootPath, cfg.listener, &cfg.cache)
	cfg.listener.ListenConfigurationEvent(cfg.rootPath, cfg.cacheListener)
	defer cfg.listener.Close()

	require.NoError(t, cfg.PublishConfig("file.properties", "grp", "v1"))
	value, err := cfg.GetProperties("file.properties", config_center.WithGroup("grp"))
	require.NoError(t, err)
	require.Equal(t, "v1", value)

	// ListenConfigurationEvent registers asynchronously; wait before triggering the watch.
	time.Sleep(50 * time.Millisecond)
	_, stat, err := client.GetContent("/dubbo/config/grp/file.properties")
	require.NoError(t, err)
	_, err = client.SetContent("/dubbo/config/grp/file.properties", []byte("v2"), stat.Version)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		value, getErr := cfg.GetProperties("file.properties", config_center.WithGroup("grp"))
		return getErr == nil && value == "v2"
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, cfg.RemoveConfig("file.properties", "grp"))
	require.Eventually(t, func() bool {
		entry, ok := cfg.cache.getFresh("/dubbo/config/grp/file.properties")
		return ok && !entry.exists
	}, time.Second, 10*time.Millisecond)
}

func TestGetPropertiesDecodesCachedBase64(t *testing.T) {
	cfg := &zookeeperDynamicConfiguration{
		rootPath:      "/dubbo/config",
		url:           mustURL(t, "registry://127.0.0.1:2181"),
		cache:         newConfigCache(time.Minute),
		base64Enabled: true,
	}
	path := cfg.getPropertiesPath("key", config_center.WithGroup("group"))
	cfg.cache.store(path, configCacheEntry{
		content: base64.StdEncoding.EncodeToString([]byte("value")),
		exists:  true,
	})

	value, err := cfg.GetProperties("key", config_center.WithGroup("group"))
	require.NoError(t, err)
	require.Equal(t, "value", value)
}

func TestRestartCallBackResetsCache(t *testing.T) {
	cluster, client, _, err := gxzookeeper.NewMockZookeeperClient("restart-watch-reset", 5*time.Second)
	if err != nil {
		t.Skipf("skip mock zk setup: %v", err)
	}
	defer cluster.Stop()

	cfg := &zookeeperDynamicConfiguration{cache: newConfigCache(time.Minute), client: client}
	path := "/dubbo/config/group/key"
	pendingPath := "/dubbo/config/group/pending"
	_, _, watcher, err := client.Conn.ExistsW(path)
	require.NoError(t, err)
	cfg.cache.store(path, configCacheEntry{content: "value", exists: true})
	cfg.cache.setWatch(path, configWatchState{watcher: watcher, auto: true})
	cfg.cache.setWatch(pendingPath, configWatchState{auto: true, pending: true})

	require.True(t, cfg.RestartCallBack())
	_, ok := cfg.cache.getFresh(path)
	require.False(t, ok)
	_, watchState := cfg.cache.snapshot(path)
	require.Nil(t, watchState.watcher)
	_, pendingWatchState := cfg.cache.snapshot(pendingPath)
	require.False(t, pendingWatchState.tracked())
	require.Zero(t, cfg.cache.autoWatchCount)
	require.Zero(t, cfg.cache.autoWatchReservations)
	select {
	case event := <-watcher.EvtCh:
		require.ErrorIs(t, event.Err, zk.ErrWatcherRemoved)
	case <-time.After(time.Second):
		t.Fatal("watcher was not removed")
	}
}

func TestRestartCallBackRestoresBusinessListener(t *testing.T) {
	cluster, client, _, err := gxzookeeper.NewMockZookeeperClient("restart-business-watch", 5*time.Second)
	if err != nil {
		t.Skipf("skip mock zk setup: %v", err)
	}
	defer cluster.Stop()
	go (&gxzookeeper.DefaultHandler{}).HandleZkEvent(client)

	cfg := &zookeeperDynamicConfiguration{
		rootPath: "/dubbo/config",
		client:   client,
		url:      mustURL(t, "registry://127.0.0.1:2181"),
		cache:    newConfigCache(time.Minute),
	}
	cfg.listener = remotingzookeeper.NewZkEventListener(client)
	cfg.cacheListener = newCacheListener(cfg.rootPath, cfg.listener, &cfg.cache)
	cfg.listener.ListenConfigurationEvent(cfg.rootPath, cfg.cacheListener)
	defer cfg.listener.Close()

	key := "app.properties"
	group := "group"
	path := cfg.getPropertiesPath(key, config_center.WithGroup(group))
	recorder := &channelConfigListener{events: make(chan *config_center.ConfigChangeEvent, 2)}
	require.NoError(t, cfg.PublishConfig(key, group, "v1"))
	time.Sleep(50 * time.Millisecond)
	cfg.AddListener(key, recorder, config_center.WithGroup(group))
	_, previousWatch := cfg.cache.snapshot(path)
	require.NotNil(t, previousWatch.watcher)
	require.False(t, previousWatch.auto)

	require.True(t, cfg.RestartCallBack())
	_, restoredWatch := cfg.cache.snapshot(path)
	require.NotNil(t, restoredWatch.watcher)
	require.NotSame(t, previousWatch.watcher, restoredWatch.watcher)
	require.False(t, restoredWatch.auto)

	for _, value := range []string{"v2", "v3"} {
		_, stat, getErr := client.GetContent(path)
		require.NoError(t, getErr)
		_, setErr := client.SetContent(path, []byte(value), stat.Version)
		require.NoError(t, setErr)

		select {
		case event := <-recorder.events:
			require.Equal(t, key, event.Key)
			require.Equal(t, value, event.Value)
		case <-time.After(time.Second):
			t.Fatalf("listener did not receive configuration value %q", value)
		}
	}
}

func TestRestartCallBackRestoresBusinessListenerWhenCacheDisabled(t *testing.T) {
	cluster, client, _, err := gxzookeeper.NewMockZookeeperClient("restart-business-watch-cache-disabled", 5*time.Second)
	if err != nil {
		t.Skipf("skip mock zk setup: %v", err)
	}
	defer cluster.Stop()
	go (&gxzookeeper.DefaultHandler{}).HandleZkEvent(client)

	cfg := &zookeeperDynamicConfiguration{
		rootPath: "/dubbo/config",
		client:   client,
		url:      mustURL(t, "registry://127.0.0.1:2181"),
		cache:    newConfigCache(0),
	}
	cfg.listener = remotingzookeeper.NewZkEventListener(client)
	cfg.cacheListener = newCacheListener(cfg.rootPath, cfg.listener, &cfg.cache)
	defer cfg.listener.Close()

	key := "app.properties"
	group := "group"
	path := cfg.getPropertiesPath(key, config_center.WithGroup(group))
	recorder := &channelConfigListener{events: make(chan *config_center.ConfigChangeEvent, 2)}
	require.NoError(t, cfg.PublishConfig(key, group, "v1"))
	time.Sleep(50 * time.Millisecond)
	// A reconnect preserves business listeners but may invalidate their watches.
	cfg.cacheListener.keyListeners.Store(path, map[config_center.ConfigurationListener]struct{}{recorder: {}})

	cfg.listener.ListenConfigurationEvent(cfg.rootPath, cfg.cacheListener)
	require.True(t, cfg.RestartCallBack())

	for _, value := range []string{"v2", "v3"} {
		_, stat, getErr := client.GetContent(path)
		require.NoError(t, getErr)
		_, setErr := client.SetContent(path, []byte(value), stat.Version)
		require.NoError(t, setErr)

		select {
		case event := <-recorder.events:
			require.Equal(t, key, event.Key)
			require.Equal(t, value, event.Value)
		case <-time.After(time.Second):
			t.Fatalf("listener did not receive configuration value %q", value)
		}
	}
}

func mustURL(t *testing.T, raw string) *common.URL {
	t.Helper()
	u, err := common.NewURL(raw)
	require.NoError(t, err)
	return u
}
