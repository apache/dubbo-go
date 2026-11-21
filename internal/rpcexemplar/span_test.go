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

package rpcexemplar

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestEnable(t *testing.T) {
	require.False(t, Enabled())
	t.Cleanup(func() { enabled.Store(false) })
	Enable()
	require.True(t, Enabled())
}

func TestSpanOwnershipAfterRewrite(t *testing.T) {
	ctx := context.Background()
	inv := invocation.NewRPCInvocation("Call", nil, nil)
	rewritten := invocation.NewRPCInvocation("Call", nil, nil)
	nested := invocation.NewRPCInvocation("Call", nil, nil)
	provider := sdktrace.NewTracerProvider()
	for _, side := range []struct {
		kind  trace.SpanKind
		role  string
		other string
	}{
		{trace.SpanKindClient, constant.SideConsumer, constant.SideProvider},
		{trace.SpanKindServer, constant.SideProvider, constant.SideConsumer},
	} {
		t.Run(side.role, func(t *testing.T) {
			_, span := provider.Tracer("test").Start(ctx, "Call", trace.WithSpanKind(side.kind))
			defer span.End()
			ids := IDs{TraceID: span.SpanContext().TraceID(), SpanID: span.SpanContext().SpanID()}

			local := WithSpan(ctx, inv, span, side.kind)
			assert.Nil(t, ForInvocation(ctx, inv, side.role))
			assert.Equal(t, &ids, ForInvocation(local, inv, side.role))
			assert.Nil(t, ForInvocation(local, inv, side.other))
			assert.Nil(t, ForInvocation(local, nested, side.role))
			assert.Same(t, local, Rebind(local, nested, rewritten))
			assert.Same(t, local, Rebind(local, inv, nil))
			assert.Same(t, local, Rebind(local, inv, valueInvocation{inv}))
			assert.Equal(t, ctx, Rebind(ctx, inv, rewritten))

			rebound := Rebind(local, inv, rewritten)
			assert.Equal(t, &ids, ForInvocation(rebound, rewritten, side.role))
			assert.Nil(t, ForInvocation(rebound, inv, side.role))
			assert.Equal(t, &ids, ForInvocation(local, inv, side.role))
		})
	}
}

type valueInvocation struct{ base.Invocation }

type spanWithContext struct {
	trace.Span
	context trace.SpanContext
}

func (s spanWithContext) SpanContext() trace.SpanContext { return s.context }

func TestWithSpanRequiresSampledLocalSpanAndPointerInvocation(t *testing.T) {
	ctx := context.Background()
	inv := invocation.NewRPCInvocation("Call", nil, nil)
	_, sampled := sdktrace.NewTracerProvider().Tracer("test").Start(ctx, "Call")
	defer sampled.End()
	_, nonRecording := noop.NewTracerProvider().Tracer("test").Start(ctx, "Call")
	unsampled := sampled.SpanContext().WithTraceFlags(0)
	var nilInvocation *invocation.RPCInvocation

	for _, tc := range []struct {
		name string
		inv  base.Invocation
		span trace.Span
	}{
		{"non-recording", inv, nonRecording},
		{"invalid", inv, spanWithContext{Span: sampled}},
		{"unsampled", inv, spanWithContext{Span: sampled, context: unsampled}},
		{"non-pointer", valueInvocation{inv}, sampled},
		{"typed nil", nilInvocation, sampled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, ctx, WithSpan(ctx, tc.inv, tc.span, trace.SpanKindClient))
		})
	}
}
