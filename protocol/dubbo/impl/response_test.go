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
	"errors"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewResponsePayload tests creating a new ResponsePayload with various inputs
func TestNewResponsePayload(t *testing.T) {
	// Pre-define error variables to ensure consistent comparison
	testError := errors.New("test error")
	errorOccurred := errors.New("error occurred")

	tests := []struct {
		desc              string
		rspObj            any
		exception         error
		attachments       map[string]any
		expectRspObj      any
		expectException   error
		expectAttachments map[string]any
	}{
		{
			desc:              "with all parameters",
			rspObj:            "response data",
			exception:         testError,
			attachments:       map[string]any{"key": "value"},
			expectRspObj:      "response data",
			expectException:   testError,
			expectAttachments: map[string]any{"key": "value"},
		},
		{
			desc:              "with nil attachments",
			rspObj:            "response data",
			exception:         nil,
			attachments:       nil,
			expectRspObj:      "response data",
			expectException:   nil,
			expectAttachments: make(map[string]any),
		},
		{
			desc:              "with nil rspObj and exception",
			rspObj:            nil,
			exception:         errorOccurred,
			attachments:       map[string]any{"error_code": 500},
			expectRspObj:      nil,
			expectException:   errorOccurred,
			expectAttachments: map[string]any{"error_code": 500},
		},
		{
			desc:              "with empty attachments",
			rspObj:            123,
			exception:         nil,
			attachments:       map[string]any{},
			expectRspObj:      123,
			expectException:   nil,
			expectAttachments: map[string]any{},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			payload := NewResponsePayload(test.rspObj, test.exception, test.attachments)

			assert.NotNil(t, payload)
			assert.Equal(t, test.expectRspObj, payload.RspObj)
			if test.expectException != nil {
				require.Error(t, payload.Exception)
			} else {
				require.NoError(t, payload.Exception)
			}
			assert.NotNil(t, payload.Attachments)
		})
	}
}

// TestEnsureResponsePayload tests EnsureResponsePayload with various body types
func TestEnsureResponsePayload(t *testing.T) {
	tests := []struct {
		desc              string
		body              any
		expectRspObj      any
		expectException   error
		expectAttachments map[string]any
		expectNotNil      bool
	}{
		{
			desc:            "with ResponsePayload object",
			body:            NewResponsePayload("data", nil, map[string]any{"key": "value"}),
			expectRspObj:    "data",
			expectException: nil,
			expectNotNil:    true,
		},
		{
			desc:         "with error object",
			body:         errors.New("test error"),
			expectRspObj: nil,
			expectNotNil: true,
		},
		{
			desc:            "with string object",
			body:            "string response",
			expectRspObj:    "string response",
			expectException: nil,
			expectNotNil:    true,
		},
		{
			desc:            "with integer object",
			body:            42,
			expectRspObj:    42,
			expectException: nil,
			expectNotNil:    true,
		},
		{
			desc:            "with map object",
			body:            map[string]any{"result": "success"},
			expectRspObj:    map[string]any{"result": "success"},
			expectException: nil,
			expectNotNil:    true,
		},
		{
			desc:            "with nil object",
			body:            nil,
			expectRspObj:    nil,
			expectException: nil,
			expectNotNil:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			payload := EnsureResponsePayload(test.body)

			if test.expectNotNil {
				assert.NotNil(t, payload)
				assert.NotNil(t, payload.Attachments)
			}
			assert.Equal(t, test.expectRspObj, payload.RspObj)
		})
	}
}

// TestNewResponsePayloadNilAttachments tests that nil attachments are converted to empty map
func TestNewResponsePayloadNilAttachments(t *testing.T) {
	payload := NewResponsePayload("response", nil, nil)

	assert.NotNil(t, payload.Attachments)
	assert.Empty(t, payload.Attachments)
}

// TestResponsePayloadFields tests that ResponsePayload fields are correctly set
func TestResponsePayloadFields(t *testing.T) {
	tests := []struct {
		desc        string
		rspObj      any
		exception   error
		attachments map[string]any
	}{
		{
			desc:        "response with data and attachments",
			rspObj:      "user data",
			exception:   nil,
			attachments: map[string]any{"status": "ok", "code": 200},
		},
		{
			desc:        "response with exception",
			rspObj:      nil,
			exception:   errors.New("service error"),
			attachments: map[string]any{"error_id": "ERR_001"},
		},
		{
			desc:        "response with only data",
			rspObj:      []int{1, 2, 3},
			exception:   nil,
			attachments: map[string]any{},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			payload := NewResponsePayload(test.rspObj, test.exception, test.attachments)

			assert.Equal(t, test.rspObj, payload.RspObj)
			assert.Equal(t, test.exception, payload.Exception)
			assert.Equal(t, test.attachments, payload.Attachments)
		})
	}
}

// TestEnsureResponsePayloadWithResponsePayload tests EnsureResponsePayload returns the same ResponsePayload
func TestEnsureResponsePayloadWithResponsePayload(t *testing.T) {
	original := NewResponsePayload("data", nil, map[string]any{"key": "value"})
	result := EnsureResponsePayload(original)

	assert.Equal(t, original, result)
	assert.Equal(t, original.RspObj, result.RspObj)
	assert.Equal(t, original.Attachments, result.Attachments)
}
