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

package servicediscovery

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	perrors "github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// retryNotifyListener is a concurrency-safe notify listener: retry rebuilds run
// on the retry timer goroutine, so test listeners must synchronize access.
type retryNotifyListener struct {
	mu     sync.Mutex
	events []*registry.ServiceEvent
}

func (c *retryNotifyListener) Notify(event *registry.ServiceEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
}

func (c *retryNotifyListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append([]*registry.ServiceEvent(nil), events...)
	if callback != nil {
		callback()
	}
}

func (c *retryNotifyListener) snapshot() []*registry.ServiceEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*registry.ServiceEvent(nil), c.events...)
}

// stubMetadataFetch replaces the metadata fetcher and restores it on cleanup.
func stubMetadataFetch(t *testing.T, fetch func(app string, instance registry.ServiceInstance, revision, registryId string) (*info.MetadataInfo, error)) {
	t.Helper()
	original := metadataInfoFetcher
	metadataInfoFetcher = fetch
	t.Cleanup(func() { metadataInfoFetcher = original })
}

// stubRetryDelays shrinks the backoff so retries fire within test time budgets.
func stubRetryDelays(t *testing.T, initial, max time.Duration) {
	t.Helper()
	origInitial, origMax := metadataRetryInitialDelay, metadataRetryMaxDelay
	metadataRetryInitialDelay, metadataRetryMaxDelay = initial, max
	t.Cleanup(func() { metadataRetryInitialDelay, metadataRetryMaxDelay = origInitial, origMax })
}

// settleRetryListener stops any pending retry by flushing an empty snapshot:
// with no instances left, every unresolved revision is dropped and the timer
// is canceled. Register after stub cleanups so it runs before them.
func settleRetryListener(t *testing.T, listener *ServiceInstancesChangedListenerImpl) {
	t.Helper()
	t.Cleanup(func() {
		_ = listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{}))
	})
}

// TestMetadataRetryRecoversWithoutFurtherEvent is the core regression test for
// the permanent empty-directory issue: the first metadata fetch fails, no
// further registry event arrives, and the listener must still recover on its own.
func TestMetadataRetryRecoversWithoutFurtherEvent(t *testing.T) {
	const revision = "rev-retry-recover"
	const port = 22101
	stubRetryDelays(t, 5*time.Millisecond, 20*time.Millisecond)

	meta := newTestMetadataInfo(t, revision, port, "")
	var calls atomic.Int32
	stubMetadataFetch(t, func(string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		if calls.Add(1) == 1 {
			return nil, perrors.New("transient metadata failure")
		}
		return meta, nil
	})

	listener := NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp)).(*ServiceInstancesChangedListenerImpl)
	notify := &retryNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)
	t.Cleanup(func() { metaCache.Delete(testApp + ":" + constant.DefaultKey + ":" + revision) })

	err := listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		newTestServiceInstanceOnly(port, "", revision),
	}))
	require.Error(t, err, "first fetch failed, OnEvent must surface the unresolved revision")
	require.Empty(t, notify.snapshot(), "first fetch failed, nothing should be notified yet")

	require.Eventually(t, func() bool {
		return len(notify.snapshot()) > 0
	}, 3*time.Second, 10*time.Millisecond, "retry must rebuild service URLs without any further registry event")
}

// TestMetadataRetryStopsWhenInstanceRemoved verifies retries do not resurrect or
// keep probing instances that disappeared from the registry snapshot.
func TestMetadataRetryStopsWhenInstanceRemoved(t *testing.T) {
	const revision = "rev-retry-stop"
	const port = 22102
	stubRetryDelays(t, 5*time.Millisecond, 10*time.Millisecond)

	var calls atomic.Int32
	stubMetadataFetch(t, func(string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		calls.Add(1)
		return nil, perrors.New("metadata unreachable")
	})

	listener := NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp)).(*ServiceInstancesChangedListenerImpl)
	notify := &retryNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)

	require.Error(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		newTestServiceInstanceOnly(port, "", revision),
	})), "fetch failed, OnEvent must surface the unresolved revision")
	require.Eventually(t, func() bool { return calls.Load() >= 2 }, time.Second, 5*time.Millisecond,
		"retry should have run at least once")

	// The instance leaves the snapshot: retries must stop.
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{})))
	time.Sleep(30 * time.Millisecond)
	before := calls.Load()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, before, calls.Load(), "no fetch should happen after the instance is removed")
	assert.Nil(t, listener.retryTimer, "retry timer must be canceled")
}

// TestMetadataRetryFollowsLatestRevision verifies a superseded revision is not
// retried once the snapshot moved to a newer revision.
func TestMetadataRetryFollowsLatestRevision(t *testing.T) {
	const revOld = "rev-retry-old"
	const revNew = "rev-retry-new"
	const port = 22103
	// Initial delay large enough to deliver the second event before any retry fires.
	stubRetryDelays(t, 200*time.Millisecond, 400*time.Millisecond)

	metaNew := newTestMetadataInfo(t, revNew, port, "")
	var oldCalls, newCalls atomic.Int32
	stubMetadataFetch(t, func(_ string, _ registry.ServiceInstance, revision string, _ string) (*info.MetadataInfo, error) {
		if revision == revOld {
			oldCalls.Add(1)
			return nil, perrors.New("transient metadata failure")
		}
		newCalls.Add(1)
		return metaNew, nil
	})

	listener := NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp)).(*ServiceInstancesChangedListenerImpl)
	notify := &retryNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)
	t.Cleanup(func() { metaCache.Delete(testApp + ":" + constant.DefaultKey + ":" + revNew) })

	require.Error(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		newTestServiceInstanceOnly(port, "", revOld),
	})), "revOld fetch failed, OnEvent must surface the unresolved revision")
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		newTestServiceInstanceOnly(port, "", revNew),
	})))

	require.Eventually(t, func() bool { return len(notify.snapshot()) > 0 }, time.Second, 10*time.Millisecond)
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, int32(1), oldCalls.Load(), "superseded revision must not be retried")
}

// TestMetadataRetryUsesSingleTimer verifies repeated failing events share one
// retry timer instead of stacking one per event.
func TestMetadataRetryUsesSingleTimer(t *testing.T) {
	const revision = "rev-retry-single-timer"
	const port = 22104
	stubRetryDelays(t, time.Second, 2*time.Second)

	var calls atomic.Int32
	stubMetadataFetch(t, func(string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		calls.Add(1)
		return nil, perrors.New("metadata unreachable")
	})

	listener := NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp)).(*ServiceInstancesChangedListenerImpl)
	settleRetryListener(t, listener)
	notify := &retryNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)

	for range 3 {
		require.Error(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
			newTestServiceInstanceOnly(port, "", revision),
		})), "fetch failed, OnEvent must surface the unresolved revision")
	}

	require.NotNil(t, listener.retryTimer)
	assert.Equal(t, 1, listener.retryAttempts, "repeated events must share the pending retry timer")
	assert.Equal(t, int32(3), calls.Load(), "each event triggers one fetch, retries come only from the timer")
}

// TestMetadataRetryListenerDetach verifies removing the last subscriber cancels
// the retry timer, and re-attaching a subscriber resumes retries.
func TestMetadataRetryListenerDetach(t *testing.T) {
	const revision = "rev-retry-detach"
	const port = 22105
	stubRetryDelays(t, time.Second, 2*time.Second)

	stubMetadataFetch(t, func(string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		return nil, perrors.New("metadata unreachable")
	})

	listener := NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp)).(*ServiceInstancesChangedListenerImpl)
	settleRetryListener(t, listener)
	notify := &retryNotifyListener{}
	key := common.MatchKey(testInterface, constant.TriProtocol)
	listener.AddListenerAndNotify(key, notify)

	require.Error(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		newTestServiceInstanceOnly(port, "", revision),
	})), "fetch failed, OnEvent must surface the unresolved revision")
	require.NotNil(t, listener.retryTimer, "retry should be scheduled after a failed fetch")

	listener.RemoveListener(key)
	assert.Nil(t, listener.retryTimer, "removing the last subscriber must cancel the retry timer")

	listener.AddListenerAndNotify(key, notify)
	assert.NotNil(t, listener.retryTimer, "re-attaching a subscriber must resume retries")
}
