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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestNewBaseConditionMatcher(t *testing.T) {
	matcher := NewBaseConditionMatcher("test-key")
	assert.NotNil(t, matcher)
	assert.Equal(t, "test-key", matcher.key)
	assert.NotNil(t, matcher.matches)
	assert.NotNil(t, matcher.misMatches)
	assert.Equal(t, 0, len(matcher.matches))
	assert.Equal(t, 0, len(matcher.misMatches))
}

func TestBaseConditionMatcherGetValue(t *testing.T) {
	matcher := NewBaseConditionMatcher("test-key")
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	value := matcher.GetValue(nil, url, inv)
	assert.Equal(t, "", value)
}

func TestBaseConditionMatcherGetMatches(t *testing.T) {
	matcher := NewBaseConditionMatcher("test-key")
	matcher.matches["value1"] = struct{}{}
	matcher.matches["value2"] = struct{}{}

	matches := matcher.GetMatches()
	assert.Equal(t, 2, len(matches))
	assert.Contains(t, matches, "value1")
	assert.Contains(t, matches, "value2")
}

func TestBaseConditionMatcherGetMismatches(t *testing.T) {
	matcher := NewBaseConditionMatcher("test-key")
	matcher.misMatches["value1"] = struct{}{}
	matcher.misMatches["value2"] = struct{}{}

	mismatches := matcher.GetMismatches()
	assert.Equal(t, 2, len(mismatches))
	assert.Contains(t, mismatches, "value1")
	assert.Contains(t, mismatches, "value2")
}

func TestGetSampleValueFromURLWithMethodKey(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("testMethod"))
	sample := map[string]string{
		constant.MethodKey: "sampleMethod",
	}

	// Should get method name from invocation, not from sample
	value := GetSampleValueFromURL(constant.MethodKey, sample, url, inv)
	assert.Equal(t, "testMethod", value)
}

func TestGetSampleValueFromURLWithMethodsKey(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("testMethod"))
	sample := map[string]string{
		constant.MethodsKey: "sampleMethods",
	}

	// Should get method name from invocation for MethodsKey too
	value := GetSampleValueFromURL(constant.MethodsKey, sample, url, inv)
	assert.Equal(t, "testMethod", value)
}

func TestGetSampleValueFromURLWithOtherKey(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("testMethod"))
	sample := map[string]string{
		"customKey":  "customValue",
		"anotherKey": "anotherValue",
	}

	// Should get value from sample for non-method keys
	value := GetSampleValueFromURL("customKey", sample, url, inv)
	assert.Equal(t, "customValue", value)

	value = GetSampleValueFromURL("anotherKey", sample, url, inv)
	assert.Equal(t, "anotherValue", value)
}

func TestGetSampleValueFromURLWithNilInvocation(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)
	sample := map[string]string{
		constant.MethodKey: "sampleMethod",
	}

	// When invocation is nil, should get value from sample
	value := GetSampleValueFromURL(constant.MethodKey, sample, url, nil)
	assert.Equal(t, "sampleMethod", value)
}

func TestGetSampleValueFromURLWithNonExistentKey(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("testMethod"))
	sample := map[string]string{
		"existingKey": "existingValue",
	}

	// Should return empty string for non-existent key
	value := GetSampleValueFromURL("nonExistentKey", sample, url, inv)
	assert.Equal(t, "", value)
}

func TestGetSampleValueFromURLWithEmptySample(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("testMethod"))
	sample := map[string]string{}

	value := GetSampleValueFromURL("anyKey", sample, url, inv)
	assert.Equal(t, "", value)
}

func TestGetSampleValueFromURLMethodKeyPriority(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("invocationMethod"))
	sample := map[string]string{
		constant.MethodKey: "sampleMethod",
	}

	// Invocation method should take priority over sample
	value := GetSampleValueFromURL(constant.MethodKey, sample, url, inv)
	assert.Equal(t, "invocationMethod", value)
	assert.NotEqual(t, "sampleMethod", value)
}

func TestBaseConditionMatcherIsMatchEmptyValue(t *testing.T) {
	matcher := NewBaseConditionMatcher("test-key")
	matcher.matches["pattern"] = struct{}{}

	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	// Empty value should return false
	result := matcher.IsMatch("", url, inv, true)
	assert.False(t, result)
}

func TestBaseConditionMatcherIsMatchNoMatchesOrMismatches(t *testing.T) {
	matcher := NewBaseConditionMatcher("test-key")

	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	require.NoError(t, err)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	// When both matches and mismatches are empty, should return false
	result := matcher.IsMatch("someValue", url, inv, true)
	assert.False(t, result)
}

func TestSortValuePatternByPriority(t *testing.T) {
	bp := byPriority(nil)
	assert.Equal(t, 0, bp.Len())
}

func TestByPriorityMethods(t *testing.T) {
	// Test the byPriority sort interface methods directly
	bp := byPriority(nil)

	assert.Equal(t, 0, bp.Len())

	// Test with empty slice - only Len() is safe
	assert.NotPanics(t, func() {
		bp.Len()
	})

	// Test with non-empty slice requires actual ValuePattern implementations
	// Since Swap and Less access array indices, they will panic on empty slice
	// This is expected behavior - sort.Stable checks Len() > 1 before calling Swap/Less
}
