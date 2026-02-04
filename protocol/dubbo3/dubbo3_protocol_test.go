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

package dubbo3

import (
	"context"
	"reflect"
	"testing"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockUser struct {
	Name string
}

func (m *MockUser) JavaClassName() string {
	return "mockuser"
}

type MockService struct {
}

func (m *MockService) GetUser(ctx context.Context, user, user2 *MockUser) (*MockUser, error) {
	return user, nil
}

func TestDubbo3UnaryService_GetReqParamsInterfaces(t *testing.T) {
	hessian.RegisterPOJO(&MockUser{})
	srv := UnaryService{}
	valueOf := reflect.ValueOf(&MockService{})
	typeOf := valueOf.Type()
	numField := valueOf.NumMethod()
	for i := 0; i < numField; i++ {
		ft := typeOf.Method(i)
		// num in/out is checked in common/rpc_service.go
		typs := make([]reflect.Type, 0)
		for j := 2; j < ft.Type.NumIn(); j++ {
			typs = append(typs, ft.Type.In(j))
		}
		srv.setReqParamsTypes("GetUser", typs)
	}
	paramsInterfaces, ok := srv.GetReqParamsInterfaces("GetUser")
	assert.True(t, ok)
	enc := hessian.NewEncoder()
	err := enc.Encode(&MockUser{
		Name: "laurence",
	})
	require.NoError(t, err)
	data := enc.Buffer()
	decoder := hessian.NewDecoder(data)
	val, err := decoder.Decode()
	require.NoError(t, err)
	assert.Len(t, paramsInterfaces, 2)
	subTest(t, val, paramsInterfaces)
	args := make([]any, 0, 1)
	for _, v := range paramsInterfaces {
		tempParamObj := reflect.ValueOf(v).Elem().Interface()
		args = append(args, tempParamObj)
	}
	assert.Equal(t, "laurence", args[0].(*MockUser).Name)
	assert.Equal(t, "laurence", args[1].(*MockUser).Name)
}

func subTest(t *testing.T, val, paramsInterfaces any) {
	list := paramsInterfaces.([]any)
	for k := range list {
		err := hessian.ReflectResponse(val, list[k])
		require.NoError(t, err)
	}
}
