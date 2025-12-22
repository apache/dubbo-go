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

// TestNewRequestPayload tests creating a new RequestPayload with various inputs
func TestNewRequestPayload(t *testing.T) {
	tests := []struct {
		desc              string
		args              any
		attachments       map[string]any
		expectParams      any
		expectAttachments map[string]any
	}{
		{
			desc:              "with all parameters",
			args:              "request args",
			attachments:       map[string]any{"key": "value"},
			expectParams:      "request args",
			expectAttachments: map[string]any{"key": "value"},
		},
		{
			desc:              "with nil attachments",
			args:              "request args",
			attachments:       nil,
			expectParams:      "request args",
			expectAttachments: make(map[string]any),
		},
		{
			desc:              "with nil args",
			args:              nil,
			attachments:       map[string]any{"trace_id": "12345"},
			expectParams:      nil,
			expectAttachments: map[string]any{"trace_id": "12345"},
		},
		{
			desc:              "with empty attachments",
			args:              []string{"arg1", "arg2"},
			attachments:       map[string]any{},
			expectParams:      []string{"arg1", "arg2"},
			expectAttachments: map[string]any{},
		},
		{
			desc:              "with complex args",
			args:              map[string]any{"user_id": 123, "name": "test"},
			attachments:       map[string]any{"version": "1.0"},
			expectParams:      map[string]any{"user_id": 123, "name": "test"},
			expectAttachments: map[string]any{"version": "1.0"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			payload := NewRequestPayload(test.args, test.attachments)

			assert.NotNil(t, payload)
			assert.Equal(t, test.expectParams, payload.Params)
			assert.NotNil(t, payload.Attachments)
		})
	}
}

// TestEnsureRequestPayload tests EnsureRequestPayload with various body types
func TestEnsureRequestPayload(t *testing.T) {
	tests := []struct {
		desc            string
		body            any
		expectParams    any
		expectNotNil    bool
	}{
		{
			desc:         "with RequestPayload object",
			body:         NewRequestPayload("args", map[string]any{"key": "value"}),
			expectParams: "args",
			expectNotNil: true,
		},
		{
			desc:         "with string object",
			body:         "string args",
			expectParams: "string args",
			expectNotNil: true,
		},
		{
			desc:         "with integer object",
			body:         42,
			expectParams: 42,
			expectNotNil: true,
		},
		{
			desc:         "with slice object",
			body:         []int{1, 2, 3},
			expectParams: []int{1, 2, 3},
			expectNotNil: true,
		},
		{
			desc:         "with map object",
			body:         map[string]any{"method": "login", "user": "admin"},
			expectParams: map[string]any{"method": "login", "user": "admin"},
			expectNotNil: true,
		},
		{
			desc:         "with nil object",
			body:         nil,
			expectParams: nil,
			expectNotNil: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			payload := EnsureRequestPayload(test.body)

			if test.expectNotNil {
				assert.NotNil(t, payload)
				assert.NotNil(t, payload.Attachments)
			}
			assert.Equal(t, test.expectParams, payload.Params)
		})
	}
}

// TestNewRequestPayloadNilAttachments tests that nil attachments are converted to empty map
func TestNewRequestPayloadNilAttachments(t *testing.T) {
	payload := NewRequestPayload("args", nil)

	assert.NotNil(t, payload.Attachments)
	assert.Equal(t, 0, len(payload.Attachments))
}

// TestRequestPayloadFields tests that RequestPayload fields are correctly set
func TestRequestPayloadFields(t *testing.T) {
	tests := []struct {
		desc        string
		args        any
		attachments map[string]any
	}{
		{
			desc:        "request with args and attachments",
			args:        "user login",
			attachments: map[string]any{"session": "xxx", "timestamp": 1234567890},
		},
		{
			desc:        "request with only args",
			args:        []any{"param1", "param2"},
			attachments: map[string]any{},
		},
		{
			desc:        "request with complex structure",
			args:        map[string]any{"data": []int{1, 2, 3}},
			attachments: map[string]any{"trace": "abc123", "span": "def456"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			payload := NewRequestPayload(test.args, test.attachments)

			assert.Equal(t, test.args, payload.Params)
			assert.Equal(t, test.attachments, payload.Attachments)
		})
	}
}

// TestEnsureRequestPayloadWithRequestPayload tests EnsureRequestPayload returns the same RequestPayload
func TestEnsureRequestPayloadWithRequestPayload(t *testing.T) {
	original := NewRequestPayload("args data", map[string]any{"key": "value"})
	result := EnsureRequestPayload(original)

	assert.Equal(t, original, result)
	assert.Equal(t, original.Params, result.Params)
	assert.Equal(t, original.Attachments, result.Attachments)
}

// TestEnsureRequestPayloadCreateNewPayload tests EnsureRequestPayload creates new payload for non-RequestPayload objects
func TestEnsureRequestPayloadCreateNewPayload(t *testing.T) {
	testArgs := "test arguments"
	payload := EnsureRequestPayload(testArgs)

	assert.NotNil(t, payload)
	assert.Equal(t, testArgs, payload.Params)
	assert.NotNil(t, payload.Attachments)
	assert.Equal(t, 0, len(payload.Attachments))
}
