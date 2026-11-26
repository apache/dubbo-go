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
package auth

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

// TestAuthFailureKeepsInvocationOutOfSpan checks that a failed authentication
// logs the invocation but records only the method name and error on the span.
func TestAuthFailureKeepsInvocationOutOfSpan(t *testing.T) {
	var buf bytes.Buffer
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"}), zapcore.AddSync(&buf), zapcore.DebugLevel)
	prev := gostLogger.GetLogger()
	gostLogger.SetLogger(dubbozap.NewZapCtxLogger(&gostLogger.DubboLogger{Logger: zap.New(core).Sugar()}, true))
	t.Cleanup(func() { gostLogger.SetLogger(prev) })

	recorder := tracetest.NewSpanRecorder()
	ctx, span := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)).Tracer("test").Start(context.Background(), "server")

	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.example.UserProvider?interface=com.example.UserProvider")
	require.NoError(t, err)
	url.SetParam(constant.AccessKeyIDKey, "dubbo-ak")
	url.SetParam(constant.SecretAccessKeyKey, "dubbo-sk")
	url.SetParam(constant.ServiceAuthKey, "true")
	inv := invocation.NewRPCInvocation("GetUser", []any{"card-4242"}, map[string]any{
		constant.RequestSignatureKey: "forged-signature",
		constant.Consumer:            "test",
		constant.RequestTimestampKey: "1700000000000",
		constant.AKKey:               "dubbo-ak",
	})

	res := (&authFilter{}).Invoke(ctx, base.NewBaseInvoker(url), inv)
	require.Error(t, res.Error())
	span.End()

	assert.Contains(t, buf.String(), "card-4242", "the log line keeps the invocation")

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	want := "[Filter][Auth] auth the request occur exception, method=GetUser err=" + res.Error().Error()
	assert.Equal(t, want, spans[0].Status().Description)
	events := spans[0].Events()
	require.Len(t, events, 1)
	assert.Contains(t, events[0].Attributes, attribute.String("exception.message", want))
	for _, attr := range events[0].Attributes {
		assert.NotContains(t, attr.Value.Emit(), "card-4242")
		assert.NotContains(t, attr.Value.Emit(), "forged-signature")
	}
}
