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

package base

import (
	"context"
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

// mockInvocation is a simple mock implementation of Invocation interface for testing
type mockInvocation struct {
	methodName         string
	actualMethodName   string
	parameterTypeNames []string
	parameterTypes     []reflect.Type
	parameterValues    []reflect.Value
	parameterRawValues []any
	arguments          []any
	reply              any
	invoker            Invoker
	isGeneric          bool
	attachments        map[string]any
	attributes         map[string]any
}

func (m *mockInvocation) MethodName() string                      { return m.methodName }
func (m *mockInvocation) ActualMethodName() string                { return m.actualMethodName }
func (m *mockInvocation) ParameterTypeNames() []string            { return m.parameterTypeNames }
func (m *mockInvocation) ParameterTypes() []reflect.Type          { return m.parameterTypes }
func (m *mockInvocation) ParameterValues() []reflect.Value        { return m.parameterValues }
func (m *mockInvocation) ParameterRawValues() []any               { return m.parameterRawValues }
func (m *mockInvocation) Arguments() []any                        { return m.arguments }
func (m *mockInvocation) Reply() any                              { return m.reply }
func (m *mockInvocation) Invoker() Invoker                        { return m.invoker }
func (m *mockInvocation) IsGenericInvocation() bool               { return m.isGeneric }
func (m *mockInvocation) Attachments() map[string]any             { return m.attachments }
func (m *mockInvocation) SetAttachment(key string, value any)     { m.attachments[key] = value }
func (m *mockInvocation) GetAttachment(key string) (string, bool) {
	if v, ok := m.attachments[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}
func (m *mockInvocation) GetAttachmentInterface(key string) any { return m.attachments[key] }
func (m *mockInvocation) GetAttachmentWithDefaultValue(key string, defaultValue string) string {
	if v, ok := m.GetAttachment(key); ok {
		return v
	}
	return defaultValue
}
func (m *mockInvocation) GetAttachmentAsContext() context.Context {
	ctx := context.Background()
	for k, v := range m.attachments {
		ctx = context.WithValue(ctx, k, v)
	}
	return ctx
}
func (m *mockInvocation) MergeAttachmentFromContext(ctx context.Context) {
	// Mock implementation
}
func (m *mockInvocation) Attributes() map[string]any { return m.attributes }
func (m *mockInvocation) SetAttribute(key string, value any) {
	if m.attributes == nil {
		m.attributes = make(map[string]any)
	}
	m.attributes[key] = value
}
func (m *mockInvocation) GetAttribute(key string) (any, bool) {
	v, ok := m.attributes[key]
	return v, ok
}
func (m *mockInvocation) GetAttributeWithDefaultValue(key string, defaultValue any) any {
	if v, ok := m.attributes[key]; ok {
		return v
	}
	return defaultValue
}

// TestInvocationInterface verifies the Invocation interface contract with a mock implementation
func TestInvocationInterface(t *testing.T) {
	// Create mock invocation
	inv := &mockInvocation{
		methodName:         "TestMethod",
		actualMethodName:   "TestMethod",
		parameterTypeNames: []string{"string", "int"},
		parameterTypes:     []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(0)},
		arguments:          []any{"arg1", 123},
		reply:              "reply",
		isGeneric:          false,
		attachments:        make(map[string]any),
		attributes:         make(map[string]any),
	}

	// Test basic methods
	assert.Equal(t, "TestMethod", inv.MethodName())
	assert.Equal(t, "TestMethod", inv.ActualMethodName())
	assert.Equal(t, []string{"string", "int"}, inv.ParameterTypeNames())
	assert.Equal(t, 2, len(inv.ParameterTypes()))
	assert.Equal(t, []any{"arg1", 123}, inv.Arguments())
	assert.Equal(t, "reply", inv.Reply())
	assert.False(t, inv.IsGenericInvocation())

	// Test attachments
	inv.SetAttachment("key1", "value1")
	inv.SetAttachment("key2", 456)

	val, ok := inv.GetAttachment("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	val2 := inv.GetAttachmentInterface("key2")
	assert.Equal(t, 456, val2)

	val3 := inv.GetAttachmentWithDefaultValue("key3", "default")
	assert.Equal(t, "default", val3)

	attachments := inv.Attachments()
	assert.Len(t, attachments, 2)

	// Test attributes
	inv.SetAttribute("attr1", "attrValue1")
	inv.SetAttribute("attr2", 789)

	attrVal, attrOk := inv.GetAttribute("attr1")
	assert.True(t, attrOk)
	assert.Equal(t, "attrValue1", attrVal)

	attrVal2 := inv.GetAttributeWithDefaultValue("attr3", "defaultAttr")
	assert.Equal(t, "defaultAttr", attrVal2)

	attributes := inv.Attributes()
	assert.Len(t, attributes, 2)

	// Test context conversion
	ctx := inv.GetAttachmentAsContext()
	assert.NotNil(t, ctx)
}

func TestInvocationInterface_EmptyAttachments(t *testing.T) {
	inv := &mockInvocation{
		methodName:  "TestMethod",
		attachments: make(map[string]any),
		attributes:  make(map[string]any),
	}

	// Test with non-existent key
	val, ok := inv.GetAttachment("nonexistent")
	assert.False(t, ok)
	assert.Equal(t, "", val)

	valInterface := inv.GetAttachmentInterface("nonexistent")
	assert.Nil(t, valInterface)

	// Test with default value
	valDefault := inv.GetAttachmentWithDefaultValue("nonexistent", "defaultValue")
	assert.Equal(t, "defaultValue", valDefault)

	// Test attribute with non-existent key
	attrVal, attrOk := inv.GetAttribute("nonexistent")
	assert.False(t, attrOk)
	assert.Nil(t, attrVal)

	attrValDefault := inv.GetAttributeWithDefaultValue("nonexistent", "defaultAttr")
	assert.Equal(t, "defaultAttr", attrValDefault)
}

func TestInvocationInterface_GenericInvocation(t *testing.T) {
	inv := &mockInvocation{
		methodName:  "GenericMethod",
		isGeneric:   true,
		attachments: make(map[string]any),
		attributes:  make(map[string]any),
	}

	assert.True(t, inv.IsGenericInvocation())
}

func TestInvocationInterface_WithInvoker(t *testing.T) {
	// Create a mock invoker
	invoker := NewBaseInvoker(nil)

	inv := &mockInvocation{
		methodName:  "TestMethod",
		invoker:     invoker,
		attachments: make(map[string]any),
		attributes:  make(map[string]any),
	}

	assert.Equal(t, invoker, inv.Invoker())
}
