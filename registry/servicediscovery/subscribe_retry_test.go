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

// stubSubscribeRetryDelays shrinks the subscribe backoff so retries fire within
// test time budgets.
func stubSubscribeRetryDelays(t *testing.T, initial, max time.Duration) {
	t.Helper()
	origInitial, origMax := subscribeRetryInitialDelay, subscribeRetryMaxDelay
	subscribeRetryInitialDelay, subscribeRetryMaxDelay = initial, max
	t.Cleanup(func() { subscribeRetryInitialDelay, subscribeRetryMaxDelay = origInitial, origMax })
}

// retrySubscribeDiscovery is a concurrency-safe ServiceDiscovery stub whose
// AddListener keeps failing while fail is set, so tests can control exactly
// when the subscription is allowed to be established. When blockOnCall > 0,
// the Nth AddListener call blocks until blockRelease is closed, letting tests
// hold an attempt in flight deterministically.
type retrySubscribeDiscovery struct {
	mockServiceDiscovery
	mu           sync.Mutex
	fail         atomic.Bool
	addCalls     int
	instances    []registry.ServiceInstance
	blockOnCall  int
	blockEntered chan struct{}
	blockRelease chan struct{}
}

func (m *retrySubscribeDiscovery) AddListener(registry.ServiceInstancesChangedListener) error {
	m.mu.Lock()
	m.addCalls++
	call := m.addCalls
	m.mu.Unlock()
	if call == m.blockOnCall && m.blockRelease != nil {
		close(m.blockEntered)
		<-m.blockRelease
	}
	if m.fail.Load() {
		return errors.New("transient subscribe failure")
	}
	return nil
}

func (m *retrySubscribeDiscovery) GetInstances(string) []registry.ServiceInstance {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]registry.ServiceInstance(nil), m.instances...)
}

func (m *retrySubscribeDiscovery) setInstances(instances []registry.ServiceInstance) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instances = instances
}

func (m *retrySubscribeDiscovery) addListenerCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addCalls
}

func newRetryTestRegistry(sd registry.ServiceDiscovery) *serviceDiscoveryRegistry {
	registryURL, _ := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"))
	return &serviceDiscoveryRegistry{
		url:                registryURL,
		serviceDiscovery:   sd,
		serviceListeners:   make(map[string]registry.ServiceInstancesChangedListener),
		subscribeRetries:   make(map[string]*subscribeRetry),
		serviceNameMapping: &mockServiceNameMapping{data: map[string]*gxset.HashSet{testInterface: gxset.NewSet(testApp)}},
	}
}

func newRetryConsumerURL(t *testing.T) *common.URL {
	t.Helper()
	consumerURL, err := common.NewURL("tri://127.0.0.1:20000/",
		common.WithInterface(testInterface),
		common.WithParamsValue(constant.SideKey, constant.SideConsumer))
	require.NoError(t, err)
	return consumerURL
}

// TestSubscribeRetryRecoversAfterTransientFailure is the core regression test
// for issue #3624: the first AddListener fails, no further registry event
// arrives, and the subscription must still be established by the retry loop —
// including a re-sync of the latest instance snapshot so the consumer receives
// instance events without a restart.
func TestSubscribeRetryRecoversAfterTransientFailure(t *testing.T) {
	const revision = "rev-subscribe-retry"
	const port = 22301
	stubSubscribeRetryDelays(t, 5*time.Millisecond, 20*time.Millisecond)

	meta := newTestMetadataInfo(t, revision, port, "")
	stubMetadataFetch(t, func(context.Context, string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		return meta, nil
	})
	t.Cleanup(func() { metaCache.Delete(testApp + ":" + constant.DefaultKey + ":" + revision) })

	sd := &retrySubscribeDiscovery{}
	sd.fail.Store(true)
	reg := newRetryTestRegistry(sd)
	t.Cleanup(reg.Destroy)

	notify := &retryNotifyListener{}
	reg.SubscribeURL(newRetryConsumerURL(t), notify, gxset.NewSet(testApp))

	require.Eventually(t, func() bool { return sd.addListenerCalls() >= 1 }, time.Second, 5*time.Millisecond,
		"initial AddListener attempt should have run")
	require.Empty(t, notify.snapshot(), "no instance is visible while the subscription is down")

	// The provider changes while the subscription is not established.
	sd.setInstances([]registry.ServiceInstance{newTestServiceInstanceOnly(port, "", revision)})
	sd.fail.Store(false)

	require.Eventually(t, func() bool { return len(notify.snapshot()) > 0 }, 3*time.Second, 10*time.Millisecond,
		"retry must establish the subscription and re-sync the latest snapshot")
	assert.Empty(t, reg.subscribeRetries, "retry state must be cleared once the subscription is established")
}

// TestSubscribeRetryBackoffDelay verifies the backoff grows exponentially, is
// capped, and stays within the [delay, delay+25%] jitter band.
func TestSubscribeRetryBackoffDelay(t *testing.T) {
	initial, max := time.Second, 30*time.Second
	stubSubscribeRetryDelays(t, initial, max)
	for _, attempt := range []int{0, 1, 2, 3} {
		want := initial << attempt
		for range 50 {
			got := subscribeRetryDelay(attempt)
			assert.GreaterOrEqual(t, got, want)
			assert.LessOrEqual(t, got, want+want/4)
		}
	}
	// Beyond the cap every attempt lands in the capped jitter band.
	for _, attempt := range []int{5, 10, 29, 30, 100} {
		for range 50 {
			got := subscribeRetryDelay(attempt)
			assert.GreaterOrEqual(t, got, max)
			assert.LessOrEqual(t, got, max+max/4)
		}
	}
}

// TestSubscribeRetryStopsAfterUnSubscribe verifies retries stop once the last
// subscriber unsubscribes.
func TestSubscribeRetryStopsAfterUnSubscribe(t *testing.T) {
	stubSubscribeRetryDelays(t, 5*time.Millisecond, 10*time.Millisecond)

	sd := &retrySubscribeDiscovery{}
	sd.fail.Store(true)
	reg := newRetryTestRegistry(sd)
	t.Cleanup(reg.Destroy)

	consumerURL := newRetryConsumerURL(t)
	reg.SubscribeURL(consumerURL, &retryNotifyListener{}, gxset.NewSet(testApp))
	require.Eventually(t, func() bool { return sd.addListenerCalls() >= 2 }, time.Second, 5*time.Millisecond,
		"retry loop should be running")

	require.NoError(t, reg.UnSubscribe(consumerURL, &retryNotifyListener{}))
	require.Eventually(t, func() bool {
		reg.lock.RLock()
		defer reg.lock.RUnlock()
		return len(reg.subscribeRetries) == 0
	}, time.Second, 5*time.Millisecond, "pending retry must be canceled on unsubscribe")

	time.Sleep(30 * time.Millisecond)
	before := sd.addListenerCalls()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, before, sd.addListenerCalls(), "no AddListener attempt should happen after unsubscribe")
}

// TestSubscribeRetryStopsAfterDestroy verifies Destroy cancels pending retries.
func TestSubscribeRetryStopsAfterDestroy(t *testing.T) {
	stubSubscribeRetryDelays(t, 5*time.Millisecond, 10*time.Millisecond)

	sd := &retrySubscribeDiscovery{}
	sd.fail.Store(true)
	reg := newRetryTestRegistry(sd)

	reg.SubscribeURL(newRetryConsumerURL(t), &retryNotifyListener{}, gxset.NewSet(testApp))
	require.Eventually(t, func() bool { return sd.addListenerCalls() >= 2 }, time.Second, 5*time.Millisecond,
		"retry loop should be running")

	reg.Destroy()
	assert.Empty(t, reg.subscribeRetries, "Destroy must cancel pending retries")

	time.Sleep(30 * time.Millisecond)
	before := sd.addListenerCalls()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, before, sd.addListenerCalls(), "no AddListener attempt should happen after Destroy")

	// A late failure after Destroy must not arm a new retry timer.
	reg.scheduleSubscribeRetry(testApp, &subscribeRetry{})
	assert.Empty(t, reg.subscribeRetries, "retries must not be scheduled after Destroy")
}

// TestSubscribeRetryUsesSingleTimer verifies repeated failing subscribes for
// the same applications share one pending retry instead of stacking timers.
func TestSubscribeRetryUsesSingleTimer(t *testing.T) {
	stubSubscribeRetryDelays(t, time.Second, 2*time.Second)

	sd := &retrySubscribeDiscovery{}
	sd.fail.Store(true)
	reg := newRetryTestRegistry(sd)
	t.Cleanup(reg.Destroy)

	services := gxset.NewSet(testApp)
	reg.SubscribeURL(newRetryConsumerURL(t), &retryNotifyListener{}, services)
	require.Eventually(t, func() bool { return sd.addListenerCalls() >= 1 }, time.Second, 5*time.Millisecond)
	// A second subscribe hits the fast path and fails again; it must reuse the
	// pending retry, not arm a second timer.
	reg.SubscribeURL(newRetryConsumerURL(t), &retryNotifyListener{}, services)
	require.Eventually(t, func() bool { return sd.addListenerCalls() >= 2 }, time.Second, 5*time.Millisecond)
	// addCalls is incremented before the failure path schedules; give both
	// schedule attempts time to settle before asserting on the map.
	time.Sleep(100 * time.Millisecond)

	reg.lock.RLock()
	defer reg.lock.RUnlock()
	assert.Len(t, reg.subscribeRetries, 1, "repeated failures must share the pending retry timer")
	assert.Equal(t, 1, reg.subscribeRetries[testApp].attempts)
}

// TestSubscribeRetryResolvedByConcurrentSuccess covers the review finding: a
// subscribe that succeeds while a retry is pending must cancel the pending
// timer — otherwise it fires an extra AddListener, which is not safe for
// service discoveries whose AddListener is not idempotent (e.g. Polaris
// AddSubscriber, ZooKeeper ListenServiceEvent).
func TestSubscribeRetryResolvedByConcurrentSuccess(t *testing.T) {
	stubSubscribeRetryDelays(t, 500*time.Millisecond, time.Second)

	sd := &retrySubscribeDiscovery{}
	sd.fail.Store(true)
	reg := newRetryTestRegistry(sd)
	t.Cleanup(reg.Destroy)

	services := gxset.NewSet(testApp)
	reg.SubscribeURL(newRetryConsumerURL(t), &retryNotifyListener{}, services)
	require.Eventually(t, func() bool { return sd.addListenerCalls() >= 1 }, time.Second, 5*time.Millisecond,
		"initial AddListener attempt should have run")
	require.Eventually(t, func() bool {
		reg.lock.RLock()
		defer reg.lock.RUnlock()
		return len(reg.subscribeRetries) == 1
	}, time.Second, 5*time.Millisecond, "retry should be pending after the failed subscribe")

	// A later subscribe succeeds while the retry is still pending.
	sd.fail.Store(false)
	reg.SubscribeURL(newRetryConsumerURL(t), &retryNotifyListener{}, services)
	require.Eventually(t, func() bool { return sd.addListenerCalls() >= 2 }, time.Second, 5*time.Millisecond)

	// Wait well beyond the backoff: the pending retry must have been resolved.
	time.Sleep(700 * time.Millisecond)
	assert.Equal(t, 2, sd.addListenerCalls(), "successful subscribe must cancel the pending retry")
	reg.lock.RLock()
	defer reg.lock.RUnlock()
	assert.Empty(t, reg.subscribeRetries, "retry state must be resolved after success")
}

// TestSubscribeRetryInFlightStatePreserved covers the review finding: the
// retry state must stay in the map while its attempt is in flight, so a
// concurrent failing subscribe dedups against it instead of seeding a fresh
// attempts=0 state that resets the backoff.
func TestSubscribeRetryInFlightStatePreserved(t *testing.T) {
	// Large delays: no timer fires during the test; the retry is driven
	// synchronously instead.
	stubSubscribeRetryDelays(t, time.Hour, 2*time.Hour)

	sd := &retrySubscribeDiscovery{
		blockOnCall:  2,
		blockEntered: make(chan struct{}),
		blockRelease: make(chan struct{}),
	}
	sd.fail.Store(true)
	reg := newRetryTestRegistry(sd)
	t.Cleanup(reg.Destroy)

	services := gxset.NewSet(testApp)
	reg.SubscribeURL(newRetryConsumerURL(t), &retryNotifyListener{}, services)
	require.Eventually(t, func() bool { return sd.addListenerCalls() >= 1 }, time.Second, 5*time.Millisecond)

	var state *subscribeRetry
	require.Eventually(t, func() bool {
		reg.lock.RLock()
		defer reg.lock.RUnlock()
		state = reg.subscribeRetries[testApp]
		return state != nil
	}, time.Second, 5*time.Millisecond, "retry should be pending after the failed subscribe")
	reg.lock.Lock()
	if state.timer != nil {
		state.timer.Stop()
	}
	reg.lock.Unlock()

	// Drive the retry attempt synchronously; it blocks inside AddListener
	// call #2, i.e. in flight.
	retryDone := make(chan struct{})
	go func() {
		defer close(retryDone)
		reg.retryAddListener(testApp)
	}()
	select {
	case <-sd.blockEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("retry attempt never reached AddListener")
	}

	// A concurrent subscribe fails while the retry is in flight.
	reg.SubscribeURL(newRetryConsumerURL(t), &retryNotifyListener{}, services)
	require.Eventually(t, func() bool { return sd.addListenerCalls() >= 3 }, time.Second, 5*time.Millisecond)
	// Let its schedule attempt finish before the in-flight attempt resolves,
	// so the dedup check runs while the in-flight state is still present.
	time.Sleep(100 * time.Millisecond)

	reg.lock.RLock()
	current := reg.subscribeRetries[testApp]
	reg.lock.RUnlock()
	assert.Same(t, state, current,
		"in-flight retry state must be preserved, not replaced by a fresh attempts=0 state")

	// Let the in-flight attempt succeed: the state is resolved.
	sd.fail.Store(false)
	close(sd.blockRelease)
	<-retryDone
	require.Eventually(t, func() bool {
		reg.lock.RLock()
		defer reg.lock.RUnlock()
		return len(reg.subscribeRetries) == 0
	}, time.Second, 5*time.Millisecond, "successful in-flight retry must resolve the state")
	assert.Equal(t, 3, sd.addListenerCalls(), "no extra AddListener beyond the three driven attempts")
}