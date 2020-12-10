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

package getty

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

type mockClientFactory struct {
}

func (c *mockClientFactory) newGettyRPCClientConn(pool *gettyRPCClientPool, addr string) (*gettyRPCClient, error) {
	var client gettyRPCClient
	client.updateActive(time.Now().Unix())
	return &client, nil
}

func TestGetConnFromPool(t *testing.T) {
	var rpcClient Client

	clientPool := newGettyRPCClientConnPool(&rpcClient, 4, time.Duration(3*time.Second))
	clientPool.clientFactory = &mockClientFactory{}
	var conn1 gettyRPCClient
	conn1.active = time.Now().Unix()
	clientPool.lazyInit()
	<-clientPool.ch
	clientPool.putConnIntoPool(&conn1, nil)
	value, err := clientPool.getGettyRpcClient("", true)
	assert.Equal(t, nil, err)
	assert.Equal(t, &conn1, value)
	value, err = clientPool.getGettyRpcClient("", false)
	assert.Equal(t, nil, err)
	assert.Equal(t, &conn1, value)

	value, err = clientPool.getGettyRpcClient("", false)
	assert.Equal(t, nil, err)
	_, ok := clientPool.poolQueue.PopTail()
	assert.Equal(t, false, ok)

	value, err = clientPool.getGettyRpcClient("", true)
	assert.Equal(t, nil, err)
	_, ok = clientPool.poolQueue.PopTail()
	assert.Equal(t, true, ok)
	_, ok = clientPool.poolQueue.PopTail()
	assert.Equal(t, false, ok)

	client, err := clientPool.getGettyRpcClient("", true)
	assert.Equal(t, nil, err)
	assert.NotNil(t, client)
	client, err = clientPool.getGettyRpcClient("", false)
	assert.Equal(t, nil, err)
	assert.NotNil(t, client)

	client, err = clientPool.getGettyRpcClient("", true)
	assert.Equal(t, nil, err)
	assert.NotNil(t, client)

	time.Sleep(4 * time.Second)

	client2, err := clientPool.getConnFromPoll()
	assert.Equal(t, nil, err)
	assert.Nil(t, client2)
}
