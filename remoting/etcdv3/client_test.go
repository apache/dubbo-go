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

package etcdv3

import (
	"sync"
	"testing"
	"time"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type mockClientFacade struct {
	client *gxetcd.Client
	lock   sync.Mutex
	url    *common.URL
	wg     sync.WaitGroup
	done   chan struct{}
}

func (m *mockClientFacade) Client() *gxetcd.Client     { return m.client }
func (m *mockClientFacade) SetClient(c *gxetcd.Client) { m.client = c }
func (m *mockClientFacade) ClientLock() *sync.Mutex    { return &m.lock }
func (m *mockClientFacade) WaitGroup() *sync.WaitGroup { return &m.wg }
func (m *mockClientFacade) Done() chan struct{} {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.done == nil {
		m.done = make(chan struct{})
	}
	return m.done
}
func (m *mockClientFacade) RestartCallBack() bool { return true }
func (m *mockClientFacade) GetURL() *common.URL   { return m.url }
func (m *mockClientFacade) IsAvailable() bool     { return true }
func (m *mockClientFacade) Destroy()              {}

func TestValidateClient(t *testing.T) {
	// Test with nil client (will fail without real etcd)
	facade := &mockClientFacade{}
	err := ValidateClient(facade,
		gxetcd.WithName("test"),
		gxetcd.WithEndpoints("127.0.0.1:2379"),
		gxetcd.WithTimeout(100*time.Millisecond),
	)
	assert.NotNil(t, err)
}

func TestNewServiceDiscoveryClient(t *testing.T) {
	// Will return nil client without real etcd, but exercises the code
	client := NewServiceDiscoveryClient(
		gxetcd.WithName("test"),
		gxetcd.WithEndpoints("127.0.0.1:2379"),
		gxetcd.WithTimeout(100*time.Millisecond),
	)
	assert.Nil(t, client) // Expected nil without real etcd
}
