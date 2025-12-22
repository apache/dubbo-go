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

package result

import (
	"errors"
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestRPCResult_SetError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{
			name:     "set nil error",
			err:      nil,
			expected: nil,
		},
		{
			name:     "set normal error",
			err:      errors.New("test error"),
			expected: errors.New("test error"),
		},
		{
			name:     "set custom error",
			err:      errors.New("connection timeout"),
			expected: errors.New("connection timeout"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RPCResult{}
			r.SetError(tt.err)

			if tt.expected == nil {
				assert.Nil(t, r.Error())
			} else {
				assert.NotNil(t, r.Error())
				assert.Equal(t, tt.expected.Error(), r.Error().Error())
			}
		})
	}
}

func TestRPCResult_Error(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *RPCResult
		expected error
	}{
		{
			name: "error is nil",
			setup: func() *RPCResult {
				return &RPCResult{}
			},
			expected: nil,
		},
		{
			name: "error is set",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Err = errors.New("test error")
				return r
			},
			expected: errors.New("test error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.setup()

			if tt.expected == nil {
				assert.Nil(t, r.Error())
			} else {
				assert.Equal(t, tt.expected.Error(), r.Error().Error())
			}
		})
	}
}

func TestRPCResult_SetBizError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{
			name:     "set nil biz error",
			err:      nil,
			expected: nil,
		},
		{
			name:     "set business error",
			err:      errors.New("business logic error"),
			expected: errors.New("business logic error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RPCResult{}
			r.SetBizError(tt.err)

			if tt.expected == nil {
				assert.Nil(t, r.BizError())
			} else {
				assert.Equal(t, tt.expected.Error(), r.BizError().Error())
			}
		})
	}
}

func TestRPCResult_BizError(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *RPCResult
		expected error
	}{
		{
			name: "biz error is nil",
			setup: func() *RPCResult {
				return &RPCResult{}
			},
			expected: nil,
		},
		{
			name: "biz error is set",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.BizErr = errors.New("business error")
				return r
			},
			expected: errors.New("business error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.setup()

			if tt.expected == nil {
				assert.Nil(t, r.BizError())
			} else {
				assert.Equal(t, tt.expected.Error(), r.BizError().Error())
			}
		})
	}
}

func TestRPCResult_SetResult(t *testing.T) {
	tests := []struct {
		name     string
		result   any
		expected any
	}{
		{
			name:     "set nil result",
			result:   nil,
			expected: nil,
		},
		{
			name:     "set string result",
			result:   "test result",
			expected: "test result",
		},
		{
			name:     "set int result",
			result:   123,
			expected: 123,
		},
		{
			name:     "set map result",
			result:   map[string]string{"key": "value"},
			expected: map[string]string{"key": "value"},
		},
		{
			name:     "set struct result",
			result:   struct{ Name string }{"test"},
			expected: struct{ Name string }{"test"},
		},
		{
			name:     "set slice result",
			result:   []int{1, 2, 3},
			expected: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RPCResult{}
			r.SetResult(tt.result)
			assert.Equal(t, tt.expected, r.Result())
		})
	}
}

func TestRPCResult_Result(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *RPCResult
		expected any
	}{
		{
			name: "result is nil",
			setup: func() *RPCResult {
				return &RPCResult{}
			},
			expected: nil,
		},
		{
			name: "result is set",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Rest = "test value"
				return r
			},
			expected: "test value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.setup()
			assert.Equal(t, tt.expected, r.Result())
		})
	}
}

func TestRPCResult_SetAttachments(t *testing.T) {
	tests := []struct {
		name        string
		attachments map[string]any
	}{
		{
			name:        "set nil attachments",
			attachments: nil,
		},
		{
			name:        "set empty attachments",
			attachments: map[string]any{},
		},
		{
			name: "set single attachment",
			attachments: map[string]any{
				"key1": "value1",
			},
		},
		{
			name: "set multiple attachments",
			attachments: map[string]any{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
		},
		{
			name: "set attachments with various types",
			attachments: map[string]any{
				"string": "test",
				"int":    42,
				"bool":   true,
				"float":  3.14,
				"slice":  []int{1, 2, 3},
				"map":    map[string]string{"nested": "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RPCResult{}
			r.SetAttachments(tt.attachments)

			if tt.attachments == nil {
				assert.Nil(t, r.Attrs)
			} else {
				assert.Equal(t, tt.attachments, r.Attrs)
			}
		})
	}
}

func TestRPCResult_SetAttachments_Override(t *testing.T) {
	r := &RPCResult{}

	// First set
	firstAttachments := map[string]any{
		"key1": "value1",
		"key2": "value2",
	}
	r.SetAttachments(firstAttachments)
	assert.Equal(t, firstAttachments, r.Attrs)

	// Override with new attachments
	secondAttachments := map[string]any{
		"key3": "value3",
	}
	r.SetAttachments(secondAttachments)
	assert.Equal(t, secondAttachments, r.Attrs)
	assert.NotContains(t, r.Attrs, "key1")
	assert.NotContains(t, r.Attrs, "key2")
}

func TestRPCResult_Attachments(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *RPCResult
		expected map[string]any
		checkNil bool
	}{
		{
			name: "attachments is nil - should initialize",
			setup: func() *RPCResult {
				return &RPCResult{}
			},
			expected: map[string]any{},
			checkNil: false,
		},
		{
			name: "attachments is set",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Attrs = map[string]any{"key": "value"}
				return r
			},
			expected: map[string]any{"key": "value"},
			checkNil: false,
		},
		{
			name: "attachments is empty map",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Attrs = map[string]any{}
				return r
			},
			expected: map[string]any{},
			checkNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.setup()
			result := r.Attachments()

			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRPCResult_Attachments_Initialization(t *testing.T) {
	r := &RPCResult{}
	assert.Nil(t, r.Attrs)

	// First call should initialize
	attachments := r.Attachments()
	assert.NotNil(t, attachments)
	assert.NotNil(t, r.Attrs)
	assert.Equal(t, 0, len(attachments))
}

func TestRPCResult_AddAttachment(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *RPCResult
		key      string
		value    any
		expected map[string]any
	}{
		{
			name: "add to nil attachments",
			setup: func() *RPCResult {
				return &RPCResult{}
			},
			key:   "key1",
			value: "value1",
			expected: map[string]any{
				"key1": "value1",
			},
		},
		{
			name: "add to existing attachments",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Attrs = map[string]any{"existing": "value"}
				return r
			},
			key:   "new",
			value: "newValue",
			expected: map[string]any{
				"existing": "value",
				"new":      "newValue",
			},
		},
		{
			name: "add string value",
			setup: func() *RPCResult {
				return &RPCResult{}
			},
			key:   "string",
			value: "test",
			expected: map[string]any{
				"string": "test",
			},
		},
		{
			name: "add int value",
			setup: func() *RPCResult {
				return &RPCResult{}
			},
			key:   "int",
			value: 42,
			expected: map[string]any{
				"int": 42,
			},
		},
		{
			name: "add bool value",
			setup: func() *RPCResult {
				return &RPCResult{}
			},
			key:   "bool",
			value: true,
			expected: map[string]any{
				"bool": true,
			},
		},
		{
			name: "add nil value",
			setup: func() *RPCResult {
				return &RPCResult{}
			},
			key:   "nil",
			value: nil,
			expected: map[string]any{
				"nil": nil,
			},
		},
		{
			name: "add complex value",
			setup: func() *RPCResult {
				return &RPCResult{}
			},
			key:   "complex",
			value: map[string]int{"nested": 1},
			expected: map[string]any{
				"complex": map[string]int{"nested": 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.setup()
			r.AddAttachment(tt.key, tt.value)

			assert.NotNil(t, r.Attrs)
			assert.Equal(t, tt.expected, r.Attrs)
		})
	}
}

func TestRPCResult_AddAttachment_Override(t *testing.T) {
	r := &RPCResult{}

	// Add first value
	r.AddAttachment("key", "value1")
	assert.Equal(t, "value1", r.Attrs["key"])

	// Override with new value
	r.AddAttachment("key", "value2")
	assert.Equal(t, "value2", r.Attrs["key"])
}

func TestRPCResult_AddAttachment_Multiple(t *testing.T) {
	r := &RPCResult{}

	r.AddAttachment("key1", "value1")
	r.AddAttachment("key2", 123)
	r.AddAttachment("key3", true)

	assert.Equal(t, 3, len(r.Attrs))
	assert.Equal(t, "value1", r.Attrs["key1"])
	assert.Equal(t, 123, r.Attrs["key2"])
	assert.Equal(t, true, r.Attrs["key3"])
}

func TestRPCResult_Attachment(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() *RPCResult
		key          string
		defaultValue any
		expected     any
	}{
		{
			name: "get existing attachment",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Attrs = map[string]any{"key": "value"}
				return r
			},
			key:          "key",
			defaultValue: "default",
			expected:     "value",
		},
		{
			name: "get non-existing attachment with default",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Attrs = map[string]any{"key": "value"}
				return r
			},
			key:          "nonexistent",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name: "get from nil attachments",
			setup: func() *RPCResult {
				return &RPCResult{}
			},
			key:          "key",
			defaultValue: "default",
			expected:     nil,
		},
		{
			name: "get with nil default value",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Attrs = map[string]any{"key": "value"}
				return r
			},
			key:          "nonexistent",
			defaultValue: nil,
			expected:     nil,
		},
		{
			name: "get int value",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Attrs = map[string]any{"count": 42}
				return r
			},
			key:          "count",
			defaultValue: 0,
			expected:     42,
		},
		{
			name: "get bool value",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Attrs = map[string]any{"flag": true}
				return r
			},
			key:          "flag",
			defaultValue: false,
			expected:     true,
		},
		{
			name: "get nil value",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Attrs = map[string]any{"null": nil}
				return r
			},
			key:          "null",
			defaultValue: "default",
			expected:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.setup()
			result := r.Attachment(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRPCResult_Attachment_EmptyMap(t *testing.T) {
	r := &RPCResult{}
	r.Attrs = map[string]any{}

	result := r.Attachment("key", "default")
	assert.Equal(t, "default", result)
}

func TestRPCResult_String(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *RPCResult
		contains []string
	}{
		{
			name: "empty result",
			setup: func() *RPCResult {
				return &RPCResult{}
			},
			contains: []string{"&RPCResult", "Rest:", "Attrs:", "Err:"},
		},
		{
			name: "result with data",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Rest = "test result"
				r.Attrs = map[string]any{"key": "value"}
				r.Err = errors.New("test error")
				return r
			},
			contains: []string{"&RPCResult", "Rest:", "test result", "Attrs:", "key", "value", "Err:", "test error"},
		},
		{
			name: "result with nil error",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Rest = "data"
				r.Attrs = map[string]any{"a": "b"}
				return r
			},
			contains: []string{"&RPCResult", "Rest:", "data", "Attrs:", "a", "b", "Err:"},
		},
		{
			name: "result with complex types",
			setup: func() *RPCResult {
				r := &RPCResult{}
				r.Rest = map[string]int{"count": 42}
				r.Attrs = map[string]any{"slice": []int{1, 2, 3}}
				return r
			},
			contains: []string{"&RPCResult", "Rest:", "Attrs:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.setup()
			result := r.String()

			assert.NotEmpty(t, result)
			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

func TestRPCResult_InterfaceCompliance(t *testing.T) {
	var _ Result = (*RPCResult)(nil)

	// Verify all interface methods are implemented
	r := &RPCResult{}

	// Test SetError and Error
	testErr := errors.New("test")
	r.SetError(testErr)
	assert.Equal(t, testErr, r.Error())

	// Test SetResult and Result
	testResult := "test result"
	r.SetResult(testResult)
	assert.Equal(t, testResult, r.Result())

	// Test SetAttachments and Attachments
	testAttachments := map[string]any{"key": "value"}
	r.SetAttachments(testAttachments)
	assert.Equal(t, testAttachments, r.Attachments())

	// Test AddAttachment
	r.AddAttachment("new", "newValue")
	assert.Equal(t, "newValue", r.Attrs["new"])

	// Test Attachment
	val := r.Attachment("key", "default")
	assert.Equal(t, "value", val)
}

func TestRPCResult_ConcurrentOperations(t *testing.T) {
	// Test that multiple separate RPCResult instances work independently
	// Note: RPCResult is not designed to be thread-safe for concurrent access to the same instance
	done := make(chan bool, 10)

	// Test concurrent operations on different instances
	for i := 0; i < 10; i++ {
		go func(idx int) {
			r := &RPCResult{}
			r.AddAttachment("key", idx)
			assert.NotNil(t, r.Attrs)
			assert.Equal(t, idx, r.Attrs["key"])
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRPCResult_ChainedOperations(t *testing.T) {
	r := &RPCResult{}

	// Chain multiple operations
	r.SetResult("result")
	r.SetError(errors.New("error"))
	r.SetBizError(errors.New("biz error"))
	r.AddAttachment("key1", "value1")
	r.AddAttachment("key2", "value2")

	// Verify all operations worked
	assert.Equal(t, "result", r.Result())
	assert.NotNil(t, r.Error())
	assert.NotNil(t, r.BizError())
	assert.Equal(t, 2, len(r.Attachments()))
	assert.Equal(t, "value1", r.Attachment("key1", nil))
	assert.Equal(t, "value2", r.Attachment("key2", nil))
}

func TestRPCResult_EdgeCases(t *testing.T) {
	t.Run("empty string key", func(t *testing.T) {
		r := &RPCResult{}
		r.AddAttachment("", "value")
		assert.Equal(t, "value", r.Attachment("", "default"))
	})

	t.Run("large attachment map", func(t *testing.T) {
		r := &RPCResult{}
		for i := 0; i < 1000; i++ {
			r.AddAttachment(fmt.Sprintf("key-%d", i), i)
		}
		assert.Equal(t, 1000, len(r.Attachments()))
	})

	t.Run("overwrite with nil", func(t *testing.T) {
		r := &RPCResult{}
		r.SetResult("initial")
		r.SetResult(nil)
		assert.Nil(t, r.Result())
	})

	t.Run("set attachments to nil then add", func(t *testing.T) {
		r := &RPCResult{}
		r.SetAttachments(map[string]any{"key": "value"})
		r.SetAttachments(nil)
		r.AddAttachment("new", "newValue")
		assert.NotNil(t, r.Attrs)
		assert.Equal(t, "newValue", r.Attrs["new"])
		assert.NotContains(t, r.Attrs, "key")
	})
}

// Benchmark tests
func BenchmarkRPCResult_SetError(b *testing.B) {
	r := &RPCResult{}
	err := errors.New("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.SetError(err)
	}
}

func BenchmarkRPCResult_SetResult(b *testing.B) {
	r := &RPCResult{}
	result := "test result"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.SetResult(result)
	}
}

func BenchmarkRPCResult_AddAttachment(b *testing.B) {
	r := &RPCResult{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.AddAttachment(fmt.Sprintf("key-%d", i), "value")
	}
}

func BenchmarkRPCResult_Attachment(b *testing.B) {
	r := &RPCResult{}
	r.Attrs = map[string]any{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Attachment("key", "default")
	}
}

func BenchmarkRPCResult_String(b *testing.B) {
	r := &RPCResult{
		Rest:  "test result",
		Attrs: map[string]any{"key": "value"},
		Err:   errors.New("test error"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.String()
	}
}
