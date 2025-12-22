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

func TestNewArgumentConditionMatcher(t *testing.T) {
	matcher := NewArgumentConditionMatcher("arguments[0]")
	assert.NotNil(t, matcher)
	assert.Equal(t, "arguments[0]", matcher.key)
}

func TestArgumentConditionMatcherGetValue(t *testing.T) {
	matcher := NewArgumentConditionMatcher("arguments[0]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithArguments([]interface{}{"value1", "value2", "value3"}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "value1", value)
}

func TestArgumentConditionMatcherGetValueDifferentIndices(t *testing.T) {
	args := []interface{}{"first", "second", "third", "fourth"}

	tests := []struct {
		name          string
		key           string
		expectedValue string
	}{
		{
			name:          "first argument",
			key:           "arguments[0]",
			expectedValue: "first",
		},
		{
			name:          "second argument",
			key:           "arguments[1]",
			expectedValue: "second",
		},
		{
			name:          "third argument",
			key:           "arguments[2]",
			expectedValue: "third",
		},
		{
			name:          "fourth argument",
			key:           "arguments[3]",
			expectedValue: "fourth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewArgumentConditionMatcher(tt.key)
			url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
			require.NoError(t, err)

			inv := invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName("test"),
				invocation.WithArguments(args),
			)

			value := matcher.GetValue(nil, url, inv)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

func TestArgumentConditionMatcherGetValueInvalidFormat(t *testing.T) {
	tests := []string{
		"invalid",
		"arguments[]",
		"arguments",
		"argument[0]",
		"[0]",
		"arguments[abc]",
	}

	args := []interface{}{"value1", "value2"}

	for _, key := range tests {
		t.Run(key, func(t *testing.T) {
			matcher := NewArgumentConditionMatcher(key)
			url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
			require.NoError(t, err)

			inv := invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName("test"),
				invocation.WithArguments(args),
			)

			value := matcher.GetValue(nil, url, inv)
			assert.Equal(t, "", value)
		})
	}
}

func TestArgumentConditionMatcherGetValueOutOfBounds(t *testing.T) {
	matcher := NewArgumentConditionMatcher("arguments[10]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithArguments([]interface{}{"value1", "value2"}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "", value)
}

func TestArgumentConditionMatcherGetValueAtBoundary(t *testing.T) {
	// Test index == len (should be out of bounds)
	matcher := NewArgumentConditionMatcher("arguments[2]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithArguments([]interface{}{"value1", "value2"}), // len = 2, index 2 is out of bounds
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "", value, "index equal to length should be out of bounds")
}

func TestArgumentConditionMatcherGetValueNegativeIndex(t *testing.T) {
	matcher := NewArgumentConditionMatcher("arguments[-1]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithArguments([]interface{}{"value1", "value2"}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "", value)
}

func TestArgumentConditionMatcherGetValueEmptyArguments(t *testing.T) {
	matcher := NewArgumentConditionMatcher("arguments[0]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithArguments([]interface{}{}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "", value)
}

func TestArgumentConditionMatcherGetValueNumericArgument(t *testing.T) {
	matcher := NewArgumentConditionMatcher("arguments[0]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithArguments([]interface{}{123, 456}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "123", value)
}

func TestArgumentConditionMatcherGetValueBooleanArgument(t *testing.T) {
	matcher := NewArgumentConditionMatcher("arguments[0]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithArguments([]interface{}{true, false}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "true", value)
}

func TestArgumentConditionMatcherGetValueWithDotNotation(t *testing.T) {
	// Test with dot notation in the key (similar to attachment)
	matcher := NewArgumentConditionMatcher("arguments[0]\\.subkey")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithArguments([]interface{}{"value1", "value2"}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "value1", value)
}

func TestArgumentConditionMatcherGetValueLargeIndex(t *testing.T) {
	matcher := NewArgumentConditionMatcher("arguments[999]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithArguments([]interface{}{"value1"}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "", value)
}

func TestArgumentConditionMatcherGetValueComplexObject(t *testing.T) {
	type TestStruct struct {
		Name string
		Age  int
	}

	matcher := NewArgumentConditionMatcher("arguments[0]")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)

	testObj := TestStruct{Name: "test", Age: 25}
	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("test"),
		invocation.WithArguments([]interface{}{testObj}),
	)

	value := matcher.GetValue(nil, url, inv)
	assert.Contains(t, value, "test")
}
