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
)

type testUser struct {
	Name    string
	Age     int
	Email   string
	Address *testAddress
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

		err := service.InvokeWithType(context.Background(), "getUser", []string{"java.lang.String"}, []hessian.Object{"123"}, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reply cannot be nil")
	})

	t.Run("non-pointer reply error", func(t *testing.T) {
		service := NewGenericService("TestService")

		var user testUser
		err := service.InvokeWithType(context.Background(), "getUser", []string{"java.lang.String"}, []hessian.Object{"123"}, user)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reply must be a pointer")
	})
}
