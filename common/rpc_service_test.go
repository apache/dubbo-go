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

const (
	referenceTestPath             = "com.test.Path"
	referenceTestPathDistinct     = "com.test.Path1"
	testInterfaceName             = "testService"
	testProtocol                  = "testprotocol"
	testSuiteMethodExpectedString = "interface {}"
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
	return referenceTestPath
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
	return referenceTestPath
}

type TestService1 struct {
}

func (s *TestService1) Reference() string {
	return referenceTestPathDistinct
}

func TestServiceMapRegister(t *testing.T) {
	// lowercase
	s0 := &testService{}
	// methods, err := ServiceMap.Register("testporotocol", s0)
	_, err := ServiceMap.Register(testInterfaceName, "testporotocol", "", "v0", s0)
	assert.EqualError(t, err, "type testService is not exported")

	// succ
	s := &TestService{}
	methods, err := ServiceMap.Register(testInterfaceName, "testporotocol", "", "v1", s)
	assert.NoError(t, err)
	assert.Equal(t, "MethodOne,MethodThree,methodTwo", methods)

	// repeat
	_, err = ServiceMap.Register(testInterfaceName, "testporotocol", "", "v1", s)
	assert.EqualError(t, err, "service already defined: testService:v1")

	// no method
	s1 := &TestService1{}
	_, err = ServiceMap.Register(testInterfaceName, "testporotocol", "", "v2", s1)
	assert.EqualError(t, err, "type testService:v2 has no exported methods of suitable type")

	ServiceMap = &serviceMap{
		serviceMap:   make(map[string]map[string]*Service),
		interfaceMap: make(map[string][]*Service),
	}
}

func TestServiceMapUnRegister(t *testing.T) {
	s := &TestService{}
	_, err := ServiceMap.Register("TestService", testProtocol, "", "v1", s)
	assert.NoError(t, err)
	assert.NotNil(t, ServiceMap.GetService(testProtocol, "TestService", "", "v1"))
	assert.Equal(t, 1, len(ServiceMap.GetInterface("TestService")))

	err = ServiceMap.UnRegister("", "", ServiceKey("TestService", "", "v1"))
	assert.EqualError(t, err, "protocol or serviceKey is nil")

	err = ServiceMap.UnRegister("", "protocol", ServiceKey("TestService", "", "v1"))
	assert.EqualError(t, err, "no services for protocol")

	err = ServiceMap.UnRegister("", testProtocol, ServiceKey("TestService", "", "v0"))
	assert.EqualError(t, err, "no service for TestService:v0")

	// succ
	err = ServiceMap.UnRegister("TestService", testProtocol, ServiceKey("TestService", "", "v1"))
	assert.NoError(t, err)
}

func TestMethodTypeSuiteContext(t *testing.T) {
	mt := &MethodType{ctxType: reflect.TypeOf(context.TODO())}
	ctx := context.WithValue(context.Background(), "key", "value")
	assert.Equal(t, reflect.ValueOf(ctx), mt.SuiteContext(ctx))

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
	assert.Equal(t, testSuiteMethodExpectedString, at[0].String())
	assert.Equal(t, testSuiteMethodExpectedString, at[1].String())
	assert.Equal(t, testSuiteMethodExpectedString, at[2].String())
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
	assert.Equal(t, testSuiteMethodExpectedString, at[0].String())
	assert.Equal(t, testSuiteMethodExpectedString, at[1].String())
	assert.Equal(t, testSuiteMethodExpectedString, at[2].String())
	assert.Nil(t, methodType.CtxType())
	rt = methodType.ReplyType()
	assert.Equal(t, testSuiteMethodExpectedString, rt.String())

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
