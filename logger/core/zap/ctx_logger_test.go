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

package zap

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

import (
	dubbogoLogger "github.com/dubbogo/gost/log/logger"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/sdk/trace"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestZapCtxLogger_CtxInfof_WithTrace(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	})
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.InfoLevel)
	zapLogger := zap.New(core).Sugar()

	baseLogger := &dubbogoLogger.DubboLogger{Logger: zapLogger}
	ctxLogger := NewZapCtxLogger(baseLogger, false)

	// Create context with trace using SDK tracer
	tp := trace.NewTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Log with context
	ctxLogger.CtxInfof(ctx, "test message")

	// Parse log output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	// Verify log contains message
	assert.Equal(t, "test message", logEntry["msg"])
	assert.Equal(t, "info", logEntry["level"])

	// Verify trace fields are present
	assert.Contains(t, logEntry, "trace_id")
	assert.Contains(t, logEntry, "span_id")
	assert.Contains(t, logEntry, "trace_flags")
}

func TestZapCtxLogger_CtxInfof_WithoutTrace(t *testing.T) {
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	})
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.InfoLevel)
	zapLogger := zap.New(core).Sugar()

	baseLogger := &dubbogoLogger.DubboLogger{Logger: zapLogger}
	ctxLogger := NewZapCtxLogger(baseLogger, false)

	// Log without trace context
	ctx := context.Background()
	ctxLogger.CtxInfof(ctx, "test message")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	// Verify log contains message but no trace fields
	assert.Equal(t, "test message", logEntry["msg"])
	assert.NotContains(t, logEntry, "trace_id")
	assert.NotContains(t, logEntry, "span_id")
}

func TestZapCtxLogger_AllLevels(t *testing.T) {
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	})
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	zapLogger := zap.New(core).Sugar()

	baseLogger := &dubbogoLogger.DubboLogger{Logger: zapLogger}
	ctxLogger := NewZapCtxLogger(baseLogger, false)

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

func TestZapCtxLogger_NonFormattedMethods(t *testing.T) {
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	})
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	zapLogger := zap.New(core).Sugar()

	baseLogger := &dubbogoLogger.DubboLogger{Logger: zapLogger}
	ctxLogger := NewZapCtxLogger(baseLogger, false)

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
