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
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTripleGenericService_ConvertSingleArgForSerializationDetailed detailed test for single argument serialization
func TestTripleGenericService_ConvertSingleArgForSerializationDetailed(t *testing.T) {
	tests := []struct {
		name        string
		arg         any
		targetType  string
		expected    any
		expectError bool
	}{
		// int32 conversion tests
		{"int32_from_int32", int32(123), "int32", int32(123), false},
		{"int32_from_int", int(456), "int32", int32(456), false},
		{"int32_from_int64", int64(789), "int32", int32(789), false},
		{"int32_from_float32", float32(12.5), "int32", int32(12), false},
		{"int32_from_float64", float64(34.7), "int32", int32(34), false},
		{"int32_from_string_valid", "678", "int32", int32(678), false},
		{"int32_from_string_invalid", "invalid", "int32", int32(0), true},
		{"int32_from_bool", true, "int32", int32(0), true},

		// int64 conversion tests
		{"int64_from_int64", int64(123456789), "int64", int64(123456789), false},
		{"int64_from_int", int(456), "int64", int64(456), false},
		{"int64_from_float64", float64(789.12), "int64", int64(789), false},
		{"int64_from_string_valid", "9876543210", "int64", int64(9876543210), false},
		{"int64_from_string_invalid", "not_a_number", "int64", int64(0), true},

		// float32 conversion tests
		{"float32_from_float32", float32(3.14), "float32", float32(3.14), false},
		{"float32_from_float64", float64(2.718), "float32", float32(2.718), false},
		{"float32_from_int", int(42), "float32", float32(42.0), false},
		{"float32_from_string_valid", "3.14159", "float32", float32(3.14159), false},
		{"float32_from_string_invalid", "not_float", "float32", float32(0), true},

		// float64 conversion tests
		{"float64_from_float64", float64(3.141592653589793), "float64", float64(3.141592653589793), false},
		{"float64_from_int64", int64(123), "float64", float64(123.0), false},
		{"float64_from_string_valid", "2.718281828", "float64", float64(2.718281828), false},
		{"float64_from_string_invalid", "invalid_float", "float64", float64(0), true},

		// string conversion tests
		{"string_from_string", "hello", "string", "hello", false},
		{"string_from_int", 123, "string", "123", false},
		{"string_from_float", 3.14, "string", "3.14", false},
		{"string_from_bool", true, "string", "true", false},
		{"string_from_nil", nil, "string", "", false},

		// bool conversion tests
		{"bool_from_bool_true", true, "bool", true, false},
		{"bool_from_bool_false", false, "bool", false, false},
		{"bool_from_int", 1, "bool", false, false}, // non-bool types convert to false
		{"bool_from_nil", nil, "bool", false, false},

		// bytes conversion tests
		{"bytes_from_bytes", []byte("hello"), "bytes", []byte("hello"), false},
		{"bytes_from_string", "world", "bytes", []byte("world"), false},
		{"bytes_from_int", 123, "bytes", []byte("123"), false},
		{"bytes_from_nil", nil, "bytes", []byte(nil), false},

		// complex types remain as-is
		{"map_passthrough", map[string]any{"key": "value"}, "map", map[string]any{"key": "value"}, false},
		{"slice_passthrough", []string{"a", "b", "c"}, "[]string", []string{"a", "b", "c"}, false},
		{"unknown_type", "test", "unknown_type", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertSingleArgForSerialization(tt.arg, tt.targetType)

			if tt.expectError {
				assert.Error(t, err, "Expected error for %s", tt.name)
			} else {
				assert.NoError(t, err, "Unexpected error for %s: %v", tt.name, err)
				assert.Equal(t, tt.expected, result, "Conversion result mismatch for %s", tt.name)
			}
		})
	}
}

// TestTripleGenericService_ParameterValidation parameter validation tests
func TestTripleGenericService_ParameterValidation(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20001/com.example.TestService")
	ctx := context.Background()

	tests := []struct {
		name          string
		methodName    string
		types         []string
		args          []any
		expectedError string
	}{
		{
			name:          "empty_method_name",
			methodName:    "",
			types:         []string{"string"},
			args:          []any{"test"},
			expectedError: "method name cannot be empty",
		},
		{
			name:          "nil_types",
			methodName:    "testMethod",
			types:         nil,
			args:          []any{"test"},
			expectedError: "parameter count mismatch",
		},
		{
			name:          "nil_args",
			methodName:    "testMethod",
			types:         []string{"string"},
			args:          nil,
			expectedError: "parameter count mismatch",
		},
		{
			name:          "empty_types_empty_args",
			methodName:    "testMethod",
			types:         []string{},
			args:          []any{},
			expectedError: "", // should succeed
		},
		{
			name:          "mismatched_param_count_more_types",
			methodName:    "testMethod",
			types:         []string{"string", "int32"},
			args:          []any{"test"},
			expectedError: "parameter count mismatch",
		},
		{
			name:          "mismatched_param_count_more_args",
			methodName:    "testMethod",
			types:         []string{"string"},
			args:          []any{"test", "extra"},
			expectedError: "parameter count mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tgs.Invoke(ctx, tt.methodName, tt.types, tt.args)

			if tt.expectedError == "" {
				// these calls should pass parameter validation successfully, but will fail due to network errors
				assert.Error(t, err, "Should have network error")
				assert.Contains(t, err.Error(), "network connection failed", "Should be network error, not parameter error")
			} else {
				assert.Error(t, err, "Expected error for %s", tt.name)
				assert.Contains(t, err.Error(), tt.expectedError, "Error message should contain expected text for %s", tt.name)
			}
		})
	}
}

// TestTripleGenericService_EdgeCases edge case tests
func TestTripleGenericService_EdgeCases(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20001/com.example.TestService")
	ctx := context.Background()

	t.Run("extremely_long_method_name", func(t *testing.T) {
		longMethodName := string(make([]byte, 10000)) // 10KB method name
		for i := range longMethodName {
			longMethodName = longMethodName[:i] + "a" + longMethodName[i+1:]
		}

		_, err := tgs.Invoke(ctx, longMethodName, []string{"string"}, []any{"test"})
		assert.Error(t, err)
		// should be network error, not parameter error
		assert.Contains(t, err.Error(), "network connection failed")
	})

	t.Run("extremely_large_parameter", func(t *testing.T) {
		largeData := make([]byte, 1024*1024) // 1MB data
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		_, err := tgs.Invoke(ctx, "testMethod", []string{"bytes"}, []any{largeData})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network connection failed")
	})

	t.Run("deeply_nested_structures", func(t *testing.T) {
		// create deeply nested structure
		nested := make(map[string]any)
		current := nested
		for i := 0; i < 100; i++ {
			next := make(map[string]any)
			current["level"] = i
			current["next"] = next
			current = next
		}
		current["final"] = "value"

		_, err := tgs.Invoke(ctx, "testMethod", []string{"map"}, []any{nested})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network connection failed")
	})

	t.Run("unicode_and_special_characters", func(t *testing.T) {
		specialChars := "testdata 🚀 Special chars: !@#$%^&*()_+-={}[]|\\:;\"'<>?,./"

		_, err := tgs.Invoke(ctx, "unicodetestMethod", []string{"string"}, []any{specialChars})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network connection failed")
	})
}

// TestTripleGenericService_ConcurrencyStress concurrent stress test
func TestTripleGenericService_ConcurrencyStress(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20001/com.example.StressTestService")
	ctx := context.Background()

	t.Run("high_concurrency_stress", func(t *testing.T) {
		const numGoroutines = 100
		const callsPerGoroutine = 10

		var wg sync.WaitGroup
		errorChan := make(chan error, numGoroutines*callsPerGoroutine)

		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < callsPerGoroutine; j++ {
					_, err := tgs.Invoke(ctx, "stressMethod",
						[]string{"int32", "string"},
						[]any{int32(goroutineID*callsPerGoroutine + j), "stress_test"})
					if err != nil {
						errorChan <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errorChan)
		duration := time.Since(start)

		// collect errors
		var errors []error
		for err := range errorChan {
			errors = append(errors, err)
		}

		// all calls should have network errors
		assert.Len(t, errors, numGoroutines*callsPerGoroutine, "All calls should have errors")

		// verify no concurrency-related panics or race conditions
		for _, err := range errors {
			assert.Contains(t, err.Error(), "network connection failed", "Should be network errors")
			assert.NotContains(t, err.Error(), "panic", "Should not have panics")
			assert.NotContains(t, err.Error(), "race", "Should not have race conditions")
		}

		t.Logf("stresstestcompleted: %d goroutines, %d calls each, total duration: %v",
			numGoroutines, callsPerGoroutine, duration)
	})
}

// TestTripleGenericService_AsyncManagerStress async managerstresstest
func TestTripleGenericService_AsyncManagerStress(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20001/com.example.AsyncStressService")
	ctx := context.Background()

	t.Run("async_manager_stress", func(t *testing.T) {
		const numCalls = 200
		var wg sync.WaitGroup
		callIDs := make(chan string, numCalls)

		// startlargeasynccalls
		for i := 0; i < numCalls; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				callID, err := tgs.InvokeAsync(ctx, "asyncStressMethod",
					[]string{"int32"}, []any{int32(index)}, nil,
					func(result any, err error) {
						// callback function
					})

				if err == nil {
					callIDs <- callID
				}
			}(i)
		}

		wg.Wait()
		close(callIDs)

		// collect callsID
		var collectedIDs []string
		for id := range callIDs {
			collectedIDs = append(collectedIDs, id)
		}

		assert.Equal(t, numCalls, len(collectedIDs), "All async calls should succeed")

		// verifyIDuniqueness
		idSet := make(map[string]bool)
		for _, id := range collectedIDs {
			assert.False(t, idSet[id], "Call ID should be unique: %s", id)
			idSet[id] = true
		}

		// Wait for async calls to complete
		time.Sleep(100 * time.Millisecond)

		// Verify async manager state
		manager := GetTripleAsyncManager()
		activeCalls := manager.getAllActiveCalls()

		t.Logf("asyncstresstestcompleted: %d calls, %d active calls", numCalls, len(activeCalls))
	})
}

// TestTripleGenericService_AttachmentBuilderEdgeCases attachment builder boundariestest
func TestTripleGenericService_AttachmentBuilderEdgeCases(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20001/com.example.TestService")

	t.Run("builder_edge_cases", func(t *testing.T) {
		builder := tgs.CreateAttachmentBuilder()

		// testemptykey
		result := builder.SetString("", "empty_key").Build()
		assert.Equal(t, "empty_key", result[""])

		// testemptyvalue
		builder = tgs.CreateAttachmentBuilder()
		result = builder.SetString("empty_value", "").Build()
		assert.Equal(t, "", result["empty_value"])

		// testoverride existingkey
		builder = tgs.CreateAttachmentBuilder()
		result = builder.
			SetString("key", "value1").
			SetString("key", "value2").
			Build()
		assert.Equal(t, "value2", result["key"])

		// testmixed types
		builder = tgs.CreateAttachmentBuilder()
		result = builder.
			SetString("str", "string_value").
			SetInt("int", 123).
			SetBool("bool", true).
			SetInt("zero", 0).
			SetBool("false", false).
			Build()

		expected := map[string]any{
			"str":   "string_value",
			"int":   123,
			"bool":  true,
			"zero":  0,
			"false": false,
		}
		assert.Equal(t, expected, result)

		// testmultipleBuild()calls
		result1 := builder.Build()
		result2 := builder.Build()
		assert.Equal(t, result1, result2, "Multiple Build() calls should return same result")

		// verifymodifying one result does not affect another
		result1["new_key"] = "new_value"
		assert.NotContains(t, result2, "new_key", "Results should be independent")
	})
}

// TestTripleGenericService_ContextCancellation context cancellationtest
func TestTripleGenericService_ContextCancellation(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20001/com.example.TestService")

	t.Run("context_cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// cancel context immediately
		cancel()

		_, err := tgs.Invoke(ctx, "testMethod", []string{"string"}, []any{"test"})
		assert.Error(t, err)
		// mayiscontext cancellationerroror networkerror，depending on implementation details
	})

	t.Run("context_timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// wait timeout
		time.Sleep(1 * time.Millisecond)

		_, err := tgs.Invoke(ctx, "testMethod", []string{"string"}, []any{"test"})
		assert.Error(t, err)
	})
}

// TestTripleGenericService_BatchInvokeEdgeCases batchcallsboundariestest
func TestTripleGenericService_BatchInvokeEdgeCases(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20001/com.example.BatchTestService")
	ctx := context.Background()

	t.Run("batch_with_invalid_params", func(t *testing.T) {
		invocations := []TripleInvocationRequest{
			{
				MethodName: "validMethod",
				Types:      []string{"string"},
				Args:       []any{"valid"},
			},
			{
				MethodName: "", // invalidmethod name
				Types:      []string{"string"},
				Args:       []any{"test"},
			},
			{
				MethodName: "invalidParams",
				Types:      []string{"int32"},
				Args:       []any{"not_a_number"}, // type mismatch
			},
		}

		results, err := tgs.BatchInvoke(ctx, invocations)
		assert.NoError(t, err, "Batch operation should not fail")
		assert.Len(t, results, 3)

		// verifyindividual results
		assert.Error(t, results[1].Error, "Empty method name should cause error")
		assert.Contains(t, results[1].Error.Error(), "method name cannot be empty")

		assert.Error(t, results[2].Error, "Type conversion error should occur")
		assert.Contains(t, results[2].Error.Error(), "cannot convert")
	})

	t.Run("batch_with_zero_concurrency", func(t *testing.T) {
		invocations := []TripleInvocationRequest{
			{
				MethodName: "testMethod",
				Types:      []string{"string"},
				Args:       []any{"test"},
			},
		}

		options := BatchInvokeOptions{
			MaxConcurrency: 0, // zeroconcurrent
			FailFast:       false,
		}

		// shoulduse defaultconcurrentor handlezeroconcurrentcase
		results, err := tgs.BatchInvokeWithOptions(ctx, invocations, options)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
	})
}
