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
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	"github.com/go-zookeeper/zk"

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

// zkAddrEnvKey mirrors the environment variable gost's
// NewZookeeperClientFromEnv reads to locate a ZooKeeper server (see
// database/kv/zk/client.go in dubbogo/gost); it isn't exported there, so the
// name is duplicated here.
const zkAddrEnvKey = "ZK_ADDR"

// failOrSkipZkUnavailable reports that a ZooKeeper connection could not be
// established. Whether that's a failure or a skip is keyed off ZK_ADDR
// rather than a generic "am I in CI" heuristic: if it's set, whoever is
// running this test explicitly pointed it at a real ZooKeeper (as our CI
// workflow does), so an unreachable server is a genuine regression, not
// something to silently skip past; when it's unset - the common case on a
// developer machine that hasn't set one up - it skips instead so local runs
// aren't blocked.
func failOrSkipZkUnavailable(t *testing.T, err error) {
	t.Helper()
	if addr := os.Getenv(zkAddrEnvKey); addr != "" {
		t.Fatalf("%s=%q was set but zookeeper is unavailable: %v", zkAddrEnvKey, addr, err)
	}
	t.Skipf("skip zk setup: %v", err)
}

func newZookeeperTestClient(t *testing.T, name string) (*gxzookeeper.ZookeeperClient, <-chan zk.Event) {
	t.Helper()
	client, events, err := gxzookeeper.NewZookeeperClientFromEnv(name, 5*time.Second)
	if err != nil {
		failOrSkipZkUnavailable(t, err)
		return nil, nil
	}
	t.Cleanup(func() { client.Close() })
	return client, events
}

// newTestRoot returns a randomly named zookeeper root path unique to the
// calling test, and registers a t.Cleanup that recursively removes it via
// client once the test finishes, so runs stay isolated on a shared
// ZooKeeper server instead of colliding on a fixed root.
func newTestRoot(t *testing.T, client *gxzookeeper.ZookeeperClient) string {
	t.Helper()
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("newTestRoot: %v", err)
	}
	root := "/dubbo-test-" + hex.EncodeToString(buf)
	t.Cleanup(func() { cleanupZkPath(client, root) })
	return root
}

// cleanupZkPath best-effort recursively removes zkPath and all of its
// descendants via client.
func cleanupZkPath(client *gxzookeeper.ZookeeperClient, zkPath string) {
	if client == nil {
		return
	}
	children, err := client.GetChildren(zkPath)
	if err == nil {
		for _, c := range children {
			cleanupZkPath(client, zkPath+"/"+c)
		}
	}
	_ = client.Delete(zkPath)
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

func TestPublishAndRemoveConfigWithZk(t *testing.T) {
	client, _, err := gxzookeeper.NewZookeeperClientFromEnv("test", 5e9)
	if err != nil {
		failOrSkipZkUnavailable(t, err)
		return
	}
	// Registered before newTestRoot's own t.Cleanup below so that, since
	// t.Cleanup runs its callbacks in LIFO order (and always after any
	// defer in this function), the zk path cleanup runs first, against a
	// still-open client, before Close() runs.
	t.Cleanup(func() { client.Close() })

	root := newTestRoot(t, client)
	cfg := &zookeeperDynamicConfiguration{
		rootPath: root,
		client:   client,
		done:     make(chan struct{}),
		url:      mustURL(t, "registry://127.0.0.1:2181"),
	}

	err = cfg.PublishConfig("k", "g", "v1")
	require.NoError(t, err)

	content, _, err := client.GetContent(root + "/g/k")
	require.NoError(t, err)
	require.Equal(t, "v1", string(content))

	// update existing node path
	err = cfg.PublishConfig("k", "g", "v2")
	require.NoError(t, err)
	content, _, err = client.GetContent(root + "/g/k")
	require.NoError(t, err)
	require.Equal(t, "v2", string(content))

	// remove
	err = cfg.RemoveConfig("k", "g")
	require.NoError(t, err)
	_, _, err = client.GetContent(root + "/g/k")
	require.ErrorIs(t, err, zk.ErrNoNode)
}

func TestGetPropertiesWithZk(t *testing.T) {
	client, _, err := gxzookeeper.NewZookeeperClientFromEnv("test2", 5e9)
	if err != nil {
		failOrSkipZkUnavailable(t, err)
		return
	}
	// Registered before newTestRoot's own t.Cleanup below so that, since
	// t.Cleanup runs its callbacks in LIFO order (and always after any
	// defer in this function), the zk path cleanup runs first, against a
	// still-open client, before Close() runs.
	t.Cleanup(func() { client.Close() })

	root := newTestRoot(t, client)
	cfg := &zookeeperDynamicConfiguration{
		rootPath: root,
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
	client, events := newZookeeperTestClient(t, "watch-selection")
	root := newTestRoot(t, client)

	cfg := &zookeeperDynamicConfiguration{
		rootPath: root,
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

	_, registration, err := cfg.loadProperties(activePath, false)
	require.NoError(t, err)
	require.Nil(t, registration.events)
	_, stat, err := client.GetContent(activePath)
	require.NoError(t, err)
	_, err = client.SetContent(activePath, []byte("v2"), stat.Version)
	require.NoError(t, err)
	require.False(t, waitForEvent(activePath, time.Second))

	_, registration, err = cfg.loadProperties(inactivePath, true)
	require.NoError(t, err)
	require.NotNil(t, registration.events)
	_, stat, err = client.GetContent(inactivePath)
	require.NoError(t, err)
	_, err = client.SetContent(inactivePath, []byte("v2"), stat.Version)
	require.NoError(t, err)
	require.True(t, waitForEvent(inactivePath, time.Second))
}

func TestListenerUsesGroupOption(t *testing.T) {
	client, _ := newZookeeperTestClient(t, "listener-group")
	root := newTestRoot(t, client)

	zkListener := remotingzookeeper.NewZkEventListener(client)
	defer zkListener.Close()
	cfg := &zookeeperDynamicConfiguration{
		rootPath: root,
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
	client, events := newZookeeperTestClient(t, "watch-limit")
	root := newTestRoot(t, client)

	cfg := &zookeeperDynamicConfiguration{
		rootPath: root,
		client:   client,
		url:      mustURL(t, "registry://127.0.0.1:2181"),
		cache:    newConfigCache(time.Minute),
	}
	for i := range maxAutoWatches {
		require.True(t, cfg.cache.setWatch(fmt.Sprintf("/watch/%d", i), configWatchState{
			registered: true,
			auto:       true,
			sessionID:  client.Conn.SessionID(),
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
	client, _ := newZookeeperTestClient(t, "cache-watch")
	go (&gxzookeeper.DefaultHandler{}).HandleZkEvent(client)
	root := newTestRoot(t, client)

	cfg := &zookeeperDynamicConfiguration{
		rootPath: root,
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
	watchPath := cfg.getPropertiesPath("file.properties", config_center.WithGroup("grp"))
	_, stat, err := client.GetContent(watchPath)
	require.NoError(t, err)
	_, err = client.SetContent(watchPath, []byte("v2"), stat.Version)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		value, getErr := cfg.GetProperties("file.properties", config_center.WithGroup("grp"))
		return getErr == nil && value == "v2"
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, cfg.RemoveConfig("file.properties", "grp"))
	require.Eventually(t, func() bool {
		entry, ok := cfg.cache.getFresh(watchPath)
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
	client, _ := newZookeeperTestClient(t, "restart-watch-reset")

	cfg := &zookeeperDynamicConfiguration{cache: newConfigCache(time.Minute), client: client}
	path := "/dubbo/config/group/key"
	pendingPath := "/dubbo/config/group/pending"
	_, _, _, err := client.Conn.ExistsW(path)
	require.NoError(t, err)
	cfg.cache.store(path, configCacheEntry{content: "value", exists: true})
	cfg.cache.setWatch(path, configWatchState{
		registered: true,
		auto:       true,
		sessionID:  client.Conn.SessionID(),
	})
	cfg.cache.setWatch(pendingPath, configWatchState{
		auto:      true,
		pending:   true,
		sessionID: client.Conn.SessionID(),
	})

	require.True(t, cfg.RestartCallBack())
	_, ok := cfg.cache.getFresh(path)
	require.False(t, ok)
	_, watchState := cfg.cache.snapshot(path)
	require.True(t, watchState.registered)
	_, pendingWatchState := cfg.cache.snapshot(pendingPath)
	require.True(t, pendingWatchState.pending)
	require.Equal(t, 1, cfg.cache.autoWatchCount)
	require.Equal(t, 1, cfg.cache.autoWatchReservations)
}

func TestRestartCallBackRestoresBusinessListener(t *testing.T) {
	client, _ := newZookeeperTestClient(t, "restart-business-watch")
	go (&gxzookeeper.DefaultHandler{}).HandleZkEvent(client)
	root := newTestRoot(t, client)

	cfg := &zookeeperDynamicConfiguration{
		rootPath: root,
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
	require.True(t, previousWatch.registered)
	require.False(t, previousWatch.auto)

	require.True(t, cfg.RestartCallBack())
	_, restoredWatch := cfg.cache.snapshot(path)
	require.True(t, restoredWatch.registered)
	require.Equal(t, previousWatch.sessionID, restoredWatch.sessionID)
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
	client, _ := newZookeeperTestClient(t, "restart-business-watch-cache-disabled")
	go (&gxzookeeper.DefaultHandler{}).HandleZkEvent(client)
	root := newTestRoot(t, client)

	cfg := &zookeeperDynamicConfiguration{
		rootPath: root,
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
	cfg.listener.ListenConfigurationEvent(cfg.rootPath, cfg.cacheListener)
	cfg.AddListener(key, recorder, config_center.WithGroup(group))
	_, watchState := cfg.cache.snapshot(path)
	require.True(t, watchState.registered)
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
