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

package cluster

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestNewMockInvoker(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	successCount := 3

	invoker := NewMockInvoker(url, successCount)

	assert.NotNil(t, invoker)
	assert.Equal(t, url, invoker.url)
	assert.Equal(t, successCount, invoker.successCount)
	assert.True(t, invoker.available)
	assert.False(t, invoker.destroyed)
}

func TestMockInvokerGetURL(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	invoker := NewMockInvoker(url, 1)

	result := invoker.GetURL()
	assert.Equal(t, url, result)
}

func TestMockInvokerIsAvailable(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	invoker := NewMockInvoker(url, 1)

	assert.True(t, invoker.IsAvailable())
}

func TestMockInvokerIsDestroyed(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	invoker := NewMockInvoker(url, 1)

	assert.False(t, invoker.IsDestroyed())
}

func TestMockInvokerInvokeFailure(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	successCount := 3
	invoker := NewMockInvoker(url, successCount)

	// Reset global count
	Count = 0

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	// First invocation should fail
	result := invoker.Invoke(context.Background(), inv)
	assert.NotNil(t, result)
	assert.Error(t, result.Error())
	assert.Equal(t, 1, Count)

	rest, ok := result.Result().(Rest)
	assert.True(t, ok)
	assert.Equal(t, 1, rest.Tried)
	assert.False(t, rest.Success)
}

func TestMockInvokerInvokeSuccess(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	successCount := 2
	invoker := NewMockInvoker(url, successCount)

	// Reset global count
	Count = 0

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	// First invocation should fail
	result1 := invoker.Invoke(context.Background(), inv)
	assert.Error(t, result1.Error())

	// Second invocation should succeed (Count == successCount)
	result2 := invoker.Invoke(context.Background(), inv)
	assert.NoError(t, result2.Error())
	assert.Equal(t, 2, Count)

	rest, ok := result2.Result().(Rest)
	assert.True(t, ok)
	assert.Equal(t, 2, rest.Tried)
	assert.True(t, rest.Success)
}

func TestMockInvokerInvokeMultipleTimes(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	successCount := 3
	invoker := NewMockInvoker(url, successCount)

	// Reset global count
	Count = 0

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	// Invoke multiple times
	for i := 1; i <= 5; i++ {
		result := invoker.Invoke(context.Background(), inv)
		assert.NotNil(t, result)
		assert.Equal(t, i, Count)

		rest, ok := result.Result().(Rest)
		assert.True(t, ok)
		assert.Equal(t, i, rest.Tried)

		if i < successCount {
			assert.Error(t, result.Error())
			assert.False(t, rest.Success)
		} else {
			assert.NoError(t, result.Error())
			assert.True(t, rest.Success)
		}
	}
}

func TestMockInvokerDestroy(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	invoker := NewMockInvoker(url, 1)

	assert.True(t, invoker.IsAvailable())
	assert.False(t, invoker.IsDestroyed())

	invoker.Destroy()

	assert.False(t, invoker.IsAvailable())
	assert.True(t, invoker.IsDestroyed())
}

func TestMockInvokerSuccessCountZero(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	invoker := NewMockInvoker(url, 0)

	// Reset global count
	Count = 0

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))

	// All invocations should succeed since Count becomes 1 after increment, and 1 >= successCount (0)
	result := invoker.Invoke(context.Background(), inv)
	assert.NoError(t, result.Error())
	assert.Equal(t, 1, Count)

	rest, ok := result.Result().(Rest)
	assert.True(t, ok)
	assert.True(t, rest.Success)
}

func TestNewMockCluster(t *testing.T) {
	cluster := NewMockCluster()
	assert.NotNil(t, cluster)
}

func TestRestStruct(t *testing.T) {
	rest := Rest{
		Tried:   5,
		Success: true,
	}

	assert.Equal(t, 5, rest.Tried)
	assert.True(t, rest.Success)
}
