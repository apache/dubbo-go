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
	"errors"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// mockClient implements Client interface for testing
type mockClient struct {
	mu           sync.Mutex
	connected    bool
	available    bool
	connectErr   error
	connectCount int
	closeCount   int
}

func newMockClient() *mockClient {
	return &mockClient{available: true}
}

func (m *mockClient) SetExchangeClient(client *ExchangeClient) {}

func (m *mockClient) Connect(url *common.URL) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectCount++
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected = true
	return nil
}

func (m *mockClient) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCount++
	m.connected = false
}

func (m *mockClient) Request(request *Request, timeout time.Duration, response *PendingResponse) error {
	return nil
}

func (m *mockClient) IsAvailable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.available
}

// Compile-time check
var _ Client = (*mockClient)(nil)

func createTestURL() *common.URL {
	return common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
}

func TestNewExchangeClient(t *testing.T) {
	t.Run("eager init", func(t *testing.T) {
		mockC := newMockClient()
		ec := NewExchangeClient(createTestURL(), mockC, 5*time.Second, false)
		assert.NotNil(t, ec)
		assert.True(t, mockC.connectCount > 0)
	})

	t.Run("lazy init", func(t *testing.T) {
		mockC := newMockClient()
		ec := NewExchangeClient(createTestURL(), mockC, 5*time.Second, true)
		assert.NotNil(t, ec)
		assert.Equal(t, 0, mockC.connectCount)
	})

	t.Run("connect fail", func(t *testing.T) {
		mockC := newMockClient()
		mockC.connectErr = errors.New("connection failed")
		ec := NewExchangeClient(createTestURL(), mockC, 5*time.Second, false)
		assert.Nil(t, ec)
	})
}

func TestExchangeClientActiveNumber(t *testing.T) {
	mockC := newMockClient()
	ec := NewExchangeClient(createTestURL(), mockC, 5*time.Second, true)

	assert.Equal(t, uint32(1), ec.GetActiveNumber())

	ec.IncreaseActiveNumber()
	assert.Equal(t, uint32(2), ec.GetActiveNumber())

	ec.DecreaseActiveNumber()
	assert.Equal(t, uint32(1), ec.GetActiveNumber())
}

func TestExchangeClientActiveNumberConcurrent(t *testing.T) {
	mockC := newMockClient()
	ec := NewExchangeClient(createTestURL(), mockC, 5*time.Second, true)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ec.IncreaseActiveNumber()
		}()
	}
	wg.Wait()

	assert.Equal(t, uint32(101), ec.GetActiveNumber())
}

func TestExchangeClientClose(t *testing.T) {
	mockC := newMockClient()
	ec := NewExchangeClient(createTestURL(), mockC, 5*time.Second, true)

	ec.Close()
	assert.Equal(t, 1, mockC.closeCount)
	assert.False(t, ec.init)
}

func TestExchangeClientIsAvailable(t *testing.T) {
	mockC := newMockClient()
	ec := NewExchangeClient(createTestURL(), mockC, 5*time.Second, true)

	assert.True(t, ec.IsAvailable())

	mockC.mu.Lock()
	mockC.available = false
	mockC.mu.Unlock()

	assert.False(t, ec.IsAvailable())
}

func TestExchangeClientDoInitRetry(t *testing.T) {
	// Mock that fails first then succeeds
	retryMock := &retryMockClient{failCount: 1}

	ec := &ExchangeClient{
		ConnectTimeout: 5 * time.Second,
		address:        "127.0.0.1:20880",
		client:         retryMock,
		init:           false,
	}

	err := ec.doInit(createTestURL())
	assert.Nil(t, err)
	assert.True(t, ec.init)
	assert.Equal(t, 2, retryMock.currentCount)
}

type retryMockClient struct {
	failCount    int
	currentCount int
	mu           sync.Mutex
}

func (r *retryMockClient) SetExchangeClient(client *ExchangeClient) {}
func (r *retryMockClient) Close()                                   {}
func (r *retryMockClient) IsAvailable() bool                        { return true }

func (r *retryMockClient) Request(request *Request, timeout time.Duration, response *PendingResponse) error {
	return nil
}

func (r *retryMockClient) Connect(url *common.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentCount++
	if r.currentCount <= r.failCount {
		return errors.New("connection failed")
	}
	return nil
}
