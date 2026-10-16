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
	"net/url"
	"strings"
	"testing"
)

import (
	dubbogoLogger "github.com/dubbogo/gost/log/logger"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oteltrace "go.opentelemetry.io/otel/trace"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/logger"
)

var _ logger.OpsLogger = (*LogrusCtxLogger)(nil)

const (
	logrusTestTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	logrusTestSpanID  = "00f067aa0ba902b7"
)

func TestLogrusCtxLogger_DynamicLevelAndTraceCorrelation(t *testing.T) {
	ctxLogger, buf, inner := newLogrusCtxLoggerWithBuffer(logrus.InfoLevel)
	traceCtx, wantTraceID, wantSpanID, wantFlags := mustLogrusTraceContext(t)
	plainCtx := t.Context()

	samples := []struct {
		name   string
		level  logrus.Level
		traced bool
		emit   func(string)
	}{
		{"debug-plain", logrus.DebugLevel, false, func(msg string) { ctxLogger.Debug(msg) }},
		{"debug-fmt", logrus.DebugLevel, false, func(msg string) { ctxLogger.Debugf("%s", msg) }},
		{"debug-ctx", logrus.DebugLevel, true, func(msg string) { ctxLogger.CtxDebug(traceCtx, msg) }},
		{"debug-ctx-fmt", logrus.DebugLevel, true, func(msg string) { ctxLogger.CtxDebugf(traceCtx, "%s", msg) }},
		{"debug-no-trace", logrus.DebugLevel, false, func(msg string) { ctxLogger.CtxDebug(plainCtx, msg) }},
		{"debug-no-trace-fmt", logrus.DebugLevel, false, func(msg string) { ctxLogger.CtxDebugf(plainCtx, "%s", msg) }},
		{"info-plain", logrus.InfoLevel, false, func(msg string) { ctxLogger.Info(msg) }},
		{"info-fmt", logrus.InfoLevel, false, func(msg string) { ctxLogger.Infof("%s", msg) }},
		{"info-ctx", logrus.InfoLevel, true, func(msg string) { ctxLogger.CtxInfo(traceCtx, msg) }},
		{"info-ctx-fmt", logrus.InfoLevel, true, func(msg string) { ctxLogger.CtxInfof(traceCtx, "%s", msg) }},
		{"info-no-trace", logrus.InfoLevel, false, func(msg string) { ctxLogger.CtxInfo(plainCtx, msg) }},
		{"info-no-trace-fmt", logrus.InfoLevel, false, func(msg string) { ctxLogger.CtxInfof(plainCtx, "%s", msg) }},
		{"warn-plain", logrus.WarnLevel, false, func(msg string) { ctxLogger.Warn(msg) }},
		{"warn-fmt", logrus.WarnLevel, false, func(msg string) { ctxLogger.Warnf("%s", msg) }},
		{"warn-ctx", logrus.WarnLevel, true, func(msg string) { ctxLogger.CtxWarn(traceCtx, msg) }},
		{"warn-ctx-fmt", logrus.WarnLevel, true, func(msg string) { ctxLogger.CtxWarnf(traceCtx, "%s", msg) }},
		{"warn-no-trace", logrus.WarnLevel, false, func(msg string) { ctxLogger.CtxWarn(plainCtx, msg) }},
		{"warn-no-trace-fmt", logrus.WarnLevel, false, func(msg string) { ctxLogger.CtxWarnf(plainCtx, "%s", msg) }},
	}

	// Keep the transitions ordered on one logger instance.
	for _, step := range []struct {
		name  string
		level logrus.Level
	}{
		{"info", logrus.InfoLevel},
		{"debug", logrus.DebugLevel},
		{"warn", logrus.WarnLevel},
	} {
		t.Run(step.name, func(t *testing.T) {
			require.True(t, ctxLogger.SetLoggerLevel(step.name))
			assert.Equal(t, step.level, inner.GetLevel())
			for _, sample := range samples {
				t.Run(sample.name, func(t *testing.T) {
					buf.Reset()
					msg := sample.name + "-" + step.name
					sample.emit(msg)
					out := buf.String()
					if sample.level > step.level {
						assert.Empty(t, out, "suppressed log %q", msg)
						return
					}
					if sample.traced {
						assertLogrusTraceFields(t, out, msg, wantTraceID, wantSpanID, wantFlags)
					} else {
						assertLogrusNoTraceFields(t, out, msg)
					}
				})
			}
		})
	}

	require.False(t, ctxLogger.SetLoggerLevel("not-a-level"))
	assert.Equal(t, logrus.WarnLevel, inner.GetLevel())
	buf.Reset()
	ctxLogger.Info("info-after-invalid")
	ctxLogger.Debug("debug-after-invalid")
	ctxLogger.Warn("warn-after-invalid")
	invalidOut := buf.String()
	assert.NotContains(t, invalidOut, "info-after-invalid")
	assert.NotContains(t, invalidOut, "debug-after-invalid")
	assert.Contains(t, invalidOut, "warn-after-invalid")
}

func TestInstantiateLogrus_TraceIntegration_SetLoggerLevel(t *testing.T) {
	u := &common.URL{}
	u.ReplaceParams(url.Values{
		constant.LoggerLevelKey:        []string{"info"},
		constant.LoggerAppenderKey:     []string{"console"},
		constant.LoggerFormatKey:       []string{"json"},
		constant.LoggerTraceEnabledKey: []string{"true"},
	})
	lg, err := instantiate(u)
	require.NoError(t, err)

	ctxLogger, ok := lg.(*LogrusCtxLogger)
	require.True(t, ok)
	inner, ok := ctxLogger.Logger.(*logrus.Logger)
	require.True(t, ok)

	require.True(t, ctxLogger.SetLoggerLevel("debug"))
	assert.Equal(t, logrus.DebugLevel, inner.GetLevel())
	require.False(t, ctxLogger.SetLoggerLevel("not-a-level"))
	assert.Equal(t, logrus.DebugLevel, inner.GetLevel())
}

func newLogrusCtxLoggerWithBuffer(level logrus.Level) (*LogrusCtxLogger, *bytes.Buffer, *logrus.Logger) {
	var buf bytes.Buffer
	lg := logrus.New()
	lg.SetOutput(&buf)
	lg.SetFormatter(&logrus.JSONFormatter{})
	lg.SetLevel(level)
	base := &dubbogoLogger.DubboLogger{Logger: lg}
	return NewLogrusCtxLogger(base, false), &buf, lg
}

func mustLogrusTraceContext(t *testing.T) (context.Context, string, string, string) {
	t.Helper()
	traceID, err := oteltrace.TraceIDFromHex(logrusTestTraceID)
	require.NoError(t, err)
	spanID, err := oteltrace.SpanIDFromHex(logrusTestSpanID)
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

func assertLogrusTraceFields(t *testing.T, raw, msg, traceID, spanID, flags string) {
	t.Helper()
	entry := findLogrusLog(t, raw, msg)
	require.NotNil(t, entry, "missing log %q in %s", msg, raw)
	assert.Equal(t, traceID, entry["trace_id"])
	assert.Equal(t, spanID, entry["span_id"])
	assert.Equal(t, flags, entry["trace_flags"])
}

func assertLogrusNoTraceFields(t *testing.T, raw, msg string) {
	t.Helper()
	entry := findLogrusLog(t, raw, msg)
	require.NotNil(t, entry, "missing log %q in %s", msg, raw)
	assert.NotContains(t, entry, "trace_id")
	assert.NotContains(t, entry, "span_id")
	assert.NotContains(t, entry, "trace_flags")
}

func findLogrusLog(t *testing.T, raw, msg string) map[string]any {
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
