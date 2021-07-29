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

var mockGsonGeneralizer = GetGsonGeneralizer()

type mockGsonParent struct {
	Gender, Email, Name string
	Age                 int
	Child               *mockGsonChild
}

func (p mockGsonParent) JavaClassName() string {
	return "org.apache.dubbo.mockGsonParent"
}

type mockGsonChild struct {
	Gender, Email, Name string
	Age                 int
}

func (p mockGsonChild) JavaClassName() string {
	return "org.apache.dubbo.mockGsonChild"
}

func TestGsonGeneralizer(t *testing.T) {
	c := &mockGsonChild{
		Age:    20,
		Gender: "male",
		Email:  "childName@example.com",
		Name:   "childName",
	}
	p := mockGsonParent{
		Age:    30,
		Gender: "male",
		Email:  "enableasync@example.com",
		Name:   "enableasync",
		Child:  c,
	}

	m, err := mockGsonGeneralizer.Generalize(p)
	assert.Nil(t, err)
	assert.Equal(t, "{\"Gender\":\"male\",\"Email\":\"enableasync@example.com\",\"Name\":\"enableasync\",\"Age\":30,\"Child\":{\"Gender\":\"male\",\"Email\":\"childName@example.com\",\"Name\":\"childName\",\"Age\":20}}", m)

	r, err := mockGsonGeneralizer.Realize(m, reflect.TypeOf(p))
	assert.Nil(t, err)
	rMockParent, ok := r.(*mockGsonParent)
	assert.True(t, ok)
	// parent
	assert.Equal(t, "enableasync", rMockParent.Name)
	assert.Equal(t, 30, rMockParent.Age)
	// child
	assert.Equal(t, "childName", rMockParent.Child.Name)
	assert.Equal(t, 20, rMockParent.Child.Age)
}

func TestGsonPointer(t *testing.T) {
	c := &mockGsonChild{
		Age:    20,
		Gender: "male",
		Email:  "childName@example.com",
		Name:   "childName",
	}

	m, err := mockMapGeneralizer.Generalize(c)
	assert.Nil(t, err)
	newC, err := mockMapGeneralizer.Realize(m, reflect.TypeOf(c))
	assert.Nil(t, err)
	rMockChild, ok := newC.(*mockGsonChild)
	assert.True(t, ok)
	assert.Equal(t, "childName", rMockChild.Name)
	assert.Equal(t, 20, rMockChild.Age)
}
