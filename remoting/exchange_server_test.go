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
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// ============================================
// Mock Server Implementation
// ============================================

type mockServer struct {
	mu           sync.Mutex
	started      bool
	stopped      bool
	startCalled  int
	stopCalled   int
}

func newMockServer() *mockServer {
	return &mockServer{}
}

func (m *mockServer) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCalled++
	m.started = true
	m.stopped = false
}

func (m *mockServer) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCalled++
	m.stopped = true
	m.started = false
}

func (m *mockServer) isStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}

func (m *mockServer) isStopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

func (m *mockServer) getStartCalled() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startCalled
}

func (m *mockServer) getStopCalled() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopCalled
}

// ============================================
// NewExchangeServer Tests
// ============================================

func TestNewExchangeServer(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		ip       string
		port     string
	}{
		{
			name:     "dubbo server",
			protocol: "dubbo",
			ip:       "127.0.0.1",
			port:     "20880",
		},
		{
			name:     "triple server",
			protocol: "triple",
			ip:       "0.0.0.0",
			port:     "50051",
		},
		{
			name:     "grpc server",
			protocol: "grpc",
			ip:       "localhost",
			port:     "9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := common.NewURLWithOptions(
				common.WithProtocol(tt.protocol),
				common.WithIp(tt.ip),
				common.WithPort(tt.port),
			)
			mockS := newMockServer()

			es := NewExchangeServer(url, mockS)
			assert.NotNil(t, es)
			assert.Equal(t, url, es.URL)
			assert.Equal(t, mockS, es.Server)
		})
	}
}

func TestNewExchangeServerWithNilURL(t *testing.T) {
	mockS := newMockServer()
	es := NewExchangeServer(nil, mockS)
	assert.NotNil(t, es)
	assert.Nil(t, es.URL)
	assert.Equal(t, mockS, es.Server)
}

// ============================================
// Start Tests
// ============================================

func TestExchangeServerStart(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockS := newMockServer()

	es := NewExchangeServer(url, mockS)
	assert.NotNil(t, es)

	es.Start()
	assert.True(t, mockS.isStarted())
	assert.Equal(t, 1, mockS.getStartCalled())
}

func TestExchangeServerStartMultipleTimes(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockS := newMockServer()

	es := NewExchangeServer(url, mockS)

	// Start multiple times
	for i := 0; i < 5; i++ {
		es.Start()
	}

	assert.Equal(t, 5, mockS.getStartCalled())
	assert.True(t, mockS.isStarted())
}

// ============================================
// Stop Tests
// ============================================

func TestExchangeServerStop(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockS := newMockServer()

	es := NewExchangeServer(url, mockS)
	es.Start()
	assert.True(t, mockS.isStarted())

	es.Stop()
	assert.True(t, mockS.isStopped())
	assert.False(t, mockS.isStarted())
	assert.Equal(t, 1, mockS.getStopCalled())
}

func TestExchangeServerStopWithoutStart(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockS := newMockServer()

	es := NewExchangeServer(url, mockS)

	// Stop without starting
	es.Stop()
	assert.True(t, mockS.isStopped())
	assert.Equal(t, 1, mockS.getStopCalled())
}

// ============================================
// Lifecycle Tests
// ============================================

func TestExchangeServerLifecycle(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockS := newMockServer()

	es := NewExchangeServer(url, mockS)

	// Start -> Stop -> Start -> Stop
	es.Start()
	assert.True(t, mockS.isStarted())

	es.Stop()
	assert.True(t, mockS.isStopped())

	es.Start()
	assert.True(t, mockS.isStarted())

	es.Stop()
	assert.True(t, mockS.isStopped())

	assert.Equal(t, 2, mockS.getStartCalled())
	assert.Equal(t, 2, mockS.getStopCalled())
}

// ============================================
// Server Interface Tests
// ============================================

func TestServerInterfaceImplementation(t *testing.T) {
	mockS := newMockServer()

	// Verify mockServer implements Server interface
	var _ Server = mockS

	mockS.Start()
	assert.True(t, mockS.isStarted())

	mockS.Stop()
	assert.True(t, mockS.isStopped())
}

// ============================================
// Concurrent Access Tests
// ============================================

func TestExchangeServerConcurrentStartStop(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockS := newMockServer()

	es := NewExchangeServer(url, mockS)

	var wg sync.WaitGroup

	// Concurrent start
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			es.Start()
		}()
	}

	// Concurrent stop
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			es.Stop()
		}()
	}

	wg.Wait()

	assert.Equal(t, 50, mockS.getStartCalled())
	assert.Equal(t, 50, mockS.getStopCalled())
}

// ============================================
// Edge Cases Tests
// ============================================

func TestExchangeServerEdgeCases(t *testing.T) {
	t.Run("server with empty url location", func(t *testing.T) {
		url := common.NewURLWithOptions(
			common.WithProtocol("dubbo"),
		)
		mockS := newMockServer()

		es := NewExchangeServer(url, mockS)
		assert.NotNil(t, es)
		assert.Equal(t, ":", es.URL.Location)
	})

	t.Run("server fields access", func(t *testing.T) {
		url := common.NewURLWithOptions(
			common.WithProtocol("dubbo"),
			common.WithIp("192.168.1.100"),
			common.WithPort("8080"),
		)
		mockS := newMockServer()

		es := NewExchangeServer(url, mockS)
		assert.Equal(t, "dubbo", es.URL.Protocol)
		assert.Equal(t, "192.168.1.100:8080", es.URL.Location)
	})
}

// ============================================
// URL Configuration Tests
// ============================================

func TestExchangeServerURLConfigurations(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		ip       string
		port     string
		params   map[string]string
	}{
		{
			name:     "basic configuration",
			protocol: "dubbo",
			ip:       "127.0.0.1",
			port:     "20880",
			params:   nil,
		},
		{
			name:     "with timeout param",
			protocol: "dubbo",
			ip:       "127.0.0.1",
			port:     "20880",
			params:   map[string]string{"timeout": "5000"},
		},
		{
			name:     "with multiple params",
			protocol: "triple",
			ip:       "0.0.0.0",
			port:     "50051",
			params:   map[string]string{"timeout": "3000", "retries": "3"},
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

			mockS := newMockServer()
			es := NewExchangeServer(url, mockS)

			assert.NotNil(t, es)
			assert.Equal(t, tt.protocol, es.URL.Protocol)
			assert.Equal(t, tt.ip+":"+tt.port, es.URL.Location)
		})
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkNewExchangeServer(b *testing.B) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockS := newMockServer()
		NewExchangeServer(url, mockS)
	}
}

func BenchmarkExchangeServerStart(b *testing.B) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockS := newMockServer()
	es := NewExchangeServer(url, mockS)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		es.Start()
	}
}

func BenchmarkExchangeServerStop(b *testing.B) {
	url := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20880"),
	)
	mockS := newMockServer()
	es := NewExchangeServer(url, mockS)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		es.Stop()
	}
}
