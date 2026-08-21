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
	"encoding/hex"
	"os"
	"testing"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	"github.com/go-zookeeper/zk"

	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

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

func mustURL(t *testing.T, raw string) *common.URL {
	t.Helper()
	u, err := common.NewURL(raw)
	require.NoError(t, err)
	return u
}
