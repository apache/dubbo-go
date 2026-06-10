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

package etcd

import (
	"encoding/json"
	"strings"
	"testing"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"

	perrors "github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clientv3 "go.etcd.io/etcd/client/v3"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
)

// --- Mock etcdClient ---
// mockEtcdClient implements etcdClient for testing.
type mockEtcdClient struct {
	data map[string]string // key -> value
	rev  map[string]int64  // key -> revision (monotonic counter)
	seq  int64             // global revision counter
}

func newMockEtcdClient() *mockEtcdClient {
	return &mockEtcdClient{
		data: make(map[string]string),
		rev:  make(map[string]int64),
	}
}

func (m *mockEtcdClient) Get(key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", gxetcd.ErrKVPairNotFound
	}
	return v, nil
}

func (m *mockEtcdClient) Put(key, value string) error {
	m.data[key] = value
	m.seq++
	m.rev[key] = m.seq
	return nil
}

func (m *mockEtcdClient) Delete(key string) error {
	if _, ok := m.data[key]; !ok {
		return nil // etcd delete is idempotent
	}
	delete(m.data, key)
	return nil
}

func (m *mockEtcdClient) GetChildren(key string) ([]string, []string, error) {
	var keys, values []string
	for k, v := range m.data {
		if strings.HasPrefix(k, key) {
			keys = append(keys, k)
			values = append(values, v)
		}
	}
	if len(keys) == 0 {
		return nil, nil, gxetcd.ErrKVPairNotFound
	}
	return keys, values, nil
}

func (m *mockEtcdClient) GetValAndRev(key string) (string, int64, error) {
	v, ok := m.data[key]
	if !ok {
		return "", 0, gxetcd.ErrKVPairNotFound
	}
	return v, m.rev[key], nil
}

func (m *mockEtcdClient) Create(key, value string) error {
	if _, ok := m.data[key]; ok {
		return gxetcd.ErrCompareFail
	}
	m.data[key] = value
	m.seq++
	m.rev[key] = m.seq
	return nil
}

func (m *mockEtcdClient) UpdateWithRev(key, value string, rev int64, _ ...clientv3.OpOption) error {
	cur := m.rev[key] // zero value (0) if missing
	if cur != rev {
		return gxetcd.ErrCompareFail
	}
	m.data[key] = value
	m.seq++
	m.rev[key] = m.seq
	return nil
}

// --- Helper ---

func newTestReport() (*etcdMetadataReport, *mockEtcdClient) {
	mc := newMockEtcdClient()
	r := &etcdMetadataReport{
		client:  mc,
		rootDir: "/dubbo",
	}
	return r, mc
}

// --- Tests ---

func TestPublishAndGetAppMetadata(t *testing.T) {
	r, _ := newTestReport()

	meta := &info.MetadataInfo{
		App:      "my-app",
		Revision: "r1",
		Services: map[string]*info.ServiceInfo{
			"com.example.Foo": {Name: "com.example.Foo", Protocol: "dubbo"},
		},
	}

	err := r.PublishAppMetadata("my-app", "r1", meta)
	require.NoError(t, err)

	got, err := r.GetAppMetadata("my-app", "r1")
	require.NoError(t, err)
	assert.Equal(t, "my-app", got.App)
	assert.Equal(t, "r1", got.Revision)
	assert.Contains(t, got.Services, "com.example.Foo")

	// Get non-existent returns error
	_, err = r.GetAppMetadata("my-app", "nonexistent")
	require.Error(t, err)
	assert.True(t, perrors.Is(err, gxetcd.ErrKVPairNotFound))
}

func TestPublishAppMetadata_Update(t *testing.T) {
	r, _ := newTestReport()

	meta := &info.MetadataInfo{App: "my-app", Revision: "r1"}
	err := r.PublishAppMetadata("my-app", "r1", meta)
	require.NoError(t, err)

	meta.Revision = "r2"
	err = r.PublishAppMetadata("my-app", "r1", meta)
	require.NoError(t, err)

	got, err := r.GetAppMetadata("my-app", "r1")
	require.NoError(t, err)
	assert.Equal(t, "r2", got.Revision)
}

func TestUnPublishAppMetadata(t *testing.T) {
	r, _ := newTestReport()

	meta := &info.MetadataInfo{App: "my-app", Revision: "r1"}
	err := r.PublishAppMetadata("my-app", "r1", meta)
	require.NoError(t, err)

	err = r.UnPublishAppMetadata("my-app", "r1")
	require.NoError(t, err)

	_, err = r.GetAppMetadata("my-app", "r1")
	require.Error(t, err)
}

func TestUnPublishAppMetadata_Idempotent(t *testing.T) {
	r, _ := newTestReport()

	// Deleting non-existent key should not error (etcd is idempotent)
	err := r.UnPublishAppMetadata("my-app", "nonexistent")
	require.NoError(t, err)
}

func TestListAppRevisions(t *testing.T) {
	r, _ := newTestReport()

	// No revisions for unknown app
	revisions, err := r.ListAppRevisions("unknown-app")
	require.NoError(t, err)
	assert.Empty(t, revisions)

	// Publish multiple revisions with explicit lastUpdatedTime values
	require.NoError(t, r.PublishAppMetadata("my-app", "r1", &info.MetadataInfo{App: "my-app", Revision: "r1", LastUpdatedTime: 1000}))
	require.NoError(t, r.PublishAppMetadata("my-app", "r2", &info.MetadataInfo{App: "my-app", Revision: "r2", LastUpdatedTime: 3000}))
	require.NoError(t, r.PublishAppMetadata("my-app", "r3", &info.MetadataInfo{App: "my-app", Revision: "r3", LastUpdatedTime: 2000}))

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

func TestListAppRevisions_ReturnsAppRevisionType(t *testing.T) {
	r, mc := newTestReport()

	mc.data["/dubbo/my-app/rev-abc"] = `{"app":"my-app","revision":"rev-abc","lastUpdatedTime":5000}`

	revisions, err := r.ListAppRevisions("my-app")
	require.NoError(t, err)
	require.Len(t, revisions, 1)
	assert.Equal(t, "rev-abc", revisions[0].Revision)
	assert.Equal(t, int64(5000), revisions[0].ModifyTime)
}

func TestListAppRevisions_ZeroLastUpdatedTime(t *testing.T) {
	// Old data without lastUpdatedTime returns ModifyTime=0; callers guard with > 0.
	r, mc := newTestReport()

	mc.data["/dubbo/my-app/old-rev"] = `{"app":"my-app","revision":"old-rev"}`

	revisions, err := r.ListAppRevisions("my-app")
	require.NoError(t, err)
	require.Len(t, revisions, 1)
	assert.Equal(t, "old-rev", revisions[0].Revision)
	assert.Equal(t, int64(0), revisions[0].ModifyTime)
}

func TestRegisterServiceAppMapping_NewKey(t *testing.T) {
	r, mc := newTestReport()

	err := r.RegisterServiceAppMapping("com.example.Foo", "mapping", "app1")
	require.NoError(t, err)
	assert.Equal(t, "app1", mc.data["/dubbo/mapping/com.example.Foo"])
}

func TestRegisterServiceAppMapping_Append(t *testing.T) {
	r, mc := newTestReport()

	mc.data["/dubbo/mapping/com.example.Foo"] = "app1"

	err := r.RegisterServiceAppMapping("com.example.Foo", "mapping", "app2")
	require.NoError(t, err)
	assert.Equal(t, "app1,app2", mc.data["/dubbo/mapping/com.example.Foo"])
}

func TestRegisterServiceAppMapping_Duplicate(t *testing.T) {
	r, mc := newTestReport()

	mc.data["/dubbo/mapping/com.example.Foo"] = "app1,app2"

	err := r.RegisterServiceAppMapping("com.example.Foo", "mapping", "app1")
	require.NoError(t, err)
	assert.Equal(t, "app1,app2", mc.data["/dubbo/mapping/com.example.Foo"])
}

func TestGetServiceAppMapping(t *testing.T) {
	r, mc := newTestReport()

	mc.data["/dubbo/mapping/com.example.Foo"] = "app1,app2"

	set, err := r.GetServiceAppMapping("com.example.Foo", "mapping", nil)
	require.NoError(t, err)
	assert.True(t, set.Contains("app1"))
	assert.True(t, set.Contains("app2"))
}

func TestGetServiceAppMapping_NotFound(t *testing.T) {
	r, _ := newTestReport()

	_, err := r.GetServiceAppMapping("com.example.Foo", "mapping", nil)
	require.Error(t, err)
}

func TestRemoveServiceAppMappingListener(t *testing.T) {
	r, _ := newTestReport()
	err := r.RemoveServiceAppMappingListener("key", "group")
	require.NoError(t, err)
}

// --- Pure logic test ---

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
}
