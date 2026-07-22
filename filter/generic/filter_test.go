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
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/filter/generic/generalizer"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type mockGenericPOJO struct {
	Name string
}

func (*mockGenericPOJO) JavaClassName() string {
	return "org.apache.dubbo.test.MockGenericPOJO"
}

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

func TestFilter_InvokeUsesConfiguredGenericMode(t *testing.T) {
	tests := []struct {
		name       string
		mode       string
		arg        any
		resultType any
	}{
		{
			name:       "map",
			mode:       constant.GenericSerializationDefault,
			arg:        &mockGenericPOJO{Name: "map"},
			resultType: map[string]any{},
		},
		{
			name:       "bean",
			mode:       constant.GenericSerializationBean,
			arg:        &mockGenericPOJO{Name: "bean"},
			resultType: &generalizer.JavaBeanDescriptor{},
		},
		{
			name:       "gson",
			mode:       constant.GenericSerializationGson,
			arg:        &mockGenericPOJO{Name: "gson"},
			resultType: "",
		},
		{
			name:       "protobuf-json",
			mode:       constant.GenericSerializationProtobufJson,
			arg:        &generalizer.RequestType{Id: 1},
			resultType: "",
		},
		{
			name:       "protobuf-legacy",
			mode:       constant.GenericSerializationProtobuf,
			arg:        &mockGenericPOJO{Name: "protobuf"},
			resultType: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invokeURL := common.NewURLWithOptions(
				common.WithParams(url.Values{}),
				common.WithParamsValue(constant.GenericKey, tt.mode),
			)
			inv := invocation.NewRPCInvocation("Hello", []any{tt.arg}, nil)

			ctrl := gomock.NewController(t)
			mockInvoker := mock.NewMockInvoker(ctrl)
			mockInvoker.EXPECT().GetURL().Return(invokeURL).Times(2)
			mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, outgoing base.Invocation) result.Result {
					assert.Equal(t, tt.mode, outgoing.GetAttachmentWithDefaultValue(constant.GenericKey, ""))
					assert.IsType(t, tt.resultType, outgoing.Arguments()[2].([]hessian.Object)[0])
					return &result.RPCResult{}
				},
			)

			res := (&genericFilter{}).Invoke(context.Background(), mockInvoker, inv)

			require.NoError(t, res.Error())
		})
	}
}

func TestFilter_InvokeInvocationGenericModeOverridesURL(t *testing.T) {
	invokeURL := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GenericKey, constant.GenericSerializationDefault),
	)
	inv := invocation.NewRPCInvocation("Hello", []any{&mockGenericPOJO{Name: "bean"}}, map[string]any{
		constant.GenericKey: constant.GenericSerializationBean,
	})

	ctrl := gomock.NewController(t)
	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetURL().Return(invokeURL).Times(2)
	mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, outgoing base.Invocation) result.Result {
			assert.Equal(t, constant.GenericSerializationBean,
				outgoing.GetAttachmentWithDefaultValue(constant.GenericKey, ""))
			assert.IsType(t, &generalizer.JavaBeanDescriptor{}, outgoing.Arguments()[2].([]hessian.Object)[0])
			return &result.RPCResult{}
		},
	)

	res := (&genericFilter{}).Invoke(context.Background(), mockInvoker, inv)

	require.NoError(t, res.Error())
}

func TestFilter_InvokeWithUnsupportedGenericMode(t *testing.T) {
	filter := &genericFilter{}

	tests := []struct {
		name    string
		generic string
	}{
		{
			name:    "unknown",
			generic: "unsupported_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invokeURL := common.NewURLWithOptions(
				common.WithParams(url.Values{}),
				common.WithParamsValue(constant.GenericKey, tt.generic),
			)
			inv := invocation.NewRPCInvocation("Hello", []any{"arg1"}, nil)

			ctrl := gomock.NewController(t)
			mockInvoker := mock.NewMockInvoker(ctrl)
			mockInvoker.EXPECT().GetURL().Return(invokeURL).Times(1)
			mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Times(0)

			res := filter.Invoke(context.Background(), mockInvoker, inv)

			require.EqualError(t, res.Error(), `unsupported generic mode "`+tt.generic+`"`)
			assert.Nil(t, res.Result())
		})
	}
}

func TestFilter_InvokeRejectsUnsupportedInvocationGenericMode(t *testing.T) {
	filter := &genericFilter{}
	invokeURL := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GenericKey, constant.GenericSerializationDefault),
	)

	for _, generic := range []string{"unsupported_type"} {
		t.Run(generic, func(t *testing.T) {
			inv := invocation.NewRPCInvocation("Hello", []any{"arg1"}, map[string]any{
				constant.GenericKey: generic,
			})
			ctrl := gomock.NewController(t)
			mockInvoker := mock.NewMockInvoker(ctrl)
			mockInvoker.EXPECT().GetURL().Return(invokeURL).Times(1)
			mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Times(0)

			res := filter.Invoke(context.Background(), mockInvoker, inv)

			require.EqualError(t, res.Error(), `unsupported generic mode "`+generic+`"`)
			assert.Nil(t, res.Result())
		})
	}
}

func TestFilter_InvokeEmptyInvocationGenericModeUsesConfiguredMode(t *testing.T) {
	filter := &genericFilter{}
	invokeURL := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GenericKey, constant.GenericSerializationDefault),
	)
	inv := invocation.NewRPCInvocation("Hello", []any{&mockGenericPOJO{Name: "map"}}, map[string]any{
		constant.GenericKey: "",
	})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetURL().Return(invokeURL).Times(2)
	mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, outgoing base.Invocation) result.Result {
			assert.Equal(t, constant.GenericSerializationDefault,
				outgoing.GetAttachmentWithDefaultValue(constant.GenericKey, ""))
			assert.IsType(t, map[string]any{}, outgoing.Arguments()[2].([]hessian.Object)[0])
			return &result.RPCResult{}
		},
	)

	res := filter.Invoke(context.Background(), mockInvoker, inv)

	require.NoError(t, res.Error())
}

func TestFilter_InvokeWithoutGenericModePassesThrough(t *testing.T) {
	filter := &genericFilter{}
	invokeURL := common.NewURLWithOptions(common.WithParams(url.Values{}))
	inv := invocation.NewRPCInvocation("Hello", []any{"arg1"}, nil)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetURL().Return(invokeURL).Times(1)
	mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Eq(inv)).Return(&result.RPCResult{Rest: "ok"})

	res := filter.Invoke(context.Background(), mockInvoker, inv)

	require.NoError(t, res.Error())
	assert.Equal(t, "ok", res.Result())
}

func TestFilter_InvokeWithGenericFalsePassesThrough(t *testing.T) {
	filter := &genericFilter{}
	invokeURL := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GenericKey, "false"),
	)
	inv := invocation.NewRPCInvocation("Hello", []any{"arg1"}, nil)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetURL().Return(invokeURL).Times(1)
	mockInvoker.EXPECT().Invoke(gomock.Any(), gomock.Eq(inv)).Return(&result.RPCResult{Rest: "ok"})

	res := filter.Invoke(context.Background(), mockInvoker, inv)

	require.NoError(t, res.Error())
	assert.Equal(t, "ok", res.Result())
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

func TestFilter_OnResponse_WithUnsupportedGenericMode(t *testing.T) {
	invokeURL := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GenericKey, "unsupported_type"),
	)
	filter := &genericFilter{}

	ctrl := gomock.NewController(t)
	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetURL().Return(invokeURL).Times(1)

	res := &result.RPCResult{Rest: map[string]any{"name": "test"}}
	newRes := filter.OnResponse(context.Background(), res, mockInvoker, invocation.NewRPCInvocation("Hello", nil, nil))

	require.EqualError(t, newRes.Error(), `unsupported generic mode "unsupported_type"`)
	assert.Nil(t, newRes.Result())
}

// TestFilter_OnResponse_DeserializationError tests that OnResponse gracefully handles
// deserialization failures by logging a warning and returning the original result.
func TestFilter_OnResponse_DeserializationError(t *testing.T) {
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GenericKey, constant.GenericSerializationDefault))
	filter := &genericFilter{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetURL().Return(invokeUrl).AnyTimes()

	t.Run("type mismatch - string to int", func(t *testing.T) {
		var user mockUser
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName(constant.Generic),
			invocation.WithReply(&user),
		)

		// Return a map with incompatible type (string instead of int for age)
		mapResult := map[string]any{
			"name": "testUser",
			"age":  "not_an_int", // This should cause deserialization to fail
		}
		res := &result.RPCResult{Rest: mapResult}

		newRes := filter.OnResponse(context.Background(), res, mockInvoker, inv)

		// OnResponse should return the original result when deserialization fails
		// The user struct should remain unchanged (zero values)
		assert.Empty(t, user.Name)
		assert.Equal(t, 0, user.Age)
		// The result should still be the original map
		assert.Equal(t, mapResult, newRes.Result())
	})
}
