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
	arguments := []any{"arg1", 123}
	attachments := map[string]any{"key1": "value1"}

	inv := NewRPCInvocation(methodName, arguments, attachments)

	assert.Equal(t, methodName, inv.MethodName())
	assert.Equal(t, arguments, inv.Arguments())
	assert.Equal(t, attachments, inv.Attachments())
	assert.NotNil(t, inv.Attributes())
}

func TestNewRPCInvocationWithOptions(t *testing.T) {
	t.Run("basic options", func(t *testing.T) {
		inv := NewRPCInvocationWithOptions(
			WithMethodName("testMethod"),
			WithArguments([]any{"arg1", 123}),
			WithAttachments(map[string]any{"key1": "value1"}),
			WithReply("testReply"),
		)

		assert.Equal(t, "testMethod", inv.MethodName())
		assert.Equal(t, []any{"arg1", 123}, inv.Arguments())
		assert.Equal(t, "testReply", inv.Reply())
		assert.NotNil(t, inv.Attributes())

		val, ok := inv.GetAttachment("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", val)
	})

	t.Run("parameter types", func(t *testing.T) {
		paramTypes := []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(0)}
		paramTypeNames := []string{"string", "int"}

		inv := NewRPCInvocationWithOptions(
			WithParameterTypes(paramTypes),
			WithParameterTypeNames(paramTypeNames),
		)

		assert.Equal(t, paramTypes, inv.ParameterTypes())
		assert.Equal(t, paramTypeNames, inv.ParameterTypeNames())
	})

	t.Run("with invoker", func(t *testing.T) {
		invoker := base.NewBaseInvoker(nil)
		inv := NewRPCInvocationWithOptions(
			WithInvoker(invoker),
		)

		assert.Equal(t, invoker, inv.Invoker())
	})

	t.Run("attachment operations", func(t *testing.T) {
		inv := NewRPCInvocationWithOptions(
			WithAttachment("key1", "value1"),
			WithAttachment("key2", 456),
		)

		// Test GetAttachment
		val, ok := inv.GetAttachment("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", val)

		// Test GetAttachmentInterface
		val2 := inv.GetAttachmentInterface("key2")
		assert.Equal(t, 456, val2)

		// Test GetAttachmentWithDefaultValue
		val3 := inv.GetAttachmentWithDefaultValue("nonexistent", "default")
		assert.Equal(t, "default", val3)

		// Test SetAttachment
		inv.SetAttachment("key3", "value3")
		val4, ok := inv.GetAttachment("key3")
		assert.True(t, ok)
		assert.Equal(t, "value3", val4)
	})

	t.Run("attribute operations", func(t *testing.T) {
		inv := NewRPCInvocationWithOptions(
			WithMethodName("test"),
		)

		// Test SetAttribute
		inv.SetAttribute("attr1", "attrValue1")
		inv.SetAttribute("attr2", 789)

		// Test GetAttribute
		attrVal, ok := inv.GetAttribute("attr1")
		assert.True(t, ok)
		assert.Equal(t, "attrValue1", attrVal)

		// Test GetAttributeWithDefaultValue
		attrVal2 := inv.GetAttributeWithDefaultValue("nonexistent", "defaultAttr")
		assert.Equal(t, "defaultAttr", attrVal2)

		// Test Attributes
		attributes := inv.Attributes()
		assert.Len(t, attributes, 2)
	})

	t.Run("nil attachments initialization", func(t *testing.T) {
		inv := NewRPCInvocationWithOptions()

		// SetAttachment should initialize nil map
		assert.NotPanics(t, func() {
			inv.SetAttachment("key1", "value1")
		})
		val, ok := inv.GetAttachment("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", val)
	})

	t.Run("nil attributes initialization", func(t *testing.T) {
		inv := NewRPCInvocationWithOptions()

		// SetAttribute should initialize nil map
		assert.NotPanics(t, func() {
			inv.SetAttribute("attr1", "value1")
		})
		attrVal, ok := inv.GetAttribute("attr1")
		assert.True(t, ok)
		assert.Equal(t, "value1", attrVal)
	})
}
