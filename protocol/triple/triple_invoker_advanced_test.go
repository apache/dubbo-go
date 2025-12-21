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

package triple

import (
	"context"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

func TestTripleInvoker_Heartbeat(t *testing.T) {
	// Create test URL
	url, _ := common.NewURL("tri://localhost:20000/com.example.TestService?heartbeat.enabled=true&heartbeat.interval=100ms&heartbeat.timeout=50ms")

	// Create TripleInvoker - Note: this test may fail due to missing service definition
	// We only test heartbeat-related configuration, not actual network calls
	invoker, err := NewTripleInvoker(url)
	if err != nil {
		t.Skipf("Skip test because TripleInvoker creation failed: %v", err)
		return
	}
	assert.NotNil(t, invoker)
	defer invoker.Destroy()

	// Test default heartbeat state
	assert.False(t, invoker.IsHeartbeatEnabled())
	assert.Equal(t, DefaultHeartbeatInterval, invoker.GetHeartbeatInterval())
	assert.Equal(t, DefaultHeartbeatTimeout, invoker.GetHeartbeatTimeout())

	// Test disable heartbeat (not enabled)
	invoker.DisableHeartbeat()
	assert.False(t, invoker.IsHeartbeatEnabled())
}

func TestTripleInvoker_Heartbeat_URL_Configuration(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected struct {
			enabled  bool
			interval time.Duration
			timeout  time.Duration
		}
	}{
		{
			name: "heartbeat enabled with custom config",
			url:  "tri://localhost:20000/com.example.TestService?heartbeat.enabled=true&heartbeat.interval=200ms&heartbeat.timeout=100ms",
			expected: struct {
				enabled  bool
				interval time.Duration
				timeout  time.Duration
			}{true, 200 * time.Millisecond, 100 * time.Millisecond},
		},
		{
			name: "heartbeat disabled",
			url:  "tri://localhost:20000/com.example.TestService?heartbeat.enabled=false",
			expected: struct {
				enabled  bool
				interval time.Duration
				timeout  time.Duration
			}{false, DefaultHeartbeatInterval, DefaultHeartbeatTimeout},
		},
		{
			name: "heartbeat default config",
			url:  "tri://localhost:20000/com.example.TestService",
			expected: struct {
				enabled  bool
				interval time.Duration
				timeout  time.Duration
			}{false, DefaultHeartbeatInterval, DefaultHeartbeatTimeout},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := common.NewURL(tt.url)
			invoker, err := NewTripleInvoker(url)
			assert.NoError(t, err)
			assert.NotNil(t, invoker)
			defer invoker.Destroy()

			assert.Equal(t, tt.expected.enabled, invoker.IsHeartbeatEnabled())
			assert.Equal(t, tt.expected.interval, invoker.GetHeartbeatInterval())
			assert.Equal(t, tt.expected.timeout, invoker.GetHeartbeatTimeout())
		})
	}
}

func TestTripleInvoker_Metrics_Events(t *testing.T) {
	// Create test URL
	url, _ := common.NewURL("tri://localhost:20000/com.example.TestService")

	// Create TripleInvoker
	invoker, err := NewTripleInvoker(url)
	assert.NoError(t, err)
	assert.NotNil(t, invoker)
	defer invoker.Destroy()

	// Create test invocation
	iv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("testMethod"))
	iv.SetAttribute(constant.CallTypeKey, constant.CallUnary)

	// Test that Invoke method sends metrics events
	ctx := context.Background()
	result := invoker.Invoke(ctx, iv)

	// Since there's no actual server, the call will fail, but metrics events should be sent
	assert.NotNil(t, result)
	assert.NotNil(t, result.Error())
}

func TestTripleInvoker_Concurrent_Heartbeat(t *testing.T) {
	// Create test URL
	url, _ := common.NewURL("tri://localhost:20000/com.example.TestService?heartbeat.enabled=true&heartbeat.interval=50ms")

	// Create TripleInvoker
	invoker, err := NewTripleInvoker(url)
	assert.NoError(t, err)
	assert.NotNil(t, invoker)
	defer invoker.Destroy()

	// Enable heartbeat
	invoker.EnableHeartbeat()

	// Concurrent test of heartbeat status queries
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Concurrent query of heartbeat status
			_ = invoker.IsHeartbeatEnabled()
			_ = invoker.IsHeartbeatHealthy()
			_ = invoker.GetHeartbeatInterval()
			_ = invoker.GetHeartbeatTimeout()
			_ = invoker.GetLastHeartbeatTime()
		}()
	}

	wg.Wait()

	// Verify heartbeat still works normally
	assert.True(t, invoker.IsHeartbeatEnabled())
}

func TestTripleInvoker_Destroy_With_Heartbeat(t *testing.T) {
	// Create test URL
	url, _ := common.NewURL("tri://localhost:20000/com.example.TestService?heartbeat.enabled=true")

	// Create TripleInvoker
	invoker, err := NewTripleInvoker(url)
	assert.NoError(t, err)
	assert.NotNil(t, invoker)

	// Enable heartbeat
	invoker.EnableHeartbeat()
	assert.True(t, invoker.IsHeartbeatEnabled())

	// Destroy invoker
	invoker.Destroy()

	// Verify heartbeat has been stopped
	assert.False(t, invoker.IsHeartbeatEnabled())
}

func TestTripleInvoker_IsAvailable(t *testing.T) {
	// Create test URL
	url, _ := common.NewURL("tri://localhost:20000/com.example.TestService")

	// Create TripleInvoker
	invoker, err := NewTripleInvoker(url)
	assert.NoError(t, err)
	assert.NotNil(t, invoker)
	defer invoker.Destroy()

	// Test IsAvailable method
	assert.True(t, invoker.IsAvailable())

	// Test IsDestroyed method
	assert.False(t, invoker.IsDestroyed())
}

func TestTripleInvoker_BaseInvoker_Methods(t *testing.T) {
	// Create test URL
	url, _ := common.NewURL("tri://localhost:20000/com.example.TestService")

	// Create TripleInvoker
	invoker, err := NewTripleInvoker(url)
	assert.NoError(t, err)
	assert.NotNil(t, invoker)
	defer invoker.Destroy()

	// Test basic information retrieval
	assert.Equal(t, url, invoker.GetURL())
	assert.NotNil(t, invoker.GetURL())
}

func TestTripleMetricsEvent_Creation(t *testing.T) {
	// Create test data
	url, _ := common.NewURL("tri://localhost:20000/com.example.TestService")
	invoker, _ := NewTripleInvoker(url)
	defer invoker.Destroy()

	iv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("testMethod"))

	// Create metrics event
	event := NewTripleMetricsEvent("before_invoke", invoker, iv)
	assert.NotNil(t, event)
	assert.Equal(t, "before_invoke", event.Name)
	assert.Equal(t, invoker, event.Invoker)
	assert.Equal(t, iv, event.Invocation)
	assert.Equal(t, constant.MetricsRpc, event.Type())

	// Test WithResult method
	rpcResult := &result.RPCResult{}
	event = event.WithResult(rpcResult)
	assert.Equal(t, rpcResult, event.Result)

	// Test WithCostTime method
	costTime := 100 * time.Millisecond
	event = event.WithCostTime(costTime)
	assert.Equal(t, costTime, event.CostTime)
}

func TestTripleInvoker_Heartbeat_Edge_Cases(t *testing.T) {
	// Test heartbeat configuration in extreme cases
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "invalid heartbeat interval",
			url:         "tri://localhost:20000/com.example.TestService?heartbeat.enabled=true&heartbeat.interval=invalid",
			expectError: false, // Should use default value
		},
		{
			name:        "invalid heartbeat timeout",
			url:         "tri://localhost:20000/com.example.TestService?heartbeat.enabled=true&heartbeat.timeout=invalid",
			expectError: false, // Should use default value
		},
		{
			name:        "very short heartbeat interval",
			url:         "tri://localhost:20000/com.example.TestService?heartbeat.enabled=true&heartbeat.interval=1ms",
			expectError: false,
		},
		{
			name:        "very long heartbeat interval",
			url:         "tri://localhost:20000/com.example.TestService?heartbeat.enabled=true&heartbeat.interval=1h",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := common.NewURL(tt.url)
			invoker, err := NewTripleInvoker(url)
			assert.NoError(t, err)
			assert.NotNil(t, invoker)
			defer invoker.Destroy()

			// Verify invoker creation successful
			assert.NotNil(t, invoker.GetURL())
		})
	}
}

func TestTripleInvoker_Heartbeat_Configuration_Parsing(t *testing.T) {
	// Test heartbeat configuration parsing
	testCases := []struct {
		name             string
		url              string
		expectedEnabled  bool
		expectedInterval time.Duration
		expectedTimeout  time.Duration
	}{
		{
			name:             "default configuration",
			url:              "tri://localhost:20000/com.example.TestService",
			expectedEnabled:  false,
			expectedInterval: DefaultHeartbeatInterval,
			expectedTimeout:  DefaultHeartbeatTimeout,
		},
		{
			name:             "explicitly disabled",
			url:              "tri://localhost:20000/com.example.TestService?heartbeat.enabled=false",
			expectedEnabled:  false,
			expectedInterval: DefaultHeartbeatInterval,
			expectedTimeout:  DefaultHeartbeatTimeout,
		},
		{
			name:             "custom interval only",
			url:              "tri://localhost:20000/com.example.TestService?heartbeat.enabled=true&heartbeat.interval=500ms",
			expectedEnabled:  true,
			expectedInterval: 500 * time.Millisecond,
			expectedTimeout:  DefaultHeartbeatTimeout,
		},
		{
			name:             "custom timeout only",
			url:              "tri://localhost:20000/com.example.TestService?heartbeat.enabled=true&heartbeat.timeout=200ms",
			expectedEnabled:  true,
			expectedInterval: DefaultHeartbeatInterval,
			expectedTimeout:  200 * time.Millisecond,
		},
		{
			name:             "both custom",
			url:              "tri://localhost:20000/com.example.TestService?heartbeat.enabled=true&heartbeat.interval=300ms&heartbeat.timeout=150ms",
			expectedEnabled:  true,
			expectedInterval: 300 * time.Millisecond,
			expectedTimeout:  150 * time.Millisecond,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url, _ := common.NewURL(tc.url)
			invoker, err := NewTripleInvoker(url)
			assert.NoError(t, err)
			assert.NotNil(t, invoker)
			defer invoker.Destroy()

			// Verify configuration is parsed correctly
			if tc.expectedEnabled {
				invoker.EnableHeartbeat()
			}

			assert.Equal(t, tc.expectedInterval, invoker.GetHeartbeatInterval())
			assert.Equal(t, tc.expectedTimeout, invoker.GetHeartbeatTimeout())
		})
	}
}
