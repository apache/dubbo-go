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
	"net/url"
	"strings"
	"testing"
)

import (
	dubbogoLogger "github.com/dubbogo/gost/log/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oteltrace "go.opentelemetry.io/otel/trace"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/logger"
)

var _ logger.OpsLogger = (*ZapCtxLogger)(nil)

const (
	zapTestTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	zapTestSpanID  = "00f067aa0ba902b7"
)

func TestZapCtxLogger_DynamicLevelAndTraceCorrelation(t *testing.T) {
	ctxLogger, buf, flush := newZapCtxLoggerWithBuffer(t, zapcore.InfoLevel)
	traceCtx, wantTraceID, wantSpanID, wantFlags := mustZapTraceContext(t)
	plainCtx := t.Context()

	samples := []struct {
		name   string
		level  zapcore.Level
		traced bool
		emit   func(string)
	}{
		{"debug-plain", zapcore.DebugLevel, false, func(msg string) { ctxLogger.Debug(msg) }},
		{"debug-fmt", zapcore.DebugLevel, false, func(msg string) { ctxLogger.Debugf("%s", msg) }},
		{"debug-ctx", zapcore.DebugLevel, true, func(msg string) { ctxLogger.CtxDebug(traceCtx, msg) }},
		{"debug-ctx-fmt", zapcore.DebugLevel, true, func(msg string) { ctxLogger.CtxDebugf(traceCtx, "%s", msg) }},
		{"debug-no-trace", zapcore.DebugLevel, false, func(msg string) { ctxLogger.CtxDebug(plainCtx, msg) }},
		{"debug-no-trace-fmt", zapcore.DebugLevel, false, func(msg string) { ctxLogger.CtxDebugf(plainCtx, "%s", msg) }},
		{"info-plain", zapcore.InfoLevel, false, func(msg string) { ctxLogger.Info(msg) }},
		{"info-fmt", zapcore.InfoLevel, false, func(msg string) { ctxLogger.Infof("%s", msg) }},
		{"info-ctx", zapcore.InfoLevel, true, func(msg string) { ctxLogger.CtxInfo(traceCtx, msg) }},
		{"info-ctx-fmt", zapcore.InfoLevel, true, func(msg string) { ctxLogger.CtxInfof(traceCtx, "%s", msg) }},
		{"info-no-trace", zapcore.InfoLevel, false, func(msg string) { ctxLogger.CtxInfo(plainCtx, msg) }},
		{"info-no-trace-fmt", zapcore.InfoLevel, false, func(msg string) { ctxLogger.CtxInfof(plainCtx, "%s", msg) }},
		{"warn-plain", zapcore.WarnLevel, false, func(msg string) { ctxLogger.Warn(msg) }},
		{"warn-fmt", zapcore.WarnLevel, false, func(msg string) { ctxLogger.Warnf("%s", msg) }},
		{"warn-ctx", zapcore.WarnLevel, true, func(msg string) { ctxLogger.CtxWarn(traceCtx, msg) }},
		{"warn-ctx-fmt", zapcore.WarnLevel, true, func(msg string) { ctxLogger.CtxWarnf(traceCtx, "%s", msg) }},
		{"warn-no-trace", zapcore.WarnLevel, false, func(msg string) { ctxLogger.CtxWarn(plainCtx, msg) }},
		{"warn-no-trace-fmt", zapcore.WarnLevel, false, func(msg string) { ctxLogger.CtxWarnf(plainCtx, "%s", msg) }},
	}

	// Keep the transitions ordered on one logger instance.
	for _, step := range []struct {
		name  string
		level zapcore.Level
	}{
		{"info", zapcore.InfoLevel},
		{"debug", zapcore.DebugLevel},
		{"warn", zapcore.WarnLevel},
	} {
		t.Run(step.name, func(t *testing.T) {
			require.True(t, ctxLogger.SetLoggerLevel(step.name))
			assert.Equal(t, step.level, ctxLogger.DynamicLevel.Level())
			for _, sample := range samples {
				t.Run(sample.name, func(t *testing.T) {
					buf.Reset()
					msg := sample.name + "-" + step.name
					sample.emit(msg)
					flush()
					out := buf.String()
					if sample.level < step.level {
						assert.Empty(t, out, "suppressed log %q", msg)
						return
					}
					if sample.traced {
						assertZapTraceFields(t, out, msg, wantTraceID, wantSpanID, wantFlags)
					} else {
						assertZapNoTraceFields(t, out, msg)
					}
				})
			}
		})
	}

	require.False(t, ctxLogger.SetLoggerLevel("not-a-level"))
	buf.Reset()
	ctxLogger.Info("info-after-invalid")
	ctxLogger.Debug("debug-after-invalid")
	ctxLogger.Warn("warn-after-invalid")
	flush()
	invalidOut := buf.String()
	assert.NotContains(t, invalidOut, "info-after-invalid")
	assert.NotContains(t, invalidOut, "debug-after-invalid")
	assert.Contains(t, invalidOut, "warn-after-invalid")
}

func TestInstantiateZap_TraceIntegration_SetLoggerLevel(t *testing.T) {
	u := &common.URL{}
	u.ReplaceParams(url.Values{
		constant.LoggerLevelKey:        []string{"info"},
		constant.LoggerAppenderKey:     []string{"console"},
		constant.LoggerFormatKey:       []string{"json"},
		constant.LoggerTraceEnabledKey: []string{"true"},
	})
	lg, err := instantiate(u)
	require.NoError(t, err)

	ctxLogger, ok := lg.(*ZapCtxLogger)
	require.True(t, ok)
	require.True(t, ctxLogger.SetLoggerLevel("debug"))
	assert.Equal(t, zapcore.DebugLevel, ctxLogger.DynamicLevel.Level())
	require.False(t, ctxLogger.SetLoggerLevel("not-a-level"))
	assert.Equal(t, zapcore.DebugLevel, ctxLogger.DynamicLevel.Level())
}

func newZapCtxLoggerWithBuffer(t *testing.T, level zapcore.Level) (*ZapCtxLogger, *bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	atomicLevel := zap.NewAtomicLevelAt(level)
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
	})
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), atomicLevel)
	sugar := zap.New(core).Sugar()
	base := &dubbogoLogger.DubboLogger{Logger: sugar, DynamicLevel: atomicLevel}
	return NewZapCtxLogger(base, false), &buf, func() { _ = sugar.Sync() }
}

func mustZapTraceContext(t *testing.T) (context.Context, string, string, string) {
	t.Helper()
	traceID, err := oteltrace.TraceIDFromHex(zapTestTraceID)
	require.NoError(t, err)
	spanID, err := oteltrace.SpanIDFromHex(zapTestSpanID)
	require.NoError(t, err)
	sc := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})
	require.True(t, sc.IsValid())
	return oteltrace.ContextWithSpanContext(t.Context(), sc), traceID.String(), spanID.String(), sc.TraceFlags().String()
}

func assertZapTraceFields(t *testing.T, raw, msg, traceID, spanID, flags string) {
	t.Helper()
	entry := findZapLog(t, raw, msg)
	require.NotNil(t, entry, "missing log %q in %s", msg, raw)
	assert.Equal(t, traceID, entry["trace_id"])
	assert.Equal(t, spanID, entry["span_id"])
	assert.Equal(t, flags, entry["trace_flags"])
}

func assertZapNoTraceFields(t *testing.T, raw, msg string) {
	t.Helper()
	entry := findZapLog(t, raw, msg)
	require.NotNil(t, entry, "missing log %q in %s", msg, raw)
	assert.NotContains(t, entry, "trace_id")
	assert.NotContains(t, entry, "span_id")
	assert.NotContains(t, entry, "trace_flags")
}

func findZapLog(t *testing.T, raw, msg string) map[string]any {
	t.Helper()
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		if entry["msg"] == msg {
			return entry
		}
	}
	return nil
}
