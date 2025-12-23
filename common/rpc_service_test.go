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

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

const (
	referenceTestPath             = "com.test.Path"
	referenceTestPathDistinct     = "com.test.Path1"
	testInterfaceName             = "testService"
	testProtocol                  = "testprotocol"
	testSuiteMethodExpectedString = "interface {}"
)

type TestService struct{}

func (s *TestService) MethodOne(ctx context.Context, arg1, arg2, arg3 any) error {
	return nil
}

func (s *TestService) MethodTwo(arg1, arg2, arg3 any) (any, error) {
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

type testService struct{}

func (s *testService) Method1(ctx context.Context, args testService, rsp *struct{}) error {
	return nil
}

func (s *testService) Method2(ctx context.Context, args []any) (testService, error) {
	return testService{}, nil
}

func (s *testService) Method3(ctx context.Context, args []any, rsp *struct{}) {
}

func (s *testService) Method4(ctx context.Context, args []any, rsp *struct{}) *testService {
	return nil
}

func (s *testService) Reference() string {
	return referenceTestPath
}

type TestService1 struct{}

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
	assert.Equal(t, "MethodOne,methodOne,MethodThree,methodThree,methodTwo,MethodTwo", methods)

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
	assert.EqualError(t, err, "protocol or ServiceKey is nil")

	err = ServiceMap.UnRegister("", "protocol", ServiceKey("TestService", "", "v1"))
	assert.EqualError(t, err, "no services for protocol")

	err = ServiceMap.UnRegister("", testProtocol, ServiceKey("TestService", "", "v0"))
	assert.EqualError(t, err, "no service for TestService:v0")

	// success
	err = ServiceMap.UnRegister("TestService", testProtocol, ServiceKey("TestService", "", "v1"))
	assert.NoError(t, err)
}

func TestMethodTypeSuiteContext(t *testing.T) {
	mt := &MethodType{ctxType: reflect.TypeOf(context.TODO())}
	ctx := context.Background()
	key := constant.DubboCtxKey("key")
	ctx = context.WithValue(ctx, key, "value")
	assert.Equal(t, reflect.ValueOf(ctx), mt.SuiteContext(ctx))
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

type ServiceWithoutRef struct{}

func TestGetReference(t *testing.T) {
	s0 := &TestService{}
	ref0 := GetReference(s0)
	assert.Equal(t, referenceTestPath, ref0)

	//s1 := TestService{}
	//ref1 := GetReference(s1)
	//assert.Equal(t, referenceTestPath, ref1)

	s2 := &struct {
		TestService
	}{}
	ref2 := GetReference(s2)
	assert.Equal(t, referenceTestPath, ref2)

	expectedReference := "ServiceWithoutRef"
	s3 := &ServiceWithoutRef{}
	ref3 := GetReference(s3)
	assert.Equal(t, expectedReference, ref3)

	s4 := ServiceWithoutRef{}
	ref4 := GetReference(s4)
	assert.Equal(t, expectedReference, ref4)

	s5 := &struct {
		ServiceWithoutRef
	}{}
	ref5 := GetReference(s5)
	assert.Equal(t, expectedReference, ref5)
}

// Additional tests for better coverage

func TestServiceMethods(t *testing.T) {
	s := &TestService{}
	_, err := ServiceMap.Register("TestServiceMethods", testProtocol, "group1", "v1", s)
	assert.NoError(t, err)

	service := ServiceMap.GetService(testProtocol, "TestServiceMethods", "group1", "v1")
	assert.NotNil(t, service)

	// Test Service.Method()
	methods := service.Method()
	assert.NotNil(t, methods)
	assert.True(t, len(methods) > 0)

	// Test Service.Name()
	assert.Equal(t, "group1/TestServiceMethods:v1", service.Name())

	// Test Service.ServiceType()
	svcType := service.ServiceType()
	assert.NotNil(t, svcType)
	assert.Equal(t, "*common.TestService", svcType.String())

	// Test Service.Service()
	svcValue := service.Service()
	assert.True(t, svcValue.IsValid())

	// Cleanup
	ServiceMap.UnRegister("TestServiceMethods", testProtocol, ServiceKey("TestServiceMethods", "group1", "v1"))
}

func TestGetServiceByServiceKey(t *testing.T) {
	s := &TestService{}
	_, err := ServiceMap.Register("TestGetServiceByKey", testProtocol, "", "v1", s)
	assert.NoError(t, err)

	// Test GetServiceByServiceKey - found
	serviceKey := ServiceKey("TestGetServiceByKey", "", "v1")
	service := ServiceMap.GetServiceByServiceKey(testProtocol, serviceKey)
	assert.NotNil(t, service)

	// Test GetServiceByServiceKey - protocol not found
	service = ServiceMap.GetServiceByServiceKey("nonexistent", serviceKey)
	assert.Nil(t, service)

	// Test GetServiceByServiceKey - service key not found
	service = ServiceMap.GetServiceByServiceKey(testProtocol, "nonexistent:v1")
	assert.Nil(t, service)

	// Cleanup
	ServiceMap.UnRegister("TestGetServiceByKey", testProtocol, serviceKey)
}

func TestGetInterface(t *testing.T) {
	s := &TestService{}
	_, err := ServiceMap.Register("TestGetInterface", testProtocol, "", "v1", s)
	assert.NoError(t, err)

	// Test GetInterface - found
	services := ServiceMap.GetInterface("TestGetInterface")
	assert.NotNil(t, services)
	assert.Equal(t, 1, len(services))

	// Test GetInterface - not found
	services = ServiceMap.GetInterface("nonexistent")
	assert.Nil(t, services)

	// Cleanup
	ServiceMap.UnRegister("TestGetInterface", testProtocol, ServiceKey("TestGetInterface", "", "v1"))
}

func TestMethodTypeSuiteContextInvalid(t *testing.T) {
	mt := &MethodType{ctxType: reflect.TypeOf((*context.Context)(nil)).Elem()}

	// Test with nil context (invalid)
	var nilCtx context.Context = nil
	result := mt.SuiteContext(nilCtx)
	assert.True(t, result.IsValid())
	assert.True(t, result.IsZero())
}

func TestIsExported(t *testing.T) {
	assert.True(t, isExported("Exported"))
	assert.True(t, isExported("A"))
	assert.False(t, isExported("unexported"))
	assert.False(t, isExported("a"))
	assert.False(t, isExported(""))
}

func TestIsExportedOrBuiltinType(t *testing.T) {
	// Exported type
	assert.True(t, isExportedOrBuiltinType(reflect.TypeOf(TestService{})))

	// Pointer to exported type
	assert.True(t, isExportedOrBuiltinType(reflect.TypeOf(&TestService{})))

	// Builtin type (string)
	assert.True(t, isExportedOrBuiltinType(reflect.TypeOf("")))

	// Builtin type (int)
	assert.True(t, isExportedOrBuiltinType(reflect.TypeOf(0)))

	// Pointer to builtin
	var i int
	assert.True(t, isExportedOrBuiltinType(reflect.TypeOf(&i)))

	// Unexported type
	assert.False(t, isExportedOrBuiltinType(reflect.TypeOf(testService{})))
}

// Test service with XXX prefix methods (should be skipped)
type TestServiceWithXXX struct{}

func (s *TestServiceWithXXX) XXX_InterfaceName() string {
	return "test"
}

func (s *TestServiceWithXXX) NormalMethod(ctx context.Context) error {
	return nil
}

func (s *TestServiceWithXXX) Reference() string {
	return "TestServiceWithXXX"
}

func TestSuiteMethodSkipsXXXPrefix(t *testing.T) {
	s := &TestServiceWithXXX{}
	sType := reflect.TypeOf(s)

	// XXX_ prefixed method should be skipped
	method, ok := sType.MethodByName("XXX_InterfaceName")
	assert.True(t, ok)
	mt := suiteMethod(method)
	assert.Nil(t, mt)

	// Reference method should be skipped
	method, ok = sType.MethodByName("Reference")
	assert.True(t, ok)
	mt = suiteMethod(method)
	assert.Nil(t, mt)

	// Normal method should not be skipped
	method, ok = sType.MethodByName("NormalMethod")
	assert.True(t, ok)
	mt = suiteMethod(method)
	assert.NotNil(t, mt)
}

// Test service with SetGRPCServer method
type TestServiceWithGRPC struct{}

func (s *TestServiceWithGRPC) SetGRPCServer(server any) {}

func (s *TestServiceWithGRPC) ValidMethod(ctx context.Context, arg any) error {
	return nil
}

func TestSuiteMethodSkipsSetGRPCServer(t *testing.T) {
	s := &TestServiceWithGRPC{}
	sType := reflect.TypeOf(s)

	// SetGRPCServer should be skipped
	method, ok := sType.MethodByName("SetGRPCServer")
	assert.True(t, ok)
	mt := suiteMethod(method)
	assert.Nil(t, mt)
}

func TestGetReferenceWithStruct(t *testing.T) {
	// Test with struct (not pointer)
	s := TestService{}
	ref := GetReference(s)
	assert.Equal(t, "TestService", ref)
}

func TestGetReferenceWithAnonymousStruct(t *testing.T) {
	// Anonymous struct embedded in pointer
	s := &struct {
		ServiceWithoutRef
	}{}
	ref := GetReference(s)
	assert.Equal(t, "ServiceWithoutRef", ref)
}

func TestRegisterWithEmptyServiceName(t *testing.T) {
	// This tests the edge case where service name cannot be determined
	// Using a non-struct type
	var fn func()
	_, err := ServiceMap.Register("test", "proto", "", "v1", fn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no service name")
}

func TestUnRegisterInterfaceNotFound(t *testing.T) {
	s := &TestService{}
	_, err := ServiceMap.Register("TestUnRegisterInterface", testProtocol, "", "v1", s)
	assert.NoError(t, err)

	// Manually remove from interfaceMap to simulate inconsistent state
	ServiceMap.mutex.Lock()
	delete(ServiceMap.interfaceMap, "TestUnRegisterInterface")
	ServiceMap.mutex.Unlock()

	err = ServiceMap.UnRegister("TestUnRegisterInterface", testProtocol, ServiceKey("TestUnRegisterInterface", "", "v1"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no service for TestUnRegisterInterface")

	// Cleanup
	ServiceMap.mutex.Lock()
	delete(ServiceMap.serviceMap[testProtocol], ServiceKey("TestUnRegisterInterface", "", "v1"))
	ServiceMap.mutex.Unlock()
}

func TestSuitableMethodsWithMethodMapper(t *testing.T) {
	s := &TestService{}
	methods, methodMap := suitableMethods(reflect.TypeOf(s))

	// Check that MethodMapper renamed MethodTwo to methodTwo
	assert.Contains(t, methods, "methodTwo")
	assert.Contains(t, methods, "MethodTwo") // swapped case version

	// Verify the method exists in map
	_, ok := methodMap["methodTwo"]
	assert.True(t, ok)
}

// Service with method that has unexported method (not exported via PkgPath)
type TestServiceUnexportedMethod struct{}

func (s *TestServiceUnexportedMethod) ValidMethod(ctx context.Context) error {
	return nil
}

func TestSuiteMethodWithUnexportedMethod(t *testing.T) {
	s := &TestServiceUnexportedMethod{}
	sType := reflect.TypeOf(s)

	method, ok := sType.MethodByName("ValidMethod")
	assert.True(t, ok)

	// Method should be exported (PkgPath is empty for exported methods)
	assert.Equal(t, "", method.PkgPath)

	mt := suiteMethod(method)
	assert.NotNil(t, mt)
}

func TestServiceInfoStruct(t *testing.T) {
	// Test ServiceInfo struct initialization
	info := ServiceInfo{
		InterfaceName: "com.test.Service",
		ServiceType:   &TestService{},
		Methods: []MethodInfo{
			{
				Name: "TestMethod",
				Type: "unary",
				Meta: map[string]any{"key": "value"},
			},
		},
		Meta: map[string]any{"version": "1.0"},
	}

	assert.Equal(t, "com.test.Service", info.InterfaceName)
	assert.NotNil(t, info.ServiceType)
	assert.Equal(t, 1, len(info.Methods))
	assert.Equal(t, "TestMethod", info.Methods[0].Name)
	assert.Equal(t, "unary", info.Methods[0].Type)
	assert.Equal(t, "value", info.Methods[0].Meta["key"])
	assert.Equal(t, "1.0", info.Meta["version"])
}

func TestMethodInfoFunctions(t *testing.T) {
	// Test MethodInfo with function fields
	reqInitCalled := false
	streamInitCalled := false
	methodFuncCalled := false

	info := MethodInfo{
		Name: "TestMethod",
		Type: "unary",
		ReqInitFunc: func() any {
			reqInitCalled = true
			return struct{}{}
		},
		StreamInitFunc: func(baseStream any) any {
			streamInitCalled = true
			return baseStream
		},
		MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
			methodFuncCalled = true
			return nil, nil
		},
	}

	// Call the functions
	info.ReqInitFunc()
	assert.True(t, reqInitCalled)

	info.StreamInitFunc(nil)
	assert.True(t, streamInitCalled)

	info.MethodFunc(context.Background(), nil, nil)
	assert.True(t, methodFuncCalled)
}

func TestRegisterMultipleServicesForSameInterface(t *testing.T) {
	s1 := &TestService{}
	s2 := &TestService{}

	_, err := ServiceMap.Register("MultiService", testProtocol, "group1", "v1", s1)
	assert.NoError(t, err)

	_, err = ServiceMap.Register("MultiService", testProtocol, "group2", "v1", s2)
	assert.NoError(t, err)

	// Should have 2 services for the same interface
	services := ServiceMap.GetInterface("MultiService")
	assert.Equal(t, 2, len(services))

	// Cleanup
	ServiceMap.UnRegister("MultiService", testProtocol, ServiceKey("MultiService", "group1", "v1"))
	ServiceMap.UnRegister("MultiService", testProtocol, ServiceKey("MultiService", "group2", "v1"))
}

func TestUnRegisterLastServiceInProtocol(t *testing.T) {
	s := &TestService{}
	protocol := "uniqueProtocol"

	_, err := ServiceMap.Register("TestLastService", protocol, "", "v1", s)
	assert.NoError(t, err)

	// Verify protocol exists
	ServiceMap.mutex.RLock()
	_, exists := ServiceMap.serviceMap[protocol]
	ServiceMap.mutex.RUnlock()
	assert.True(t, exists)

	// Unregister the only service
	err = ServiceMap.UnRegister("TestLastService", protocol, ServiceKey("TestLastService", "", "v1"))
	assert.NoError(t, err)

	// Protocol should be removed from serviceMap
	ServiceMap.mutex.RLock()
	_, exists = ServiceMap.serviceMap[protocol]
	ServiceMap.mutex.RUnlock()
	assert.False(t, exists)
}

// Test with method that has wrong return type (not error)
type TestServiceWrongReturn struct{}

func (s *TestServiceWrongReturn) WrongReturn(ctx context.Context) string {
	return ""
}

func TestSuiteMethodWrongReturnType(t *testing.T) {
	s := &TestServiceWrongReturn{}
	sType := reflect.TypeOf(s)

	method, ok := sType.MethodByName("WrongReturn")
	assert.True(t, ok)

	mt := suiteMethod(method)
	assert.Nil(t, mt) // Should be nil because return type is not error
}

// Test with method that has too many return values
type TestServiceTooManyReturns struct{}

func (s *TestServiceTooManyReturns) TooManyReturns(ctx context.Context) (any, any, error) {
	return nil, nil, nil
}

func TestSuiteMethodTooManyReturns(t *testing.T) {
	s := &TestServiceTooManyReturns{}
	sType := reflect.TypeOf(s)

	method, ok := sType.MethodByName("TooManyReturns")
	assert.True(t, ok)

	mt := suiteMethod(method)
	assert.Nil(t, mt) // Should be nil because too many return values
}
