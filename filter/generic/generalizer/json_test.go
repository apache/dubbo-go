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

var mockJsonGeneralizer = GetJsonrpcGeneralizer()

type mockJsonParent struct {
	Gender, Email, Name string
	Age                 int
	Child               *mockJsonChild
}

type mockJsonChild struct {
	Gender, Email, Name string
	Age                 int
}

func TestJsonGeneralizer(t *testing.T) {
	c := &mockJsonChild{
		Age:    20,
		Gender: "male",
		Email:  "childName@example.com",
		Name:   "childName",
	}
	p := mockJsonParent{
		Age:    30,
		Gender: "male",
		Email:  "enableasync@example.com",
		Name:   "enableasync",
		Child:  c,
	}

	m, err := mockJsonGeneralizer.Generalize(p)
	assert.Nil(t, err)
	assert.Equal(t, "{\"Gender\":\"male\",\"Email\":\"enableasync@example.com\",\"Name\":\"enableasync\",\"Age\":30,\"Child\":{\"Gender\":\"male\",\"Email\":\"childName@example.com\",\"Name\":\"childName\",\"Age\":20}}", m)

	r, err := mockJsonGeneralizer.Realize(m, reflect.TypeOf(p))
	assert.Nil(t, err)
	rMockParent, ok := r.(*mockJsonParent)
	assert.True(t, ok)
	// parent
	assert.Equal(t, "enableasync", rMockParent.Name)
	assert.Equal(t, 30, rMockParent.Age)
	// child
	assert.Equal(t, "childName", rMockParent.Child.Name)
	assert.Equal(t, 20, rMockParent.Child.Age)
}
