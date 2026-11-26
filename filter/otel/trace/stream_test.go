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

package trace

import (
	"context"
	"errors"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

import (
	directorybase "dubbo.apache.org/dubbo-go/v3/cluster/directory/base"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func newStreamInvocation() *invocation.RPCInvocation {
	inv := invocation.NewRPCInvocation("Chat", nil, nil)
	inv.SetAttribute(constant.CallTypeKey, constant.CallBidiStream)
	return inv
}

// openedStream behaves like a protocol that opened the stream: it claims the
// watches and keeps the function that ends the stream.
func openedStream(end *func(error)) func(context.Context, base.Invocation) result.Result {
	return func(ctx context.Context, inv base.Invocation) result.Result {
		*end = base.ClaimStreamWatches(ctx, inv)
		return &result.RPCResult{Rest: "stream"}
	}
}

func spanByKind(t *testing.T, spans tracetest.SpanStubs, kind trace.SpanKind) tracetest.SpanStub {
	t.Helper()
	for _, span := range spans {
		if span.SpanKind == kind {
			return span
		}
	}
	t.Fatalf("no %s span", kind)
	return tracetest.SpanStub{}
}

func TestClientSpanEndsWithStream(t *testing.T) {
	for _, test := range []struct {
		name     string
		err      error
		wantCode codes.Code
		wantType string
	}{
		{name: "normal end", wantCode: codes.Ok},
		{name: "stream error", err: tri.NewError(tri.CodeUnavailable, errors.New("provider failed")),
			wantCode: codes.Error, wantType: "service_unavailable"},
		{name: "deadline", err: tri.NewError(tri.CodeDeadlineExceeded, context.DeadlineExceeded),
			wantCode: codes.Error, wantType: "timeout"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tp, exporter := traceTestProvider(t)
			filter := &otelClientFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}
			var end func(error)
			transport := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(traceTestURL(true)), call: openedStream(&end)}

			res := filter.Invoke(context.Background(), transport, newStreamInvocation())
			require.NoError(t, res.Error())
			require.NotNil(t, end)
			require.Empty(t, exporter.GetSpans(), "the client span ended when the stream opened")

			time.Sleep(20 * time.Millisecond)
			end(test.err)
			spans := exporter.GetSpans()
			require.Len(t, spans, 1)
			span := spans[0]
			require.Equal(t, test.wantCode, span.Status.Code)
			require.GreaterOrEqual(t, span.EndTime.Sub(span.StartTime), 20*time.Millisecond)
			if test.wantType != "" {
				require.Equal(t, test.wantType, spanValue(span, "error.type").AsString())
			}
		})
	}
}

func TestClientSpanOfUnclaimedStreamEndsOnReturn(t *testing.T) {
	tp, exporter := traceTestProvider(t)
	filter := &otelClientFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}
	openErr := rpcerror.Mark(errors.New("connection refused"), rpcerror.Network)
	transport := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(traceTestURL(true)),
		call: func(context.Context, base.Invocation) result.Result { return &result.RPCResult{Err: openErr} }}

	res := filter.Invoke(context.Background(), transport, newStreamInvocation())
	require.ErrorIs(t, res.Error(), openErr)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, codes.Error, spans[0].Status.Code)
	require.Equal(t, "network_failure", spanValue(spans[0], "error.type").AsString())
}

func TestClusterLogicalSpanEndsWithStream(t *testing.T) {
	tp, exporter := traceTestProvider(t)
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })
	clientFilter := &otelClientFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}
	makeAttempt := func(port string, call func(context.Context, base.Invocation) result.Result) base.Invoker {
		url := traceTestURL(true)
		url.Port = port
		transport := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(url), call: call}
		return &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(url), call: func(ctx context.Context, inv base.Invocation) result.Result {
			return clientFilter.Invoke(ctx, transport, inv)
		}}
	}
	// The first attempt cannot open the stream, the retry opens it.
	first := makeAttempt("20001", func(context.Context, base.Invocation) result.Result {
		return &result.RPCResult{Err: rpcerror.Mark(errors.New("connection refused"), rpcerror.Network)}
	})
	var end func(error)
	second := makeAttempt("20002", openedStream(&end))
	url := traceTestURL(true)
	dir := &traceTestDirectory{Directory: directorybase.NewDirectory(url), invokers: []base.Invoker{first, second}}
	strategy, err := extension.GetCluster(constant.ClusterKeyFailover)
	require.NoError(t, err)

	res := strategy.Join(dir).Invoke(context.Background(), newStreamInvocation())
	require.NoError(t, res.Error())
	require.NotNil(t, end)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1, "only the failed attempt ends before the stream does")
	require.Equal(t, codes.Error, spans[0].Status.Code)

	end(tri.NewError(tri.CodeUnavailable, errors.New("provider failed")))
	spans = exporter.GetSpans()
	require.Len(t, spans, 3)
	logical := spanByKind(t, spans, trace.SpanKindInternal)
	require.Equal(t, codes.Error, logical.Status.Code)
	require.EqualValues(t, 1, spanValue(logical, "cluster.retry_count").AsInt64())
	stream := spans[1]
	require.Equal(t, trace.SpanKindClient, stream.SpanKind)
	require.Equal(t, codes.Error, stream.Status.Code)
	require.Equal(t, logical.SpanContext.SpanID(), stream.Parent.SpanID())
	require.False(t, logical.EndTime.Before(stream.EndTime), "the logical span ends with its stream")
}

func TestClientSpanOfUnaryCallRecordsPanic(t *testing.T) {
	tp, exporter := traceTestProvider(t)
	filter := &otelClientFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}
	transport := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(traceTestURL(true)),
		call: func(context.Context, base.Invocation) result.Result { panic("invoker panicked") }}

	require.Panics(t, func() {
		filter.Invoke(context.Background(), transport, invocation.NewRPCInvocation("Call", nil, nil))
	})
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	require.Len(t, spanEvents(spans[0], "exception"), 1, "the span ended by a deferred End records the panic")
}
