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

package nacos

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

// noopListener is a no-op ConfigurationListener used by the concurrency tests.
// It carries an id so each instance has a distinct address — an empty struct
// would have Go collapse every &noopListener{} to a single address
// (runtime.zerobase), making every listener the same map key and invalidating
// the add/remove bookkeeping under test.
type noopListener struct {
	id int
}

var listenerSeq int32

func newNoopListener() *noopListener {
	return &noopListener{id: int(atomic.AddInt32(&listenerSeq, 1))}
}

func (r *noopListener) Process(*config_center.ConfigChangeEvent) {}

// countingConfigClient is a concurrency-safe IConfigClient that mirrors the
// real Nacos SDK semantics relevant to the TOCTOU race under review: both
// ListenConfig and CancelListenConfig operate on a single per-(dataId, group)
// slot. ListenConfig sets the slot; CancelListenConfig clears it
// unconditionally — exactly like the SDK's cacheMap.Remove, which drops
// whatever subscription currently occupies the slot regardless of when it was
// registered. This is what makes the race dangerous: a late cancel can wipe a
// subscription a racing listen just established.
//
// The optional onListenDone / onCancelEnter hooks let a deterministic test
// choreograph the exact interleaving the reviewer described (a cancel landing
// after a racing listen) without relying on timing.
type countingConfigClient struct {
	mu            sync.Mutex
	slot          map[string]struct{} // (dataId|group) -> present when a subscription is live
	listens       int32
	cancels       int32
	onListenDone  func() // invoked after ListenConfig sets the slot
	onCancelEnter func() // invoked before CancelListenConfig clears the slot
}

func newCountingConfigClient() *countingConfigClient {
	return &countingConfigClient{slot: make(map[string]struct{})}
}

func (c *countingConfigClient) keyOf(p vo.ConfigParam) string { return p.DataId + "|" + p.Group }

func (c *countingConfigClient) ListenConfig(p vo.ConfigParam) error {
	c.mu.Lock()
	c.slot[c.keyOf(p)] = struct{}{}
	c.mu.Unlock()
	atomic.AddInt32(&c.listens, 1)
	if c.onListenDone != nil {
		c.onListenDone()
	}
	return nil
}

func (c *countingConfigClient) CancelListenConfig(p vo.ConfigParam) error {
	if c.onCancelEnter != nil {
		c.onCancelEnter()
	}
	c.mu.Lock()
	delete(c.slot, c.keyOf(p))
	c.mu.Unlock()
	atomic.AddInt32(&c.cancels, 1)
	return nil
}

// hasSubscription reports whether a live subscription occupies the slot.
func (c *countingConfigClient) hasSubscription(key, group string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.slot[key+"|"+group]
	return ok
}

func (c *countingConfigClient) GetConfig(vo.ConfigParam) (string, error)   { return "", nil }
func (c *countingConfigClient) PublishConfig(vo.ConfigParam) (bool, error) { return true, nil }
func (c *countingConfigClient) DeleteConfig(vo.ConfigParam) (bool, error)  { return true, nil }
func (c *countingConfigClient) SearchConfig(vo.SearchConfigParam) (*model.ConfigPage, error) {
	return nil, nil
}
func (c *countingConfigClient) CloseClient() {}

// newTestConfig builds a nacosDynamicConfiguration wired to a counting client.
func newTestConfig(t *testing.T, client *countingConfigClient) *nacosDynamicConfiguration {
	t.Helper()
	nc := &nacosClient.NacosConfigClient{}
	nc.SetClient(client)
	u, err := common.NewURL("registry://127.0.0.1:8848",
		common.WithParamsValue(constant.NacosGroupKey, "test-group"))
	assert.NoError(t, err)
	return &nacosDynamicConfiguration{
		url:    u,
		client: nc,
	}
}

// TestAddRemoveListenerConcurrentChurnKeepsPersistentSubscription verifies the
// non-empty listener path under concurrent churn. A persistent listener remains
// registered while many temporary listeners are added and removed, so the key
// should never be canceled and the persistent subscription must remain live.
func TestAddRemoveListenerConcurrentChurnKeepsPersistentSubscription(t *testing.T) {
	const key = "race-key"
	client := newCountingConfigClient()
	n := newTestConfig(t, client)

	// A persistent listener that stays registered for the whole test; its
	// subscription must never be dropped by a racing remove's cancel.
	persistent := newNoopListener()
	n.addListener(key, persistent)
	assert.True(t, client.hasSubscription(key, "test-group"),
		"initial add should establish a subscription")

	var wg sync.WaitGroup
	workers := 32
	start := make(chan struct{})
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < 200; i++ {
				l := newNoopListener()
				n.addListener(key, l)
				n.removeListener(key, l)
			}
		}()
	}
	close(start)
	wg.Wait()

	// The persistent listener is still registered, so a live subscription must
	// remain. Under the unfixed code a racing cancel could have cleared the slot
	// after a worker's add repopulated it, leaving the slot empty while the
	// persistent listener is still registered — the assertion below catches
	// that. (Each worker balances its own add/remove, so no worker listener
	// should remain; only the persistent one.)
	assert.True(t, client.hasSubscription(key, "test-group"),
		"persistent listener's subscription must survive concurrent churn")
	assert.Equal(t, int32(0), atomic.LoadInt32(&client.cancels),
		"persistent listener keeps the key non-empty, so this test should not exercise cancel")

	// Sanity: the persistent listener is still the only one tracked for the key.
	rawSet, ok := n.keyListeners.Load(key)
	assert.True(t, ok, "key entry must remain while a listener is registered")
	set := rawSet.(*keyListenerSet)
	assert.Len(t, set.snapshot(), 1, "only the persistent listener should remain")
}

// TestRemoveThenAddNoLostSubscriptionDeterministic forces the exact interleaving
// the reviewer described, with no reliance on timing:
//
//  1. remover: removeListener observes the last listener leaving, deletes the
//     keyListeners entry, and enters CancelListenConfig — where we park it.
//  2. adder: while the remover is parked inside CancelListenConfig, addListener
//     runs: it sees no entry, stores a fresh set, and calls ListenConfig (which
//     populates the slot).
//  3. remover: we release CancelListenConfig; it clears the slot unconditionally
//     (mirroring the SDK's cacheMap.Remove).
//
// Under the unfixed code the adder's slot is wiped by the late cancel, leaving a
// registered listener with no live Nacos subscription. Under the fix the
// listener lifecycle lock makes step 2 wait until the remover has fully finished
// (including its CancelListenConfig), so the adder's ListenConfig wins and the
// slot survives.
func TestRemoveThenAddNoLostSubscriptionDeterministic(t *testing.T) {
	const key = "det-key"

	// cancelEntered is closed once the remover is parked inside CancelListenConfig;
	// cancelRelease unblocks it to finish clearing the slot.
	cancelEntered := make(chan struct{})
	cancelRelease := make(chan struct{})

	client := newCountingConfigClient()
	client.onCancelEnter = func() {
		close(cancelEntered)
		<-cancelRelease
	}
	n := newTestConfig(t, client)

	// One registered listener so removeListener has a real last-listener to remove.
	first := newNoopListener()
	n.addListener(key, first)
	assert.True(t, client.hasSubscription(key, "test-group"))

	removerDone := make(chan struct{})
	go func() {
		defer close(removerDone)
		n.removeListener(key, first)
	}()

	// Wait until the remover is parked inside CancelListenConfig (i.e. it has
	// already deleted the keyListeners entry). This is the dangerous window.
	<-cancelEntered

	// Now the adder races in: under the fix it blocks on the listener lifecycle
	// lock held by the remover, so ListenConfig has NOT run yet. Under the
	// unfixed code the adder proceeds, stores a fresh set, and ListenConfig
	// populates the slot while the remover is still parked.
	var adderWG sync.WaitGroup
	adderWG.Add(1)
	go func() {
		defer adderWG.Done()
		l := newNoopListener()
		n.addListener(key, l)
	}()

	// Probe whether the adder could complete before we release the cancel. With
	// the lifecycle lock the adder is still blocked; without it the adder
	// finished and populated the slot. Either way we then release the cancel and
	// wait for both goroutines to finish.
	time.Sleep(200 * time.Millisecond)
	close(cancelRelease)

	<-removerDone
	adderWG.Wait()

	// The adder registered a listener, so a live subscription must remain. On the
	// unfixed code the late cancel wiped the adder's slot, leaving it empty —
	// this assertion fails there and passes with the fix.
	assert.True(t, client.hasSubscription(key, "test-group"),
		"adder's subscription must survive the racing remover's cancel")
	rawSet, ok := n.keyListeners.Load(key)
	assert.True(t, ok, "key entry must remain while the adder's listener is registered")
	assert.Len(t, rawSet.(*keyListenerSet).snapshot(), 1, "only the adder's listener should remain")
}

// TestListenerConcurrentStress runs many goroutines adding and removing
// listeners across several keys. Run with -race to catch data races. The
// invariant checked at the end: for every key, a live Nacos subscription
// exists iff at least one listener remains registered. Under the unfixed code
// the TOCTOU window can leave a key with registered listeners but no slot
// (or, less likely, an orphaned slot after all listeners are gone).
func TestListenerConcurrentStress(t *testing.T) {
	client := newCountingConfigClient()
	n := newTestConfig(t, client)

	const numKeys = 8
	const goroutines = 64
	const iterations = 200

	// remaining[k] is the number of listeners we intentionally left registered
	// for key k (those we did not remove).
	var remaining [numKeys]int32

	keyFor := func(k int) string { return "stress-key-" + strconv.Itoa(k) }

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				k := g % numKeys
				key := keyFor(k)
				l := newNoopListener()
				n.addListener(key, l)
				// Keep roughly half of the listeners so some keys retain
				// subscriptions and others churn to empty (exercising the
				// cancel path) and back.
				if (i+g)%2 == 0 {
					n.removeListener(key, l)
				} else {
					atomic.AddInt32(&remaining[k], 1)
				}
			}
		}(g)
	}
	wg.Wait()

	// A live subscription must exist iff at least one listener remains.
	for k := 0; k < numKeys; k++ {
		key := keyFor(k)
		got := client.hasSubscription(key, "test-group")
		want := atomic.LoadInt32(&remaining[k]) > 0
		assert.Equal(t, want, got,
			"key %s: subscription live=%v but remaining listeners=%d", key, got, remaining[k])
	}
}
