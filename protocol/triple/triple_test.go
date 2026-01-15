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

package triple

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

func TestNewTripleProtocol(t *testing.T) {
	tp := NewTripleProtocol()

	assert.NotNil(t, tp)
	assert.NotNil(t, tp.serverMap)
	assert.Empty(t, tp.serverMap)
}

func TestGetProtocol(t *testing.T) {
	// reset singleton for test isolation
	tripleProtocol = nil

	p1 := GetProtocol()
	assert.NotNil(t, p1)

	// should return same instance (singleton)
	p2 := GetProtocol()
	assert.Same(t, p1, p2)
}

func TestTripleProtocolRegistration(t *testing.T) {
	// verify protocol is registered via init()
	p := extension.GetProtocol(TRIPLE)
	assert.NotNil(t, p)
}

func TestTripleConstant(t *testing.T) {
	assert.Equal(t, "tri", TRIPLE)
}

func TestTripleProtocol_Destroy_EmptyServerMap(t *testing.T) {
	tp := NewTripleProtocol()

	// should not panic when serverMap is empty
	assert.NotPanics(t, func() {
		tp.Destroy()
	})
}

// Test isGenericCall checks if the generic parameter indicates a generic call
func Test_isGenericCall(t *testing.T) {
	tests := []struct {
		name     string
		generic  string
		expected bool
	}{
		// valid generic serialization types
		{"empty string", "", false},
		{"true", "true", true},
		{"TRUE", "TRUE", true},
		{"True", "True", true},
		{"gson", "gson", true},
		{"GSON", "GSON", true},
		{"Gson", "Gson", true},
		{"protobuf", "protobuf", true},
		{"PROTOBUF", "PROTOBUF", true},
		{"Protobuf", "Protobuf", true},
		{"protobuf-json", "protobuf-json", true},
		{"PROTOBUF-JSON", "PROTOBUF-JSON", true},
		{"Protobuf-Json", "Protobuf-Json", true},

		// invalid generic serialization types
		{"false", "false", false},
		{"random", "random", false},
		{"json", "json", false},
		{"xml", "xml", false},
		{"hessian", "hessian", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGenericCall(tt.generic)
			assert.Equal(t, tt.expected, result)
		})
	}
}
