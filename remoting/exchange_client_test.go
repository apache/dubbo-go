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

package remoting

import (
	"context"
	"errors"
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
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type mockClient struct {
	mu                    sync.Mutex
	available             bool
	connectErr            error
	connCount             int
	requestErr            error
	contextRequestStarted chan struct{}
	blockContextRequest   bool
	connectFunc           func() error
}

func (m *mockClient) SetExchangeClient(client *ExchangeClient) {}
func (m *mockClient) Close()                                   {}
func (m *mockClient) IsAvailable() bool                        { m.mu.Lock(); defer m.mu.Unlock(); return m.available }

func (m *mockClient) Request(request *Request, timeout time.Duration, response *PendingResponse) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestErr
}

func (m *mockClient) RequestContext(ctx context.Context, request *Request, timeout time.Duration, response *PendingResponse) error {
	if m.contextRequestStarted != nil {
		close(m.contextRequestStarted)
		m.contextRequestStarted = nil
	}
	if m.blockContextRequest {
		<-ctx.Done()
		return ctx.Err()
	}
	return m.Request(request, timeout, response)
}

func (m *mockClient) Connect(url *common.URL) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connCount++
	if m.connectFunc != nil {
		return m.connectFunc()
	}
	return m.connectErr
}

// initWaitMutex signals when a waiter releases the condition lock in Wait.
type initWaitMutex struct {
	sync.Mutex
	once    sync.Once
	waiting chan struct{}
}

func (m *initWaitMutex) Unlock() {
	m.once.Do(func() { close(m.waiting) })
	m.Mutex.Unlock()
}

func waitForInitSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal(message)
	}
}

func waitForInitGoroutines(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	waitForInitSignal(t, done, "timed out waiting for concurrent doInit calls to finish")
}

func testURL() *common.URL {
	return common.NewURLWithOptions(common.WithProtocol("dubbo"), common.WithIp("127.0.0.1"), common.WithPort("20880"))
}

func TestNewExchangeClient(t *testing.T) {
	t.Run("eager init", func(t *testing.T) {
		m := &mockClient{available: true}
		ec := NewExchangeClient(testURL(), m, 5*time.Second, false)
		assert.NotNil(t, ec)
		assert.Positive(t, m.connCount)
	})

	t.Run("lazy init", func(t *testing.T) {
		m := &mockClient{available: true}
		ec := NewExchangeClient(testURL(), m, 5*time.Second, true)
		assert.NotNil(t, ec)
		assert.Equal(t, 0, m.connCount)
	})

	t.Run("connect fail", func(t *testing.T) {
		m := &mockClient{connectErr: errors.New("fail")}
		assert.Nil(t, NewExchangeClient(testURL(), m, 5*time.Second, false))
	})
}

func TestExchangeClientActiveNumber(t *testing.T) {
	ec := NewExchangeClient(testURL(), &mockClient{available: true}, 5*time.Second, true)
	assert.Equal(t, uint32(1), ec.GetActiveNumber())
	ec.IncreaseActiveNumber()
	assert.Equal(t, uint32(2), ec.GetActiveNumber())
	ec.DecreaseActiveNumber()
	assert.Equal(t, uint32(1), ec.GetActiveNumber())
}

func TestExchangeClientClose(t *testing.T) {
	m := &mockClient{available: true}
	ec := NewExchangeClient(testURL(), m, 5*time.Second, true)
	ec.Close()
	require.ErrorIs(t, ec.doInit(testURL()), base.ErrClientClosed)
	assert.Zero(t, m.connCount, "a closed client must not connect")
}

func TestExchangeClientIsAvailable(t *testing.T) {
	m := &mockClient{available: true}
	ec := NewExchangeClient(testURL(), m, 5*time.Second, true)
	assert.True(t, ec.IsAvailable())
	m.mu.Lock()
	m.available = false
	m.mu.Unlock()
	assert.False(t, ec.IsAvailable())
}

func TestExchangeClientConcurrentDoInit(t *testing.T) {
	// Verify that concurrent calls to doInit only result in a single Connect call.
	m := &mockClient{available: true}
	ec := NewExchangeClient(testURL(), m, 5*time.Second, true)

	var wg sync.WaitGroup
	const goroutines = 20
	errCh := make(chan error, goroutines)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			errCh <- ec.doInit(testURL())
		}()
	}
	waitForInitGoroutines(t, &wg)
	close(errCh)

	assert.Len(t, errCh, goroutines, "all doInit calls should finish")
	for err := range errCh {
		require.NoError(t, err)
	}

	m.mu.Lock()
	count := m.connCount
	m.mu.Unlock()
	assert.Equal(t, 1, count, "Connect should be called exactly once despite concurrent doInit")
	assert.Equal(t, initInitialized, ec.initState.Load(), "client should be initialized after doInit")
}

func TestExchangeClientDoInitFailureResets(t *testing.T) {
	// Verify that a failed doInit resets the init state so future calls can retry.
	m := &mockClient{connectErr: errors.New("fail")}
	ec := NewExchangeClient(testURL(), m, 5*time.Second, true)

	err := ec.doInit(testURL())
	require.Error(t, err)
	assert.Equal(t, initUninitialized, ec.initState.Load(), "client should be uninitialized after failed doInit")

	// Now make Connect succeed and retry
	m.mu.Lock()
	m.connectErr = nil
	m.mu.Unlock()
	err = ec.doInit(testURL())
	require.NoError(t, err)
	assert.Equal(t, initInitialized, ec.initState.Load(), "client should be initialized after successful retry")
}

func TestExchangeClientConcurrentDoInitPropagatesError(t *testing.T) {
	// Verify that when initialization fails, all concurrent waiters receive the same error.
	m := &mockClient{connectErr: errors.New("connect refused")}
	ec := NewExchangeClient(testURL(), m, 5*time.Second, true)

	var wg sync.WaitGroup
	const goroutines = 10
	errCh := make(chan error, goroutines)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			errCh <- ec.doInit(testURL())
		}()
	}
	waitForInitGoroutines(t, &wg)
	close(errCh)

	errorCount := 0
	for err := range errCh {
		if err != nil {
			errorCount++
		}
	}
	assert.Equal(t, goroutines, errorCount, "all goroutines should receive the init error")
	assert.Equal(t, initUninitialized, ec.initState.Load(), "client should be uninitialized after failure")
}

func TestExchangeClientCloseDuringInit(t *testing.T) {
	connectStarted := make(chan struct{})
	releaseConnect := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseConnect) }) }
	defer release()

	m := &mockClient{connectFunc: func() error {
		close(connectStarted)
		<-releaseConnect
		return nil
	}}
	ec := NewExchangeClient(testURL(), m, 5*time.Second, true)
	waiting := make(chan struct{})
	ec.initCond = sync.NewCond(&initWaitMutex{waiting: waiting})

	initErr := make(chan error, 1)
	initDone := make(chan struct{})
	go func() {
		defer close(initDone)
		initErr <- ec.doInit(testURL())
	}()
	waitForInitSignal(t, connectStarted, "Connect did not start")

	waiterErr := make(chan error, 1)
	waiterDone := make(chan struct{})
	go func() {
		defer close(waiterDone)
		waiterErr <- ec.doInit(testURL())
	}()
	waitForInitSignal(t, waiting, "concurrent caller did not wait for initialization")

	ec.Close()
	waitForInitSignal(t, waiterDone, "Close did not release the initialization waiter")
	waiterResult := <-waiterErr
	ec.Close()
	release()
	waitForInitSignal(t, initDone, "initializer remained blocked after Connect returned")
	require.ErrorIs(t, waiterResult, base.ErrClientClosed)
	require.ErrorIs(t, <-initErr, base.ErrClientClosed, "late Connect success must not undo Close")
	require.ErrorIs(t, ec.doInit(testURL()), base.ErrClientClosed)
	m.mu.Lock()
	assert.Equal(t, 1, m.connCount, "a closed client must not reconnect")
	m.mu.Unlock()
}

func TestExchangeClientDoInitPanicReleasesWaiters(t *testing.T) {
	for _, tc := range []struct {
		name        string
		panicOnCall int
	}{
		{name: "first connect", panicOnCall: 1},
		{name: "retry connect", panicOnCall: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			connectStarted := make(chan struct{})
			releaseConnect := make(chan struct{})
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(releaseConnect) }) }
			defer release()

			panicValue := errors.New("transport panic")
			m := &mockClient{}
			m.connectFunc = func() error {
				if m.connCount < tc.panicOnCall {
					return errors.New("retry connection")
				}
				close(connectStarted)
				<-releaseConnect
				panic(panicValue)
			}
			ec := NewExchangeClient(testURL(), m, 5*time.Second, true)
			waiting := make(chan struct{})
			ec.initCond = sync.NewCond(&initWaitMutex{waiting: waiting})

			panicCh := make(chan any, 1)
			initDone := make(chan struct{})
			go func() {
				defer close(initDone)
				defer func() { panicCh <- recover() }()
				_ = ec.doInit(testURL())
			}()
			waitForInitSignal(t, connectStarted, "Connect did not start")

			errCh := make(chan error, 1)
			waiterDone := make(chan struct{})
			go func() {
				defer close(waiterDone)
				errCh <- ec.doInit(testURL())
			}()
			// Wait registers the waiter before unlocking, so no sleeps are needed.
			waitForInitSignal(t, waiting, "concurrent caller did not wait for initialization")
			release()

			waitForInitSignal(t, initDone, "initializer did not propagate the panic")
			assert.Same(t, panicValue, <-panicCh, "initializer should preserve the original panic")
			waitForInitSignal(t, waiterDone, "waiter remained blocked after Connect panicked")
			require.EqualError(t, <-errCh, "exchange client init panic: transport panic")
			assert.Equal(t, initUninitialized, ec.initState.Load())

			m.mu.Lock()
			assert.Equal(t, tc.panicOnCall, m.connCount)
			m.connectFunc = nil
			m.mu.Unlock()
			require.NoError(t, ec.doInit(testURL()), "initialization should be retryable after panic")
			assert.Equal(t, initInitialized, ec.initState.Load())
			m.mu.Lock()
			assert.Equal(t, tc.panicOnCall+1, m.connCount)
			m.mu.Unlock()
		})
	}
}

// newTestInvocation builds a *base.Invocation usable by ExchangeClient.Request/AsyncRequest.
func newTestInvocation() *base.Invocation {
	var inv base.Invocation = invocation.NewRPCInvocation("test", nil, nil)
	return &inv
}

// TestExchangeClientRequestErrorCleanup is a regression test for the map leak fix:
// when the underlying client.Request returns an error, the pending response for the
// request ID must be removed from pendingResponses so the global map does not leak.
func TestExchangeClientRequestErrorCleanup(t *testing.T) {
	m := &mockClient{available: true, requestErr: errors.New("request failed")}
	ec := NewExchangeClient(testURL(), m, 5*time.Second, true)

	before := countPendingResponses()
	res := &result.RPCResult{}
	err := ec.Request(newTestInvocation(), testURL(), time.Second, res)

	require.Error(t, err)
	assert.Equal(t, err, res.Err)
	// No pending response should be left behind on the error path.
	assert.Equal(t, before, countPendingResponses(), "pendingResponses leaked on Request error path")
}

// TestExchangeClientAsyncRequestErrorCleanup mirrors the sync case for AsyncRequest.
func TestExchangeClientAsyncRequestErrorCleanup(t *testing.T) {
	m := &mockClient{available: true, requestErr: errors.New("request failed")}
	ec := NewExchangeClient(testURL(), m, 5*time.Second, true)

	before := countPendingResponses()
	res := &result.RPCResult{}
	cb := func(response common.CallbackResponse) {}
	err := ec.AsyncRequest(newTestInvocation(), testURL(), time.Second, cb, res)

	require.Error(t, err)
	assert.Equal(t, err, res.Err)
	assert.Equal(t, before, countPendingResponses(), "pendingResponses leaked on AsyncRequest error path")
}

func TestExchangeClientRequestContextCancellationCleanup(t *testing.T) {
	requestStarted := make(chan struct{})
	m := &mockClient{
		available:             true,
		contextRequestStarted: requestStarted,
		blockContextRequest:   true,
	}
	ec := NewExchangeClient(testURL(), m, 5*time.Second, true)

	before := countPendingResponses()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	res := &result.RPCResult{}
	errCh := make(chan error, 1)
	go func() {
		errCh <- ec.RequestContext(ctx, newTestInvocation(), testURL(), time.Minute, res)
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("request was not sent to the transport")
	}
	cancel()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorIs(t, res.Err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("request did not stop after context cancellation")
	}
	assert.Equal(t, before, countPendingResponses(), "pendingResponses leaked after context cancellation")
}

// countPendingResponses returns the number of entries currently held in the global
// pendingResponses map.
func countPendingResponses() int {
	n := 0
	pendingResponses.Range(func(_, _ any) bool {
		n++
		return true
	})
	return n
}
