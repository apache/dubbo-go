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
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

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
func stubMetadataFetch(t *testing.T, fetch func(ctx context.Context, app string, instance registry.ServiceInstance, revision, registryId string) (*info.MetadataInfo, error)) {
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
	stubMetadataFetch(t, func(context.Context, string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		if calls.Add(1) == 1 {
			return nil, errors.New("transient metadata failure")
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
	stubMetadataFetch(t, func(context.Context, string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		calls.Add(1)
		return nil, errors.New("metadata unreachable")
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
	stubMetadataFetch(t, func(_ context.Context, _ string, _ registry.ServiceInstance, revision string, _ string) (*info.MetadataInfo, error) {
		if revision == revOld {
			oldCalls.Add(1)
			return nil, errors.New("transient metadata failure")
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
	stubMetadataFetch(t, func(context.Context, string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		calls.Add(1)
		return nil, errors.New("metadata unreachable")
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

	stubMetadataFetch(t, func(context.Context, string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		return nil, errors.New("metadata unreachable")
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

// TestMetadataRetryWaitsForSubscriber verifies a failed refresh with no
// attached subscriber does not arm the retry timer: a listener that is never
// installed (e.g. discarded after losing the subscribe install race) must not
// keep probing metadata. Attaching a subscriber re-arms the retry.
func TestMetadataRetryWaitsForSubscriber(t *testing.T) {
	const revision = "rev-retry-no-subscriber"
	const port = 22106
	stubRetryDelays(t, time.Second, 2*time.Second)

	stubMetadataFetch(t, func(context.Context, string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		return nil, errors.New("metadata unreachable")
	})

	listener := NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp)).(*ServiceInstancesChangedListenerImpl)
	settleRetryListener(t, listener)

	// Snapshot arrives before any subscriber attaches (the SubscribeURL order).
	require.Error(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		newTestServiceInstanceOnly(port, "", revision),
	})))
	assert.Nil(t, listener.retryTimer, "no subscriber attached: retry must not be armed")

	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), &retryNotifyListener{})
	assert.NotNil(t, listener.retryTimer, "attaching a subscriber must arm the pending retry")
}

// TestMetadataRetryStopsAfterClose verifies a closed listener cannot re-arm the
// retry timer — neither via a direct schedule nor via an in-flight refresh
// that finishes after the owning registry was destroyed.
func TestMetadataRetryStopsAfterClose(t *testing.T) {
	const revision = "rev-retry-closed"
	const port = 22107
	stubRetryDelays(t, time.Second, 2*time.Second)

	stubMetadataFetch(t, func(context.Context, string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		return nil, errors.New("metadata unreachable")
	})

	listener := NewServiceInstancesChangedListener(testApp, constant.DefaultKey, gxset.NewSet(testApp)).(*ServiceInstancesChangedListenerImpl)
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), &retryNotifyListener{})

	require.Error(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		newTestServiceInstanceOnly(port, "", revision),
	})))
	require.NotNil(t, listener.retryTimer, "retry should be scheduled after a failed fetch")

	listener.stopMetadataRetry()
	assert.Nil(t, listener.retryTimer, "close must cancel the pending retry")

	listener.scheduleMetadataRetry()
	assert.Nil(t, listener.retryTimer, "closed listener must not re-arm the retry")

	// Simulate a refresh that was in flight while the registry was destroyed:
	// its trailing schedule must not arm a new timer.
	listener.refreshServiceURLs()
	assert.Nil(t, listener.retryTimer, "in-flight refresh finishing after close must not arm a retry")
}

// TestUnSubscribeStopsMetadataRetry is the regression test for the public
// SubscribeURL -> UnSubscribe lifecycle: the listener is registered under the
// protocol-qualified key, so UnSubscribe must remove it with the same key.
// With an unresolved revision and a pending retry timer, UnSubscribe must leave
// the listener subscriber-less and stop the timer from probing metadata.
func TestUnSubscribeStopsMetadataRetry(t *testing.T) {
	const revision = "rev-unsubscribe-retry"
	const port = 22108
	stubRetryDelays(t, 5*time.Millisecond, 20*time.Millisecond)

	var fetchCalls atomic.Int32
	stubMetadataFetch(t, func(context.Context, string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		fetchCalls.Add(1)
		return nil, errors.New("metadata unreachable")
	})

	setupEnvironment(t)
	registryURL, _ := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"))
	reg, err := newServiceDiscoveryRegistry(registryURL)
	require.NoError(t, err)
	sdReg, ok := reg.(*serviceDiscoveryRegistry)
	require.True(t, ok)
	sdReg.serviceDiscovery = &destroyRaceDiscovery{
		instances: []registry.ServiceInstance{newTestServiceInstanceOnly(port, "", revision)},
	}

	consumerURL, err := common.NewURL("dubbo://127.0.0.1:20000/",
		common.WithInterface(testInterface),
		common.WithParamsValue(constant.SideKey, constant.SideConsumer),
		common.WithParamsValue(constant.ProvidedBy, testApp))
	require.NoError(t, err)

	sdReg.SubscribeURL(consumerURL, &retryNotifyListener{}, gxset.NewSet(testApp))

	listener, ok := sdReg.getServiceListener(testApp).(*ServiceInstancesChangedListenerImpl)
	require.True(t, ok, "SubscribeURL must install the listener")
	require.Eventually(t, func() bool {
		listener.mutex.Lock()
		defer listener.mutex.Unlock()
		return listener.retryTimer != nil
	}, 2*time.Second, time.Millisecond, "failed metadata fetch must arm the retry timer")

	require.NoError(t, sdReg.UnSubscribe(consumerURL, &retryNotifyListener{}))

	listener.mutex.Lock()
	remaining := len(listener.listeners)
	timer := listener.retryTimer
	listener.mutex.Unlock()
	assert.Zero(t, remaining, "UnSubscribe must remove the subscriber registered by SubscribeURL")
	assert.Nil(t, timer, "removing the last subscriber must cancel the retry timer")

	// No retry may fire after the last subscriber left.
	before := fetchCalls.Load()
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, before, fetchCalls.Load(), "no metadata fetch may happen after UnSubscribe")
}
