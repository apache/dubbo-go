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

package logrus

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

import (
	dubbogoLogger "github.com/dubbogo/gost/log/logger"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestLogrusCtxLogger_CtxInfof_WithTrace(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	lg := logrus.New()
	lg.SetOutput(&buf)
	lg.SetFormatter(&logrus.JSONFormatter{})
	lg.SetLevel(logrus.InfoLevel)

	baseLogger := &dubbogoLogger.DubboLogger{Logger: lg}
	ctxLogger := NewLogrusCtxLogger(baseLogger, false)

	// Create context with trace using SDK tracer
	tp := trace.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Log with context
	ctxLogger.CtxInfof(ctx, "test message")

	// Parse log output
	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Verify log contains message
	assert.Equal(t, "test message", logEntry["msg"])
	assert.Equal(t, "info", logEntry["level"])

	// Verify trace fields are present
	assert.Contains(t, logEntry, "trace_id")
	assert.Contains(t, logEntry, "span_id")
	assert.Contains(t, logEntry, "trace_flags")
}

func TestLogrusCtxLogger_CtxInfof_WithoutTrace(t *testing.T) {
	var buf bytes.Buffer
	lg := logrus.New()
	lg.SetOutput(&buf)
	lg.SetFormatter(&logrus.JSONFormatter{})
	lg.SetLevel(logrus.InfoLevel)

	baseLogger := &dubbogoLogger.DubboLogger{Logger: lg}
	ctxLogger := NewLogrusCtxLogger(baseLogger, false)

	// Log without trace context
	ctx := context.Background()
	ctxLogger.CtxInfof(ctx, "test message")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Verify log contains message but no trace fields
	assert.Equal(t, "test message", logEntry["msg"])
	assert.NotContains(t, logEntry, "trace_id")
	assert.NotContains(t, logEntry, "span_id")
}

func TestLogrusCtxLogger_AllLevels(t *testing.T) {
	var buf bytes.Buffer
	lg := logrus.New()
	lg.SetOutput(&buf)
	lg.SetFormatter(&logrus.JSONFormatter{})
	lg.SetLevel(logrus.DebugLevel)

	baseLogger := &dubbogoLogger.DubboLogger{Logger: lg}
	ctxLogger := NewLogrusCtxLogger(baseLogger, false)

	tp := trace.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Test all log levels
	ctxLogger.CtxDebugf(ctx, "debug message")
	ctxLogger.CtxInfof(ctx, "info message")
	ctxLogger.CtxWarnf(ctx, "warn message")
	ctxLogger.CtxErrorf(ctx, "error message")

	// Verify all messages were logged
	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestLogrusCtxLogger_NonFormattedMethods(t *testing.T) {
	var buf bytes.Buffer
	lg := logrus.New()
	lg.SetOutput(&buf)
	lg.SetFormatter(&logrus.JSONFormatter{})
	lg.SetLevel(logrus.DebugLevel)

	baseLogger := &dubbogoLogger.DubboLogger{Logger: lg}
	ctxLogger := NewLogrusCtxLogger(baseLogger, false)

	tp := trace.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Test non-formatted methods
	ctxLogger.CtxDebug(ctx, "debug")
	ctxLogger.CtxInfo(ctx, "info")
	ctxLogger.CtxWarn(ctx, "warn")
	ctxLogger.CtxError(ctx, "error")

	output := buf.String()
	assert.Contains(t, output, "debug")
	assert.Contains(t, output, "info")
	assert.Contains(t, output, "warn")
	assert.Contains(t, output, "error")
}

func TestLogrusCtxLogger_RecordErrorToSpan_Errorf(t *testing.T) {
	var buf bytes.Buffer
	lg := logrus.New()
	lg.SetOutput(&buf)
	lg.SetFormatter(&logrus.JSONFormatter{})
	lg.SetLevel(logrus.ErrorLevel)

	baseLogger := &dubbogoLogger.DubboLogger{Logger: lg}
	// Enable recordErrorToSpan
	ctxLogger := NewLogrusCtxLogger(baseLogger, true)

	// Create span recorder to capture span events
	spanRecorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")

	// Log error with context
	ctxLogger.CtxErrorf(ctx, "test error: %s", "something went wrong")
	span.End()

	// Verify error was logged
	output := buf.String()
	assert.Contains(t, output, "test error: something went wrong")

	// Verify span recorded the error
	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	recordedSpan := spans[0]

	// Check span status
	assert.Equal(t, codes.Error, recordedSpan.Status().Code)
	assert.Equal(t, "test error: something went wrong", recordedSpan.Status().Description)

	// Check span events (error recording)
	events := recordedSpan.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "exception", events[0].Name)
}

func TestLogrusCtxLogger_RecordErrorToSpan_Error(t *testing.T) {
	var buf bytes.Buffer
	lg := logrus.New()
	lg.SetOutput(&buf)
	lg.SetFormatter(&logrus.JSONFormatter{})
	lg.SetLevel(logrus.ErrorLevel)

	baseLogger := &dubbogoLogger.DubboLogger{Logger: lg}
	// Enable recordErrorToSpan
	ctxLogger := NewLogrusCtxLogger(baseLogger, true)

	// Create span recorder to capture span events
	spanRecorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")

	// Log error with context
	ctxLogger.CtxError(ctx, "test error message")
	span.End()

	// Verify error was logged
	output := buf.String()
	assert.Contains(t, output, "test error message")

	// Verify span recorded the error
	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	recordedSpan := spans[0]

	// Check span status
	assert.Equal(t, codes.Error, recordedSpan.Status().Code)
	assert.Equal(t, "test error message", recordedSpan.Status().Description)

	// Check span events (error recording)
	events := recordedSpan.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "exception", events[0].Name)
}

func TestLogrusCtxLogger_RecordErrorToSpan_Disabled(t *testing.T) {
	var buf bytes.Buffer
	lg := logrus.New()
	lg.SetOutput(&buf)
	lg.SetFormatter(&logrus.JSONFormatter{})
	lg.SetLevel(logrus.ErrorLevel)

	baseLogger := &dubbogoLogger.DubboLogger{Logger: lg}
	// Disable recordErrorToSpan
	ctxLogger := NewLogrusCtxLogger(baseLogger, false)

	// Create span recorder to capture span events
	spanRecorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")

	// Log error with context
	ctxLogger.CtxErrorf(ctx, "test error")
	span.End()

	// Verify error was logged
	output := buf.String()
	assert.Contains(t, output, "test error")

	// Verify span did NOT record the error
	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	recordedSpan := spans[0]

	// Check span status should be Unset (not Error)
	assert.NotEqual(t, codes.Error, recordedSpan.Status().Code)

	// Check span events should be empty
	events := recordedSpan.Events()
	assert.Empty(t, events)
}
