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

package grpc

import (
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc/internal/helloworld"
)

func TestGrpcProtocolRefer(t *testing.T) {
	server, err := helloworld.NewServer("127.0.0.1:30000")
	require.NoError(t, err)
	go server.Start()
	defer server.Stop()

	url, err := common.NewURL(helloworldURL)
	require.NoError(t, err)

	proto := GetProtocol()
	invoker := proto.Refer(url)

	// make sure url
	eq := invoker.GetURL().URLEqual(url)
	assert.True(t, eq)

	// make sure invokers after 'Destroy'
	invokersLen := len(proto.(*GrpcProtocol).Invokers())
	assert.Equal(t, 1, invokersLen)
	proto.Destroy()
	invokersLen = len(proto.(*GrpcProtocol).Invokers())
	assert.Equal(t, 0, invokersLen)
}

func TestGrpcProtocolDestroyDoesNotHoldServerLockWhileGracefulStopping(t *testing.T) {
	proto := NewGRPCProtocol()
	proto.serverMap["127.0.0.1:20000"] = &Server{}

	originalStop := grpcServerGracefulStop
	t.Cleanup(func() {
		grpcServerGracefulStop = originalStop
	})

	entered := make(chan struct{})
	release := make(chan struct{})
	var stopCalls atomic.Int32
	grpcServerGracefulStop = func(server *Server) {
		stopCalls.Add(1)
		close(entered)
		<-release
	}

	done := make(chan struct{})
	go func() {
		proto.Destroy()
		close(done)
	}()

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("Destroy did not reach graceful stop")
	}

	lockAcquired := make(chan struct{})
	go func() {
		proto.serverLock.Lock()
		_ = proto.serverMap
		proto.serverLock.Unlock()
		close(lockAcquired)
	}()

	select {
	case <-lockAcquired:
	case <-time.After(time.Second):
		t.Fatal("serverLock remained held during graceful stop")
	}

	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Destroy did not finish")
	}

	assert.Equal(t, int32(1), stopCalls.Load())
	assert.Empty(t, proto.serverMap)
}
