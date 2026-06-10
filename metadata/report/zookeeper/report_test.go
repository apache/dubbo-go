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
	"encoding/json"
	"strings"
	"testing"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

// --- Mock zkClient ---
// mockZkClient implements zkClient for testing.
type mockZkClient struct {
	data   map[string][]byte   // path -> value
	stats  map[string]*zk.Stat // path -> stat
	errors map[string]error    // path -> error for specific operations (optional override)
}

func newMockZkClient() *mockZkClient {
	return &mockZkClient{
		data:   make(map[string][]byte),
		stats:  make(map[string]*zk.Stat),
		errors: make(map[string]error),
	}
}

func (m *mockZkClient) GetContent(path string) ([]byte, *zk.Stat, error) {
	if err, ok := m.errors["GetContent:"+path]; ok {
		return nil, nil, err
	}
	v, ok := m.data[path]
	if !ok {
		return nil, nil, zk.ErrNoNode
	}
	return v, m.stats[path], nil
}

func (m *mockZkClient) SetContent(path string, data []byte, version int32) (*zk.Stat, error) {
	if err, ok := m.errors["SetContent:"+path]; ok {
		return nil, err
	}
	m.data[path] = data
	stat := &zk.Stat{Version: version + 1, Mtime: int64(version + 1)}
	m.stats[path] = stat
	return stat, nil
}

func (m *mockZkClient) CreateWithValue(path string, data []byte) error {
	if err, ok := m.errors["CreateWithValue:"+path]; ok {
		return err
	}
	if _, exists := m.data[path]; exists {
		return zk.ErrNodeExists
	}
	m.data[path] = data
	stat := &zk.Stat{Version: 0, Mtime: 0}
	m.stats[path] = stat
	return nil
}

func (m *mockZkClient) Delete(path string) error {
	if err, ok := m.errors["Delete:"+path]; ok {
		return err
	}
	if _, ok := m.data[path]; !ok {
		return zk.ErrNoNode
	}
	delete(m.data, path)
	delete(m.stats, path)
	return nil
}

func (m *mockZkClient) Exists(path string) (bool, *zk.Stat, error) {
	_, ok := m.data[path]
	if !ok {
		return false, nil, nil
	}
	return true, m.stats[path], nil
}

func (m *mockZkClient) Children(path string) ([]string, *zk.Stat, error) {
	if err, ok := m.errors["Children:"+path]; ok {
		return nil, nil, err
	}
	prefix := path + "/"
	seen := make(map[string]bool)
	for k := range m.data {
		if strings.HasPrefix(k, prefix) {
			child := k[len(prefix):]
			// only direct children
			if idx := strings.Index(child, "/"); idx >= 0 {
				child = child[:idx]
			}
			seen[child] = true
		}
	}
	if len(seen) == 0 {
		return nil, nil, zk.ErrNoNode
	}
	result := make([]string, 0, len(seen))
	for k := range seen {
		result = append(result, k)
	}
	return result, &zk.Stat{}, nil
}

func (m *mockZkClient) Get(path string) ([]byte, *zk.Stat, error) {
	v, ok := m.data[path]
	if !ok {
		return nil, nil, zk.ErrNoNode
	}
	return v, m.stats[path], nil
}

// --- Helper ---

func newTestReportWithMock() (*zookeeperMetadataReport, *mockZkClient) {
	mc := newMockZkClient()
	r := &zookeeperMetadataReport{
		client:  mc,
		rootDir: "/dubbo/",
	}
	return r, mc
}

// --- Tests ---

func TestPublishAndGetAppMetadata(t *testing.T) {
	r, mc := newTestReportWithMock()

	meta := &info.MetadataInfo{
		App:      "my-app",
		Revision: "r1",
		Services: map[string]*info.ServiceInfo{
			"com.example.Foo": {Name: "com.example.Foo", Protocol: "dubbo"},
		},
	}

	err := r.PublishAppMetadata("my-app", "r1", meta)
	require.NoError(t, err)

	// Verify data was stored
	data, err := json.Marshal(meta)
	require.NoError(t, err)
	assert.Equal(t, data, mc.data["/dubbo/my-app/r1"])

	// Get it back
	got, err := r.GetAppMetadata("my-app", "r1")
	require.NoError(t, err)
	assert.Equal(t, "my-app", got.App)
	assert.Equal(t, "r1", got.Revision)
	assert.Contains(t, got.Services, "com.example.Foo")

	// Get non-existent returns error
	_, err = r.GetAppMetadata("my-app", "nonexistent")
	require.Error(t, err)
}

func TestPublishAppMetadata_Update(t *testing.T) {
	r, _ := newTestReportWithMock()

	meta := &info.MetadataInfo{App: "my-app", Revision: "r1"}
	err := r.PublishAppMetadata("my-app", "r1", meta)
	require.NoError(t, err)

	// Update existing node
	meta.Revision = "r2"
	err = r.PublishAppMetadata("my-app", "r1", meta)
	require.NoError(t, err)

	got, err := r.GetAppMetadata("my-app", "r1")
	require.NoError(t, err)
	assert.Equal(t, "r2", got.Revision)
}

func TestUnPublishAppMetadata(t *testing.T) {
	r, mc := newTestReportWithMock()

	meta := &info.MetadataInfo{App: "my-app", Revision: "r1"}
	err := r.PublishAppMetadata("my-app", "r1", meta)
	require.NoError(t, err)

	err = r.UnPublishAppMetadata("my-app", "r1")
	require.NoError(t, err)

	// Data should be gone
	_, ok := mc.data["/dubbo/my-app/r1"]
	assert.False(t, ok)

	// Get should fail
	_, err = r.GetAppMetadata("my-app", "r1")
	require.Error(t, err)
}

func TestUnPublishAppMetadata_NonExistent(t *testing.T) {
	r, _ := newTestReportWithMock()

	// Deleting a non-existent node should not error (idempotent per interface contract)
	err := r.UnPublishAppMetadata("my-app", "nonexistent")
	require.NoError(t, err)
}

func TestListAppRevisions(t *testing.T) {
	r, mc := newTestReportWithMock()

	// No revisions for unknown app
	revisions, err := r.ListAppRevisions("unknown-app")
	require.NoError(t, err)
	assert.Empty(t, revisions)

	// Manually set up revisions with different lastUpdatedTime in content
	mc.data["/dubbo/my-app/r1"] = []byte(`{"app":"my-app","revision":"r1","lastUpdatedTime":1000}`)
	mc.data["/dubbo/my-app/r2"] = []byte(`{"app":"my-app","revision":"r2","lastUpdatedTime":3000}`)
	mc.data["/dubbo/my-app/r3"] = []byte(`{"app":"my-app","revision":"r3","lastUpdatedTime":2000}`)

	revisions, err = r.ListAppRevisions("my-app")
	require.NoError(t, err)
	require.Len(t, revisions, 3)

	names := make(map[string]int64)
	for _, rev := range revisions {
		names[rev.Revision] = rev.ModifyTime
	}
	assert.Equal(t, int64(1000), names["r1"])
	assert.Equal(t, int64(3000), names["r2"])
	assert.Equal(t, int64(2000), names["r3"])
}

func TestRegisterServiceAppMapping_NewKey(t *testing.T) {
	r, mc := newTestReportWithMock()

	err := r.RegisterServiceAppMapping("com.example.Foo", "mapping", "app1")
	require.NoError(t, err)
	assert.Equal(t, []byte("app1"), mc.data["/dubbo/mapping/com.example.Foo"])
}

func TestRegisterServiceAppMapping_Append(t *testing.T) {
	r, mc := newTestReportWithMock()

	mc.data["/dubbo/mapping/com.example.Foo"] = []byte("app1")
	mc.stats["/dubbo/mapping/com.example.Foo"] = &zk.Stat{Version: 0}

	err := r.RegisterServiceAppMapping("com.example.Foo", "mapping", "app2")
	require.NoError(t, err)
	assert.Equal(t, []byte("app1,app2"), mc.data["/dubbo/mapping/com.example.Foo"])
}

func TestRegisterServiceAppMapping_Duplicate(t *testing.T) {
	r, mc := newTestReportWithMock()

	mc.data["/dubbo/mapping/com.example.Foo"] = []byte("app1,app2")
	mc.stats["/dubbo/mapping/com.example.Foo"] = &zk.Stat{Version: 0}

	err := r.RegisterServiceAppMapping("com.example.Foo", "mapping", "app1")
	require.NoError(t, err)
	// Should not change — app1 is already in the value
	assert.Equal(t, []byte("app1,app2"), mc.data["/dubbo/mapping/com.example.Foo"])
}

func TestGetServiceAppMapping(t *testing.T) {
	r, mc := newTestReportWithMock()
	r.cacheListener = NewCacheListener("/dubbo/", nil)

	mc.data["/dubbo/mapping/com.example.Foo"] = []byte("app1,app2")
	mc.stats["/dubbo/mapping/com.example.Foo"] = &zk.Stat{}

	set, err := r.GetServiceAppMapping("com.example.Foo", "mapping", nil)
	require.NoError(t, err)
	assert.True(t, set.Contains("app1"))
	assert.True(t, set.Contains("app2"))
}

// --- Existing pure-logic tests ---

func TestMetadataInfoSerialization(t *testing.T) {
	original := &info.MetadataInfo{
		App:      "test-app",
		Revision: "1.0.0",
		Services: map[string]*info.ServiceInfo{
			"com.example.TestService": {
				Name: "com.example.TestService", Protocol: "dubbo",
			},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored info.MetadataInfo
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)
	assert.Equal(t, original.App, restored.App)
	assert.Equal(t, original.Revision, restored.Revision)

	// Invalid JSON
	err = json.Unmarshal([]byte(`{invalid}`), &restored)
	require.Error(t, err)
}

func TestRegisterServiceAppMappingValueMerge(t *testing.T) {
	tests := []struct {
		oldValue, newValue, expected string
		wantChanged                  bool
	}{
		{"app1", "app2", "app1,app2", true},
		{"app1,app2", "app1", "app1,app2", false},
		// empty old value must not produce a leading comma (was ",app1")
		{"", "app1", "app1", true},
		// substring must not be mistaken for membership (was wrongly treated as present)
		{"app1-extra", "app1", "app1-extra,app1", true},
	}
	for _, tt := range tests {
		result, changed := report.MergeServiceAppMapping(tt.oldValue, tt.newValue)
		assert.Equal(t, tt.expected, result)
		assert.Equal(t, tt.wantChanged, changed)
	}
}

func TestCreateMetadataReportURLParsing(t *testing.T) {
	tests := []struct {
		group, expectedRootDir string
	}{
		{"", "/dubbo/"},
		{"custom", "/custom/"},
		{"/custom", "/custom/"},
		{"/", "/"},
	}
	for _, tt := range tests {
		url := common.NewURLWithOptions(
			common.WithProtocol("zookeeper"),
			common.WithLocation("127.0.0.1:2181"),
		)
		if tt.group != "" {
			url.SetParam(constant.MetadataReportGroupKey, tt.group)
		}
		rootDir := url.GetParam(constant.MetadataReportGroupKey, "dubbo")
		if len(rootDir) > 0 && rootDir[0] != '/' {
			rootDir = "/" + rootDir
		}
		if rootDir != "/" {
			rootDir = rootDir + "/"
		}
		assert.Equal(t, tt.expectedRootDir, rootDir)
	}
}

func TestRemoveServiceAppMappingListener(t *testing.T) {
	r := &zookeeperMetadataReport{
		rootDir:       "/dubbo/",
		cacheListener: NewCacheListener("/dubbo/", nil),
	}
	err := r.RemoveServiceAppMappingListener("test.service", "mapping")
	require.NoError(t, err)
}

func TestListAppRevisions_ReturnsAppRevisionType(t *testing.T) {
	// Verify the AppRevision struct is populated correctly from lastUpdatedTime in content
	mc := newMockZkClient()
	mc.data["/dubbo/my-app/rev-abc"] = []byte(`{"app":"my-app","revision":"rev-abc","lastUpdatedTime":5000}`)

	r := &zookeeperMetadataReport{client: mc, rootDir: "/dubbo/"}
	revisions, err := r.ListAppRevisions("my-app")
	require.NoError(t, err)
	require.Len(t, revisions, 1)
	assert.Equal(t, "rev-abc", revisions[0].Revision)
	assert.Equal(t, int64(5000), revisions[0].ModifyTime)
}

func TestListAppRevisions_ZeroLastUpdatedTime(t *testing.T) {
	// Old data without lastUpdatedTime returns ModifyTime=0; callers guard with > 0.
	mc := newMockZkClient()
	mc.data["/dubbo/my-app/old-rev"] = []byte(`{"app":"my-app","revision":"old-rev"}`)

	r := &zookeeperMetadataReport{client: mc, rootDir: "/dubbo/"}
	revisions, err := r.ListAppRevisions("my-app")
	require.NoError(t, err)
	require.Len(t, revisions, 1)
	assert.Equal(t, "old-rev", revisions[0].Revision)
	assert.Equal(t, int64(0), revisions[0].ModifyTime)
}
