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

package invocation

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestRPCInvocation_ServiceKey(t *testing.T) {
	providerURL := "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245"

	sameInfPathConsumerURL := "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245"
	diffInfPathConsumerURL := "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProviderFoo&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245"

	providerUrl, err := common.NewURL(providerURL)
	assert.NoError(t, err)

	// invocation with same interface and path value
	sameInfPathConsumerUrl, err := common.NewURL(sameInfPathConsumerURL)
	assert.NoError(t, err)
	invocation := NewRPCInvocationWithOptions(WithAttachments(map[string]any{
		constant.InterfaceKey: sameInfPathConsumerUrl.GetParam(constant.InterfaceKey, ""),
		constant.PathKey:      sameInfPathConsumerUrl.Path,
		constant.GroupKey:     sameInfPathConsumerUrl.GetParam(constant.GroupKey, ""),
		constant.VersionKey:   sameInfPathConsumerUrl.GetParam(constant.VersionKey, ""),
	}))
	assert.Equal(t, providerUrl.ServiceKey(), invocation.ServiceKey())

	// invocation with different interface and path value
	diffInfPathConsumerUrl, err := common.NewURL(diffInfPathConsumerURL)
	assert.NoError(t, err)
	invocation = NewRPCInvocationWithOptions(WithAttachments(map[string]any{
		constant.InterfaceKey: diffInfPathConsumerUrl.GetParam(constant.InterfaceKey, ""),
		constant.PathKey:      diffInfPathConsumerUrl.Path,
		constant.GroupKey:     diffInfPathConsumerUrl.GetParam(constant.GroupKey, ""),
		constant.VersionKey:   diffInfPathConsumerUrl.GetParam(constant.VersionKey, ""),
	}))
	assert.Equal(t, providerUrl.ServiceKey(), invocation.ServiceKey())
}

func TestNewRPCInvocation(t *testing.T) {
	methodName := "testMethod"
	arguments := []any{"arg1", 123, true}
	attachments := map[string]any{
		"key1": "value1",
		"key2": "value2",
	}

	invocation := NewRPCInvocation(methodName, arguments, attachments)

	assert.NotNil(t, invocation)
	assert.Equal(t, methodName, invocation.MethodName())
	assert.Equal(t, arguments, invocation.Arguments())
	assert.Equal(t, attachments, invocation.Attachments())
	assert.NotNil(t, invocation.Attributes())
	assert.Empty(t, invocation.Attributes())
}

func TestNewRPCInvocationWithOptions(t *testing.T) {
	methodName := "testMethod"
	paramTypes := []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(0)}
	paramTypeNames := []string{"string", "int"}
	paramValues := []reflect.Value{reflect.ValueOf("test"), reflect.ValueOf(123)}
	paramRawValues := []any{"test", 123}
	arguments := []any{"arg1", 456}
	reply := "test_reply"
	callback := func() {}
	attachments := map[string]any{"key": "value"}

	invocation := NewRPCInvocationWithOptions(
		WithMethodName(methodName),
		WithParameterTypes(paramTypes),
		WithParameterTypeNames(paramTypeNames),
		WithParameterValues(paramValues),
		WithParameterRawValues(paramRawValues),
		WithArguments(arguments),
		WithReply(reply),
		WithCallBack(callback),
		WithAttachments(attachments),
	)

	assert.NotNil(t, invocation)
	assert.Equal(t, methodName, invocation.MethodName())
	assert.Equal(t, paramTypes, invocation.ParameterTypes())
	assert.Equal(t, paramTypeNames, invocation.ParameterTypeNames())
	assert.Equal(t, paramValues, invocation.ParameterValues())
	assert.Equal(t, paramRawValues, invocation.ParameterRawValues())
	assert.Equal(t, arguments, invocation.Arguments())
	assert.Equal(t, reply, invocation.Reply())
	assert.NotNil(t, invocation.CallBack())
	assert.Equal(t, attachments, invocation.Attachments())
}

func TestRPCInvocation_MethodName(t *testing.T) {
	invocation := NewRPCInvocationWithOptions(WithMethodName("testMethod"))
	assert.Equal(t, "testMethod", invocation.MethodName())
}

func TestRPCInvocation_ActualMethodName(t *testing.T) {
	// Test non-generic invocation
	invocation := NewRPCInvocationWithOptions(
		WithMethodName("normalMethod"),
		WithArguments([]any{"arg1"}),
	)
	assert.Equal(t, "normalMethod", invocation.ActualMethodName())

	// Test generic invocation with $invoke
	invocation = NewRPCInvocationWithOptions(
		WithMethodName(constant.Generic),
		WithArguments([]any{"actualMethod", []string{"java.lang.String"}, []any{"param"}}),
	)
	assert.Equal(t, "actualMethod", invocation.ActualMethodName())

	// Test generic invocation with $invokeAsync
	invocation = NewRPCInvocationWithOptions(
		WithMethodName(constant.GenericAsync),
		WithArguments([]any{"actualAsyncMethod", []string{"java.lang.String"}, []any{"param"}}),
	)
	assert.Equal(t, "actualAsyncMethod", invocation.ActualMethodName())
}

func TestRPCInvocation_IsGenericInvocation(t *testing.T) {
	// Test non-generic invocation
	invocation := NewRPCInvocationWithOptions(
		WithMethodName("normalMethod"),
		WithArguments([]any{"arg1"}),
	)
	assert.False(t, invocation.IsGenericInvocation())

	// Test generic invocation with $invoke
	invocation = NewRPCInvocationWithOptions(
		WithMethodName(constant.Generic),
		WithArguments([]any{"actualMethod", []string{"java.lang.String"}, []any{"param"}}),
	)
	assert.True(t, invocation.IsGenericInvocation())

	// Test generic invocation with $invokeAsync
	invocation = NewRPCInvocationWithOptions(
		WithMethodName(constant.GenericAsync),
		WithArguments([]any{"actualMethod", []string{"java.lang.String"}, []any{"param"}}),
	)
	assert.True(t, invocation.IsGenericInvocation())

	// Test generic method name but insufficient arguments
	invocation = NewRPCInvocationWithOptions(
		WithMethodName(constant.Generic),
		WithArguments([]any{"arg1"}),
	)
	assert.False(t, invocation.IsGenericInvocation())

	// Test generic method name but nil arguments
	invocation = NewRPCInvocationWithOptions(
		WithMethodName(constant.Generic),
	)
	assert.False(t, invocation.IsGenericInvocation())
}

func TestRPCInvocation_SetAndGetReply(t *testing.T) {
	invocation := NewRPCInvocationWithOptions()
	assert.Nil(t, invocation.Reply())

	reply := "test_reply"
	invocation.SetReply(reply)
	assert.Equal(t, reply, invocation.Reply())
}

func TestRPCInvocation_SetAndGetCallBack(t *testing.T) {
	invocation := NewRPCInvocationWithOptions()
	assert.Nil(t, invocation.CallBack())

	callback := func() string { return "callback" }
	invocation.SetCallBack(callback)
	assert.NotNil(t, invocation.CallBack())
}

func TestRPCInvocation_Attachments(t *testing.T) {
	attachments := map[string]any{
		"key1": "value1",
		"key2": 123,
	}
	invocation := NewRPCInvocationWithOptions(WithAttachments(attachments))
	assert.Equal(t, attachments, invocation.Attachments())
}

func TestRPCInvocation_GetAttachmentInterface(t *testing.T) {
	attachments := map[string]any{
		"key1": "value1",
		"key2": 123,
	}
	invocation := NewRPCInvocationWithOptions(WithAttachments(attachments))

	// Test existing key
	val := invocation.GetAttachmentInterface("key1")
	assert.Equal(t, "value1", val)

	val = invocation.GetAttachmentInterface("key2")
	assert.Equal(t, 123, val)

	// Test non-existing key
	val = invocation.GetAttachmentInterface("non-existing")
	assert.Nil(t, val)

	// Test nil attachments
	invocation2 := NewRPCInvocationWithOptions()
	val = invocation2.GetAttachmentInterface("key1")
	assert.Nil(t, val)
}

func TestRPCInvocation_SetAndGetAttachment(t *testing.T) {
	invocation := NewRPCInvocationWithOptions()

	// Test setting attachment
	invocation.SetAttachment("key1", "value1")
	val, ok := invocation.GetAttachment("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// Test getting non-existing key
	val, ok = invocation.GetAttachment("non-existing")
	assert.False(t, ok)
	assert.Empty(t, val)

	// Test string array attachment (for triple protocol)
	invocation.SetAttachment("key2", []string{"value2", "value3"})
	val, ok = invocation.GetAttachment("key2")
	assert.True(t, ok)
	assert.Equal(t, "value2", val)

	// Test non-string attachment
	invocation.SetAttachment("key3", 123)
	val, ok = invocation.GetAttachment("key3")
	assert.False(t, ok)
	assert.Empty(t, val)

	// Test empty string array
	invocation.SetAttachment("key4", []string{})
	val, ok = invocation.GetAttachment("key4")
	assert.False(t, ok)
	assert.Empty(t, val)
}

func TestRPCInvocation_GetAttachmentWithDefaultValue(t *testing.T) {
	invocation := NewRPCInvocationWithOptions()
	invocation.SetAttachment("key1", "value1")

	// Test existing key
	val := invocation.GetAttachmentWithDefaultValue("key1", "default")
	assert.Equal(t, "value1", val)

	// Test non-existing key
	val = invocation.GetAttachmentWithDefaultValue("non-existing", "default")
	assert.Equal(t, "default", val)
}

func TestRPCInvocation_Attributes(t *testing.T) {
	invocation := NewRPCInvocationWithOptions()
	assert.NotNil(t, invocation.Attributes())
	assert.Empty(t, invocation.Attributes())
}

func TestRPCInvocation_SetAndGetAttribute(t *testing.T) {
	invocation := NewRPCInvocationWithOptions()

	// Test setting attribute
	invocation.SetAttribute("key1", "value1")
	val, ok := invocation.GetAttribute("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// Test getting non-existing key
	val, ok = invocation.GetAttribute("non-existing")
	assert.False(t, ok)
	assert.Nil(t, val)

	// Test with different types
	invocation.SetAttribute("key2", 123)
	val, ok = invocation.GetAttribute("key2")
	assert.True(t, ok)
	assert.Equal(t, 123, val)
}

func TestRPCInvocation_GetAttributeWithDefaultValue(t *testing.T) {
	invocation := NewRPCInvocationWithOptions()
	invocation.SetAttribute("key1", "value1")

	// Test existing key
	val := invocation.GetAttributeWithDefaultValue("key1", "default")
	assert.Equal(t, "value1", val)

	// Test non-existing key
	val = invocation.GetAttributeWithDefaultValue("non-existing", "default")
	assert.Equal(t, "default", val)

	// Test with nil attributes (should not happen but test defensive code)
	invocation2 := &RPCInvocation{}
	val = invocation2.GetAttributeWithDefaultValue("key1", "default")
	assert.Equal(t, "default", val)
}

func TestRPCInvocation_SetAndGetInvoker(t *testing.T) {
	invocation := NewRPCInvocationWithOptions()
	assert.Nil(t, invocation.Invoker())

	// Create a mock invoker
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/test")
	mockInvoker := &mockInvoker{url: url}

	invocation.SetInvoker(mockInvoker)
	assert.NotNil(t, invocation.Invoker())
	assert.Equal(t, mockInvoker, invocation.Invoker())
}

func TestRPCInvocation_GetAttachmentAsContext(t *testing.T) {
	attachments := map[string]any{
		"key1": "value1",
		"key2": []string{"value2", "value3"},
		"key3": 123, // non-string value should be skipped
	}
	invocation := NewRPCInvocationWithOptions(WithAttachments(attachments))

	ctx := invocation.GetAttachmentAsContext()
	assert.NotNil(t, ctx)

	// Extract header from context
	header := triple_protocol.ExtractFromOutgoingContext(ctx)
	assert.NotNil(t, header)

	// Verify that string attachments are in the header
	// NewOutgoingContext stores keys as lowercase, so check both ways
	assert.Contains(t, header, "key1")
	assert.Equal(t, []string{"value1"}, header["key1"]) //nolint:staticcheck

	assert.Contains(t, header, "key2")
	assert.Equal(t, []string{"value2", "value3"}, header["key2"]) //nolint:staticcheck

	// key3 (int) should not be in the header since it's not a string
	assert.NotContains(t, header, "key3")
}

func TestRPCInvocation_MergeAttachmentFromContext(t *testing.T) {
	invocation := NewRPCInvocationWithOptions()

	// Create a context with outgoing metadata
	ctx := context.Background()
	header := http.Header{}
	header.Set("key1", "value1")
	header.Add("key2", "value2")
	header.Add("key2", "value3")
	ctx = triple_protocol.NewOutgoingContext(ctx, header)

	invocation.MergeAttachmentFromContext(ctx)

	// Verify merged attachments
	val, ok := invocation.GetAttachment("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// For multiple values, it should be stored as array
	valInterface := invocation.GetAttachmentInterface("key2")
	assert.NotNil(t, valInterface)
	valArray, ok := valInterface.([]string)
	assert.True(t, ok)
	assert.Equal(t, 2, len(valArray))

	// Test with nil context header (context without outgoing header)
	invocation2 := NewRPCInvocationWithOptions()
	invocation2.MergeAttachmentFromContext(context.Background())
	// Should not panic and attachments should remain nil or empty
	if invocation2.Attachments() != nil {
		assert.Empty(t, invocation2.Attachments())
	}
}

func TestRPCInvocation_WithAttachment(t *testing.T) {
	invocation := NewRPCInvocationWithOptions(
		WithAttachment("key1", "value1"),
		WithAttachment("key2", 123),
	)

	val, ok := invocation.GetAttachment("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	val2 := invocation.GetAttachmentInterface("key2")
	assert.Equal(t, 123, val2)
}

func TestRPCInvocation_WithInvoker(t *testing.T) {
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/test")
	mockInvoker := &mockInvoker{url: url}

	invocation := NewRPCInvocationWithOptions(WithInvoker(mockInvoker))
	assert.NotNil(t, invocation.Invoker())
	assert.Equal(t, mockInvoker, invocation.Invoker())
}

func TestRPCInvocation_WithParameterTypeNames_EmptySlice(t *testing.T) {
	// Test with empty slice
	invocation := NewRPCInvocationWithOptions(
		WithParameterTypeNames([]string{}),
	)
	assert.Nil(t, invocation.ParameterTypeNames())

	// Test with non-empty slice
	invocation = NewRPCInvocationWithOptions(
		WithParameterTypeNames([]string{"string", "int"}),
	)
	assert.Equal(t, []string{"string", "int"}, invocation.ParameterTypeNames())
}

func TestRPCInvocation_ServiceKeyWithGroupAndVersion(t *testing.T) {
	// Test with group and version
	invocation := NewRPCInvocationWithOptions(WithAttachments(map[string]any{
		constant.InterfaceKey: "com.test.Service",
		constant.GroupKey:     "testGroup",
		constant.VersionKey:   "1.0.0",
	}))
	assert.Equal(t, "testGroup/com.test.Service:1.0.0", invocation.ServiceKey())

	// Test with path having leading slash
	invocation = NewRPCInvocationWithOptions(WithAttachments(map[string]any{
		constant.PathKey:    "/com.test.Service",
		constant.GroupKey:   "testGroup",
		constant.VersionKey: "1.0.0",
	}))
	assert.Equal(t, "testGroup/com.test.Service:1.0.0", invocation.ServiceKey())

	// Test with only interface
	invocation = NewRPCInvocationWithOptions(WithAttachments(map[string]any{
		constant.InterfaceKey: "com.test.Service",
	}))
	assert.Equal(t, "com.test.Service", invocation.ServiceKey())
}

func TestRPCInvocation_ConcurrentAccess(t *testing.T) {
	invocation := NewRPCInvocationWithOptions()

	// Test concurrent SetAttachment and GetAttachment
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := "key"
			invocation.SetAttachment(key, n)
			_, _ = invocation.GetAttachment(key)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent SetAttribute and GetAttribute
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := "attrKey"
			invocation.SetAttribute(key, n)
			_, _ = invocation.GetAttribute(key)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent SetInvoker and Invoker
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/test")
	mockInvoker := &mockInvoker{url: url}
	for i := 0; i < 10; i++ {
		go func() {
			invocation.SetInvoker(mockInvoker)
			_ = invocation.Invoker()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// mockInvoker is a simple mock implementation of base.Invoker for testing
type mockInvoker struct {
	url *common.URL
}

func (m *mockInvoker) GetURL() *common.URL {
	return m.url
}

func (m *mockInvoker) IsAvailable() bool {
	return true
}

func (m *mockInvoker) Destroy() {
}

func (m *mockInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	return &result.RPCResult{}
}
