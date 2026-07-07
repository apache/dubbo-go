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

package metadata

import (
	"fmt"
	"sync"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

// versionedStore is an in-memory key/value store with a per-key version, modeling the
// compare-and-swap primitive a real metadata center (etcd ModRevision, zk Stat.Version,
// nacos content MD5) provides.
type versionedStore struct {
	mu   sync.Mutex
	data map[string]versionedEntry
}

type versionedEntry struct {
	val string
	ver int64
}

func newVersionedStore() *versionedStore {
	return &versionedStore{data: make(map[string]versionedEntry)}
}

func (s *versionedStore) get(key string) (string, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.data[key]
	return e.val, e.ver
}

// cas writes val only if the current version equals ver, returning whether it was applied.
func (s *versionedStore) cas(key, val string, ver int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[key].ver != ver {
		return false
	}
	s.data[key] = versionedEntry{val: val, ver: ver + 1}
	return true
}

// put writes unconditionally, modeling the old read-modify-write behavior without CAS.
func (s *versionedStore) put(key, val string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = versionedEntry{val: val, ver: s.data[key].ver + 1}
}

// stubReport satisfies the non-mapping parts of report.MetadataReport.
type stubReport struct{}

func (stubReport) GetAppMetadata(string, string) (*info.MetadataInfo, error)   { return nil, nil }
func (stubReport) PublishAppMetadata(string, string, *info.MetadataInfo) error { return nil }
func (stubReport) GetServiceAppMapping(string, string, mapping.MappingListener) (*gxset.HashSet, error) {
	return nil, nil
}
func (stubReport) RemoveServiceAppMappingListener(string, string) error { return nil }
func (stubReport) UnPublishAppMetadata(string, string) error            { return nil }
func (stubReport) ListAppRevisions(string) ([]report.AppRevision, error) {
	return nil, nil
}
func (stubReport) URL() *common.URL { return nil }

// casReport registers mappings with optimistic concurrency against a versionedStore, exactly
// as the etcd/zk/nacos reports now do, returning report.ErrMappingCASConflict on conflict.
type casReport struct {
	stubReport
	store *versionedStore
}

func (r *casReport) RegisterServiceAppMapping(key, group, value string) error {
	full := group + "/" + key
	old, ver := r.store.get(full)
	merged, changed := report.MergeServiceAppMapping(old, value)
	if !changed {
		return nil
	}
	if !r.store.cas(full, merged, ver) {
		return report.ErrMappingCASConflict
	}
	return nil
}

func (r *casReport) GetServiceAppMapping(key, group string, _ mapping.MappingListener) (*gxset.HashSet, error) {
	val, _ := r.store.get(group + "/" + key)
	return report.DecodeServiceAppNames(val), nil
}

// fastRetry sets a near-zero backoff and the given retry budget, returning a restore func.
func fastRetry(times int) func() {
	ot, ob, om := retryTimes, retryBaseInterval, retryMaxInterval
	retryTimes, retryBaseInterval, retryMaxInterval = times, 0, 0
	return func() { retryTimes, retryBaseInterval, retryMaxInterval = ot, ob, om }
}

// TestNaiveReadModifyWriteLosesConcurrentUpdate reproduces the bug the issue is about: two
// providers that read the same value and write back unconditionally clobber each other.
func TestNaiveReadModifyWriteLosesConcurrentUpdate(t *testing.T) {
	store := newVersionedStore()
	const key = "mapping/Iface"

	// Both providers read the initial empty value...
	oldA, _ := store.get(key)
	oldB, _ := store.get(key)
	// ...each merges its own app and writes back without a version check.
	mergedA, _ := report.MergeServiceAppMapping(oldA, "appA")
	store.put(key, mergedA)
	mergedB, _ := report.MergeServiceAppMapping(oldB, "appB")
	store.put(key, mergedB)

	val, _ := store.get(key)
	got := report.DecodeServiceAppNames(val)
	assert.False(t, got.Contains("appA"), "appA was silently lost by the second write")
	assert.True(t, got.Contains("appB"))
}

// TestCASRejectsConcurrentUpdate shows the fix: the second writer's CAS fails, and after a
// re-read both apps survive.
func TestCASRejectsConcurrentUpdate(t *testing.T) {
	store := newVersionedStore()
	const key = "mapping/Iface"

	oldA, verA := store.get(key)
	oldB, verB := store.get(key) // both read version 0

	mergedA, _ := report.MergeServiceAppMapping(oldA, "appA")
	assert.True(t, store.cas(key, mergedA, verA)) // A wins, version -> 1

	mergedB, _ := report.MergeServiceAppMapping(oldB, "appB")
	assert.False(t, store.cas(key, mergedB, verB)) // B's stale version is rejected

	// B re-reads and retries; nothing is lost.
	oldB2, verB2 := store.get(key)
	mergedB2, _ := report.MergeServiceAppMapping(oldB2, "appB")
	assert.True(t, store.cas(key, mergedB2, verB2))

	val, _ := store.get(key)
	got := report.DecodeServiceAppNames(val)
	assert.True(t, got.Contains("appA"))
	assert.True(t, got.Contains("appB"))
}

// TestRegisterWithRetryConcurrentNoLostUpdate drives the real registerWithRetry loop with many
// writers racing on the same interface key, while readers concurrently read it. It asserts no
// app is lost and that a reader never observes the set shrink. Run with -race to also catch
// data races.
func TestRegisterWithRetryConcurrentNoLostUpdate(t *testing.T) {
	defer fastRetry(10000)()
	store := newVersionedStore()
	r := &casReport{store: store}

	const writers = 200
	const readers = 20

	// concurrent readers: every successful registration only appends, so a reader must never
	// see the set get smaller or contain a malformed entry.
	stop := make(chan struct{})
	var readerWg sync.WaitGroup
	for range readers {
		readerWg.Go(func() {
			prev := 0
			for {
				select {
				case <-stop:
					return
				default:
					set, err := r.GetServiceAppMapping("Iface", DefaultGroup, nil)
					assert.NoError(t, err)
					assert.GreaterOrEqual(t, set.Size(), prev)
					assert.False(t, set.Contains(""))
					prev = set.Size()
				}
			}
		})
	}

	var writerWg sync.WaitGroup
	for i := range writers {
		writerWg.Add(1)
		go func(i int) {
			defer writerWg.Done()
			assert.NoError(t, registerWithRetry(r, "Iface", DefaultGroup, fmt.Sprintf("app-%d", i)))
		}(i)
	}
	writerWg.Wait()
	close(stop)
	readerWg.Wait()

	val, _ := store.get(DefaultGroup + "/Iface")
	got := report.DecodeServiceAppNames(val)
	assert.Equal(t, writers, got.Size())
	for i := range writers {
		assert.True(t, got.Contains(fmt.Sprintf("app-%d", i)), "app-%d was lost", i)
	}
}
