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

package common

import (
	"context"
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

type TestService struct {
}

func (s *TestService) MethodOne(ctx context.Context, arg1, arg2, arg3 interface{}) error {
	return nil
}
func (s *TestService) MethodTwo(arg1, arg2, arg3 interface{}) (interface{}, error) {
	return struct{}{}, nil
}
func (s *TestService) MethodThree() error {
	return nil
}
func (s *TestService) Reference() string {
	return "com.test.Path"
}
func (s *TestService) MethodMapper() map[string]string {
	return map[string]string{
		"MethodTwo": "methodTwo",
	}
}

type testService struct {
}

func (s *testService) Method1(ctx context.Context, args testService, rsp *struct{}) error {
	return nil
}
func (s *testService) Method2(ctx context.Context, args []interface{}) (testService, error) {
	return testService{}, nil
}
func (s *testService) Method3(ctx context.Context, args []interface{}, rsp *struct{}) {
}
func (s *testService) Method4(ctx context.Context, args []interface{}, rsp *struct{}) *testService {
	return nil
}
func (s *testService) Reference() string {
	return "com.test.Path"
}

type TestService1 struct {
}

func (s *TestService1) Reference() string {
	return "com.test.Path1"
}

func TestServiceMap_Register(t *testing.T) {
	// lowercase
	s0 := &testService{}
	// methods, err := ServiceMap.Register("testporotocol", s0)
	_, err := ServiceMap.Register("testporotocol", s0)
	assert.EqualError(t, err, "type testService is not exported")

	// succ
	s := &TestService{}
	methods, err := ServiceMap.Register("testporotocol", s)
	assert.NoError(t, err)
	assert.Equal(t, "MethodOne,MethodThree,methodTwo", methods)

	// repeat
	_, err = ServiceMap.Register("testporotocol", s)
	assert.EqualError(t, err, "service already defined: com.test.Path")

	// no method
	s1 := &TestService1{}
	_, err = ServiceMap.Register("testporotocol", s1)
	assert.EqualError(t, err, "type com.test.Path1 has no exported methods of suitable type")

	ServiceMap = &serviceMap{
		serviceMap: make(map[string]map[string]*Service),
	}
}

func TestServiceMap_UnRegister(t *testing.T) {
	s := &TestService{}
	_, err := ServiceMap.Register("testprotocol", s)
	assert.NoError(t, err)
	assert.NotNil(t, ServiceMap.GetService("testprotocol", "com.test.Path"))

	err = ServiceMap.UnRegister("", "com.test.Path")
	assert.EqualError(t, err, "protocol or serviceName is nil")

	err = ServiceMap.UnRegister("protocol", "com.test.Path")
	assert.EqualError(t, err, "no services for protocol")

	err = ServiceMap.UnRegister("testprotocol", "com.test.Path1")
	assert.EqualError(t, err, "no service for com.test.Path1")

	// succ
	err = ServiceMap.UnRegister("testprotocol", "com.test.Path")
	assert.NoError(t, err)
}

func TestMethodType_SuiteContext(t *testing.T) {
	mt := &MethodType{ctxType: reflect.TypeOf(context.TODO())}
	c := context.TODO()
	c = context.WithValue(c, "key", "value")
	assert.Equal(t, reflect.ValueOf(c), mt.SuiteContext(c))

	assert.Equal(t, reflect.Zero(mt.ctxType), mt.SuiteContext(nil))
}

func TestSuiteMethod(t *testing.T) {

	s := &TestService{}
	method, ok := reflect.TypeOf(s).MethodByName("MethodOne")
	assert.True(t, ok)
	methodType := suiteMethod(method)
	method = methodType.Method()
	assert.Equal(t, "func(*common.TestService, context.Context, interface {}, interface {}, interface {}) error", method.Type.String())
	at := methodType.ArgsType()
	assert.Equal(t, "interface {}", at[0].String())
	assert.Equal(t, "interface {}", at[1].String())
	assert.Equal(t, "interface {}", at[2].String())
	ct := methodType.CtxType()
	assert.Equal(t, "context.Context", ct.String())
	rt := methodType.ReplyType()
	assert.Nil(t, rt)

	method, ok = reflect.TypeOf(s).MethodByName("MethodTwo")
	assert.True(t, ok)
	methodType = suiteMethod(method)
	method = methodType.Method()
	assert.Equal(t, "func(*common.TestService, interface {}, interface {}, interface {}) (interface {}, error)", method.Type.String())
	at = methodType.ArgsType()
	assert.Equal(t, "interface {}", at[0].String())
	assert.Equal(t, "interface {}", at[1].String())
	assert.Equal(t, "interface {}", at[2].String())
	assert.Nil(t, methodType.CtxType())
	rt = methodType.ReplyType()
	assert.Equal(t, "interface {}", rt.String())

	method, ok = reflect.TypeOf(s).MethodByName("MethodThree")
	assert.True(t, ok)
	methodType = suiteMethod(method)
	method = methodType.Method()
	assert.Equal(t, "func(*common.TestService) error", method.Type.String())
	at = methodType.ArgsType()
	assert.Equal(t, 0, len(at))
	assert.Nil(t, methodType.CtxType())
	rt = methodType.ReplyType()
	assert.Nil(t, rt)

	// wrong number of in return
	s1 := &testService{}
	method, ok = reflect.TypeOf(s1).MethodByName("Reference")
	assert.True(t, ok)
	methodType = suiteMethod(method)
	assert.Nil(t, methodType)

	// args not exported
	method, ok = reflect.TypeOf(s1).MethodByName("Method1")
	assert.True(t, ok)
	methodType = suiteMethod(method)
	assert.Nil(t, methodType)

	// Reply not exported
	method, ok = reflect.TypeOf(s1).MethodByName("Method2")
	assert.True(t, ok)
	methodType = suiteMethod(method)
	assert.Nil(t, methodType)

	// no return
	method, ok = reflect.TypeOf(s1).MethodByName("Method3")
	assert.True(t, ok)
	methodType = suiteMethod(method)
	assert.Nil(t, methodType)

	// return value is not error
	method, ok = reflect.TypeOf(s1).MethodByName("Method4")
	assert.True(t, ok)
	methodType = suiteMethod(method)
	assert.Nil(t, methodType)
}
