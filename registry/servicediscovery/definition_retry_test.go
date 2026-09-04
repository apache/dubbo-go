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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

// definitionPublishStub records calls and replays a scripted outcome.
type definitionPublishStub struct {
	mu sync.Mutex
	// failUntil is the number of leading calls that report failure.
	failUntil int
	calls     int
	done      chan struct{}
	// signalAt fires done once this many calls have been made.
	signalAt int
}

func (p *definitionPublishStub) publish(_ report.MetadataReport, urls []*common.URL) []*common.URL {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	if p.signalAt > 0 && p.calls == p.signalAt && p.done != nil {
		close(p.done)
	}
	if p.calls <= p.failUntil {
		return urls
	}
	return nil
}

func (p *definitionPublishStub) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

// installDefinitionPublishStub swaps the publish seam and shrinks the backoff so
// the retry loop runs in test time rather than minutes.
func installDefinitionPublishFunc(
	t *testing.T,
	publish func(report.MetadataReport, []*common.URL) []*common.URL,
) {
	t.Helper()

	prevPublish := publishDefinitions
	prevInitial := definitionRetryInitialDelay
	prevMax := definitionRetryMaxDelay
	prevAttempts := definitionRetryMaxAttempts

	publishDefinitions = publish
	definitionRetryInitialDelay = time.Millisecond
	definitionRetryMaxDelay = 4 * time.Millisecond

	t.Cleanup(func() {
		publishDefinitions = prevPublish
		definitionRetryInitialDelay = prevInitial
		definitionRetryMaxDelay = prevMax
		definitionRetryMaxAttempts = prevAttempts
	})
}

func installDefinitionPublishStub(t *testing.T, stub *definitionPublishStub) {
	t.Helper()
	installDefinitionPublishFunc(t, stub.publish)
}

// newDefinitionRetryRegistry builds a registry and cancels whatever retries it
// still holds when the test ends. Without that, a pending timer outlives its
// test and lands on the next test's stub — which is exactly the leak
// cancelDefinitionRetriesLocked exists to prevent in production.
func newDefinitionRetryRegistry(t *testing.T) *serviceDiscoveryRegistry {
	t.Helper()
	reg := &serviceDiscoveryRegistry{
		url:               common.NewURLWithOptions(),
		subscribeRetries:  make(map[string]*subscribeRetry),
		definitionRetries: make(map[string]*definitionRetry),
	}
	t.Cleanup(func() {
		reg.lock.Lock()
		reg.destroyed = true
		reg.cancelDefinitionRetriesLocked()
		reg.cancelDefinitionPublishLocked()
		reg.lock.Unlock()
	})
	return reg
}

func definitionTestURL(iface string) *common.URL {
	return common.NewURLWithOptions(
		common.WithProtocol(constant.DubboProtocol),
		common.WithParamsValue(constant.InterfaceKey, iface),
	)
}

func (s *serviceDiscoveryRegistry) pendingDefinitionRetries() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.definitionRetries)
}

// TestDefinitionRetrySucceedsAfterTransientFailure covers the case this
// mechanism exists for: the metadata center is briefly unavailable while the
// provider starts, and the definition lands on a later attempt instead of
// waiting for the next daily pass.
func TestDefinitionRetrySucceedsAfterTransientFailure(t *testing.T) {
	stub := &definitionPublishStub{failUntil: 2, signalAt: 3, done: make(chan struct{})}
	installDefinitionPublishStub(t, stub)

	reg := newDefinitionRetryRegistry(t)
	reg.publishServiceDefinitions([]*common.URL{definitionTestURL("org.test.Retry")})

	select {
	case <-stub.done:
	case <-time.After(5 * time.Second):
		t.Fatal("definition publish was never retried to success")
	}

	assert.Eventually(t, func() bool { return reg.pendingDefinitionRetries() == 0 },
		2*time.Second, 5*time.Millisecond,
		"a successful retry must drop its state instead of rescheduling")
}

// TestDefinitionRetryGivesUpAfterMaxAttempts pins the bound. Retrying forever
// would only add noise to a failure the daily cycle report already owns.
func TestDefinitionRetryGivesUpAfterMaxAttempts(t *testing.T) {
	stub := &definitionPublishStub{failUntil: 1 << 30}
	installDefinitionPublishStub(t, stub)
	definitionRetryMaxAttempts = 3

	reg := newDefinitionRetryRegistry(t)
	reg.publishServiceDefinitions([]*common.URL{definitionTestURL("org.test.Doomed")})

	assert.Eventually(t, func() bool { return reg.pendingDefinitionRetries() == 0 },
		2*time.Second, 5*time.Millisecond, "the retry must eventually give up")

	// One initial publish plus definitionRetryMaxAttempts retries.
	assert.Equal(t, 1+3, stub.callCount())
}

// TestDefinitionRetryIsOnePerService keeps repeated failures from stacking
// timers, which would both multiply the load and reset the backoff.
func TestDefinitionRetryIsOnePerService(t *testing.T) {
	stub := &definitionPublishStub{failUntil: 1 << 30}
	installDefinitionPublishStub(t, stub)
	definitionRetryMaxAttempts = 1 << 30 // keep the entry pending for the assertion

	reg := newDefinitionRetryRegistry(t)
	u := definitionTestURL("org.test.Duplicate")

	for range 5 {
		reg.scheduleDefinitionRetry(u)
	}
	assert.Equal(t, 1, reg.pendingDefinitionRetries())
}

func TestDefinitionRetryTracksServicesSeparately(t *testing.T) {
	stub := &definitionPublishStub{failUntil: 1 << 30}
	installDefinitionPublishStub(t, stub)
	definitionRetryMaxAttempts = 1 << 30

	reg := newDefinitionRetryRegistry(t)
	reg.scheduleDefinitionRetry(definitionTestURL("org.test.A"))
	reg.scheduleDefinitionRetry(definitionTestURL("org.test.B"))

	assert.Equal(t, 2, reg.pendingDefinitionRetries())
}

// TestDefinitionRetryCancelledOnDestroy guards against a timer outliving the
// registry it belongs to.
func TestDefinitionRetryCancelledOnDestroy(t *testing.T) {
	stub := &definitionPublishStub{failUntil: 1 << 30}
	installDefinitionPublishStub(t, stub)
	definitionRetryMaxAttempts = 1 << 30
	// Long enough that the first timer cannot have fired before the cancel
	// below, so the assertion measures cancellation rather than timing.
	definitionRetryInitialDelay = 300 * time.Millisecond
	definitionRetryMaxDelay = time.Second

	reg := newDefinitionRetryRegistry(t)
	reg.scheduleDefinitionRetry(definitionTestURL("org.test.Destroyed"))
	require.Equal(t, 1, reg.pendingDefinitionRetries())
	require.Equal(t, 0, stub.callCount(), "scheduling alone must not publish")

	reg.lock.Lock()
	reg.destroyed = true
	reg.cancelDefinitionRetriesLocked()
	reg.lock.Unlock()

	assert.Equal(t, 0, reg.pendingDefinitionRetries())

	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, 0, stub.callCount(), "a canceled retry must not fire")
}

// TestDefinitionRetryNotArmedAfterDestroy covers the race where a publish
// started before Destroy reports its failure after it.
func TestDefinitionRetryNotArmedAfterDestroy(t *testing.T) {
	stub := &definitionPublishStub{failUntil: 1 << 30}
	installDefinitionPublishStub(t, stub)

	reg := newDefinitionRetryRegistry(t)
	reg.lock.Lock()
	reg.destroyed = true
	reg.lock.Unlock()

	reg.scheduleDefinitionRetry(definitionTestURL("org.test.Late"))
	assert.Equal(t, 0, reg.pendingDefinitionRetries())
}

func TestDefinitionRetryDelayBackoff(t *testing.T) {
	prevInitial, prevMax := definitionRetryInitialDelay, definitionRetryMaxDelay
	definitionRetryInitialDelay = time.Second
	definitionRetryMaxDelay = 30 * time.Second
	defer func() {
		definitionRetryInitialDelay, definitionRetryMaxDelay = prevInitial, prevMax
	}()

	// Jitter is up to 25% on top, so each attempt lands in [base, base*1.25].
	for attempt, base := range map[int]time.Duration{
		0: time.Second,
		1: 2 * time.Second,
		2: 4 * time.Second,
	} {
		delay := definitionRetryDelay(attempt)
		assert.GreaterOrEqual(t, delay, base, "attempt %d", attempt)
		assert.LessOrEqual(t, delay, base+base/4, "attempt %d", attempt)
	}

	// Far past the cap, including values that would overflow a naive shift.
	for _, attempt := range []int{10, 29, 30, 1000} {
		delay := definitionRetryDelay(attempt)
		assert.GreaterOrEqual(t, delay, definitionRetryMaxDelay, "attempt %d", attempt)
		assert.LessOrEqual(t, delay, definitionRetryMaxDelay+definitionRetryMaxDelay/4,
			"attempt %d", attempt)
	}
}

// TestPublishServiceDefinitionsArmsNoRetryOnSuccess keeps the happy path free of
// timers.
func TestPublishServiceDefinitionsArmsNoRetryOnSuccess(t *testing.T) {
	stub := &definitionPublishStub{}
	installDefinitionPublishStub(t, stub)

	reg := newDefinitionRetryRegistry(t)
	reg.publishServiceDefinitions([]*common.URL{definitionTestURL("org.test.Fine")})

	assert.Equal(t, 1, stub.callCount())
	assert.Equal(t, 0, reg.pendingDefinitionRetries())
}

type serializedDefinitionPublishStub struct {
	mu           sync.Mutex
	calls        int
	active       int
	maxActive    int
	reports      []report.MetadataReport
	batches      [][]*common.URL
	firstStarted chan struct{}
	releaseFirst chan struct{}
	secondDone   chan struct{}
}

func (p *serializedDefinitionPublishStub) publish(
	r report.MetadataReport,
	urls []*common.URL,
) []*common.URL {
	p.mu.Lock()
	p.calls++
	call := p.calls
	p.active++
	if p.active > p.maxActive {
		p.maxActive = p.active
	}
	p.reports = append(p.reports, r)
	p.batches = append(p.batches, append([]*common.URL(nil), urls...))
	if call == 1 {
		close(p.firstStarted)
	}
	p.mu.Unlock()

	if call == 1 {
		<-p.releaseFirst
	}

	p.mu.Lock()
	p.active--
	if call == 2 {
		close(p.secondDone)
	}
	p.mu.Unlock()
	return nil
}

func (p *serializedDefinitionPublishStub) snapshot() (
	calls int,
	maxActive int,
	reports []report.MetadataReport,
	batches [][]*common.URL,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	batches = make([][]*common.URL, len(p.batches))
	for i := range p.batches {
		batches[i] = append([]*common.URL(nil), p.batches[i]...)
	}
	return p.calls, p.maxActive, append([]report.MetadataReport(nil), p.reports...), batches
}

func TestDefinitionPublishWorkerSerializesAndCoalesces(t *testing.T) {
	stub := &serializedDefinitionPublishStub{
		firstStarted: make(chan struct{}),
		releaseFirst: make(chan struct{}),
		secondDone:   make(chan struct{}),
	}
	installDefinitionPublishFunc(t, stub.publish)

	reportA := &mockMetadataReportForGC{}
	reg := newDefinitionRetryRegistry(t)
	reg.metadataReport = reportA
	first := definitionTestURL("org.test.First")
	second := definitionTestURL("org.test.Second")
	latest := definitionTestURL("org.test.Latest")

	reg.scheduleServiceDefinitionPublish([]*common.URL{first})
	select {
	case <-stub.firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("initial definition publish did not start")
	}

	// Both calls must return while the first backend write is blocked. The
	// worker keeps only the latest full snapshot for the next pass.
	reg.scheduleServiceDefinitionPublish([]*common.URL{second})
	reg.scheduleServiceDefinitionPublish([]*common.URL{latest})
	close(stub.releaseFirst)

	select {
	case <-stub.secondDone:
	case <-time.After(2 * time.Second):
		t.Fatal("coalesced definition publish did not run")
	}

	assert.Eventually(t, func() bool {
		reg.lock.RLock()
		defer reg.lock.RUnlock()
		return !reg.definitionPublishRunning
	}, 2*time.Second, 5*time.Millisecond)

	calls, maxActive, reports, batches := stub.snapshot()
	assert.Equal(t, 2, calls)
	assert.Equal(t, 1, maxActive, "one registry must never write definitions concurrently")
	assert.Equal(t, []report.MetadataReport{reportA, reportA}, reports)
	require.Len(t, batches, 2)
	assert.Equal(t, []*common.URL{first}, batches[0])
	assert.Equal(t, []*common.URL{latest}, batches[1], "intermediate full snapshots should be coalesced")
}

func TestDefinitionPublishWorkerDropsPendingWorkOnDestroy(t *testing.T) {
	stub := &serializedDefinitionPublishStub{
		firstStarted: make(chan struct{}),
		releaseFirst: make(chan struct{}),
		secondDone:   make(chan struct{}),
	}
	installDefinitionPublishFunc(t, stub.publish)

	reg := newDefinitionRetryRegistry(t)
	reg.scheduleServiceDefinitionPublish([]*common.URL{definitionTestURL("org.test.InFlight")})
	select {
	case <-stub.firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("initial definition publish did not start")
	}
	reg.scheduleServiceDefinitionPublish([]*common.URL{definitionTestURL("org.test.Pending")})

	reg.lock.Lock()
	reg.destroyed = true
	reg.cancelDefinitionPublishLocked()
	reg.lock.Unlock()
	close(stub.releaseFirst)

	assert.Eventually(t, func() bool {
		reg.lock.RLock()
		defer reg.lock.RUnlock()
		return !reg.definitionPublishRunning
	}, 2*time.Second, 5*time.Millisecond)
	calls, maxActive, _, _ := stub.snapshot()
	assert.Equal(t, 1, calls, "destroy must discard a full publish that has not started")
	assert.Equal(t, 1, maxActive)
}

func TestDefinitionRetrySerializesWithFullPublish(t *testing.T) {
	stub := &serializedDefinitionPublishStub{
		firstStarted: make(chan struct{}),
		releaseFirst: make(chan struct{}),
		secondDone:   make(chan struct{}),
	}
	installDefinitionPublishFunc(t, stub.publish)

	reg := newDefinitionRetryRegistry(t)
	reg.scheduleServiceDefinitionPublish([]*common.URL{definitionTestURL("org.test.Full")})
	select {
	case <-stub.firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("full definition publish did not start")
	}

	retryURL := definitionTestURL("org.test.RetryWhileFull")
	key := retryURL.ServiceKey()
	reg.lock.Lock()
	reg.definitionRetries[key] = &definitionRetry{url: retryURL}
	reg.lock.Unlock()
	go reg.retryPublishDefinition(key)

	// The retry goroutine may be waiting, but it must not enter the backend
	// while the full publish is in flight.
	assert.Eventually(t, func() bool {
		reg.lock.RLock()
		defer reg.lock.RUnlock()
		return reg.definitionRetries[key].inFlight
	}, 2*time.Second, 5*time.Millisecond)
	calls, maxActive, _, _ := stub.snapshot()
	assert.Equal(t, 1, calls)
	assert.Equal(t, 1, maxActive)

	close(stub.releaseFirst)
	select {
	case <-stub.secondDone:
	case <-time.After(2 * time.Second):
		t.Fatal("serialized definition retry did not run")
	}
	calls, maxActive, _, _ = stub.snapshot()
	assert.Equal(t, 2, calls)
	assert.Equal(t, 1, maxActive)
}
