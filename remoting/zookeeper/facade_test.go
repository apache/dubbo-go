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
)

// Verify ZkClientFacade interface can be implemented
type testZkClientFacade struct {
	client *gxzookeeper.ZookeeperClient
	lock   sync.Mutex
	wg     sync.WaitGroup
	done   chan struct{}
	url    *common.URL
}

func (f *testZkClientFacade) ZkClient() *gxzookeeper.ZookeeperClient  { return f.client }
func (f *testZkClientFacade) SetZkClient(c *gxzookeeper.ZookeeperClient) { f.client = c }
func (f *testZkClientFacade) ZkClientLock() *sync.Mutex               { return &f.lock }
func (f *testZkClientFacade) WaitGroup() *sync.WaitGroup              { return &f.wg }
func (f *testZkClientFacade) Done() chan struct{}                     { return f.done }
func (f *testZkClientFacade) RestartCallBack() bool                   { return true }
func (f *testZkClientFacade) GetURL() *common.URL                     { return f.url }

// Compile-time check that testZkClientFacade implements ZkClientFacade
var _ ZkClientFacade = (*testZkClientFacade)(nil)

func TestZkClientFacadeInterface(t *testing.T) {
	facade := &testZkClientFacade{done: make(chan struct{})}

	assert.Nil(t, facade.ZkClient())
	assert.NotNil(t, facade.ZkClientLock())
	assert.NotNil(t, facade.WaitGroup())
	assert.NotNil(t, facade.Done())
	assert.True(t, facade.RestartCallBack())
	assert.Nil(t, facade.GetURL())

	facade.SetZkClient(nil)
	assert.Nil(t, facade.ZkClient())
}
