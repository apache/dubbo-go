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
	"testing"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/structpb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/filter/generic/generalizer"
)

type testUser struct {
	Name    string
	Age     int
	Email   string
	Address *testAddress
}

func (testUser) JavaClassName() string {
	return "org.apache.dubbo.test.User"
}

type testAddress struct {
	City    string
	Country string
}

func TestGenericService(t *testing.T) {
	service := NewGenericService("HelloService")
	reference := service.Reference()
	assert.Equal(t, "HelloService", reference)
}

func TestGenericService_InvokeWithType(t *testing.T) {
	t.Run("simple struct", func(t *testing.T) {
		service := NewGenericService("TestService")
		service.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
			assert.Equal(t, "getUser", methodName)
			return map[string]any{
				"name":  "testUser",
				"age":   25,
				"email": "test@example.com",
			}, nil
		}

		var user testUser
		err := service.InvokeWithType(context.Background(), "getUser", []string{"java.lang.String"}, []hessian.Object{"123"}, &user)

		require.NoError(t, err)
		assert.Equal(t, "testUser", user.Name)
		assert.Equal(t, 25, user.Age)
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("nested struct", func(t *testing.T) {
		service := NewGenericService("TestService")
		service.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
			return map[string]any{
				"name":  "nestedUser",
				"age":   30,
				"email": "nested@example.com",
				"address": map[string]any{
					"city":    "Beijing",
					"country": "China",
				},
			}, nil
		}

		var user testUser
		err := service.InvokeWithType(context.Background(), "getUser", []string{"java.lang.String"}, []hessian.Object{"456"}, &user)

		require.NoError(t, err)
		assert.Equal(t, "nestedUser", user.Name)
		assert.Equal(t, 30, user.Age)
		require.NotNil(t, user.Address)
		assert.Equal(t, "Beijing", user.Address.City)
		assert.Equal(t, "China", user.Address.Country)
	})

	t.Run("slice result", func(t *testing.T) {
		service := NewGenericService("TestService")
		service.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
			return []any{
				map[string]any{"name": "user1", "age": 20},
				map[string]any{"name": "user2", "age": 25},
			}, nil
		}

		var users []testUser
		err := service.InvokeWithType(context.Background(), "listUsers", []string{}, []hessian.Object{}, &users)

		require.NoError(t, err)
		require.Len(t, users, 2)
		assert.Equal(t, "user1", users[0].Name)
		assert.Equal(t, 20, users[0].Age)
		assert.Equal(t, "user2", users[1].Name)
		assert.Equal(t, 25, users[1].Age)
	})

	t.Run("nil result", func(t *testing.T) {
		service := NewGenericService("TestService")
		service.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
			return nil, nil
		}

		var user testUser
		err := service.InvokeWithType(context.Background(), "getUser", []string{"java.lang.String"}, []hessian.Object{"789"}, &user)

		require.NoError(t, err)
		assert.Empty(t, user.Name)
		assert.Zero(t, user.Age)
	})

	t.Run("nil reply error", func(t *testing.T) {
		service := NewGenericService("TestService")
		// Add dummy invoke to prevent nil pointer dereference
		service.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
			return nil, nil
		}

		err := service.InvokeWithType(context.Background(), "getUser", []string{"java.lang.String"}, []hessian.Object{"123"}, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reply cannot be nil")
	})

	t.Run("non-pointer reply error", func(t *testing.T) {
		service := NewGenericService("TestService")
		// Add dummy invoke to prevent nil pointer dereference
		service.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
			return nil, nil
		}

		var user testUser
		err := service.InvokeWithType(context.Background(), "getUser", []string{"java.lang.String"}, []hessian.Object{"123"}, user)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reply must be a pointer")
	})

	t.Run("invoke error", func(t *testing.T) {
		service := NewGenericService("TestService")
		service.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
			return nil, assert.AnError
		}

		var user testUser
		err := service.InvokeWithType(context.Background(), "getUser", []string{"java.lang.String"}, []hessian.Object{"123"}, &user)

		require.Error(t, err)
		assert.Equal(t, assert.AnError, err)
	})

	t.Run("deserialization error", func(t *testing.T) {
		service := NewGenericService("TestService")
		service.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
			// Return a type that mismatches the target struct
			return map[string]any{
				"age": "invalid_age_type", // string cannot be unmarshaled to int
			}, nil
		}

		var user testUser
		err := service.InvokeWithType(context.Background(), "getUser", []string{"java.lang.String"}, []hessian.Object{"123"}, &user)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to deserialize result")
	})
}

func TestGenericService_InvokeWithTypeUsesGenericMode(t *testing.T) {
	tests := []struct {
		name   string
		mode   string
		result any
	}{
		{
			name: constant.GenericSerializationDefault,
			mode: constant.GenericSerializationDefault,
			result: map[string]any{
				"name": "mapUser",
				"age":  31,
			},
		},
		{
			name:   constant.GenericSerializationGson,
			mode:   constant.GenericSerializationGson,
			result: `{"name":"gsonUser","age":32}`,
		},
		{
			name: constant.GenericSerializationBean,
			mode: constant.GenericSerializationBean,
			result: mustGeneralize(t, generalizer.GetBeanGeneralizer(), testUser{
				Name: "beanUser",
				Age:  33,
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewGenericService("TestService")
			require.NoError(t, service.SetGenericType(tt.mode))
			service.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
				return tt.result, nil
			}

			var user testUser
			err := service.InvokeWithType(context.Background(), "getUser", nil, nil, &user)

			require.NoError(t, err)
			assert.NotEmpty(t, user.Name)
			assert.NotZero(t, user.Age)
		})
	}
}

func TestGenericService_InvokeWithTypeUsesProtobufJsonMode(t *testing.T) {
	service := NewGenericService("TestService")
	require.NoError(t, service.SetGenericType(constant.GenericSerializationProtobufJson))
	service.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
		return `{"name":"protoUser"}`, nil
	}

	var reply structpb.Struct
	err := service.InvokeWithType(context.Background(), "getUser", nil, nil, &reply)

	require.NoError(t, err)
	assert.Equal(t, "protoUser", reply.Fields["name"].GetStringValue())
}

func TestGenericService_InvokeWithTypeRejectsDisabledGenericMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "empty", mode: ""},
		{name: "false", mode: "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewGenericService("TestService")
			require.NoError(t, service.SetGenericType(tt.mode))
			service.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
				t.Fatal("InvokeWithType should reject disabled generic mode before invoking")
				return nil, nil
			}

			var user testUser
			err := service.InvokeWithType(context.Background(), "getUser", nil, nil, &user)

			require.EqualError(t, err, `generic mode "`+tt.mode+`" does not support typed result`)
		})
	}
}

func mustGeneralize(t *testing.T, g generalizer.Generalizer, obj any) any {
	t.Helper()
	result, err := g.Generalize(obj)
	require.NoError(t, err)
	return result
}
