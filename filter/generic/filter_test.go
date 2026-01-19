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

package generic

import (
	"context"
	"net/url"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"

	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// test isCallingToGenericService branch
func TestFilter_Invoke(t *testing.T) {
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GenericKey, constant.GenericSerializationDefault))
	filter := &genericFilter{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	normalInvocation := invocation.NewRPCInvocation("Hello", []any{"arg1"}, make(map[string]any))

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetURL().Return(invokeUrl).Times(2)
	mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Not(normalInvocation)).DoAndReturn(
		func(ctx context.Context, invocation base.Invocation) result.Result {
			assert.Equal(t, constant.Generic, invocation.MethodName())
			args := invocation.Arguments()
			assert.Equal(t, "Hello", args[0])
			assert.Equal(t, "java.lang.String", args[1].([]string)[0])
			assert.Equal(t, "arg1", args[2].([]hessian.Object)[0].(string))
			assert.Equal(t, constant.GenericSerializationDefault, invocation.GetAttachmentWithDefaultValue(constant.GenericKey, ""))
			return &result.RPCResult{}
		})

	r := filter.Invoke(context.Background(), mockInvoker, normalInvocation)
	assert.NotNil(t, r)
}

// test isMakingAGenericCall branch
func TestFilter_InvokeWithGenericCall(t *testing.T) {
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GenericKey, constant.GenericSerializationDefault))
	filter := &genericFilter{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	genericInvocation := invocation.NewRPCInvocation(constant.Generic, []any{
		"hello",
		[]string{"java.lang.String"},
		[]string{"arg1"},
	}, make(map[string]any))

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetURL().Return(invokeUrl).Times(3)
	mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(
		func(arg0 context.Context, invocation base.Invocation) result.Result {
			assert.Equal(t, constant.Generic, invocation.MethodName())
			args := invocation.Arguments()
			assert.Equal(t, "hello", args[0])
			assert.Equal(t, "java.lang.String", args[1].([]string)[0])
			assert.Equal(t, "arg1", args[2].([]string)[0])
			assert.Equal(t, constant.GenericSerializationDefault, invocation.GetAttachmentWithDefaultValue(constant.GenericKey, ""))
			return &result.RPCResult{}
		})

	r := filter.Invoke(context.Background(), mockInvoker, genericInvocation)
	assert.NotNil(t, r)
}

// mockUser is a test struct for OnResponse deserialization
type mockUser struct {
	Name    string
	Age     int
	Email   string
	Address *mockAddress
}

type mockAddress struct {
	City    string
	Country string
}

// test OnResponse with struct deserialization
func TestFilter_OnResponse_WithStructDeserialization(t *testing.T) {
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GenericKey, constant.GenericSerializationDefault))
	filter := &genericFilter{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetURL().Return(invokeUrl).AnyTimes()

	t.Run("simple struct deserialization", func(t *testing.T) {
		// Create a reply pointer for the target struct
		var user mockUser
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName(constant.Generic),
			invocation.WithReply(&user),
		)

		// Simulate a map result from generic call
		mapResult := map[string]any{
			"name":  "testUser",
			"age":   25,
			"email": "test@example.com",
		}
		res := &result.RPCResult{Rest: mapResult}

		// Call OnResponse
		newRes := filter.OnResponse(context.Background(), res, mockInvoker, inv)

		// Verify the result is deserialized into user struct
		require.NoError(t, newRes.Error())
		assert.Equal(t, "testUser", user.Name)
		assert.Equal(t, 25, user.Age)
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("nested struct deserialization", func(t *testing.T) {
		var user mockUser
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName(constant.Generic),
			invocation.WithReply(&user),
		)

		// Simulate a nested map result
		mapResult := map[string]any{
			"name":  "nestedUser",
			"age":   30,
			"email": "nested@example.com",
			"address": map[string]any{
				"city":    "Beijing",
				"country": "China",
			},
		}
		res := &result.RPCResult{Rest: mapResult}

		newRes := filter.OnResponse(context.Background(), res, mockInvoker, inv)

		require.NoError(t, newRes.Error())
		assert.Equal(t, "nestedUser", user.Name)
		assert.Equal(t, 30, user.Age)
		require.NotNil(t, user.Address)
		assert.Equal(t, "Beijing", user.Address.City)
		assert.Equal(t, "China", user.Address.Country)
	})

	t.Run("nil reply - no deserialization", func(t *testing.T) {
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName(constant.Generic),
			invocation.WithReply(nil),
		)

		mapResult := map[string]any{"name": "test"}
		res := &result.RPCResult{Rest: mapResult}

		newRes := filter.OnResponse(context.Background(), res, mockInvoker, inv)

		// Result should remain as map since no reply pointer provided
		assert.Equal(t, mapResult, newRes.Result())
	})

	t.Run("nil result - no deserialization", func(t *testing.T) {
		var user mockUser
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName(constant.Generic),
			invocation.WithReply(&user),
		)

		res := &result.RPCResult{Rest: nil}

		newRes := filter.OnResponse(context.Background(), res, mockInvoker, inv)

		// User should remain zero value
		assert.Empty(t, user.Name)
		assert.Equal(t, 0, user.Age)
		assert.Nil(t, newRes.Result())
	})

	t.Run("primitive result - no deserialization", func(t *testing.T) {
		var count int
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName(constant.Generic),
			invocation.WithReply(&count),
		)

		// Primitive result should pass through without deserialization
		res := &result.RPCResult{Rest: 42}

		newRes := filter.OnResponse(context.Background(), res, mockInvoker, inv)

		// Result should remain as primitive
		assert.Equal(t, 42, newRes.Result())
	})

	t.Run("error result - no deserialization", func(t *testing.T) {
		var user mockUser
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName(constant.Generic),
			invocation.WithReply(&user),
		)

		res := &result.RPCResult{
			Rest: map[string]any{"name": "test"},
			Err:  assert.AnError,
		}

		newRes := filter.OnResponse(context.Background(), res, mockInvoker, inv)

		// Should return immediately without deserialization when there's an error
		require.Error(t, newRes.Error())
		assert.Empty(t, user.Name)
	})
}

func TestFilter_OnResponse_WithSliceDeserialization(t *testing.T) {
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GenericKey, constant.GenericSerializationDefault))
	filter := &genericFilter{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetURL().Return(invokeUrl).AnyTimes()

	var users []mockUser
	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName(constant.Generic),
		invocation.WithReply(&users),
	)

	// Simulate a slice of maps result
	sliceResult := []any{
		map[string]any{
			"name": "user1",
			"age":  20,
		},
		map[string]any{
			"name": "user2",
			"age":  25,
		},
	}
	res := &result.RPCResult{Rest: sliceResult}

	newRes := filter.OnResponse(context.Background(), res, mockInvoker, inv)

	require.NoError(t, newRes.Error())
	require.Len(t, users, 2)
	assert.Equal(t, "user1", users[0].Name)
	assert.Equal(t, 20, users[0].Age)
	assert.Equal(t, "user2", users[1].Name)
	assert.Equal(t, 25, users[1].Age)
}
