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

package getty

import (
	"testing"
)

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// ============================================
// filterContext Tests
// ============================================
func TestFilterContext(t *testing.T) {
	tests := []struct {
		name        string
		attachments map[string]any
		expected    map[string]string
	}{
		{
			name:        "empty attachments",
			attachments: map[string]any{},
			expected:    map[string]string{},
		},
		{
			name: "all string values",
			attachments: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "mixed types - only strings extracted",
			attachments: map[string]any{
				"string_key": "string_value",
				"int_key":    123,
				"bool_key":   true,
				"float_key":  3.14,
			},
			expected: map[string]string{
				"string_key": "string_value",
			},
		},
		{
			name: "nil value",
			attachments: map[string]any{
				"key1":    "value1",
				"nil_key": nil,
			},
			expected: map[string]string{
				"key1": "value1",
			},
		},
		{
			name: "empty string value",
			attachments: map[string]any{
				"empty": "",
				"key":   "value",
			},
			expected: map[string]string{
				"empty": "",
				"key":   "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterContext(tt.attachments)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================
// fillTraceAttachments Tests
// ============================================

func TestFillTraceAttachments(t *testing.T) {
	tests := []struct {
		name             string
		attachments      map[string]any
		traceAttachments map[string]string
		expectedKeys     []string
	}{
		{
			name:        "fill empty attachments",
			attachments: map[string]any{},
			traceAttachments: map[string]string{
				"trace_id": "abc123",
				"span_id":  "def456",
			},
			expectedKeys: []string{"trace_id", "span_id"},
		},
		{
			name: "fill existing attachments",
			attachments: map[string]any{
				"existing": "value",
			},
			traceAttachments: map[string]string{
				"trace_id": "abc123",
			},
			expectedKeys: []string{"existing", "trace_id"},
		},
		{
			name: "override existing key",
			attachments: map[string]any{
				"trace_id": "old_value",
			},
			traceAttachments: map[string]string{
				"trace_id": "new_value",
			},
			expectedKeys: []string{"trace_id"},
		},
		{
			name:             "empty trace attachments",
			attachments:      map[string]any{"key": "value"},
			traceAttachments: map[string]string{},
			expectedKeys:     []string{"key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fillTraceAttachments(tt.attachments, tt.traceAttachments)

			for _, key := range tt.expectedKeys {
				_, exists := tt.attachments[key]
				assert.True(t, exists, "key %s should exist", key)
			}

			// Verify trace attachments are correctly filled
			for k, v := range tt.traceAttachments {
				assert.Equal(t, v, tt.attachments[k])
			}
		})
	}
}

// ============================================
// injectTraceCtx Tests
// ============================================

func TestInjectTraceCtx(t *testing.T) {
	// Setup mock tracer
	tracer := mocktracer.New()
	opentracing.SetGlobalTracer(tracer)

	t.Run("inject trace context successfully", func(t *testing.T) {
		attachments := map[string]any{
			"key1": "value1",
		}
		inv := invocation.NewRPCInvocation("TestMethod", []any{"arg1"}, attachments)

		span := tracer.StartSpan("test-operation")
		err := injectTraceCtx(span, inv)
		span.Finish()

		assert.Nil(t, err)
		// Verify trace headers were injected
		invAttachments := inv.Attachments()
		assert.NotNil(t, invAttachments)
	})

	t.Run("inject with empty attachments", func(t *testing.T) {
		attachments := map[string]any{}
		inv := invocation.NewRPCInvocation("TestMethod", []any{}, attachments)

		span := tracer.StartSpan("test-operation")
		err := injectTraceCtx(span, inv)
		span.Finish()

		assert.Nil(t, err)
	})

	t.Run("inject with mixed type attachments", func(t *testing.T) {
		attachments := map[string]any{
			"string_key": "string_value",
			"int_key":    42,
			"bool_key":   true,
		}
		inv := invocation.NewRPCInvocation("TestMethod", []any{}, attachments)

		span := tracer.StartSpan("test-operation")
		err := injectTraceCtx(span, inv)
		span.Finish()

		assert.Nil(t, err)
		// Original non-string values should still exist
		assert.Equal(t, 42, inv.Attachments()["int_key"])
		assert.Equal(t, true, inv.Attachments()["bool_key"])
	})
}

// ============================================
// Integration Tests
// ============================================

func TestOpenTracingIntegration(t *testing.T) {
	tracer := mocktracer.New()
	opentracing.SetGlobalTracer(tracer)

	t.Run("full trace injection and extraction flow", func(t *testing.T) {
		// Create invocation with attachments
		attachments := map[string]any{
			"service": "test-service",
			"version": "1.0.0",
		}
		inv := invocation.NewRPCInvocation("GetUser", []any{"userId"}, attachments)

		// Start a span and inject context
		span := tracer.StartSpan("client-call")
		err := injectTraceCtx(span, inv)
		assert.Nil(t, err)

		// Verify trace context was injected
		traceAttachments := filterContext(inv.Attachments())
		assert.NotEmpty(t, traceAttachments)

		span.Finish()

		// Verify original attachments are preserved
		assert.Equal(t, "test-service", inv.Attachments()["service"])
		assert.Equal(t, "1.0.0", inv.Attachments()["version"])
	})

	t.Run("multiple spans injection", func(t *testing.T) {
		// Create a fresh tracer for this test
		freshTracer := mocktracer.New()
		opentracing.SetGlobalTracer(freshTracer)

		for i := 0; i < 5; i++ {
			attachments := map[string]any{"iteration": i}
			inv := invocation.NewRPCInvocation("Method", []any{}, attachments)

			span := freshTracer.StartSpan("operation")
			err := injectTraceCtx(span, inv)
			assert.Nil(t, err)
			span.Finish()
		}

		// Verify all spans were recorded
		assert.Equal(t, 5, len(freshTracer.FinishedSpans()))
	})
}

// ============================================
// Edge Cases Tests
// ============================================

func TestOpenTracingEdgeCases(t *testing.T) {
	tracer := mocktracer.New()
	opentracing.SetGlobalTracer(tracer)

	t.Run("filter context with special characters", func(t *testing.T) {
		attachments := map[string]any{
			"key-with-dash":       "value1",
			"key_with_underscore": "value2",
			"key.with.dot":        "value3",
			"key:with:colon":      "value4",
		}
		result := filterContext(attachments)
		assert.Equal(t, 4, len(result))
	})

	t.Run("filter context with unicode", func(t *testing.T) {
		attachments := map[string]any{
			"chinese": "中文",
			"emoji":   "🎉",
			"mixed":   "hello世界",
		}
		result := filterContext(attachments)
		assert.Equal(t, "中文", result["chinese"])
		assert.Equal(t, "🎉", result["emoji"])
		assert.Equal(t, "hello世界", result["mixed"])
	})

	t.Run("fill attachments preserves type", func(t *testing.T) {
		attachments := map[string]any{
			"int_val":   100,
			"float_val": 3.14,
			"bool_val":  false,
		}
		traceAttachments := map[string]string{
			"trace": "value",
		}
		fillTraceAttachments(attachments, traceAttachments)

		// Original types should be preserved
		assert.Equal(t, 100, attachments["int_val"])
		assert.Equal(t, 3.14, attachments["float_val"])
		assert.Equal(t, false, attachments["bool_val"])
		// New trace value should be string
		assert.Equal(t, "value", attachments["trace"])
	})
}

// ============================================
// Concurrent Access Tests
// ============================================

func TestFilterContextConcurrent(t *testing.T) {
	attachments := map[string]any{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			result := filterContext(attachments)
			assert.Equal(t, 3, len(result))
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkFilterContext(b *testing.B) {
	attachments := map[string]any{
		"key1": "value1",
		"key2": "value2",
		"key3": 123,
		"key4": true,
		"key5": "value5",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filterContext(attachments)
	}
}

func BenchmarkFillTraceAttachments(b *testing.B) {
	traceAttachments := map[string]string{
		"trace_id": "abc123",
		"span_id":  "def456",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attachments := map[string]any{"existing": "value"}
		fillTraceAttachments(attachments, traceAttachments)
	}
}
