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

package generalizer

import (
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUser is a test struct for bean serialization
type TestUser struct {
	Name    string
	Age     int
	Address *TestAddress
}

func (t *TestUser) JavaClassName() string {
	return "com.test.User"
}

// TestAddress is a nested test struct
type TestAddress struct {
	City    string
	ZipCode string
}

func (t *TestAddress) JavaClassName() string {
	return "com.test.Address"
}

func TestBeanGeneralizer_Generalize_Primitive(t *testing.T) {
	g := GetBeanGeneralizer()

	// Test string
	result, err := g.Generalize("hello")
	require.NoError(t, err)
	desc := result.(*JavaBeanDescriptor)
	assert.Equal(t, TypePrimitive, desc.Type)
	assert.Equal(t, "hello", desc.Properties["value"])

	// Test int
	result, err = g.Generalize(42)
	require.NoError(t, err)
	desc = result.(*JavaBeanDescriptor)
	assert.Equal(t, TypePrimitive, desc.Type)
	assert.Equal(t, 42, desc.Properties["value"])
}

func TestBeanGeneralizer_Generalize_Nil(t *testing.T) {
	g := GetBeanGeneralizer()
	result, err := g.Generalize(nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestBeanGeneralizer_Generalize_TypedNilPointer(t *testing.T) {
	g := GetBeanGeneralizer()

	// Test typed nil pointer (e.g., (*TestAddress)(nil) stored in an any)
	var addr *TestAddress = nil
	result, err := g.Generalize(addr)
	require.NoError(t, err)
	assert.Nil(t, result)

	// Test typed nil pointer in struct field
	user := &TestUser{
		Name:    "John",
		Age:     30,
		Address: nil, // typed nil pointer field
	}
	result, err = g.Generalize(user)
	require.NoError(t, err)
	desc := result.(*JavaBeanDescriptor)
	assert.Equal(t, TypeBean, desc.Type)
	// The nil Address field should be nil in properties
	assert.Nil(t, desc.Properties["address"])
}

func TestBeanGeneralizer_Generalize_Slice(t *testing.T) {
	g := GetBeanGeneralizer()

	result, err := g.Generalize([]string{"a", "b", "c"})
	require.NoError(t, err)
	desc := result.(*JavaBeanDescriptor)
	assert.Equal(t, TypeArray, desc.Type)
	assert.Len(t, desc.Properties, 3)
}

func TestBeanGeneralizer_Generalize_Map(t *testing.T) {
	g := GetBeanGeneralizer()

	result, err := g.Generalize(map[string]int{"one": 1, "two": 2})
	require.NoError(t, err)
	desc := result.(*JavaBeanDescriptor)
	assert.Equal(t, TypeMap, desc.Type)
	assert.Len(t, desc.Properties, 2)
}

func TestBeanGeneralizer_Generalize_Struct(t *testing.T) {
	g := GetBeanGeneralizer()

	user := &TestUser{
		Name: "John",
		Age:  30,
		Address: &TestAddress{
			City:    "Beijing",
			ZipCode: "100000",
		},
	}

	result, err := g.Generalize(user)
	require.NoError(t, err)
	desc := result.(*JavaBeanDescriptor)
	assert.Equal(t, TypeBean, desc.Type)
	assert.Equal(t, "com.test.User", desc.ClassName)
}

func TestBeanGeneralizer_Realize_Primitive(t *testing.T) {
	g := GetBeanGeneralizer()

	desc := &JavaBeanDescriptor{
		Type:       TypePrimitive,
		Properties: map[any]any{"value": "hello"},
	}

	result, err := g.Realize(desc, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestBeanGeneralizer_Realize_Nil(t *testing.T) {
	g := GetBeanGeneralizer()
	result, err := g.Realize(nil, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestBeanGeneralizer_RoundTrip_Struct(t *testing.T) {
	g := GetBeanGeneralizer()

	original := &TestUser{
		Name: "Alice",
		Age:  25,
		Address: &TestAddress{
			City:    "Shanghai",
			ZipCode: "200000",
		},
	}

	generalized, err := g.Generalize(original)
	require.NoError(t, err)

	realized, err := g.Realize(generalized, reflect.TypeOf(&TestUser{}))
	require.NoError(t, err)

	user := realized.(*TestUser)
	assert.Equal(t, original.Name, user.Name)
	assert.Equal(t, original.Age, user.Age)
	assert.NotNil(t, user.Address)
	assert.Equal(t, original.Address.City, user.Address.City)
}

func TestBeanGeneralizer_RoundTrip_Slice(t *testing.T) {
	g := GetBeanGeneralizer()

	original := []int{1, 2, 3, 4, 5}

	generalized, err := g.Generalize(original)
	require.NoError(t, err)

	realized, err := g.Realize(generalized, reflect.TypeOf([]int{}))
	require.NoError(t, err)

	slice := realized.([]int)
	assert.Len(t, slice, len(original))
}

func TestBeanGeneralizer_RoundTrip_Map(t *testing.T) {
	g := GetBeanGeneralizer()

	original := map[string]string{"key1": "value1", "key2": "value2"}

	generalized, err := g.Generalize(original)
	require.NoError(t, err)

	realized, err := g.Realize(generalized, reflect.TypeOf(map[string]string{}))
	require.NoError(t, err)

	m := realized.(map[string]string)
	assert.Len(t, m, len(original))
}

func TestJavaBeanDescriptor_JavaClassName(t *testing.T) {
	desc := &JavaBeanDescriptor{}
	assert.Equal(t, "org.apache.dubbo.common.beanutil.JavaBeanDescriptor", desc.JavaClassName())
}

func TestGetBeanGeneralizer_Singleton(t *testing.T) {
	g1 := GetBeanGeneralizer()
	g2 := GetBeanGeneralizer()
	assert.Same(t, g1, g2)
}

func TestBeanGeneralizer_GetType(t *testing.T) {
	g := GetBeanGeneralizer()
	typ, err := g.GetType(&TestUser{})
	require.NoError(t, err)
	assert.Equal(t, "com.test.User", typ)
}

func TestBeanGeneralizer_GetType_Nil(t *testing.T) {
	g := GetBeanGeneralizer()
	typ, err := g.GetType(nil)
	require.NoError(t, err)
	assert.Equal(t, "java.lang.Object", typ)
}

func TestBeanGeneralizer_Realize_ArrayWithInt32Keys(t *testing.T) {
	g := GetBeanGeneralizer()

	// Simulate JavaBeanDescriptor from Java Hessian with int32 keys
	// Java Hessian decodes integer keys as int32, not int
	desc := &JavaBeanDescriptor{
		Type:      TypeArray,
		ClassName: "[Ljava.lang.String;",
		Properties: map[any]any{
			int32(0): &JavaBeanDescriptor{Type: TypePrimitive, Properties: map[any]any{"value": "a"}},
			int32(1): &JavaBeanDescriptor{Type: TypePrimitive, Properties: map[any]any{"value": "b"}},
			int32(2): &JavaBeanDescriptor{Type: TypePrimitive, Properties: map[any]any{"value": "c"}},
		},
	}

	realized, err := g.Realize(desc, reflect.TypeOf([]string{}))
	require.NoError(t, err)

	slice := realized.([]string)
	assert.Len(t, slice, 3)
	assert.Equal(t, "a", slice[0])
	assert.Equal(t, "b", slice[1])
	assert.Equal(t, "c", slice[2])
}

// CircularNode is a test struct for circular reference testing
type CircularNode struct {
	Name string
	Next *CircularNode
}

func TestBeanGeneralizer_Generalize_CircularReference(t *testing.T) {
	g := GetBeanGeneralizer()

	// Create a circular reference: node -> node (self-referencing)
	node := &CircularNode{Name: "node1"}
	node.Next = node

	// Should not panic or infinite loop
	result, err := g.Generalize(node)
	require.NoError(t, err)
	assert.NotNil(t, result)

	desc := result.(*JavaBeanDescriptor)
	assert.Equal(t, TypeBean, desc.Type)
	assert.Equal(t, "node1", desc.Properties["name"].(*JavaBeanDescriptor).Properties["value"])
	// The circular reference should be nil to break the cycle
	assert.Nil(t, desc.Properties["next"])
}

func TestBeanGeneralizer_Generalize_MutualCircularReference(t *testing.T) {
	g := GetBeanGeneralizer()

	// Create mutual circular reference: node1 -> node2 -> node1
	node1 := &CircularNode{Name: "node1"}
	node2 := &CircularNode{Name: "node2"}
	node1.Next = node2
	node2.Next = node1

	// Should not panic or infinite loop
	result, err := g.Generalize(node1)
	require.NoError(t, err)
	assert.NotNil(t, result)

	desc := result.(*JavaBeanDescriptor)
	assert.Equal(t, TypeBean, desc.Type)
	assert.Equal(t, "node1", desc.Properties["name"].(*JavaBeanDescriptor).Properties["value"])

	// node1.Next should be node2
	node2Desc := desc.Properties["next"].(*JavaBeanDescriptor)
	assert.Equal(t, "node2", node2Desc.Properties["name"].(*JavaBeanDescriptor).Properties["value"])

	// node2.Next should be nil (circular reference back to node1)
	assert.Nil(t, node2Desc.Properties["next"])
}
