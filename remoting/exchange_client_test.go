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
	return m.connectErr
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
	assert.False(t, ec.init)
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
