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
	"errors"
	"testing"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"

	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

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
	require.True(t, errors.Is(err, zk.ErrNoNode))
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
	require.Equal(t, "", empty)
}

func mustURL(t *testing.T, raw string) *common.URL {
	t.Helper()
	u, err := common.NewURL(raw)
	require.NoError(t, err)
	return u
}
