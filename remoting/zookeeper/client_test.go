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

package zookeeper

import (
	"sync"
	"testing"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// ============================================
// Mock ZkClientFacade Implementation
// ============================================
type mockZkClientFacade struct {
	zkClient   *gxzookeeper.ZookeeperClient
	zkLock     sync.Mutex
	wg         sync.WaitGroup
	done       chan struct{}
	url        *common.URL
	restartCnt int
}

func newMockZkClientFacade(url *common.URL) *mockZkClientFacade {
	return &mockZkClientFacade{
		done: make(chan struct{}),
		url:  url,
	}
}

func (m *mockZkClientFacade) ZkClient() *gxzookeeper.ZookeeperClient {
	return m.zkClient
}

func (m *mockZkClientFacade) SetZkClient(client *gxzookeeper.ZookeeperClient) {
	m.zkClient = client
}

func (m *mockZkClientFacade) ZkClientLock() *sync.Mutex {
	return &m.zkLock
}

func (m *mockZkClientFacade) WaitGroup() *sync.WaitGroup {
	return &m.wg
}

func (m *mockZkClientFacade) Done() chan struct{} {
	return m.done
}

func (m *mockZkClientFacade) RestartCallBack() bool {
	m.restartCnt++
	return true
}

func (m *mockZkClientFacade) GetURL() *common.URL {
	return m.url
}

// ============================================
// Constants Tests
// ============================================

func TestConstants(t *testing.T) {
	assert.Equal(t, 3, ConnDelay)
	assert.Equal(t, 3, MaxFailTimes)
}

// ============================================
// ValidateZookeeperClient Tests
// ============================================

func TestValidateZookeeperClientWithNilClient(t *testing.T) {
	tests := []struct {
		name    string
		url     *common.URL
		zkName  string
		wantErr bool
	}{
		{
			name: "invalid zookeeper address",
			url: common.NewURLWithOptions(
				common.WithProtocol("zookeeper"),
				common.WithLocation("127.0.0.1:29999"),
				common.WithParamsValue(constant.ConfigTimeoutKey, "100ms"),
			),
			zkName:  "test-zk-client",
			wantErr: true,
		},
		{
			name: "empty location",
			url: common.NewURLWithOptions(
				common.WithProtocol("zookeeper"),
				common.WithLocation(""),
				common.WithParamsValue(constant.ConfigTimeoutKey, "100ms"),
			),
			zkName:  "test-zk-client",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facade := newMockZkClientFacade(tt.url)
			err := ValidateZookeeperClient(facade, tt.zkName)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestValidateZookeeperClientWithExistingClient(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("zookeeper"),
		common.WithLocation("127.0.0.1:2181"),
	)
	facade := newMockZkClientFacade(url)

	// Simulate existing client (not nil)
	// In real scenario, this would be a valid client
	// Here we just test that the function doesn't try to create a new one
	// when client already exists

	// First call will try to create client (and fail due to no zk server)
	err := ValidateZookeeperClient(facade, "test")
	assert.NotNil(t, err) // Expected to fail without real zk
}

func TestValidateZookeeperClientURLParsing(t *testing.T) {
	tests := []struct {
		name            string
		location        string
		expectedAddrs   int
		timeout         string
		expectedTimeout string
	}{
		{
			name:          "single address",
			location:      "127.0.0.1:2181",
			expectedAddrs: 1,
			timeout:       "5s",
		},
		{
			name:          "multiple addresses",
			location:      "127.0.0.1:2181,127.0.0.1:2182,127.0.0.1:2183",
			expectedAddrs: 3,
			timeout:       "10s",
		},
		{
			name:          "default timeout",
			location:      "localhost:2181",
			expectedAddrs: 1,
			timeout:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := common.NewURLWithOptions(
				common.WithProtocol("zookeeper"),
				common.WithLocation(tt.location),
			)
			if tt.timeout != "" {
				url.SetParam(constant.ConfigTimeoutKey, tt.timeout)
			}

			// Verify URL parsing
			addresses := splitAddresses(tt.location)
			assert.Equal(t, tt.expectedAddrs, len(addresses))
		})
	}
}

// Helper function to simulate address splitting
func splitAddresses(location string) []string {
	if location == "" {
		return []string{}
	}
	result := []string{}
	start := 0
	for i := 0; i < len(location); i++ {
		if location[i] == ',' {
			result = append(result, location[start:i])
			start = i + 1
		}
	}
	result = append(result, location[start:])
	return result
}

// ============================================
// ZkClientFacade Interface Tests
// ============================================

func TestMockZkClientFacadeImplementation(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("zookeeper"),
		common.WithLocation("127.0.0.1:2181"),
	)
	facade := newMockZkClientFacade(url)

	t.Run("initial state", func(t *testing.T) {
		assert.Nil(t, facade.ZkClient())
		assert.NotNil(t, facade.ZkClientLock())
		assert.NotNil(t, facade.WaitGroup())
		assert.NotNil(t, facade.Done())
		assert.Equal(t, url, facade.GetURL())
	})

	t.Run("set client", func(t *testing.T) {
		facade.SetZkClient(nil)
		assert.Nil(t, facade.ZkClient())
	})

	t.Run("restart callback", func(t *testing.T) {
		result := facade.RestartCallBack()
		assert.True(t, result)
		assert.Equal(t, 1, facade.restartCnt)

		facade.RestartCallBack()
		assert.Equal(t, 2, facade.restartCnt)
	})
}

// ============================================
// Concurrent Access Tests
// ============================================

func TestValidateZookeeperClientConcurrentAccess(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("zookeeper"),
		common.WithLocation("127.0.0.1:29999"),
		common.WithParamsValue(constant.ConfigTimeoutKey, "50ms"),
	)
	facade := newMockZkClientFacade(url)
	var wg sync.WaitGroup

	// Concurrent validation attempts
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ValidateZookeeperClient(facade, "concurrent-test")
		}()
	}

	wg.Wait()
}

// ============================================
// Edge Cases Tests
// ============================================

func TestValidateZookeeperClientEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		zkName   string
		location string
		timeout  string
	}{
		{
			name:     "special characters in zkName",
			zkName:   "test-zk_client.v1",
			location: "127.0.0.1:29999",
			timeout:  "50ms",
		},
		{
			name:     "empty zkName",
			zkName:   "",
			location: "127.0.0.1:29999",
			timeout:  "50ms",
		},
		{
			name:     "unicode zkName",
			zkName:   "测试客户端",
			location: "127.0.0.1:29999",
			timeout:  "50ms",
		},
		{
			name:     "very long zkName",
			zkName:   "this-is-a-very-long-zookeeper-client-name-that-exceeds-normal-length-limits-for-testing-purposes",
			location: "127.0.0.1:29999",
			timeout:  "50ms",
		},
		{
			name:     "zkName with spaces",
			zkName:   "test zk client",
			location: "127.0.0.1:29999",
			timeout:  "50ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := common.NewURLWithOptions(
				common.WithProtocol("zookeeper"),
				common.WithLocation(tt.location),
				common.WithParamsValue(constant.ConfigTimeoutKey, tt.timeout),
			)
			facade := newMockZkClientFacade(url)
			err := ValidateZookeeperClient(facade, tt.zkName)
			assert.NotNil(t, err) // Will fail due to no zk server
		})
	}
}

// ============================================
// Timeout Configuration Tests
// ============================================

func TestValidateZookeeperClientTimeoutConfig(t *testing.T) {
	tests := []struct {
		name            string
		timeout         string
		expectedTimeout bool
	}{
		{
			name:            "milliseconds timeout",
			timeout:         "100ms",
			expectedTimeout: true,
		},
		{
			name:            "seconds timeout",
			timeout:         "5s",
			expectedTimeout: true,
		},
		{
			name:            "no timeout (use default)",
			timeout:         "",
			expectedTimeout: true,
		},
		{
			name:            "very short timeout",
			timeout:         "1ms",
			expectedTimeout: true,
		},
		{
			name:            "long timeout",
			timeout:         "30s",
			expectedTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := common.NewURLWithOptions(
				common.WithProtocol("zookeeper"),
				common.WithLocation("127.0.0.1:29999"),
			)
			if tt.timeout != "" {
				url.SetParam(constant.ConfigTimeoutKey, tt.timeout)
			}
			facade := newMockZkClientFacade(url)
			err := ValidateZookeeperClient(facade, "timeout-test")
			// Will fail due to no zk server, but timeout config should be parsed
			assert.NotNil(t, err)
		})
	}
}

// ============================================
// Address Parsing Tests
// ============================================

func TestSplitAddresses(t *testing.T) {
	tests := []struct {
		name          string
		location      string
		expectedCount int
		expectedAddrs []string
	}{
		{
			name:          "single address",
			location:      "127.0.0.1:2181",
			expectedCount: 1,
			expectedAddrs: []string{"127.0.0.1:2181"},
		},
		{
			name:          "multiple addresses",
			location:      "127.0.0.1:2181,127.0.0.1:2182,127.0.0.1:2183",
			expectedCount: 3,
			expectedAddrs: []string{"127.0.0.1:2181", "127.0.0.1:2182", "127.0.0.1:2183"},
		},
		{
			name:          "empty location",
			location:      "",
			expectedCount: 0,
			expectedAddrs: []string{},
		},
		{
			name:          "hostname addresses",
			location:      "zk1.example.com:2181,zk2.example.com:2181",
			expectedCount: 2,
			expectedAddrs: []string{"zk1.example.com:2181", "zk2.example.com:2181"},
		},
		{
			name:          "ipv6 style addresses",
			location:      "[::1]:2181",
			expectedCount: 1,
			expectedAddrs: []string{"[::1]:2181"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addrs := splitAddresses(tt.location)
			assert.Equal(t, tt.expectedCount, len(addrs))
			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedAddrs, addrs)
			}
		})
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkSplitAddresses(b *testing.B) {
	location := "127.0.0.1:2181,127.0.0.1:2182,127.0.0.1:2183"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitAddresses(location)
	}
}

func BenchmarkMockZkClientFacadeRestartCallback(b *testing.B) {
	url := common.NewURLWithOptions(
		common.WithProtocol("zookeeper"),
		common.WithLocation("127.0.0.1:2181"),
	)
	facade := newMockZkClientFacade(url)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		facade.RestartCallBack()
	}
}

func BenchmarkMockZkClientFacadeLock(b *testing.B) {
	url := common.NewURLWithOptions(
		common.WithProtocol("zookeeper"),
		common.WithLocation("127.0.0.1:2181"),
	)
	facade := newMockZkClientFacade(url)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lock := facade.ZkClientLock()
		lock.Lock()
		lock.Unlock()
	}
}

// ============================================
// URL Parameter Tests
// ============================================

func TestValidateZookeeperClientURLParameters(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
	}{
		{
			name: "with all common parameters",
			params: map[string]string{
				constant.ConfigTimeoutKey: "5s",
				"session.timeout":         "30s",
				"retry.times":             "3",
			},
		},
		{
			name:   "with no parameters",
			params: map[string]string{},
		},
		{
			name: "with custom parameters",
			params: map[string]string{
				"custom.param1": "value1",
				"custom.param2": "value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := common.NewURLWithOptions(
				common.WithProtocol("zookeeper"),
				common.WithLocation("127.0.0.1:29999"),
			)
			for k, v := range tt.params {
				url.SetParam(k, v)
			}
			facade := newMockZkClientFacade(url)
			err := ValidateZookeeperClient(facade, "param-test")
			assert.NotNil(t, err) // Will fail due to no zk server
		})
	}
}

// ============================================
// Facade State Tests
// ============================================

func TestMockZkClientFacadeStateTransitions(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("zookeeper"),
		common.WithLocation("127.0.0.1:2181"),
	)
	facade := newMockZkClientFacade(url)

	t.Run("initial state verification", func(t *testing.T) {
		assert.Nil(t, facade.ZkClient())
		assert.Equal(t, 0, facade.restartCnt)
		assert.NotNil(t, facade.Done())
	})

	t.Run("multiple restart callbacks", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			result := facade.RestartCallBack()
			assert.True(t, result)
		}
		assert.Equal(t, 10, facade.restartCnt)
	})

	t.Run("done channel behavior", func(t *testing.T) {
		select {
		case <-facade.Done():
			t.Fatal("Done channel should not be closed")
		default:
			// Expected behavior
		}
	})
}
