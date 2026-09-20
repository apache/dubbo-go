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
	plainCtx := context.Background()

	emitLogrusSample(ctxLogger, traceCtx, plainCtx, "info")
	infoOut := buf.String()
	assert.NotContains(t, infoOut, "debug-plain-info")
	assert.NotContains(t, infoOut, "debug-fmt-info")
	assert.NotContains(t, infoOut, "debug-ctx-info")
	assert.NotContains(t, infoOut, "debug-ctx-fmt-info")
	assert.Contains(t, infoOut, "info-plain-info")
	assert.Contains(t, infoOut, "info-fmt-info")
	assert.Contains(t, infoOut, "info-ctx-info")
	assert.Contains(t, infoOut, "info-ctx-fmt-info")
	assert.Contains(t, infoOut, "info-no-trace-info")
	assertLogrusTraceFields(t, infoOut, "info-ctx-info", wantTraceID, wantSpanID, wantFlags)
	assertLogrusNoTraceFields(t, infoOut, "info-no-trace-info")

	require.True(t, ctxLogger.SetLoggerLevel("debug"))
	assert.Equal(t, logrus.DebugLevel, inner.GetLevel())
	buf.Reset()
	emitLogrusSample(ctxLogger, traceCtx, plainCtx, "debug")
	debugOut := buf.String()
	assert.Contains(t, debugOut, "debug-plain-debug")
	assert.Contains(t, debugOut, "debug-fmt-debug")
	assert.Contains(t, debugOut, "debug-ctx-debug")
	assert.Contains(t, debugOut, "debug-ctx-fmt-debug")
	assert.Contains(t, debugOut, "info-plain-debug")
	assertLogrusTraceFields(t, debugOut, "debug-ctx-debug", wantTraceID, wantSpanID, wantFlags)

	require.True(t, ctxLogger.SetLoggerLevel("warn"))
	assert.Equal(t, logrus.WarnLevel, inner.GetLevel())
	buf.Reset()
	emitLogrusSample(ctxLogger, traceCtx, plainCtx, "warn")
	ctxLogger.Warn("warn-plain-warn")
	ctxLogger.Warnf("warn-fmt-%s", "warn")
	ctxLogger.CtxWarn(traceCtx, "warn-ctx-warn")
	ctxLogger.CtxWarnf(traceCtx, "warn-ctx-fmt-%s", "warn")
	warnOut := buf.String()
	assert.NotContains(t, warnOut, "debug-plain-warn")
	assert.NotContains(t, warnOut, "info-plain-warn")
	assert.NotContains(t, warnOut, "info-ctx-warn")
	assert.Contains(t, warnOut, "warn-plain-warn")
	assert.Contains(t, warnOut, "warn-fmt-warn")
	assert.Contains(t, warnOut, "warn-ctx-warn")
	assert.Contains(t, warnOut, "warn-ctx-fmt-warn")
	assertLogrusTraceFields(t, warnOut, "warn-ctx-warn", wantTraceID, wantSpanID, wantFlags)

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

func emitLogrusSample(l *LogrusCtxLogger, traceCtx, plainCtx context.Context, suffix string) {
	l.Debug("debug-plain-" + suffix)
	l.Debugf("debug-fmt-%s", suffix)
	l.CtxDebug(traceCtx, "debug-ctx-"+suffix)
	l.CtxDebugf(traceCtx, "debug-ctx-fmt-%s", suffix)
	l.Info("info-plain-" + suffix)
	l.Infof("info-fmt-%s", suffix)
	l.CtxInfo(traceCtx, "info-ctx-"+suffix)
	l.CtxInfof(traceCtx, "info-ctx-fmt-%s", suffix)
	l.CtxInfo(plainCtx, "info-no-trace-"+suffix)
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
	return oteltrace.ContextWithSpanContext(context.Background(), sc), traceID.String(), spanID.String(), sc.TraceFlags().String()
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
