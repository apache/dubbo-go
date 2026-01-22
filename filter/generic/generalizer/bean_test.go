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
	assert.Nil(t, err)
	desc := result.(*JavaBeanDescriptor)
	assert.Equal(t, TypePrimitive, desc.Type)
	assert.Equal(t, "hello", desc.Properties["value"])

	// Test int
	result, err = g.Generalize(42)
	assert.Nil(t, err)
	desc = result.(*JavaBeanDescriptor)
	assert.Equal(t, TypePrimitive, desc.Type)
	assert.Equal(t, 42, desc.Properties["value"])
}

func TestBeanGeneralizer_Generalize_Nil(t *testing.T) {
	g := GetBeanGeneralizer()
	result, err := g.Generalize(nil)
	assert.Nil(t, err)
	assert.Nil(t, result)
}

func TestBeanGeneralizer_Generalize_Slice(t *testing.T) {
	g := GetBeanGeneralizer()

	result, err := g.Generalize([]string{"a", "b", "c"})
	assert.Nil(t, err)
	desc := result.(*JavaBeanDescriptor)
	assert.Equal(t, TypeArray, desc.Type)
	assert.Equal(t, 3, len(desc.Properties))
}

func TestBeanGeneralizer_Generalize_Map(t *testing.T) {
	g := GetBeanGeneralizer()

	result, err := g.Generalize(map[string]int{"one": 1, "two": 2})
	assert.Nil(t, err)
	desc := result.(*JavaBeanDescriptor)
	assert.Equal(t, TypeMap, desc.Type)
	assert.Equal(t, 2, len(desc.Properties))
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
	assert.Nil(t, err)
	desc := result.(*JavaBeanDescriptor)
	assert.Equal(t, TypeBean, desc.Type)
	assert.Equal(t, "com.test.User", desc.ClassName)
}

func TestBeanGeneralizer_Realize_Primitive(t *testing.T) {
	g := GetBeanGeneralizer()

	desc := &JavaBeanDescriptor{
		Type:       TypePrimitive,
		Properties: map[interface{}]interface{}{"value": "hello"},
	}

	result, err := g.Realize(desc, reflect.TypeOf(""))
	assert.Nil(t, err)
	assert.Equal(t, "hello", result)
}

func TestBeanGeneralizer_Realize_Nil(t *testing.T) {
	g := GetBeanGeneralizer()
	result, err := g.Realize(nil, reflect.TypeOf(""))
	assert.Nil(t, err)
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
	assert.Nil(t, err)

	realized, err := g.Realize(generalized, reflect.TypeOf(&TestUser{}))
	assert.Nil(t, err)

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
	assert.Nil(t, err)

	realized, err := g.Realize(generalized, reflect.TypeOf([]int{}))
	assert.Nil(t, err)

	slice := realized.([]int)
	assert.Equal(t, len(original), len(slice))
}

func TestBeanGeneralizer_RoundTrip_Map(t *testing.T) {
	g := GetBeanGeneralizer()

	original := map[string]string{"key1": "value1", "key2": "value2"}

	generalized, err := g.Generalize(original)
	assert.Nil(t, err)

	realized, err := g.Realize(generalized, reflect.TypeOf(map[string]string{}))
	assert.Nil(t, err)

	m := realized.(map[string]string)
	assert.Equal(t, len(original), len(m))
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
	assert.Nil(t, err)
	assert.Equal(t, "com.test.User", typ)
}
