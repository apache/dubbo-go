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
	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
)

type mockFacade struct {
	client  *ZookeeperClient
	cltLock sync.Mutex
	wg      sync.WaitGroup
	URL     *common.URL
	done    chan struct{}
}

func newMockFacade(client *ZookeeperClient, url *common.URL) ZkClientFacade {
	mock := &mockFacade{
		client: client,
		URL:    url,
	}

	mock.wg.Add(1)
	return mock
}

func (r *mockFacade) ZkClient() *ZookeeperClient {
	return r.client
}

func (r *mockFacade) SetZkClient(client *ZookeeperClient) {
	r.client = client
}

func (r *mockFacade) ZkClientLock() *sync.Mutex {
	return &r.cltLock
}

func (r *mockFacade) WaitGroup() *sync.WaitGroup {
	return &r.wg
}

func (r *mockFacade) Done() chan struct{} {
	return r.done
}

func (r *mockFacade) GetUrl() *common.URL {
	return r.URL
}

func (r *mockFacade) Destroy() {
	close(r.done)
	r.wg.Wait()
}

func (r *mockFacade) RestartCallBack() bool {
	return true
}

func (r *mockFacade) IsAvailable() bool {
	return true
}

func Test_Facade(t *testing.T) {
	ts, z, event, err := NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)
	defer func() {
		if err := ts.Stop(); err != nil {
			t.Errorf("tc.Stop() = error: %v", err)
		}
	}()
	url, _ := common.NewURL("mock://127.0.0.1")
	mock := newMockFacade(z, url)
	go HandleClientRestart(mock)
	states := []zk.State{zk.StateConnecting, zk.StateConnected, zk.StateHasSession}
	verifyEventStateOrder(t, event, states, "event channel")
}
