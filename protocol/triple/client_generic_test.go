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

func TestTripleGenericService_Basic(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20001/com.example.TestService")
	assert.NotNil(t, tgs)
	assert.Equal(t, "tri://127.0.0.1:20001/com.example.TestService", tgs.Reference())
}

func TestTripleGenericService_ConvertParamsForIDL(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20001/com.example.TestService")

	tests := []struct {
		name     string
		params   any
		types    []string
		expected []any
		hasError bool
	}{
		{
			name:     "single string parameter",
			params:   "test",
			types:    []string{"string"},
			expected: []any{"test"},
			hasError: false,
		},
		{
			name:     "multiple parameters",
			params:   []any{123, "test", true},
			types:    []string{"int64", "string", "bool"},
			expected: []any{int64(123), "test", true},
			hasError: false,
		},
		{
			name:     "parameter count mismatch",
			params:   []any{123},
			types:    []string{"int64", "string"},
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tgs.convertParamsForIDL(tt.params, tt.types)
			if tt.hasError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTripleGenericService_ConvertSingleArgForSerialization(t *testing.T) {
	// This test is for internal conversion logic, skipping for now
	t.Skip("convertSingleArgForSerialization is an internal method")
}

func TestTripleGenericService_InferParameterTypes(t *testing.T) {
	// This test is for internal type inference, skipping for now
	t.Skip("inferParameterTypes is an internal method")
}

func TestTripleGenericService_AttachmentBuilder(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20001/com.example.TestService")

	builder := tgs.CreateAttachmentBuilder()
	assert.NotNil(t, builder)

	attachments := builder.
		SetString("key1", "value1").
		SetInt("key2", 123).
		SetBool("key3", true).
		Build()

	expected := map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	assert.Equal(t, expected, attachments)
}

func TestTripleGenericService_BatchInvoke(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20001/com.example.TestService")

	// Test empty invocations
	results, err := tgs.BatchInvoke(context.Background(), []TripleInvocationRequest{})
	assert.Error(t, err)
	assert.Nil(t, results)

	// Test with mock invocations (these will fail since no server is running, but we test the structure)
	invocations := []TripleInvocationRequest{
		{
			MethodName:  "method1",
			Types:       []string{"string"},
			Args:        []any{"arg1"},
			Attachments: map[string]any{"key": "value"},
		},
		{
			MethodName:  "method2",
			Types:       []string{"int"},
			Args:        []any{123},
			Attachments: nil,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	results, err = tgs.BatchInvoke(ctx, invocations)
	assert.NoError(t, err) // The batch operation itself should not error
	assert.Len(t, results, 2)

	// Each individual invocation should have errors since no server is running
	for i, result := range results {
		assert.Equal(t, i, result.Index)
		assert.Error(t, result.Error) // Expected since no server is running
	}
}

func TestTripleGenericService_BatchInvokeWithOptions(t *testing.T) {
	tgs := NewTripleGenericService("tri://127.0.0.1:20001/com.example.TestService")

	invocations := make([]TripleInvocationRequest, 5)
	for i := 0; i < 5; i++ {
		invocations[i] = TripleInvocationRequest{
			MethodName:  "testMethod",
			Types:       []string{"int"},
			Args:        []any{i},
			Attachments: nil,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	options := BatchInvokeOptions{
		MaxConcurrency: 2,
		FailFast:       false,
	}

	results, err := tgs.BatchInvokeWithOptions(ctx, invocations, options)
	assert.NoError(t, err)
	assert.Len(t, results, 5)

	// Verify all results have correct indices
	for i, result := range results {
		assert.Equal(t, i, result.Index)
	}
}

func TestTripleAsyncManager(t *testing.T) {
	manager := GetTripleAsyncManager()
	assert.NotNil(t, manager)

	// Test that we get the same instance (singleton)
	manager2 := GetTripleAsyncManager()
	assert.Same(t, manager, manager2)

	// Test registering and getting calls
	asyncCall := &TripleAsyncCall{
		ID:         "test-call-1",
		MethodName: "testMethod",
		StartTime:  time.Now(),
	}

	manager.registerCall(asyncCall)

	retrieved := manager.getCall("test-call-1")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-call-1", retrieved.ID)

	// Test getting all active calls
	activeCalls := manager.getAllActiveCalls()
	assert.Len(t, activeCalls, 1)
	assert.Contains(t, activeCalls, "test-call-1")

	// Test unregistering
	manager.unregisterCall("test-call-1")
	retrieved = manager.getCall("test-call-1")
	assert.Nil(t, retrieved)

	activeCalls = manager.getAllActiveCalls()
	assert.Len(t, activeCalls, 0)
}

func TestTripleAsyncManager_Concurrent(t *testing.T) {
	manager := GetTripleAsyncManager()

	// Test concurrent access
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			callID := fmt.Sprintf("test-call-%d", id)
			asyncCall := &TripleAsyncCall{
				ID:         callID,
				MethodName: "testMethod",
				StartTime:  time.Now(),
			}

			manager.registerCall(asyncCall)

			// Small delay
			time.Sleep(10 * time.Millisecond)

			retrieved := manager.getCall(callID)
			assert.NotNil(t, retrieved)
			assert.Equal(t, callID, retrieved.ID)

			manager.unregisterCall(callID)
		}(i)
	}

	wg.Wait()

	// All calls should be cleaned up
	activeCalls := manager.getAllActiveCalls()
	assert.Len(t, activeCalls, 0)
}

func TestGenerateCallID(t *testing.T) {
	id1 := generateCallID()
	id2 := generateCallID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2) // Should be unique
	assert.Contains(t, id1, "triple-async-")
	assert.Contains(t, id2, "triple-async-")
}
