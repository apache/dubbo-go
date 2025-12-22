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

package impl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// TestGetSerializerById tests GetSerializerById with valid and invalid serializer IDs
func TestGetSerializerById(t *testing.T) {
	// Setup: Create a mock serializer for testing
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	tests := []struct {
		desc        string
		id          byte
		shouldPanic bool
	}{
		{
			desc:        "valid Hessian2 serializer ID",
			id:          constant.SHessian2,
			shouldPanic: false,
		},
		{
			desc:        "valid Protobuf serializer ID",
			id:          constant.SProto,
			shouldPanic: true, // Will panic because serializer is not registered
		},
		{
			desc:        "invalid serializer ID 255",
			id:          255,
			shouldPanic: true, // Will panic because ID is not in nameMaps
		},
		{
			desc:        "invalid serializer ID 0",
			id:          0,
			shouldPanic: true, // Will panic because ID is not in nameMaps
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.shouldPanic {
				assert.Panics(t, func() {
					GetSerializerById(test.id)
				})
			} else {
				assert.NotPanics(t, func() {
					serializer, err := GetSerializerById(test.id)
					assert.NoError(t, err)
					assert.NotNil(t, serializer)
					assert.Equal(t, mockSerializer, serializer)
				})
			}
		})
	}
}

// TestGetSerializerByIdWithUnregisteredSerializer tests GetSerializerById with unregistered serializer
func TestGetSerializerByIdWithUnregisteredSerializer(t *testing.T) {
	// Try to get a serializer ID that exists in nameMaps but is not registered
	assert.Panics(t, func() {
		GetSerializerById(constant.SProto)
	})
}

// TestGetSerializerByIdConsistency tests that GetSerializerById returns consistent results
func TestGetSerializerByIdConsistency(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	// Call multiple times, should get same result
	result1, err1 := GetSerializerById(constant.SHessian2)
	result2, err2 := GetSerializerById(constant.SHessian2)
	result3, err3 := GetSerializerById(constant.SHessian2)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.Equal(t, result1, result2)
	assert.Equal(t, result2, result3)
}
