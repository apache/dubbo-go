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
	"sync"
	"sync/atomic"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestGettyRPCClientUpdateActive(t *testing.T) {
	client := &gettyRPCClient{}
	client.updateActive(1234567890)
	assert.Equal(t, int64(1234567890), atomic.LoadInt64(&client.active))

	client.updateActive(0)
	assert.Equal(t, int64(0), atomic.LoadInt64(&client.active))
}

func TestGettyRPCClientSelectSession(t *testing.T) {
	client := &gettyRPCClient{sessions: nil}
	assert.Nil(t, client.selectSession())

	client.sessions = []*rpcSession{}
	assert.Nil(t, client.selectSession())
}

func TestGettyRPCClientSessionOperations(t *testing.T) {
	client := &gettyRPCClient{}

	// Remove/update nil session should not panic
	client.removeSession(nil)
	client.updateSession(nil)

	// Get from nil sessions
	_, err := client.getClientRpcSession(nil)
	assert.Equal(t, errClientClosed, err)

	// Session not found
	client.sessions = []*rpcSession{}
	_, err = client.getClientRpcSession(nil)
	assert.Contains(t, err.Error(), "session not exist")
}

func TestGettyRPCClientIsAvailable(t *testing.T) {
	client := &gettyRPCClient{sessions: nil}
	assert.False(t, client.isAvailable())

	client.sessions = []*rpcSession{}
	assert.False(t, client.isAvailable())
}

func TestGettyRPCClientClose(t *testing.T) {
	client := &gettyRPCClient{sessions: []*rpcSession{}}
	assert.Nil(t, client.close())
	err := client.close()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "close gettyRPCClient")
	assert.Contains(t, err.Error(), "again")
}

func TestGettyRPCClientConcurrent(t *testing.T) {
	client := &gettyRPCClient{sessions: []*rpcSession{}}
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(val int64) {
			defer wg.Done()
			client.updateActive(val)
			_ = client.selectSession()
			_ = client.isAvailable()
		}(int64(i))
	}
	wg.Wait()
}

func TestRpcSession(t *testing.T) {
	s := &rpcSession{reqNum: 0}

	s.AddReqNum(5)
	assert.Equal(t, int32(5), s.GetReqNum())

	s.AddReqNum(3)
	assert.Equal(t, int32(8), s.GetReqNum())

	s.AddReqNum(-3)
	assert.Equal(t, int32(5), s.GetReqNum())
}

func TestRpcSessionConcurrent(t *testing.T) {
	s := &rpcSession{}
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.AddReqNum(1)
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(100), s.GetReqNum())
}

func TestGettyRPCClientLifecycle(t *testing.T) {
	client := &gettyRPCClient{addr: "127.0.0.1:20880", sessions: []*rpcSession{}}

	assert.False(t, client.isAvailable())
	assert.Equal(t, int64(0), atomic.LoadInt64(&client.active))

	client.updateActive(1234567890)
	assert.Equal(t, int64(1234567890), atomic.LoadInt64(&client.active))

	assert.Nil(t, client.close())
	assert.Equal(t, int64(0), atomic.LoadInt64(&client.active))
}
