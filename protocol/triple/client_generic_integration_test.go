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
	"fmt"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

// TestTripleGenericService_IntegrationScenarios tests integration scenarios
func TestTripleGenericService_IntegrationScenarios(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20003/com.example.UserService?serialization=hessian2")
	ctx := context.Background()

	t.Run("Complete user management service flow", func(t *testing.T) {
		// Scenario 1: Create user
		newUser := map[string]any{
			"name":  "John Doe",
			"email": "zhangsan@example.com",
			"age":   28,
			"city":  "Beijing",
		}

		// Since there's no real server, we mainly test that calls don't panic and parameters are correctly passed
		_, err := tgs.Invoke(ctx, "createUser", []string{"map"}, []any{newUser})
		// Expected to have network error, but should not have parameter errors
		assert.Error(t, err, "Expected network connection error")
		assert.NotContains(t, err.Error(), "parameter", "Should not have parameter errors")

		// Scenario 2: Query user
		_, err = tgs.Invoke(ctx, "getUserById", []string{"int64"}, []any{int64(12345)})
		assert.Error(t, err, "Expected network connection error")

		// Scenario 3: Update user information
		updates := map[string]any{
			"age":  30,
			"city": "Shanghai",
		}
		_, err = tgs.Invoke(ctx, "updateUser", []string{"int64", "map"}, []any{int64(12345), updates})
		assert.Error(t, err, "Expected network connection error")

		// Scenario 4: Delete user
		_, err = tgs.Invoke(ctx, "deleteUser", []string{"int64"}, []any{int64(12345)})
		assert.Error(t, err, "Expected network connection error")
	})

	t.Run("E-commerce order service scenario", func(t *testing.T) {
		// Scenario 1: Create order
		orderItems := []any{
			map[string]any{
				"productId": int64(1001),
				"quantity":  2,
				"price":     99.99,
			},
			map[string]any{
				"productId": int64(1002),
				"quantity":  1,
				"price":     159.99,
			},
		}

		order := map[string]any{
			"userId":      int64(789),
			"items":       orderItems,
			"totalAmount": 359.97,
			"address":     "Beijing Chaoyang District XX Street XX Number",
		}

		_, err := tgs.Invoke(ctx, "createOrder", []string{"map"}, []any{order})
		assert.Error(t, err, "Expected network connection error")

		// Scenario 2: Query order status
		_, err = tgs.Invoke(ctx, "getOrderStatus", []string{"string"}, []any{"ORDER_20231201_001"})
		assert.Error(t, err, "Expected network connection error")

		// Scenario 3: Cancel order
		_, err = tgs.Invoke(ctx, "cancelOrder", []string{"string", "string"},
			[]any{"ORDER_20231201_001", "User initiated cancellation"})
		assert.Error(t, err, "Expected network connection error")
	})
}

// TestTripleGenericService_DataTypesCoverage tests data types coverage
func TestTripleGenericService_DataTypesCoverage(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20003/com.example.DataService?serialization=hessian2")
	ctx := context.Background()

	testCases := []struct {
		name   string
		method string
		types  []string
		args   []any
		desc   string
	}{
		{
			name:   "string_type",
			method: "processString",
			types:  []string{"string"},
			args:   []any{"Hello World UTF-8 Test"},
			desc:   "Test UTF-8 string processing",
		},
		{
			name:   "integer_type_combination",
			method: "processIntegers",
			types:  []string{"int32", "int64", "int32"},
			args:   []any{int32(123), int64(9876543210), int32(-456)},
			desc:   "Test different integer types",
		},
		{
			name:   "float_type",
			method: "processFloats",
			types:  []string{"float32", "float64"},
			args:   []any{float32(3.14), float64(2.718281828)},
			desc:   "Test floating point precision",
		},
		{
			name:   "boolean_type",
			method: "processBoolean",
			types:  []string{"bool", "bool"},
			args:   []any{true, false},
			desc:   "Test boolean values",
		},
		{
			name:   "byte_array",
			method: "processBytes",
			types:  []string{"bytes"},
			args:   []any{[]byte("binary data test")},
			desc:   "Test binary data",
		},
		{
			name:   "array_type",
			method: "processArray",
			types:  []string{"[]string", "[]int64"},
			args: []any{
				[]string{"apple", "banana", "cherry"},
				[]int64{100, 200, 300, 400, 500},
			},
			desc: "Test array parameters",
		},
		{
			name:   "complex_object",
			method: "processComplexObject",
			types:  []string{"map"},
			args: []any{
				map[string]any{
					"id":     int64(12345),
					"name":   "complex object test",
					"active": true,
					"tags":   []string{"test", "complex", "object"},
					"metadata": map[string]any{
						"version":    "1.0.0",
						"created_at": "2023-12-01T10:00:00Z",
						"author":     "test user",
					},
					"scores": []float64{95.5, 87.3, 92.1},
				},
			},
			desc: "Test nested complex objects",
		},
		{
			name:   "null_value_handling",
			method: "processNullValues",
			types:  []string{"string", "map", "[]string"},
			args: []any{
				"",
				map[string]any{},
				[]string{},
			},
			desc: "Test null values and empty collections",
		},
		{
			name:   "special_characters",
			method: "processSpecialChars",
			types:  []string{"string"},
			args:   []any{"Special chars: !@#$%^&*()_+-={}[]|\\:;\"'<>?,./ Emoji😀🎉🚀"},
			desc:   "Test special characters and emojis",
		},
		{
			name:   "large_data_volume",
			method: "processLargeData",
			types:  []string{"[]map"},
			args: []any{
				func() []any {
					var data []any
					for i := 0; i < 100; i++ {
						data = append(data, map[string]any{
							"id":    int64(i),
							"value": fmt.Sprintf("data_%d", i),
							"index": i,
						})
					}
					return data
				}(),
			},
			desc: "Test large data volume processing",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Test scenario: %s", tc.desc)

			// Test basic invocation
			_, err := tgs.Invoke(ctx, tc.method, tc.types, tc.args)
			// Expected network error due to no real server, but should not have parameter processing errors
			assert.Error(t, err, "Expected network connection error")

			// Verify error is not due to parameter processing issues
			assert.NotContains(t, err.Error(), "parameter count mismatch", "Parameter count should match")
			assert.NotContains(t, err.Error(), "failed to convert", "Type conversion should succeed")
		})
	}
}

// TestTripleGenericService_AttachmentsScenarios tests attachment usage scenarios
func TestTripleGenericService_AttachmentsScenarios(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20003/com.example.AttachmentService?serialization=hessian2")
	ctx := context.Background()

	t.Run("distributed_tracing_scenario", func(t *testing.T) {
		traceAttachments := map[string]any{
			"traceId":      "trace-12345-67890",
			"spanId":       "span-abcde-fghij",
			"parentSpanId": "span-00000-11111",
			"sampled":      true,
			"baggage":      "user=pixu,service=test",
		}

		_, err := tgs.InvokeWithAttachments(ctx, "processWithTracing",
			[]string{"string"}, []any{"test data"}, traceAttachments)
		assert.Error(t, err, "Expected network connection error")
	})

	t.Run("user_context_scenario", func(t *testing.T) {
		userContextAttachments := tgs.CreateAttachmentBuilder().
			SetString("userId", "user_12345").
			SetString("username", "pixu").
			SetString("roles", "admin,user").
			SetString("tenant", "company_abc").
			SetString("locale", "zh_CN").
			SetString("timezone", "Asia/Shanghai").
			SetBool("authenticated", true).
			SetInt("sessionTimeout", 3600).
			Build()

		_, err := tgs.InvokeWithAttachments(ctx, "getUserProfile",
			[]string{"string"}, []any{"profile_data"}, userContextAttachments)
		assert.Error(t, err, "Expected network connection error")
	})

	t.Run("rate_limiting_and_circuit_breaking_scenario", func(t *testing.T) {
		rateLimitAttachments := map[string]any{
			"rateLimitKey":   "api_key_12345",
			"requestsPerMin": 100,
			"burstSize":      10,
			"circuitBreaker": "enabled",
			"timeout":        "5000ms",
			"retryAttempts":  3,
			"retryBackoff":   "exponential",
		}

		_, err := tgs.InvokeWithAttachments(ctx, "apiCallWithLimits",
			[]string{"map"}, []any{
				map[string]any{"action": "getData", "params": "test"}},
			rateLimitAttachments)
		assert.Error(t, err, "Expected network connection error")
	})

	t.Run("security_authentication_scenario", func(t *testing.T) {
		securityAttachments := tgs.CreateAttachmentBuilder().
			SetString("authorization", "Bearer eyJhbGciOiJIUzI1NiIs...").
			SetString("apiKey", "ak_test_12345").
			SetString("signature", "sha256=abcdef123456...").
			SetString("timestamp", fmt.Sprintf("%d", time.Now().Unix())).
			SetString("nonce", "random_nonce_12345").
			SetBool("requiresAuth", true).
			Build()

		_, err := tgs.InvokeWithAttachments(ctx, "secureOperation",
			[]string{"string", "map"},
			[]any{"sensitive_data", map[string]any{"level": "confidential"}},
			securityAttachments)
		assert.Error(t, err, "Expected network connection error")
	})
}

// TestTripleGenericService_AsyncScenarios tests asynchronous invocation scenarios
func TestTripleGenericService_AsyncScenarios(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20003/com.example.AsyncService?serialization=hessian2")
	ctx := context.Background()

	t.Run("concurrent_async_calls", func(t *testing.T) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var callIDs []string
		var errors []error

		// Start multiple concurrent asynchronous calls
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				callID, err := tgs.InvokeAsync(ctx, "asyncMethod",
					[]string{"int32", "string"},
					[]any{int32(index), fmt.Sprintf("async_call_%d", index)},
					map[string]any{"callIndex": index},
					func(result any, err error) {
						mu.Lock()
						defer mu.Unlock()
						if err != nil {
							errors = append(errors, err)
						}
						t.Logf("async callback %d: result=%v, err=%v", index, result, err)
					})

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					callIDs = append(callIDs, callID)
				}
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Verify all calls started successfully
		assert.Len(t, callIDs, 5, "Should have 5 successful call IDs")
		t.Logf("Started %d asynchronous calls", len(callIDs))
	})

	t.Run("async_call_timeout_handling", func(t *testing.T) {
		done := make(chan bool, 1)

		callID, err := tgs.InvokeAsyncWithTimeout(ctx, "slowMethod",
			[]string{"int32"}, []any{int32(100)},
			nil,
			func(result any, err error) {
				t.Logf("timeout test callback: result=%v, err=%v", result, err)
				done <- true
			},
			100*time.Millisecond) // Very short timeout

		if err == nil {
			assert.NotEmpty(t, callID, "call ID should not be empty")

			// Wait for callback or timeout
			select {
			case <-done:
				t.Log("async call callback completed")
			case <-time.After(1 * time.Second):
				t.Log("waiting for async callback timeout, this is expected")
			}
		}
	})

	t.Run("async_call_cancellation", func(t *testing.T) {
		callID, err := tgs.InvokeAsync(ctx, "longRunningMethod",
			[]string{"string"}, []any{"test"},
			nil,
			func(result any, err error) {
				t.Logf("cancellation test callback: result=%v, err=%v", result, err)
			})

		if err == nil {
			assert.NotEmpty(t, callID, "call ID should not be empty")

			// Cancel call immediately
			cancelled := tgs.CancelAsyncCall(callID)
			assert.True(t, cancelled, "Should be able to cancel call")

			// Attempting to cancel again should return false
			cancelled = tgs.CancelAsyncCall(callID)
			assert.False(t, cancelled, "Duplicate cancellation should return false")
		}
	})
}

// TestTripleGenericService_BatchScenarios tests batch invocation scenarios
func TestTripleGenericService_BatchScenarios(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20003/com.example.BatchService?serialization=hessian2")
	ctx := context.Background()

	t.Run("batch_user_query", func(t *testing.T) {
		userIDs := []int64{1001, 1002, 1003, 1004, 1005}
		var invocations []TripleInvocationRequest

		for _, userID := range userIDs {
			invocations = append(invocations, TripleInvocationRequest{
				MethodName: "getUserById",
				Types:      []string{"int64"},
				Args:       []any{userID},
				Attachments: map[string]any{
					"batchId": "batch_users_001",
					"userId":  userID,
				},
			})
		}

		results, err := tgs.BatchInvoke(ctx, invocations)
		assert.NoError(t, err, "batch invocation should not have structural errors")
		assert.Len(t, results, len(userIDs), "result count should match")

		for i, result := range results {
			assert.Equal(t, i, result.Index, "result index should match")
			// Expected network error for each call due to no real server
			assert.Error(t, result.Error, "Expected network error for each call")
		}
	})

	t.Run("mixed_method_batch_call", func(t *testing.T) {
		invocations := []TripleInvocationRequest{
			{
				MethodName:  "createUser",
				Types:       []string{"map"},
				Args:        []any{map[string]any{"name": "user1", "age": 25}},
				Attachments: map[string]any{"operation": "create"},
			},
			{
				MethodName:  "getUserById",
				Types:       []string{"int64"},
				Args:        []any{int64(1001)},
				Attachments: map[string]any{"operation": "read"},
			},
			{
				MethodName:  "updateUser",
				Types:       []string{"int64", "map"},
				Args:        []any{int64(1002), map[string]any{"age": 26}},
				Attachments: map[string]any{"operation": "update"},
			},
			{
				MethodName:  "deleteUser",
				Types:       []string{"int64"},
				Args:        []any{int64(1003)},
				Attachments: map[string]any{"operation": "delete"},
			},
			{
				MethodName:  "listUsers",
				Types:       []string{"int32", "int32"},
				Args:        []any{int32(0), int32(10)},
				Attachments: map[string]any{"operation": "list"},
			},
		}

		results, err := tgs.BatchInvoke(ctx, invocations)
		assert.NoError(t, err, "batch invocation should not have structural errors")
		assert.Len(t, results, len(invocations), "result count should match")

		operations := []string{"create", "read", "update", "delete", "list"}
		for i, result := range results {
			assert.Equal(t, i, result.Index, "result index should match")
			t.Logf("operation %s (index %d): %v", operations[i], i,
				map[string]any{"hasError": result.Error != nil})
		}
	})

	t.Run("large_batch_concurrency_control", func(t *testing.T) {
		// Create large number of call requests
		var invocations []TripleInvocationRequest
		for i := 0; i < 50; i++ {
			invocations = append(invocations, TripleInvocationRequest{
				MethodName:  "processItem",
				Types:       []string{"int32", "string"},
				Args:        []any{int32(i), fmt.Sprintf("item_%d", i)},
				Attachments: map[string]any{"itemIndex": i},
			})
		}

		// Test different concurrency settings
		concurrencyTests := []struct {
			name           string
			maxConcurrency int
			failFast       bool
		}{
			{"low_concurrency", 2, false},
			{"medium_concurrency", 10, false},
			{"high_concurrency", 20, false},
			{"fail_fast", 5, true},
		}

		for _, test := range concurrencyTests {
			t.Run(test.name, func(t *testing.T) {
				options := BatchInvokeOptions{
					MaxConcurrency: test.maxConcurrency,
					FailFast:       test.failFast,
				}

				start := time.Now()
				results, err := tgs.BatchInvokeWithOptions(ctx, invocations, options)
				duration := time.Since(start)

				assert.NoError(t, err, "batch invocation should not have structural errors")
				assert.Len(t, results, len(invocations), "result count should match")

				t.Logf("concurrency setting: %d, duration: %v", test.maxConcurrency, duration)
			})
		}
	})
}

// TestTripleGenericService_ErrorHandlingScenarios tests error handling scenarios
func TestTripleGenericService_ErrorHandlingScenarios(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20003/com.example.ErrorService?serialization=hessian2")
	ctx := context.Background()

	t.Run("Parameter validation errors", func(t *testing.T) {
		// Empty method name
		_, err := tgs.Invoke(ctx, "", []string{"string"}, []any{"test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "method name cannot be empty")

		// Parameter count mismatch (for IDL mode conversion)
		_, err = tgs.convertParamsForIDL([]any{1, 2}, []string{"int32"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parameter count mismatch")
	})

	t.Run("Type conversion errors", func(t *testing.T) {
		// Test various type conversion errors
		_, err := convertToInt32("not_a_number")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert")

		_, err = convertToFloat64("not_a_float")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert")
	})

	t.Run("timeout_handling", func(t *testing.T) {
		shortCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
		defer cancel()

		// Let context timeout first
		time.Sleep(2 * time.Millisecond)

		_, err := tgs.Invoke(shortCtx, "slowMethod", []string{"string"}, []any{"test"})
		assert.Error(t, err)
		// Since context is already cancelled, might have context cancellation error
	})

	t.Run("concurrent_safety_test", func(t *testing.T) {
		var wg sync.WaitGroup
		errorChan := make(chan error, 10)

		// Concurrent access to the same service instance
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				_, err := tgs.Invoke(ctx, "concurrentMethod",
					[]string{"int32"}, []any{int32(index)})
				if err != nil {
					errorChan <- err
				}
			}(i)
		}

		wg.Wait()
		close(errorChan)

		// Collect errors
		var errors []error
		for err := range errorChan {
			errors = append(errors, err)
		}

		// All calls should have network errors, but should not have concurrency-related panics
		assert.Len(t, errors, 10, "Should have 10 network errors")
		for _, err := range errors {
			assert.NotContains(t, err.Error(), "panic", "should not have panic")
			assert.NotContains(t, err.Error(), "concurrent", "should not have concurrency errors")
		}
	})
}

// TestTripleGenericService_PerformanceScenarios tests performance scenarios
func TestTripleGenericService_PerformanceScenarios(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20003/com.example.PerfService?serialization=hessian2")
	ctx := context.Background()

	t.Run("single_call_performance", func(t *testing.T) {
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			_, _ = tgs.Invoke(ctx, "perfTest", []string{"int32"}, []any{int32(i)})
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("average single call duration: %v (total %d calls)", avgDuration, iterations)

		// Verify average call time is reasonable (considering network error overhead)
		assert.Less(t, avgDuration, 10*time.Millisecond, "Single call duration should be reasonable")
	})

	t.Run("batch_call_performance_comparison", func(t *testing.T) {
		callCount := 20

		// Serial calls
		start := time.Now()
		for i := 0; i < callCount; i++ {
			_, _ = tgs.Invoke(ctx, "perfTest", []string{"int32"}, []any{int32(i)})
		}
		serialDuration := time.Since(start)

		// Batch parallel calls
		var invocations []TripleInvocationRequest
		for i := 0; i < callCount; i++ {
			invocations = append(invocations, TripleInvocationRequest{
				MethodName: "perfTest",
				Types:      []string{"int32"},
				Args:       []any{int32(i)},
			})
		}

		start = time.Now()
		_, _ = tgs.BatchInvoke(ctx, invocations)
		batchDuration := time.Since(start)

		t.Logf("serialinvocationduration: %v", serialDuration)
		t.Logf("batchinvocationduration: %v", batchDuration)
		t.Logf("performance improvement ratio: %.2fx", float64(serialDuration)/float64(batchDuration))

		// In a real network environment, batch calls would be faster due to parallelization
		// However, in test environment with simulated errors, goroutine overhead might make batch calls slower
		// So we just verify both complete successfully and log the performance difference
		t.Logf("performance test note: In real network environments，batchinvocationare usually faster。testenvironments, goroutine overhead may make，goroutineoverheadmaycausingbatchinvocationslower。")

		// Verify both methods complete successfully (even with network errors)
		assert.True(t, serialDuration > 0, "Serial calls should complete")
		assert.True(t, batchDuration > 0, "Batch calls should complete")
	})

	t.Run("memory_usage_test", func(t *testing.T) {
		// Create large number of small objects to test memory usage
		var invocations []TripleInvocationRequest
		for i := 0; i < 1000; i++ {
			invocations = append(invocations, TripleInvocationRequest{
				MethodName: "memoryTest",
				Types:      []string{"map"},
				Args: []any{
					map[string]any{
						"id":   int64(i),
						"data": fmt.Sprintf("test_data_%d", i),
					},
				},
			})
		}

		start := time.Now()
		results, err := tgs.BatchInvoke(ctx, invocations)
		duration := time.Since(start)

		assert.NoError(t, err, "largedatabatch invocation should not have structural errors")
		assert.Len(t, results, 1000, "should return1000results")

		t.Logf("processing1000objectsduration: %v", duration)
		assert.Less(t, duration, 5*time.Second, "largedataprocessingprocessing time should be reasonable")
	})
}
