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

package graceful_shutdown

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// MockFilter implements filter.Filter and config.Setter for testing
type MockFilter struct {
	mock.Mock
}

func (m *MockFilter) Set(key string, value interface{}) {
	m.Called(key, value)
}

func (m *MockFilter) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	return nil
}

func (m *MockFilter) OnResponse(ctx context.Context, result result.Result, invoker base.Invoker, invocation base.Invocation) result.Result {
	return nil
}

func TestInit(t *testing.T) {
	// Reset initOnce and protocols for testing
	initOnce = sync.Once{}
	protocols = nil
	proMu = sync.Mutex{}

	// Register mock filters
	mockConsumerFilter := &MockFilter{}
	mockProviderFilter := &MockFilter{}

	// Expect Set method calls 
	mockConsumerFilter.On("Set", mock.Anything, mock.Anything).Return()
	mockProviderFilter.On("Set", mock.Anything, mock.Anything).Return()

	// Register mock filters
	extension.SetFilter(constant.GracefulShutdownConsumerFilterKey, func() filter.Filter {
		return mockConsumerFilter
	})
	extension.SetFilter(constant.GracefulShutdownProviderFilterKey, func() filter.Filter {
		return mockProviderFilter
	})

	// Test with default options
	Init()

	// Test with custom options
	customTimeout := 120 * time.Second
	Init(WithTimeout(customTimeout))

	// Remove mock filters
	extension.UnregisterFilter(constant.GracefulShutdownConsumerFilterKey)
	extension.UnregisterFilter(constant.GracefulShutdownProviderFilterKey)
}

func TestRegisterProtocol(t *testing.T) {
	// Reset protocols for testing
	protocols = make(map[string]struct{})
	proMu = sync.Mutex{}

	// Register some protocols
	RegisterProtocol("dubbo")
	RegisterProtocol("rest")
	RegisterProtocol("tri")

	// Check if protocols are registered correctly
	proMu.Lock()
	defer proMu.Unlock()

	assert.Contains(t, protocols, "dubbo")
	assert.Contains(t, protocols, "rest")
	assert.Contains(t, protocols, "tri")
	assert.Len(t, protocols, 3)
}

func TestTotalTimeout(t *testing.T) {
	// Test with default timeout
	config := global.DefaultShutdownConfig()
	timeout := totalTimeout(config)
	assert.Equal(t, defaultTimeout, timeout)

	// Test with custom timeout
	config.Timeout = "120s"
	timeout = totalTimeout(config)
	assert.Equal(t, 120*time.Second, timeout)

	// Test with invalid timeout
	config.Timeout = "invalid"
	timeout = totalTimeout(config)
	assert.Equal(t, defaultTimeout, timeout)

	// Test with timeout less than default
	config.Timeout = "30s"
	timeout = totalTimeout(config)
	assert.Equal(t, defaultTimeout, timeout) // Should use default if less than default
}

func TestParseDuration(t *testing.T) {
	// Test with valid duration
	res := parseDuration("10s", "test", 5*time.Second)
	assert.Equal(t, 10*time.Second, res)

	// Test with invalid duration
	res = parseDuration("invalid", "test", 5*time.Second)
	assert.Equal(t, 5*time.Second, res)

	// Test with empty string
	res = parseDuration("", "test", 5*time.Second)
	assert.Equal(t, 5*time.Second, res)
}

func TestWaitAndAcceptNewRequests(t *testing.T) {
	// Test with positive step timeout
	config := global.DefaultShutdownConfig()
	config.StepTimeout = "100ms"
	config.ProviderActiveCount.Store(0)

	start := time.Now()
	waitAndAcceptNewRequests(config)
	elapsed := time.Since(start)

	// Should wait for ConsumerUpdateWaitTime (default 3s) plus a little extra for processing
	assert.GreaterOrEqual(t, elapsed, 3*time.Second)

	// Test with negative step timeout (should skip waiting)
	config.StepTimeout = "-1s"
	start = time.Now()
	waitAndAcceptNewRequests(config)
	elapsed = time.Since(start)

	// Should only wait for ConsumerUpdateWaitTime
	assert.Less(t, elapsed, 3*time.Second+100*time.Millisecond) 
}

func TestWaitForSendingAndReceivingRequests(t *testing.T) {
	// Test with active consumer requests
	config := global.DefaultShutdownConfig()
	config.StepTimeout = "100ms"
	config.ConsumerActiveCount.Store(1)

	start := time.Now()
	waitForSendingAndReceivingRequests(config)
	elapsed := time.Since(start)

	// Should wait for step timeout
	assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond)

	// Test with no active consumer requests
	config.ConsumerActiveCount.Store(0)
	start = time.Now()
	waitForSendingAndReceivingRequests(config)
	elapsed = time.Since(start)

	// Should return immediately
	assert.Less(t, elapsed, 50*time.Millisecond)
}
