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

package logger

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestExtractTraceFields_WithValidSpan(t *testing.T) {
	// Create a real tracer with SDK to generate valid trace IDs
	tp := trace.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Extract trace fields
	fields := ExtractTraceFields(ctx)

	// Verify fields are extracted
	assert.NotEmpty(t, fields.TraceID)
	assert.NotEmpty(t, fields.SpanID)
	assert.NotEmpty(t, fields.TraceFlags)
}

func TestExtractTraceFields_WithNilContext(t *testing.T) {
	fields := ExtractTraceFields(nil)

	assert.Empty(t, fields.TraceID)
	assert.Empty(t, fields.SpanID)
	assert.Empty(t, fields.TraceFlags)
}

func TestExtractTraceFields_WithoutSpan(t *testing.T) {
	ctx := context.Background()
	fields := ExtractTraceFields(ctx)

	assert.Empty(t, fields.TraceID)
	assert.Empty(t, fields.SpanID)
	assert.Empty(t, fields.TraceFlags)
}

func TestExtractTraceFields_WithInvalidSpan(t *testing.T) {
	// Create context with invalid span context
	ctx := oteltrace.ContextWithSpan(context.Background(), noop.Span{})
	fields := ExtractTraceFields(ctx)

	// Invalid span should return empty fields
	assert.Empty(t, fields.TraceID)
	assert.Empty(t, fields.SpanID)
	assert.Empty(t, fields.TraceFlags)
}

func TestHasTrace_WithValidSpan(t *testing.T) {
	tp := trace.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	assert.True(t, HasTrace(ctx))
}

func TestHasTrace_WithNilContext(t *testing.T) {
	assert.False(t, HasTrace(nil))
}

func TestHasTrace_WithoutSpan(t *testing.T) {
	ctx := context.Background()
	assert.False(t, HasTrace(ctx))
}

func TestHasTrace_WithInvalidSpan(t *testing.T) {
	ctx := oteltrace.ContextWithSpan(context.Background(), noop.Span{})
	assert.False(t, HasTrace(ctx))
}
