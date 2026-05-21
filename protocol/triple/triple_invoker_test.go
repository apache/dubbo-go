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

package triple

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	_ "dubbo.apache.org/dubbo-go/v3/filter/otel/trace"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func headerValues(header http.Header, key string) []string {
	if vals, ok := header[http.CanonicalHeaderKey(key)]; ok {
		return vals
	}
	return header[strings.ToLower(key)]
}

func Test_parseInvocation(t *testing.T) {
	tests := []struct {
		desc   string
		ctx    func() context.Context
		url    *common.URL
		invo   func() base.Invocation
		expect func(t *testing.T, callType string, inRaw []any, methodName string, err error)
	}{
		{
			desc: "miss callType",
			ctx: func() context.Context {
				return context.Background()
			},
			url: common.NewURLWithOptions(),
			invo: func() base.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, callType string, inRaw []any, methodName string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "miss CallType")
			},
		},
		{
			desc: "wrong callType",
			ctx: func() context.Context {
				return context.Background()
			},
			url: common.NewURLWithOptions(),
			invo: func() base.Invocation {
				iv := invocation.NewRPCInvocationWithOptions()
				iv.SetAttribute(constant.CallTypeKey, 1)
				return iv
			},
			expect: func(t *testing.T, callType string, inRaw []any, methodName string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "CallType should be string")
			},
		},
		{
			desc: "empty methodName",
			ctx: func() context.Context {
				return context.Background()
			},
			url: common.NewURLWithOptions(),
			invo: func() base.Invocation {
				iv := invocation.NewRPCInvocationWithOptions()
				iv.SetAttribute(constant.CallTypeKey, constant.CallUnary)
				return iv
			},
			expect: func(t *testing.T, callType string, inRaw []any, methodName string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "miss MethodName")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			callType, inRaw, methodName, err := parseInvocation(test.ctx(), test.url, test.invo())
			test.expect(t, callType, inRaw, methodName, err)
		})
	}
}

func Test_parseAttachments(t *testing.T) {
	tests := []struct {
		desc   string
		ctx    func() context.Context
		url    *common.URL
		invo   func() base.Invocation
		expect func(t *testing.T, inv base.Invocation)
	}{
		{
			desc: "url has pre-defined keys in triAttachmentKeys",
			ctx: func() context.Context {
				return context.Background()
			},
			url: common.NewURLWithOptions(
				common.WithInterface("interface"),
				common.WithToken("token"),
			),
			invo: func() base.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, inv base.Invocation) {
				assert.Equal(t, "interface", inv.GetAttachmentInterface(constant.InterfaceKey))
				assert.Equal(t, "token", inv.GetAttachmentInterface(constant.TokenKey))
			},
		},
		{
			desc: "ctx attachments are ignored",
			ctx: func() context.Context {
				userDefined := make(map[string]any)
				userDefined["key1"] = "val1"
				userDefined["traceparent"] = "old-trace"
				return context.WithValue(context.Background(), constant.AttachmentKey, userDefined)
			},
			url: common.NewURLWithOptions(),
			invo: func() base.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, inv base.Invocation) {
				_, ok := inv.GetAttachment("key1")
				assert.False(t, ok)
				_, ok = inv.GetAttachment("traceparent")
				assert.False(t, ok)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := test.ctx()
			inv := test.invo()
			parseAttachments(ctx, test.url, inv)
			test.expect(t, inv)
		})
	}
}

// newTestTripleInvoker creates a TripleInvoker for testing without network connection
func newTestTripleInvoker(url *common.URL, cm *clientManager) *TripleInvoker {
	return &TripleInvoker{
		BaseInvoker:   *base.NewBaseInvoker(url),
		quitOnce:      sync.Once{},
		clientGuard:   &sync.RWMutex{},
		clientManager: cm,
	}
}

func TestTripleInvoker_SetGetClientManager(t *testing.T) {
	url := common.NewURLWithOptions()
	ti := newTestTripleInvoker(url, nil)

	// initially nil
	assert.Nil(t, ti.getClientManager())

	// set clientManager
	cm := &clientManager{
		isIDL:     true,
		triClient: nil,
	}
	ti.setClientManager(cm)
	assert.Equal(t, cm, ti.getClientManager())

	// set to nil
	ti.setClientManager(nil)
	assert.Nil(t, ti.getClientManager())
}

func TestTripleInvoker_IsAvailable(t *testing.T) {
	tests := []struct {
		desc          string
		clientManager *clientManager
		expect        bool
	}{
		{
			desc:          "clientManager is nil",
			clientManager: nil,
			expect:        false,
		},
		{
			desc: "clientManager is not nil",
			clientManager: &clientManager{
				isIDL:     true,
				triClient: nil,
			},
			expect: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			url := common.NewURLWithOptions()
			ti := newTestTripleInvoker(url, test.clientManager)
			assert.Equal(t, test.expect, ti.IsAvailable())
		})
	}
}

func TestTripleInvoker_IsDestroyed(t *testing.T) {
	tests := []struct {
		desc          string
		clientManager *clientManager
		destroyed     bool
		expect        bool
	}{
		{
			desc:          "clientManager is nil",
			clientManager: nil,
			destroyed:     false,
			expect:        false,
		},
		{
			desc: "clientManager is not nil and not destroyed",
			clientManager: &clientManager{
				isIDL:     true,
				triClient: nil,
			},
			destroyed: false,
			expect:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			url := common.NewURLWithOptions()
			ti := newTestTripleInvoker(url, test.clientManager)
			assert.Equal(t, test.expect, ti.IsDestroyed())
		})
	}
}

func TestTripleInvoker_Destroy(t *testing.T) {
	t.Run("destroy with clientManager", func(t *testing.T) {
		url := common.NewURLWithOptions()
		cm := &clientManager{
			isIDL:     true,
			triClient: nil,
		}
		ti := newTestTripleInvoker(url, cm)

		assert.True(t, ti.IsAvailable())
		assert.NotNil(t, ti.getClientManager())

		ti.Destroy()

		assert.False(t, ti.IsAvailable())
		assert.Nil(t, ti.getClientManager())
	})

	t.Run("destroy without clientManager", func(t *testing.T) {
		url := common.NewURLWithOptions()
		ti := newTestTripleInvoker(url, nil)

		ti.Destroy()

		assert.False(t, ti.IsAvailable())
		assert.Nil(t, ti.getClientManager())
	})

	t.Run("destroy called multiple times", func(t *testing.T) {
		url := common.NewURLWithOptions()
		cm := &clientManager{
			isIDL:     true,
			triClient: nil,
		}
		ti := newTestTripleInvoker(url, cm)

		// first destroy
		ti.Destroy()
		assert.Nil(t, ti.getClientManager())

		// second destroy should not panic
		ti.Destroy()
		assert.Nil(t, ti.getClientManager())
	})
}

func TestTripleInvoker_Invoke(t *testing.T) {
	tests := []struct {
		desc         string
		setup        func() (*TripleInvoker, base.Invocation)
		expectErr    error
		expectErrMsg string
	}{
		{
			desc: "invoker is destroyed",
			setup: func() (*TripleInvoker, base.Invocation) {
				url := common.NewURLWithOptions()
				ti := newTestTripleInvoker(url, &clientManager{
					isIDL:     true,
					triClient: nil,
				})
				ti.Destroy()
				inv := invocation.NewRPCInvocationWithOptions()
				return ti, inv
			},
			expectErr: base.ErrDestroyedInvoker,
		},
		{
			desc: "clientManager is nil",
			setup: func() (*TripleInvoker, base.Invocation) {
				url := common.NewURLWithOptions()
				ti := newTestTripleInvoker(url, nil)
				inv := invocation.NewRPCInvocationWithOptions()
				return ti, inv
			},
			expectErr: base.ErrClientClosed,
		},
		{
			desc: "parseInvocation error - miss callType",
			setup: func() (*TripleInvoker, base.Invocation) {
				url := common.NewURLWithOptions()
				ti := newTestTripleInvoker(url, &clientManager{
					isIDL:     true,
					triClient: nil,
				})
				inv := invocation.NewRPCInvocationWithOptions()
				return ti, inv
			},
			expectErrMsg: "miss CallType",
		},
		{
			desc: "mergeAttachmentToOutgoing error - invalid attachment",
			setup: func() (*TripleInvoker, base.Invocation) {
				url := common.NewURLWithOptions()
				ti := newTestTripleInvoker(url, &clientManager{
					isIDL:     true,
					triClient: nil,
				})
				inv := invocation.NewRPCInvocationWithOptions(
					invocation.WithMethodName("TestMethod"),
					invocation.WithAttachment("invalid_key", 123), // invalid attachment type
				)
				inv.SetAttribute(constant.CallTypeKey, constant.CallUnary)
				return ti, inv
			},
			expectErrMsg: "invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ti, inv := test.setup()
			result := ti.Invoke(context.Background(), inv)
			if test.expectErr != nil {
				assert.Equal(t, test.expectErr, result.Error())
			}
			if test.expectErrMsg != "" {
				require.Error(t, result.Error())
				assert.Contains(t, result.Error().Error(), test.expectErrMsg)
			}
		})
	}
}

func TestTripleInvoker_Invoke_Concurrent(t *testing.T) {
	url := common.NewURLWithOptions()
	ti := newTestTripleInvoker(url, &clientManager{
		isIDL:     true,
		triClient: nil,
	})

	var wg sync.WaitGroup
	concurrency := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			inv := invocation.NewRPCInvocationWithOptions()
			_ = ti.Invoke(context.Background(), inv)
		}()
	}

	wg.Wait()
}

func Test_mergeAttachmentToOutgoing(t *testing.T) {
	tests := []struct {
		desc   string
		ctx    context.Context
		invo   func() base.Invocation
		expect func(t *testing.T, ctx context.Context, err error)
	}{
		{
			desc: "with timeout attachment",
			ctx:  context.Background(),
			invo: func() base.Invocation {
				inv := invocation.NewRPCInvocationWithOptions(
					invocation.WithAttachment(constant.TimeoutKey, "5000"),
				)
				return inv
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.NoError(t, err)
				timeout := ctx.Value(tri.TimeoutKey{})
				assert.Equal(t, "5000", timeout)
			},
		},
		{
			desc: "with string attachment",
			ctx:  context.Background(),
			invo: func() base.Invocation {
				inv := invocation.NewRPCInvocationWithOptions(
					invocation.WithAttachment("custom-key", "custom-value"),
				)
				return inv
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.NoError(t, err)
				header := http.Header(tri.ExtractFromOutgoingContext(ctx))
				assert.Equal(t, []string{"custom-value"}, headerValues(header, "custom-key"))
			},
		},
		{
			desc: "with string slice attachment",
			ctx:  context.Background(),
			invo: func() base.Invocation {
				inv := invocation.NewRPCInvocationWithOptions(
					invocation.WithAttachment("multi-key", []string{"val1", "val2"}),
				)
				return inv
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.NoError(t, err)
				header := http.Header(tri.ExtractFromOutgoingContext(ctx))
				assert.Equal(t, []string{"val1", "val2"}, headerValues(header, "multi-key"))
			},
		},
		{
			desc: "preserves unrelated existing outgoing header",
			ctx: tri.NewOutgoingContext(context.Background(), http.Header{
				"existing-header": []string{"existing-value"},
			}),
			invo: func() base.Invocation {
				return invocation.NewRPCInvocationWithOptions(
					invocation.WithAttachment("custom-key", "custom-value"),
				)
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.NoError(t, err)
				header := http.Header(tri.ExtractFromOutgoingContext(ctx))
				assert.Equal(t, []string{"existing-value"}, headerValues(header, "existing-header"))
				assert.Equal(t, []string{"custom-value"}, headerValues(header, "custom-key"))
			},
		},
		{
			desc: "overwrites existing traceparent instead of appending",
			ctx: tri.NewOutgoingContext(context.Background(), http.Header{
				"traceparent": []string{"old-traceparent"},
			}),
			invo: func() base.Invocation {
				return invocation.NewRPCInvocationWithOptions(
					invocation.WithAttachment("traceparent", "new-traceparent"),
				)
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.NoError(t, err)
				header := http.Header(tri.ExtractFromOutgoingContext(ctx))
				assert.Equal(t, []string{"new-traceparent"}, headerValues(header, "traceparent"))
			},
		},
		{
			desc: "with invalid attachment type",
			ctx:  context.Background(),
			invo: func() base.Invocation {
				inv := invocation.NewRPCInvocationWithOptions(
					invocation.WithAttachment("invalid-key", 12345),
				)
				return inv
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid")
			},
		},
		{
			desc: "with empty attachments",
			ctx:  context.Background(),
			invo: func() base.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			inv := test.invo()
			ctx, err := mergeAttachmentToOutgoing(test.ctx, inv)
			test.expect(t, ctx, err)
		})
	}
}

func Test_mergeAttachmentToOutgoing_DoesNotMutatePreviousContext(t *testing.T) {
	inv1 := invocation.NewRPCInvocationWithOptions(
		invocation.WithAttachment("traceparent", "first-traceparent"),
	)
	ctx1, err := mergeAttachmentToOutgoing(context.Background(), inv1)
	require.NoError(t, err)

	inv2 := invocation.NewRPCInvocationWithOptions(
		invocation.WithAttachment("traceparent", "second-traceparent"),
	)
	ctx2, err := mergeAttachmentToOutgoing(ctx1, inv2)
	require.NoError(t, err)

	header1 := http.Header(tri.ExtractFromOutgoingContext(ctx1))
	header2 := http.Header(tri.ExtractFromOutgoingContext(ctx2))

	assert.Equal(t, []string{"first-traceparent"}, headerValues(header1, "traceparent"))
	assert.Equal(t, []string{"second-traceparent"}, headerValues(header2, "traceparent"))
}

type capturedTripleCall struct {
	method              string
	activeSpanContext   oteltrace.SpanContext
	outgoingTraceparent string
}

type traceCaptureInvoker struct {
	base.BaseInvoker
	calls []capturedTripleCall
}

func newTraceCaptureInvoker(url *common.URL) *traceCaptureInvoker {
	return &traceCaptureInvoker{
		BaseInvoker: *base.NewBaseInvoker(url),
	}
}

func (i *traceCaptureInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	_, _, _, err := parseInvocation(ctx, i.GetURL(), inv)
	if err != nil {
		return &result.RPCResult{Err: err}
	}

	ctx, err = mergeAttachmentToOutgoing(ctx, inv)
	if err != nil {
		return &result.RPCResult{Err: err}
	}

	header := cloneOutgoingHeader(tri.ExtractFromOutgoingContext(ctx))
	i.calls = append(i.calls, capturedTripleCall{
		method:              inv.MethodName(),
		activeSpanContext:   oteltrace.SpanContextFromContext(ctx),
		outgoingTraceparent: firstHeaderValue(header, "traceparent"),
	})
	return &result.RPCResult{}
}

func TestTripleClientOTELTraceparentIsolation(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	defer func() {
		_ = tracerProvider.Shutdown(context.Background())
	}()

	oldTracerProvider := otel.GetTracerProvider()
	oldPropagator := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer func() {
		otel.SetTracerProvider(oldTracerProvider)
		otel.SetTextMapPropagator(oldPropagator)
	}()

	clientFilter, ok := extension.GetFilter(constant.OTELClientTraceKey)
	require.True(t, ok)

	incomingTraceparent := "00-4bf92f3577b34da6a3ce929d0e0e4736-1111111111111111-01"
	upstreamCtx := propagation.TraceContext{}.Extract(
		context.Background(),
		propagation.MapCarrier{"traceparent": incomingTraceparent},
	)

	serverTracer := tracerProvider.Tracer("triple-otel-repro")
	serverCtx, serverSpan := serverTracer.Start(upstreamCtx, "service-a-handler", oteltrace.WithSpanKind(oteltrace.SpanKindServer))
	serverCtx = context.WithValue(serverCtx, constant.AttachmentKey, map[string]any{
		"traceparent": incomingTraceparent,
	})
	serverCtx = tri.NewOutgoingContext(serverCtx, http.Header{
		"x-seed": []string{"seed"},
	})

	invoker := newTraceCaptureInvoker(common.NewURLWithOptions(
		common.WithProtocol("tri"),
		common.WithInterface("org.apache.dubbo.test.DownstreamService"),
	))

	callB := newTraceInvocation("CallServiceB")
	callC := newTraceInvocation("CallServiceC")

	resB := clientFilter.Invoke(serverCtx, invoker, callB)
	require.NoError(t, resB.Error())
	afterBHeader := cloneOutgoingHeader(tri.ExtractFromOutgoingContext(serverCtx))

	resC := clientFilter.Invoke(serverCtx, invoker, callC)
	require.NoError(t, resC.Error())
	afterCHeader := cloneOutgoingHeader(tri.ExtractFromOutgoingContext(serverCtx))

	serverSpan.End()

	require.Len(t, invoker.calls, 2)
	endedSpans := spanRecorder.Ended()
	serverReadOnly := findEndedSpanByName(t, endedSpans, "service-a-handler")
	clientBReadOnly := findEndedSpanByName(t, endedSpans, "CallServiceB")
	clientCReadOnly := findEndedSpanByName(t, endedSpans, "CallServiceC")

	clientBOutgoing := parseTraceparent(t, invoker.calls[0].outgoingTraceparent)
	clientCOutgoing := parseTraceparent(t, invoker.calls[1].outgoingTraceparent)

	assert.Equal(t, incomingTraceparent, contextAttachmentTraceparent(serverCtx))
	assert.Equal(t, serverReadOnly.SpanContext().SpanID(), clientBReadOnly.Parent().SpanID())
	assert.Equal(t, serverReadOnly.SpanContext().SpanID(), clientCReadOnly.Parent().SpanID())
	assert.Equal(t, clientBReadOnly.SpanContext().SpanID().String(), clientBOutgoing.spanID)
	assert.Equal(t, clientCReadOnly.SpanContext().SpanID().String(), clientCOutgoing.spanID)
	assert.NotEqual(t, incomingTraceparent, invoker.calls[0].outgoingTraceparent)
	assert.NotEqual(t, invoker.calls[0].outgoingTraceparent, invoker.calls[1].outgoingTraceparent)
	assert.Equal(t, "seed", firstHeaderValue(afterBHeader, "x-seed"))
	assert.Equal(t, "seed", firstHeaderValue(afterCHeader, "x-seed"))
	assert.Empty(t, headerValues(afterBHeader, "traceparent"))
	assert.Empty(t, headerValues(afterCHeader, "traceparent"))

	t.Logf("incoming traceparent = %s", incomingTraceparent)
	t.Logf(
		"service A server span: trace_id=%s span_id=%s parent_span_id=%s",
		serverReadOnly.SpanContext().TraceID(),
		serverReadOnly.SpanContext().SpanID(),
		serverReadOnly.Parent().SpanID(),
	)
	t.Logf(
		"call B client span: trace_id=%s span_id=%s parent_span_id=%s outgoing_traceparent=%s",
		clientBReadOnly.SpanContext().TraceID(),
		clientBReadOnly.SpanContext().SpanID(),
		clientBReadOnly.Parent().SpanID(),
		invoker.calls[0].outgoingTraceparent,
	)
	t.Logf(
		"call C client span: trace_id=%s span_id=%s parent_span_id=%s outgoing_traceparent=%s",
		clientCReadOnly.SpanContext().TraceID(),
		clientCReadOnly.SpanContext().SpanID(),
		clientCReadOnly.Parent().SpanID(),
		invoker.calls[1].outgoingTraceparent,
	)
	t.Logf("base ctx outgoing header after call B = %v", afterBHeader)
	t.Logf("base ctx outgoing header after call C = %v", afterCHeader)
}

func newTraceInvocation(method string) *invocation.RPCInvocation {
	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName(method),
	)
	inv.SetAttribute(constant.CallTypeKey, constant.CallUnary)
	return inv
}

func findEndedSpanByName(t *testing.T, spans []sdktrace.ReadOnlySpan, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	for _, span := range spans {
		if span.Name() == name {
			return span
		}
	}
	t.Fatalf("span %q not found", name)
	return nil
}

type parsedTraceparent struct {
	traceID string
	spanID  string
	flags   string
}

func parseTraceparent(t *testing.T, traceparent string) parsedTraceparent {
	t.Helper()
	parts := strings.Split(traceparent, "-")
	require.Len(t, parts, 4, "invalid traceparent: %s", traceparent)
	return parsedTraceparent{
		traceID: parts[1],
		spanID:  parts[2],
		flags:   parts[3],
	}
}

func firstHeaderValue(header http.Header, key string) string {
	values := header[strings.ToLower(key)]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func contextAttachmentTraceparent(ctx context.Context) string {
	raw := ctx.Value(constant.AttachmentKey)
	if raw == nil {
		return ""
	}
	attachments, ok := raw.(map[string]any)
	if !ok {
		return fmt.Sprintf("%v", raw)
	}
	if value, ok := attachments["traceparent"].(string); ok {
		return value
	}
	return ""
}
