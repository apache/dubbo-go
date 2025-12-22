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
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type mockInterceptor struct {
	invoked bool
}

func (m *mockInterceptor) Invoke(ctx context.Context, invoker base.Invoker, inv base.Invocation) result.Result {
	m.invoked = true
	return invoker.Invoke(ctx, inv)
}

func TestInterceptorInvokerGetURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)

	interceptorInvoker := &InterceptorInvoker{
		next:        mockInvoker,
		interceptor: &mockInterceptor{},
	}

	url := interceptorInvoker.GetURL()
	assert.Equal(t, testURL, url)
}

func TestInterceptorInvokerIsAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().IsAvailable().Return(true).Times(1)

	interceptorInvoker := &InterceptorInvoker{
		next:        mockInvoker,
		interceptor: &mockInterceptor{},
	}

	available := interceptorInvoker.IsAvailable()
	assert.True(t, available)
}

func TestInterceptorInvokerIsAvailableFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().IsAvailable().Return(false).Times(1)

	interceptorInvoker := &InterceptorInvoker{
		next:        mockInvoker,
		interceptor: &mockInterceptor{},
	}

	available := interceptorInvoker.IsAvailable()
	assert.False(t, available)
}

func TestInterceptorInvokerInvoke(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).AnyTimes()

	expectedResult := &result.RPCResult{Rest: Rest{Tried: 0, Success: true}}
	mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(expectedResult).Times(1)

	interceptor := &mockInterceptor{}
	interceptorInvoker := &InterceptorInvoker{
		next:        mockInvoker,
		interceptor: interceptor,
	}

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))
	res := interceptorInvoker.Invoke(context.Background(), inv)

	assert.Equal(t, expectedResult, res)
	assert.True(t, interceptor.invoked)
}

func TestInterceptorInvokerDestroy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().Destroy().Times(1)

	interceptorInvoker := &InterceptorInvoker{
		next:        mockInvoker,
		interceptor: &mockInterceptor{},
	}

	interceptorInvoker.Destroy()
}

func TestSetAndGetClusterInterceptor(t *testing.T) {
	interceptorName := "test-interceptor"
	called := false

	// Save old interceptor if exists
	lock.Lock()
	oldInterceptor := interceptors[interceptorName]
	lock.Unlock()

	// Clean up after test
	defer func() {
		lock.Lock()
		if oldInterceptor != nil {
			interceptors[interceptorName] = oldInterceptor
		} else {
			delete(interceptors, interceptorName)
		}
		lock.Unlock()
	}()

	SetClusterInterceptor(interceptorName, func() Interceptor {
		called = true
		return &mockInterceptor{}
	})

	interceptor := GetClusterInterceptor(interceptorName)
	assert.NotNil(t, interceptor)
	assert.True(t, called)
}

func TestGetClusterInterceptorPanic(t *testing.T) {
	assert.Panics(t, func() {
		GetClusterInterceptor("non-existent-interceptor")
	})
}

func TestGetClusterInterceptors(t *testing.T) {
	// Clear existing interceptors by creating new ones
	SetClusterInterceptor("test-interceptor-1", func() Interceptor {
		return &mockInterceptor{}
	})
	SetClusterInterceptor("test-interceptor-2", func() Interceptor {
		return &mockInterceptor{}
	})

	interceptors := GetClusterInterceptors()
	assert.NotNil(t, interceptors)
	assert.GreaterOrEqual(t, len(interceptors), 2)
}

func TestBuildInterceptorChainEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).AnyTimes()

	// Clear interceptors
	lock.Lock()
	oldInterceptors := interceptors
	interceptors = make(map[string]func() Interceptor)
	lock.Unlock()

	// Restore interceptors using defer to ensure cleanup even if test fails
	defer func() {
		lock.Lock()
		interceptors = oldInterceptors
		lock.Unlock()
	}()

	chain := BuildInterceptorChain(mockInvoker)
	assert.Equal(t, mockInvoker, chain)
}

func TestBuildInterceptorChainWithBuiltins(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).AnyTimes()
	mockInvoker.EXPECT().IsAvailable().Return(true).AnyTimes()

	expectedResult := &result.RPCResult{Rest: Rest{Tried: 0, Success: true}}
	mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(expectedResult).Times(1)

	// Clear global interceptors for this test
	lock.Lock()
	oldInterceptors := interceptors
	interceptors = make(map[string]func() Interceptor)
	lock.Unlock()

	// Restore interceptors using defer to ensure cleanup even if test fails
	defer func() {
		lock.Lock()
		interceptors = oldInterceptors
		lock.Unlock()
	}()

	interceptor1 := &mockInterceptor{}
	interceptor2 := &mockInterceptor{}

	chain := BuildInterceptorChain(mockInvoker, interceptor1, interceptor2)
	assert.NotNil(t, chain)

	// Verify chain is built correctly
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))
	res := chain.Invoke(context.Background(), inv)

	assert.Equal(t, expectedResult, res)
	assert.True(t, interceptor1.invoked)
	assert.True(t, interceptor2.invoked)
}

func TestBuildInterceptorChainWithGlobalInterceptors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).AnyTimes()
	mockInvoker.EXPECT().IsAvailable().Return(true).AnyTimes()

	expectedResult := &result.RPCResult{Rest: Rest{Tried: 0, Success: true}}
	mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(expectedResult).Times(1)

	// Set up global interceptor
	globalInterceptor := &mockInterceptor{}
	SetClusterInterceptor("test-global-interceptor", func() Interceptor {
		return globalInterceptor
	})

	chain := BuildInterceptorChain(mockInvoker)
	assert.NotNil(t, chain)

	// Verify chain invokes global interceptor
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))
	res := chain.Invoke(context.Background(), inv)

	assert.Equal(t, expectedResult, res)
}

func TestBuildInterceptorChainOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).AnyTimes()
	mockInvoker.EXPECT().IsAvailable().Return(true).AnyTimes()

	expectedResult := &result.RPCResult{Rest: Rest{Tried: 0, Success: true}}
	mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(expectedResult).Times(1)

	// Clear global interceptors for this test
	lock.Lock()
	oldInterceptors := interceptors
	interceptors = make(map[string]func() Interceptor)
	lock.Unlock()

	// Restore interceptors using defer to ensure cleanup even if test fails
	defer func() {
		lock.Lock()
		interceptors = oldInterceptors
		lock.Unlock()
	}()

	var order []int
	interceptor1 := &orderTrackingInterceptor{id: 1, order: &order}
	interceptor2 := &orderTrackingInterceptor{id: 2, order: &order}

	chain := BuildInterceptorChain(mockInvoker, interceptor1, interceptor2)

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"))
	chain.Invoke(context.Background(), inv)

	// Interceptors should be invoked in order: 1, 2
	assert.Equal(t, []int{1, 2}, order)
}

type orderTrackingInterceptor struct {
	id    int
	order *[]int
}

func (o *orderTrackingInterceptor) Invoke(ctx context.Context, invoker base.Invoker, inv base.Invocation) result.Result {
	*o.order = append(*o.order, o.id)
	return invoker.Invoke(ctx, inv)
}

func TestInterceptorInvokerChainGetURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).Times(1)

	// Clear global interceptors
	lock.Lock()
	oldInterceptors := interceptors
	interceptors = make(map[string]func() Interceptor)
	lock.Unlock()

	// Restore interceptors using defer to ensure cleanup even if test fails
	defer func() {
		lock.Lock()
		interceptors = oldInterceptors
		lock.Unlock()
	}()

	chain := BuildInterceptorChain(mockInvoker, &mockInterceptor{})
	url := chain.GetURL()
	assert.Equal(t, testURL, url)
}

func TestInterceptorInvokerChainIsAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).AnyTimes()
	mockInvoker.EXPECT().IsAvailable().Return(true).Times(1)

	// Clear global interceptors
	lock.Lock()
	oldInterceptors := interceptors
	interceptors = make(map[string]func() Interceptor)
	lock.Unlock()

	// Restore interceptors using defer to ensure cleanup even if test fails
	defer func() {
		lock.Lock()
		interceptors = oldInterceptors
		lock.Unlock()
	}()

	chain := BuildInterceptorChain(mockInvoker, &mockInterceptor{})
	available := chain.IsAvailable()
	assert.True(t, available)
}

func TestInterceptorInvokerChainDestroy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	testURL, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	mockInvoker.EXPECT().GetURL().Return(testURL).AnyTimes()
	mockInvoker.EXPECT().Destroy().Times(1)

	// Clear global interceptors
	lock.Lock()
	oldInterceptors := interceptors
	interceptors = make(map[string]func() Interceptor)
	lock.Unlock()

	// Restore interceptors using defer to ensure cleanup even if test fails
	defer func() {
		lock.Lock()
		interceptors = oldInterceptors
		lock.Unlock()
	}()

	chain := BuildInterceptorChain(mockInvoker, &mockInterceptor{})
	chain.Destroy()
}
