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

package matcher

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestNewAttachmentConditionMatcher(t *testing.T) {
	matcher := NewAttachmentConditionMatcher("attachments[version]")
	assert.NotNil(t, matcher)
	assert.Equal(t, "attachments[version]", matcher.key)
}

func TestAttachmentConditionMatcherGetValue(t *testing.T) {
	matcher := NewAttachmentConditionMatcher("attachments[version]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithAttachments(map[string]interface{}{
			"version": "1.0.0",
		}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "1.0.0", value)
}

func TestAttachmentConditionMatcherGetValueDifferentKeys(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		attachmentKey string
		attachmentVal string
		expectedValue string
	}{
		{
			name:          "version attachment",
			key:           "attachments[version]",
			attachmentKey: "version",
			attachmentVal: "2.0.0",
			expectedValue: "2.0.0",
		},
		{
			name:          "timeout attachment",
			key:           "attachments[timeout]",
			attachmentKey: "timeout",
			attachmentVal: "3000",
			expectedValue: "3000",
		},
		{
			name:          "custom attachment",
			key:           "attachments[customKey]",
			attachmentKey: "customKey",
			attachmentVal: "customValue",
			expectedValue: "customValue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewAttachmentConditionMatcher(tt.key)
			url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
			require.NoError(t, err)

			inv := invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName("test"),
				invocation.WithAttachments(map[string]interface{}{
					tt.attachmentKey: tt.attachmentVal,
				}),
			)

			value := matcher.GetValue(nil, url, inv)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

func TestAttachmentConditionMatcherGetValueNotFound(t *testing.T) {
	matcher := NewAttachmentConditionMatcher("attachments[notexist]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithAttachments(map[string]interface{}{
			"version": "1.0.0",
		}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "", value)
}

func TestAttachmentConditionMatcherGetValueInvalidFormat(t *testing.T) {
	tests := []string{
		"invalid",
		"attachments[]",
		"attachments",
		"attachment[version]",
		"[version]",
	}

	for _, key := range tests {
		t.Run(key, func(t *testing.T) {
			matcher := NewAttachmentConditionMatcher(key)
			url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
			require.NoError(t, err)

			inv := invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName("test"),
				invocation.WithAttachments(map[string]interface{}{
					"version": "1.0.0",
				}),
			)

			value := matcher.GetValue(nil, url, inv)
			assert.Equal(t, "", value)
		})
	}
}

func TestAttachmentConditionMatcherGetValueWithDotNotation(t *testing.T) {
	// Test with dot notation in the key
	matcher := NewAttachmentConditionMatcher("attachments[version]\\.subkey")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithAttachments(map[string]interface{}{
			"version": "1.0.0",
		}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "1.0.0", value)
}

func TestAttachmentConditionMatcherGetValueEmptyAttachment(t *testing.T) {
	matcher := NewAttachmentConditionMatcher("attachments[version]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "", value)
}

func TestAttachmentConditionMatcherGetValueNumericKey(t *testing.T) {
	matcher := NewAttachmentConditionMatcher("attachments[abc123]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithAttachments(map[string]interface{}{
			"abc123": "value123",
		}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "value123", value)
}

func TestAttachmentConditionMatcherGetValueUnderscoreKey(t *testing.T) {
	matcher := NewAttachmentConditionMatcher("attachments[my_key]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithAttachments(map[string]interface{}{
			"my_key": "my_value",
		}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "my_value", value)
}
