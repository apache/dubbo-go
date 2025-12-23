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
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// Verify clientFacade interface can be implemented
type testClientFacade struct {
	client *gxetcd.Client
	lock   sync.Mutex
	wg     sync.WaitGroup
	done   chan struct{}
	url    *common.URL
}

func (f *testClientFacade) Client() *gxetcd.Client     { return f.client }
func (f *testClientFacade) SetClient(c *gxetcd.Client) { f.client = c }
func (f *testClientFacade) ClientLock() *sync.Mutex    { return &f.lock }
func (f *testClientFacade) WaitGroup() *sync.WaitGroup { return &f.wg }
func (f *testClientFacade) Done() chan struct{}        { return f.done }
func (f *testClientFacade) RestartCallBack() bool      { return true }
func (f *testClientFacade) GetURL() *common.URL        { return f.url }
func (f *testClientFacade) IsAvailable() bool          { return true }
func (f *testClientFacade) Destroy()                   { close(f.done) }

// Compile-time check
var _ clientFacade = (*testClientFacade)(nil)

func TestClientFacadeInterface(t *testing.T) {
	facade := &testClientFacade{done: make(chan struct{})}

	assert.Nil(t, facade.Client())
	assert.NotNil(t, facade.ClientLock())
	assert.NotNil(t, facade.WaitGroup())
	assert.NotNil(t, facade.Done())
	assert.True(t, facade.RestartCallBack())
	assert.True(t, facade.IsAvailable())

	facade.SetClient(nil)
	assert.Nil(t, facade.Client())
}
