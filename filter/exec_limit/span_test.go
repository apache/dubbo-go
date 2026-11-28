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
package exec_limit

import (
	"bytes"
	"context"
	"testing"
)

import (
	gostLogger "github.com/dubbogo/gost/log/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	dubbozap "dubbo.apache.org/dubbo-go/v3/logger/core/zap"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// TestRejectionKeepsURLOutOfSpan checks that a rejected invocation logs the
// full URL but records only the limit target on the span.
func TestRejectionKeepsURLOutOfSpan(t *testing.T) {
	var buf bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"}), zapcore.AddSync(&buf), zapcore.DebugLevel)
	prev := gostLogger.GetLogger()
	gostLogger.SetLogger(dubbozap.NewZapCtxLogger(&gostLogger.DubboLogger{Logger: zap.New(core).Sugar()}, true))
	t.Cleanup(func() { gostLogger.SetLogger(prev) })

	recorder := tracetest.NewSpanRecorder()
	ctx, span := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)).Tracer("test").Start(context.Background(), "server")

	invokeURL := common.NewURLWithOptions(
		common.WithUsername("user"),
		common.WithPassword("secret"),
		common.WithParamsValue(constant.InterfaceKey, "greet.GreetService"),
		common.WithParamsValue(constant.ExecuteLimitKey, "0"),
	)
	newFilter().Invoke(ctx, base.NewBaseInvoker(invokeURL), invocation.NewRPCInvocation("Greet", nil, nil))
	span.End()

	assert.Contains(t, buf.String(), "user:secret@", "the log line keeps the full URL")

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	want := "[Filter][ExecLimit] the invocation was rejected due to over the execute limitation, target=greet.GreetService"
	assert.Equal(t, want, spans[0].Status().Description)
	events := spans[0].Events()
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Attributes, attribute.String("exception.message", want))
}
