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
	"time"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, 3, ConnDelay)
	assert.Equal(t, 3, MaxFailTimes)
}

// Mock for ValidateZookeeperClient test
type mockZkClientFacade struct {
	client *gxzookeeper.ZookeeperClient
	lock   sync.Mutex
	url    *common.URL
	done   chan struct{}
}

func (m *mockZkClientFacade) ZkClient() *gxzookeeper.ZookeeperClient     { return m.client }
func (m *mockZkClientFacade) SetZkClient(c *gxzookeeper.ZookeeperClient) { m.client = c }
func (m *mockZkClientFacade) ZkClientLock() *sync.Mutex                  { return &m.lock }
func (m *mockZkClientFacade) WaitGroup() *sync.WaitGroup                 { return &sync.WaitGroup{} }
func (m *mockZkClientFacade) Done() chan struct{} {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.done == nil {
		m.done = make(chan struct{})
	}
	return m.done
}
func (m *mockZkClientFacade) RestartCallBack() bool { return true }
func (m *mockZkClientFacade) GetURL() *common.URL   { return m.url }

func TestValidateZookeeperClient(t *testing.T) {
	// Test with invalid address (will fail to connect but exercises the code path)
	facade := &mockZkClientFacade{
		url: common.NewURLWithOptions(
			common.WithParamsValue("config.timeout", "100ms"),
		),
	}

	err := ValidateZookeeperClient(facade, "test")
	require.Error(t, err) // Expected to fail without real zk

	// Test with existing client (should skip creation)
	facade2 := &mockZkClientFacade{
		client: &gxzookeeper.ZookeeperClient{},
		url:    common.NewURLWithOptions(),
	}
	err = ValidateZookeeperClient(facade2, "test")
	assert.NoError(t, err)
}

func TestValidateZookeeperClientConcurrent(t *testing.T) {
	facade := &mockZkClientFacade{
		url: common.NewURLWithOptions(
			common.WithParamsValue("config.timeout", "50ms"),
		),
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ValidateZookeeperClient(facade, "concurrent-test")
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent test timeout")
	}
}
