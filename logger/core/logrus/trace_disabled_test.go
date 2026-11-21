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

	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/logger"
)

// TestInstantiateWithoutTraceIntegration checks that the logger is still a
// CtxLogger when trace integration is off, and that its Ctx methods then add
// no trace fields and record nothing on the span.
func TestInstantiateWithoutTraceIntegration(t *testing.T) {
	url := common.NewURLWithOptions(common.WithParamsValue(constant.LoggerTraceRecordErrorKey, "true"))
	l, err := instantiate(url)
	require.NoError(t, err)
	ctxLogger, ok := l.(logger.CtxLogger)
	require.True(t, ok, "logrus logger must implement CtxLogger with trace integration off")

	var buf bytes.Buffer
	lg := l.(*LogrusCtxLogger).Logger.(*logrus.Logger)
	lg.SetOutput(&buf)
	lg.SetFormatter(&logrus.JSONFormatter{})

	recorder := tracetest.NewSpanRecorder()
	ctx, span := trace.NewTracerProvider(trace.WithSpanProcessor(recorder)).Tracer("test").Start(context.Background(), "test-span")
	ctxLogger.CtxErrorf(ctx, "failed: %s", "boom")
	span.End()

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "failed: boom", entry["msg"])
	assert.NotContains(t, entry, "trace_id")
	require.Len(t, recorder.Ended(), 1)
	assert.Empty(t, recorder.Ended()[0].Events(), "no error is recorded on the span")
}

func TestNewLogrusCtxLoggerKeepsTraceFields(t *testing.T) {
	var buf bytes.Buffer
	lg := logrus.New()
	lg.SetOutput(&buf)
	lg.SetFormatter(&logrus.JSONFormatter{})
	ctxLogger := NewLogrusCtxLogger(&dubbogoLogger.DubboLogger{Logger: lg}, false)

	ctx, span := trace.NewTracerProvider().Tracer("test").Start(context.Background(), "test-span")
	defer span.End()
	ctxLogger.CtxInfof(ctx, "msg")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, span.SpanContext().TraceID().String(), entry["trace_id"])
}
