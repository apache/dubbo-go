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

// ============================================
// Mock Client Implementation
// ============================================

type mockClient struct {
	mu              sync.Mutex
	exchangeClient  *ExchangeClient
	connected       bool
	available       bool
	connectErr      error
	requestErr      error
	connectCalled   int
	requestCalled   int
	closeCalled     int
	connectDelay    time.Duration
}

func newMockClient() *mockClient {
	return &mockClient{
		available: true,
	}
}

func (m *mockClient) SetExchangeClient(client *ExchangeClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exchangeClient = client
}

func (m *mockClient) Connect(url *common.URL) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectCalled++
	if m.connectDelay > 0 {
		time.Sleep(m.connectDelay)
	}
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected = true
	return nil
}

func (m *mockClient) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalled++
	m.connected = false
}

func (m *mockClient) Request(request *Request, timeout time.Duration, response *PendingResponse) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestCalled++
	if m.requestErr != nil {
		return m.requestErr
	}
	return nil
}

func (m *mockClient) IsAvailable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.available
}

func (m *mockClient) setConnectError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectErr = err
}

func (m *mockClient) setRequestError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestErr = err
}

func (m *mockClient) setAvailable(available bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.available = available
}

// ============================================
// NewExchangeClient Tests
// ============================================

func TestNewExchangeClient(t *testing.T) {
	tests := []struct {
		name           string
		lazyInit       bool
		connectTimeout time.Duration
		expectInit     bool
	}{
		{
			name:           "eager init",
			lazyInit:       false,
			connectTimeout: 5 * time.Second,
			expectInit:     true,
		},
		{
			name:           "lazy init",
			lazyInit:       true,
			connectTimeout: 10 * time.Second,
			expectInit:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := common.NewURLWithOptions(
				common.WithProtocol("dubbo"),
				common.WithIp("127.0.0.1"),
				common.WithPort("20880"),
			)
			mockC := newMockClient()

			ec := NewExchangeClient(url, mockC, tt.connectTimeout, tt.lazyInit)
			assert.NotNil(t, ec)
			assert.Equal(t, tt.connectTimeout, ec.ConnectTimeout)
			assert.Equal(t, "127.0.0.1:20880", ec.address)

			if tt.expectInit {
				assert.True(t, mockC.connectCalled > 0)
			}
		})
	}
}

func TestNewExchangeClientConnectFail(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockC := newMockClient()
	mockC.setConnectError(errors.New("connection failed"))

	ec := NewExchangeClient(url, mockC, 5*time.Second, false)
	assert.Nil(t, ec)
}

// ============================================
// ActiveNumber Tests
// ============================================

func TestExchangeClientActiveNumber(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockC := newMockClient()

	ec := NewExchangeClient(url, mockC, 5*time.Second, true)
	assert.NotNil(t, ec)

	// Initial active number should be 1
	assert.Equal(t, uint32(1), ec.GetActiveNumber())

	// Increase
	ec.IncreaseActiveNumber()
	assert.Equal(t, uint32(2), ec.GetActiveNumber())

	ec.IncreaseActiveNumber()
	assert.Equal(t, uint32(3), ec.GetActiveNumber())

	// Decrease
	ec.DecreaseActiveNumber()
	assert.Equal(t, uint32(2), ec.GetActiveNumber())

	ec.DecreaseActiveNumber()
	assert.Equal(t, uint32(1), ec.GetActiveNumber())
}

func TestExchangeClientActiveNumberConcurrent(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockC := newMockClient()

	ec := NewExchangeClient(url, mockC, 5*time.Second, true)
	assert.NotNil(t, ec)

	var wg sync.WaitGroup

	// Concurrent increase
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ec.IncreaseActiveNumber()
		}()
	}

	wg.Wait()
	assert.Equal(t, uint32(101), ec.GetActiveNumber()) // 1 initial + 100 increases
}

// ============================================
// Close Tests
// ============================================

func TestExchangeClientClose(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockC := newMockClient()

	ec := NewExchangeClient(url, mockC, 5*time.Second, true)
	assert.NotNil(t, ec)

	ec.Close()
	assert.Equal(t, 1, mockC.closeCalled)
	assert.False(t, ec.init)
}

// ============================================
// IsAvailable Tests
// ============================================

func TestExchangeClientIsAvailable(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockC := newMockClient()

	ec := NewExchangeClient(url, mockC, 5*time.Second, true)
	assert.NotNil(t, ec)

	assert.True(t, ec.IsAvailable())

	mockC.setAvailable(false)
	assert.False(t, ec.IsAvailable())
}

// ============================================
// doInit Tests
// ============================================

func TestExchangeClientDoInitAlreadyInit(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockC := newMockClient()

	ec := NewExchangeClient(url, mockC, 5*time.Second, false)
	assert.NotNil(t, ec)

	// Already initialized, doInit should return nil immediately
	initialConnectCalls := mockC.connectCalled
	err := ec.doInit(url)
	assert.Nil(t, err)
	assert.Equal(t, initialConnectCalls, mockC.connectCalled)
}

func TestExchangeClientDoInitRetry(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)

	// Use a special mock that fails first then succeeds
	retryMock := &retryMockClient{
		failCount:    1,
		currentCount: 0,
	}

	ec := &ExchangeClient{
		ConnectTimeout: 5 * time.Second,
		address:        url.Location,
		client:         retryMock,
		init:           false,
	}

	err := ec.doInit(url)
	assert.Nil(t, err)
	assert.True(t, ec.init)
	assert.Equal(t, 2, retryMock.currentCount) // Should have tried twice
}

// retryMockClient is a mock that fails a specified number of times before succeeding
type retryMockClient struct {
	failCount    int
	currentCount int
	mu           sync.Mutex
}

func (r *retryMockClient) SetExchangeClient(client *ExchangeClient) {}

func (r *retryMockClient) Connect(url *common.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentCount++
	if r.currentCount <= r.failCount {
		return errors.New("connection failed")
	}
	return nil
}

func (r *retryMockClient) Close() {}

func (r *retryMockClient) Request(request *Request, timeout time.Duration, response *PendingResponse) error {
	return nil
}

func (r *retryMockClient) IsAvailable() bool {
	return true
}

// ============================================
// Client Interface Tests
// ============================================

func TestClientInterfaceImplementation(t *testing.T) {
	mockC := newMockClient()

	// Verify mockClient implements Client interface
	var _ Client = mockC

	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)

	// Test SetExchangeClient
	ec := &ExchangeClient{}
	mockC.SetExchangeClient(ec)
	assert.Equal(t, ec, mockC.exchangeClient)

	// Test Connect
	err := mockC.Connect(url)
	assert.Nil(t, err)
	assert.True(t, mockC.connected)

	// Test IsAvailable
	assert.True(t, mockC.IsAvailable())

	// Test Close
	mockC.Close()
	assert.False(t, mockC.connected)
}

// ============================================
// Edge Cases Tests
// ============================================

func TestExchangeClientEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*common.URL, *mockClient)
		timeout  time.Duration
		lazyInit bool
		validate func(*testing.T, *ExchangeClient)
	}{
		{
			name: "zero connect timeout",
			setup: func() (*common.URL, *mockClient) {
				url := common.NewURLWithOptions(
					common.WithProtocol("dubbo"),
					common.WithIp("127.0.0.1"),
					common.WithPort("20880"),
				)
				return url, newMockClient()
			},
			timeout:  0,
			lazyInit: true,
			validate: func(t *testing.T, ec *ExchangeClient) {
				assert.NotNil(t, ec)
				assert.Equal(t, time.Duration(0), ec.ConnectTimeout)
			},
		},
		{
			name: "empty address",
			setup: func() (*common.URL, *mockClient) {
				url := common.NewURLWithOptions(
					common.WithProtocol("dubbo"),
				)
				return url, newMockClient()
			},
			timeout:  5 * time.Second,
			lazyInit: true,
			validate: func(t *testing.T, ec *ExchangeClient) {
				assert.NotNil(t, ec)
				assert.Equal(t, ":", ec.address)
			},
		},
		{
			name: "very long timeout",
			setup: func() (*common.URL, *mockClient) {
				url := common.NewURLWithOptions(
					common.WithProtocol("dubbo"),
					common.WithIp("127.0.0.1"),
					common.WithPort("20880"),
				)
				return url, newMockClient()
			},
			timeout:  24 * time.Hour,
			lazyInit: true,
			validate: func(t *testing.T, ec *ExchangeClient) {
				assert.NotNil(t, ec)
				assert.Equal(t, 24*time.Hour, ec.ConnectTimeout)
			},
		},
		{
			name: "ipv6 address",
			setup: func() (*common.URL, *mockClient) {
				url := common.NewURLWithOptions(
					common.WithProtocol("dubbo"),
					common.WithIp("::1"),
					common.WithPort("20880"),
				)
				return url, newMockClient()
			},
			timeout:  5 * time.Second,
			lazyInit: true,
			validate: func(t *testing.T, ec *ExchangeClient) {
				assert.NotNil(t, ec)
				assert.Contains(t, ec.address, "::1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, mockC := tt.setup()
			ec := NewExchangeClient(url, mockC, tt.timeout, tt.lazyInit)
			tt.validate(t, ec)
		})
	}
}

// ============================================
// URL Configuration Tests
// ============================================

func TestExchangeClientURLConfigurations(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		ip       string
		port     string
		params   map[string]string
	}{
		{
			name:     "dubbo protocol",
			protocol: "dubbo",
			ip:       "127.0.0.1",
			port:     "20880",
			params:   nil,
		},
		{
			name:     "triple protocol",
			protocol: "triple",
			ip:       "192.168.1.100",
			port:     "50051",
			params:   map[string]string{"timeout": "5000"},
		},
		{
			name:     "grpc protocol",
			protocol: "grpc",
			ip:       "10.0.0.1",
			port:     "9090",
			params:   map[string]string{"retries": "3", "loadbalance": "random"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := common.NewURLWithOptions(
				common.WithProtocol(tt.protocol),
				common.WithIp(tt.ip),
				common.WithPort(tt.port),
			)
			for k, v := range tt.params {
				url.SetParam(k, v)
			}

			mockC := newMockClient()
			ec := NewExchangeClient(url, mockC, 5*time.Second, true)

			assert.NotNil(t, ec)
			assert.Equal(t, tt.ip+":"+tt.port, ec.address)
		})
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkNewExchangeClient(b *testing.B) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockC := newMockClient()
		NewExchangeClient(url, mockC, 5*time.Second, true)
	}
}

func BenchmarkExchangeClientActiveNumber(b *testing.B) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockC := newMockClient()
	ec := NewExchangeClient(url, mockC, 5*time.Second, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec.IncreaseActiveNumber()
		ec.DecreaseActiveNumber()
	}
}

func BenchmarkExchangeClientIsAvailable(b *testing.B) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockC := newMockClient()
	ec := NewExchangeClient(url, mockC, 5*time.Second, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec.IsAvailable()
	}
}
