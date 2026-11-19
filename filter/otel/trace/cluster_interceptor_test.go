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
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

import (
	cluster "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	clusterbase "dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/failover"
	directorybase "dubbo.apache.org/dubbo-go/v3/cluster/directory/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

type traceTestInvoker struct {
	*base.BaseInvoker
	call func(context.Context, base.Invocation) result.Result
}

func (i *traceTestInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	return i.call(ctx, inv)
}

type traceTestDirectory struct {
	*directorybase.Directory
	invokers []base.Invoker
	list     func(base.Invocation) []base.Invoker
}

func (d *traceTestDirectory) List(inv base.Invocation) []base.Invoker {
	if d.list != nil {
		return d.list(inv)
	}
	return d.invokers
}
func (d *traceTestDirectory) IsAvailable() bool           { return true }
func (d *traceTestDirectory) Destroy()                    {}
func (d *traceTestDirectory) Subscribe(*common.URL) error { return nil }

func init() {
	extension.SetLoadbalance("trace-test-first", func() loadbalance.LoadBalance { return &traceTestBalance{} })
}

type traceTestBalance struct{}

func (*traceTestBalance) Select(invokers []base.Invoker, _ base.Invocation) base.Invoker {
	return invokers[0]
}

func traceTestURL(enabled bool) *common.URL {
	url := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithIp("127.0.0.1"),
		common.WithPort("20001"), common.WithPath("trace.test.Service"),
		common.WithParamsValue(constant.RetriesKey, "2"),
		common.WithParamsValue(constant.LoadbalanceKey, "trace-test-first"))
	if enabled {
		url.SetParam(constant.ReferenceFilterKey, constant.OTELClientTraceKey)
	}
	return url
}

func traceTestProvider(t *testing.T) (*sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { require.NoError(t, tp.Shutdown(context.Background())) })
	return tp, exporter
}

func spanValue(span tracetest.SpanStub, name string) attribute.Value {
	for _, attr := range span.Attributes {
		if string(attr.Key) == name {
			return attr.Value
		}
	}
	return attribute.Value{}
}
func spanEvents(span tracetest.SpanStub, name string) []sdktrace.Event {
	var events []sdktrace.Event
	for _, event := range span.Events {
		if event.Name == name {
			events = append(events, event)
		}
	}
	return events
}

func TestClusterLogicalRetryLoopback(t *testing.T) {
	tp, exporter := traceTestProvider(t)
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })
	url := traceTestURL(true)
	clientFilter := &otelClientFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}
	serverFilter := &otelServerFilter{TracerProvider: tp, Propagators: propagation.TraceContext{}}
	serverInvoker := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(url), call: func(context.Context, base.Invocation) result.Result {
		return &result.RPCResult{}
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attachments := map[string]any{"traceparent": r.Header.Get("traceparent")}
		inv := invocation.NewRPCInvocation("Call", nil, attachments)
		serverFilter.Invoke(r.Context(), serverInvoker, inv)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	makeAttempt := func(url *common.URL, call func(context.Context, base.Invocation) result.Result) base.Invoker {
		transport := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(url), call: call}
		return &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(url), call: func(ctx context.Context, inv base.Invocation) result.Result {
			return clientFilter.Invoke(ctx, transport, inv)
		}}
	}
	first := makeAttempt(url, func(context.Context, base.Invocation) result.Result {
		return &result.RPCResult{Err: rpcerror.Mark(errors.New("connection closed"), rpcerror.Network)}
	})
	secondURL := traceTestURL(true)
	secondURL.Port = "20002"
	second := makeAttempt(secondURL, func(ctx context.Context, inv base.Invocation) result.Result {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		if err != nil {
			return &result.RPCResult{Err: err}
		}
		parent, _ := inv.GetAttachment("traceparent")
		req.Header.Set("traceparent", parent)
		resp, err := server.Client().Do(req)
		if resp != nil {
			_ = resp.Body.Close()
		}
		return &result.RPCResult{Err: err}
	})
	dir := &traceTestDirectory{Directory: directorybase.NewDirectory(url), invokers: []base.Invoker{first, second}}
	strategy, err := extension.GetCluster(constant.ClusterKeyFailover)
	require.NoError(t, err)
	invoker := strategy.Join(dir)
	inv := invocation.NewRPCInvocation("Call", []any{"private-argument-not-in-trace"}, nil)
	require.NoError(t, invoker.Invoke(context.Background(), inv).Error())
	spans := exporter.GetSpans()
	require.Len(t, spans, 4)
	var logical, serverSpan tracetest.SpanStub
	var attempts []tracetest.SpanStub
	for _, span := range spans {
		switch span.SpanKind {
		case trace.SpanKindInternal:
			logical = span
		case trace.SpanKindClient:
			attempts = append(attempts, span)
		case trace.SpanKindServer:
			serverSpan = span
		}
		require.NotContains(t, fmt.Sprint(span.Attributes, span.Events), "private-argument-not-in-trace")
	}
	require.Len(t, attempts, 2)
	require.Equal(t, codes.Ok, logical.Status.Code)
	require.EqualValues(t, 1, spanValue(logical, "cluster.retry_count").AsInt64())
	require.Len(t, spanEvents(logical, "cluster.select"), 2)
	require.Len(t, spanEvents(logical, "cluster.retry"), 1)
	require.NotEqual(t, attempts[0].SpanContext.SpanID(), attempts[1].SpanContext.SpanID())
	for _, attempt := range attempts {
		require.Equal(t, logical.SpanContext.SpanID(), attempt.Parent.SpanID())
		require.Equal(t, logical.SpanContext.TraceID(), attempt.SpanContext.TraceID())
		require.Equal(t, "Call", attempt.Name) // Existing client naming is preserved.
	}
	require.Equal(t, codes.Error, attempts[0].Status.Code)
	require.Equal(t, "network_failure", spanValue(attempts[0], "error.type").AsString())
	require.Len(t, spanEvents(attempts[0], "exception"), 1)
	require.Equal(t, codes.Ok, attempts[1].Status.Code)
	require.Equal(t, attempts[1].SpanContext.SpanID(), serverSpan.Parent.SpanID())
	require.Equal(t, logical.SpanContext.TraceID(), serverSpan.SpanContext.TraceID())
	// Events include only the selected address, never credentials/service URLs.
	selections := spanEvents(logical, "cluster.select")
	require.Contains(t, selections[0].Attributes, attribute.String("peer", "127.0.0.1:20001"))
	require.Contains(t, selections[1].Attributes, attribute.Int("attempt", 2))
}

func TestClusterLogicalNoProvider(t *testing.T) {
	tp, exporter := traceTestProvider(t)
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })
	for _, registry := range []bool{false, true} {
		url := traceTestURL(true)
		url.Username = "private-url-user"
		url.Password = "private-url-password"
		url.SetParam(constant.NacosPassword, "private-registry-password")
		if registry {
			outer := traceTestURL(false)
			outer.SubURL = url
			outer.SetParam(constant.NacosPassword, "private-registry-password")
			url = outer
		}
		dir := &traceTestDirectory{Directory: directorybase.NewDirectory(url)}
		strategy, err := extension.GetCluster(constant.ClusterKeyFailover)
		require.NoError(t, err)
		invoker := strategy.Join(dir)
		require.Error(t, invoker.Invoke(context.Background(), invocation.NewRPCInvocation("Call", nil, nil)).Error())
	}
	spans := exporter.GetSpans()
	require.Len(t, spans, 2)
	require.NotContains(t, fmt.Sprint(spans), "private-url-user")
	require.NotContains(t, fmt.Sprint(spans), "private-url-password")
	require.NotContains(t, fmt.Sprint(spans), "private-registry-password")
	for _, span := range spans {
		require.Equal(t, trace.SpanKindInternal, span.SpanKind)
		require.Equal(t, codes.Error, span.Status.Code)
		require.Equal(t, "service_unavailable", spanValue(span, "error.type").AsString())
		require.Len(t, spanEvents(span, "cluster.no_provider"), 1)
		require.Contains(t, spanEvents(span, "cluster.no_provider")[0].Attributes, attribute.Int("candidate_count", 0))
		require.Len(t, spanEvents(span, "exception"), 1)
	}
}

func TestClusterLogicalCancellationEndsBeforeInvokeReturns(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprint(deadline), func(t *testing.T) {
			tp, exporter := traceTestProvider(t)
			ctx, cancel := context.WithCancel(context.Background())
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
			}
			defer cancel()
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
			invoker := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(traceTestURL(true)), call: func(ctx context.Context, _ base.Invocation) result.Result {
				close(entered)
				<-release
				return &result.RPCResult{Err: ctx.Err()}
			}}
			go func() {
				defer close(done)
				(&clusterInterceptor{TracerProvider: tp}).Invoke(ctx, invoker, invocation.NewRPCInvocation("Call", nil, nil))
			}()
			<-entered
			if !deadline {
				cancel()
			}
			require.Eventually(t, func() bool { return len(exporter.GetSpans()) == 1 }, time.Second, time.Millisecond)
			close(release)
			<-done
			spans := exporter.GetSpans()
			require.Len(t, spans, 1)
			require.Equal(t, codes.Error, spans[0].Status.Code)
			expected := "unknown"
			if deadline {
				expected = "timeout"
			}
			require.Equal(t, expected, spanValue(spans[0], "error.type").AsString())
			require.Len(t, spanEvents(spans[0], "exception"), 1)
		})
	}
}

// traceTestCancelResult cancels while the interceptor reads a completed RPC's
// result. Cancellation must not replace that result's success or protocol error.
type traceTestCancelResult struct {
	*result.RPCResult
	cancel context.CancelFunc
	done   <-chan struct{}
}

func (r *traceTestCancelResult) Error() error {
	r.cancel()
	<-r.done
	return r.RPCResult.Error()
}

func TestClusterLogicalCompletedResultSurvivesCancellation(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{
		{"success", nil},
		{"protocol error", tri.NewError(tri.CodeResourceExhausted, errors.New("provider quota exhausted"))},
		{"protocol timeout", tri.NewError(tri.CodeDeadlineExceeded, errors.New("provider response deadline exceeded"))},
	} {
		t.Run(test.name, func(t *testing.T) {
			tp, exporter := traceTestProvider(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			canceled := make(chan struct{})
			stop := context.AfterFunc(ctx, func() { close(canceled) })
			defer stop()
			response := &traceTestCancelResult{RPCResult: &result.RPCResult{Err: test.err}, cancel: cancel, done: canceled}
			invoker := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(traceTestURL(true)), call: func(context.Context, base.Invocation) result.Result {
				return response
			}}
			got := (&clusterInterceptor{TracerProvider: tp}).Invoke(ctx, invoker, invocation.NewRPCInvocation("Call", nil, nil))
			require.Same(t, response, got)
			require.ErrorIs(t, ctx.Err(), context.Canceled)
			require.Len(t, exporter.GetSpans(), 1)
			span := exporter.GetSpans()[0]
			if test.err == nil {
				require.Equal(t, codes.Ok, span.Status.Code)
				require.Empty(t, spanEvents(span, "exception"))
			} else {
				require.Same(t, test.err, got.Error())
				require.Equal(t, codes.Error, span.Status.Code)
				require.Equal(t, test.err.Error(), span.Status.Description)
				require.Equal(t, errorType(test.err), spanValue(span, "error.type").AsString())
				exceptions := spanEvents(span, "exception")
				require.Len(t, exceptions, 1)
				require.Contains(t, exceptions[0].Attributes, attribute.String("exception.message", test.err.Error()))
			}
		})
	}
}

type traceTestNilURLInvoker struct{ *base.BaseInvoker }

func (*traceTestNilURLInvoker) GetURL() *common.URL { return nil }

type traceTestFixedBalance struct{ selected base.Invoker }

func (b *traceTestFixedBalance) Select([]base.Invoker, base.Invocation) base.Invoker {
	return b.selected
}

func TestClusterDiagnosticsNilURLs(t *testing.T) {
	t.Run("selected invoker", func(t *testing.T) {
		tp, exporter := traceTestProvider(t)
		ctx, span := tp.Tracer("test").Start(context.Background(), "selection")
		observation := &logicalSpan{span: span}
		inv := invocation.NewRPCInvocation("Call", nil, nil)
		ctx = cluster.WithDiagnostics(ctx, inv, observation)
		first := base.NewBaseInvoker(traceTestURL(false))
		selected := &traceTestNilURLInvoker{BaseInvoker: base.NewBaseInvoker(nil)}
		invoker := clusterbase.NewBaseClusterInvoker(nil)
		require.NotPanics(t, func() {
			got := invoker.DoSelectWithContext(ctx, &traceTestFixedBalance{selected: selected},
				inv, []base.Invoker{first, selected}, nil, 1)
			require.Same(t, selected, got)
		})
		observation.finish(nil)
		require.Len(t, exporter.GetSpans(), 1)
		require.Empty(t, spanEvents(exporter.GetSpans()[0], "cluster.select"))
	})
	t.Run("directory", func(t *testing.T) {
		dir := &traceTestDirectory{Directory: directorybase.NewDirectory(nil)}
		invoker := clusterbase.NewBaseClusterInvoker(dir)
		require.NotPanics(t, func() {
			err := invoker.CheckInvokers(nil, invocation.NewRPCInvocation("Call", nil, nil))
			require.ErrorContains(t, err, "No provider available")
		})
	})
}

// Build a real failover invoker with a pass-through interceptor so nested
// routing can exercise invocation binding independently of tracing's masking.
type traceTestPassThroughInterceptor struct{}

func (*traceTestPassThroughInterceptor) Invoke(ctx context.Context, invoker base.Invoker, inv base.Invocation) result.Result {
	return invoker.Invoke(ctx, inv)
}

func traceTestFailover(t *testing.T, dir *traceTestDirectory) base.Invoker {
	t.Helper()
	cluster.SetClusterInterceptor(constant.OTELClientTraceKey, func() cluster.Interceptor {
		return &traceTestPassThroughInterceptor{}
	})
	defer cluster.SetClusterInterceptor(constant.OTELClientTraceKey, func() cluster.Interceptor {
		return &clusterInterceptor{TracerProvider: otel.GetTracerProvider()}
	})
	strategy, err := extension.GetCluster(constant.ClusterKeyFailover)
	require.NoError(t, err)
	return strategy.Join(dir)
}

func TestClusterDiagnosticsInvocationBinding(t *testing.T) {
	inv := invocation.NewRPCInvocation("Call", nil, nil)
	observer := &logicalSpan{}
	require.Nil(t, cluster.DiagnosticsFromContext(nil, inv))
	ctx := cluster.WithDiagnostics(nil, inv, observer)
	require.Same(t, observer, cluster.DiagnosticsFromContext(ctx, inv))
	for _, other := range []any{nil, (*invocation.RPCInvocation)(nil),
		invocation.NewRPCInvocation("Call", nil, nil), 1, map[string]any{"method": "Call"}} {
		require.Nil(t, cluster.DiagnosticsFromContext(ctx, other))
	}
	cleared := cluster.WithDiagnostics(ctx, nil, nil)
	require.Nil(t, cluster.DiagnosticsFromContext(cleared, inv))
	require.Same(t, observer, cluster.DiagnosticsFromContext(ctx, inv))
	require.Same(t, cleared, cluster.WithDiagnostics(cleared, nil, nil))
	require.Equal(t, context.Background(), cluster.WithDiagnostics(nil, nil, nil))
	invalid := cluster.WithDiagnostics(ctx, "non-pointer", observer)
	require.Nil(t, cluster.DiagnosticsFromContext(invalid, inv))
}

func TestClusterLogicalNilContext(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			tp, exporter := traceTestProvider(t)
			previous := otel.GetTracerProvider()
			otel.SetTracerProvider(tp)
			t.Cleanup(func() { otel.SetTracerProvider(previous) })
			dir := &traceTestDirectory{Directory: directorybase.NewDirectory(traceTestURL(enabled))}
			strategy, err := extension.GetCluster(constant.ClusterKeyFailover)
			require.NoError(t, err)
			invoker := strategy.Join(dir)
			require.NotPanics(t, func() {
				require.Error(t, invoker.Invoke(nil, invocation.NewRPCInvocation("Call", nil, nil)).Error())
			})
			if enabled {
				require.Len(t, exporter.GetSpans(), 1)
				require.Len(t, spanEvents(exporter.GetSpans()[0], "cluster.no_provider"), 1)
			} else {
				require.Empty(t, exporter.GetSpans())
			}
		})
	}
}

func TestClusterLogicalNestedIsolation(t *testing.T) {
	for _, mode := range []string{"new invocation", "new invocation no provider", "disabled", "unsampled", "recording"} {
		t.Run(mode, func(t *testing.T) {
			tp, exporter := traceTestProvider(t)
			innerTP := trace.TracerProvider(tp)
			if mode == "unsampled" {
				never := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.NeverSample()), sdktrace.WithSyncer(exporter))
				t.Cleanup(func() { require.NoError(t, never.Shutdown(context.Background())) })
				innerTP = never
			}
			url := traceTestURL(mode == "unsampled" || mode == "recording")
			first := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(url), call: func(context.Context, base.Invocation) result.Result {
				return &result.RPCResult{Err: errors.New("nested failure")}
			}}
			secondURL := traceTestURL(false)
			secondURL.Port = "20302"
			second := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(secondURL), call: func(context.Context, base.Invocation) result.Result {
				return &result.RPCResult{}
			}}
			dir := &traceTestDirectory{Directory: directorybase.NewDirectory(url), invokers: []base.Invoker{first, second}}
			if mode == "new invocation no provider" {
				dir.invokers = nil
			}
			inner := traceTestFailover(t, dir)
			parentInv := invocation.NewRPCInvocation("Parent", nil, nil)
			childInv := base.Invocation(parentInv)
			if mode == "new invocation" || mode == "new invocation no provider" {
				childInv = invocation.NewRPCInvocation("Child", nil, nil)
			}
			parent := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(traceTestURL(true)), call: func(ctx context.Context, _ base.Invocation) result.Result {
				parentObserver := cluster.DiagnosticsFromContext(ctx, parentInv)
				require.NotNil(t, parentObserver)
				var res result.Result
				if mode == "new invocation" || mode == "new invocation no provider" {
					res = inner.Invoke(ctx, childInv)
				} else {
					res = (&clusterInterceptor{TracerProvider: innerTP}).Invoke(ctx, inner, childInv)
				}
				if mode == "new invocation no provider" {
					require.Error(t, res.Error())
				} else {
					require.NoError(t, res.Error())
				}
				require.Same(t, parentObserver, cluster.DiagnosticsFromContext(ctx, parentInv))
				return &result.RPCResult{}
			}}
			require.NoError(t, (&clusterInterceptor{TracerProvider: tp}).Invoke(context.Background(), parent, parentInv).Error())
			spans := exporter.GetSpans()
			if mode == "recording" {
				require.Len(t, spans, 2)
				require.Len(t, spanEvents(spans[0], "cluster.select"), 2)
				require.Len(t, spanEvents(spans[0], "cluster.retry"), 1)
				require.EqualValues(t, 1, spanValue(spans[0], "cluster.retry_count").AsInt64())
			} else {
				require.Len(t, spans, 1)
			}
			parentSpan := spans[len(spans)-1]
			require.Equal(t, "Parent cluster", parentSpan.Name)
			require.Empty(t, parentSpan.Events)
			require.EqualValues(t, 0, spanValue(parentSpan, "cluster.retry_count").AsInt64())
			require.Equal(t, codes.Ok, parentSpan.Status.Code)
		})
	}
}

func TestClusterLogicalRetryBoundaries(t *testing.T) {
	for _, mode := range []string{"empty directory", "destroyed", "skipped selection"} {
		t.Run(mode, func(t *testing.T) {
			tp, exporter := traceTestProvider(t)
			url := traceTestURL(true)
			dir := &traceTestDirectory{Directory: directorybase.NewDirectory(url)}
			var invoker base.Invoker
			calls := 0
			first := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(url), call: func(context.Context, base.Invocation) result.Result {
				calls++
				switch mode {
				case "empty directory":
					dir.invokers = nil
				case "destroyed":
					invoker.Destroy()
				}
				return &result.RPCResult{Err: errors.New("first attempt failed")}
			}}
			dir.invokers = []base.Invoker{first}
			listCalls := 0
			if mode == "skipped selection" {
				unavailableURL := traceTestURL(false)
				unavailableURL.Port = "20402"
				unavailable := base.NewBaseInvoker(unavailableURL)
				unavailable.SetAvailable(false)
				successURL := traceTestURL(false)
				successURL.Port = "20403"
				success := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(successURL), call: func(context.Context, base.Invocation) result.Result {
					calls++
					return &result.RPCResult{}
				}}
				dir.list = func(base.Invocation) []base.Invoker {
					listCalls++
					switch listCalls {
					case 1:
						return []base.Invoker{first}
					case 2:
						return []base.Invoker{unavailable}
					default:
						return []base.Invoker{success}
					}
				}
			}
			invoker = traceTestFailover(t, dir)
			res := (&clusterInterceptor{TracerProvider: tp}).Invoke(context.Background(), invoker, invocation.NewRPCInvocation("Call", nil, nil))
			require.Len(t, exporter.GetSpans(), 1)
			span := exporter.GetSpans()[0]
			if mode == "skipped selection" {
				require.NoError(t, res.Error())
				require.Equal(t, 2, calls)
				require.Equal(t, 3, listCalls)
				require.EqualValues(t, 1, spanValue(span, "cluster.retry_count").AsInt64())
				retries := spanEvents(span, "cluster.retry")
				require.Len(t, retries, 1)
				require.Contains(t, retries[0].Attributes, attribute.Int("attempt", 2))
				selections := spanEvents(span, "cluster.select")
				require.Len(t, selections, 2)
				require.Contains(t, selections[1].Attributes, attribute.Int("attempt", 3))
				require.Equal(t, "cluster.select", span.Events[0].Name)
				require.Equal(t, "cluster.retry", span.Events[1].Name)
				require.Equal(t, "cluster.select", span.Events[2].Name)
			} else {
				require.Error(t, res.Error())
				require.Equal(t, 1, calls)
				require.EqualValues(t, 0, spanValue(span, "cluster.retry_count").AsInt64())
				require.Empty(t, spanEvents(span, "cluster.retry"))
				if mode == "empty directory" {
					require.Len(t, spanEvents(span, "cluster.no_provider"), 1)
				}
			}
		})
	}
}

func TestClusterLogicalCompletionRace(t *testing.T) {
	tp, exporter := traceTestProvider(t)
	interceptor := &clusterInterceptor{TracerProvider: tp}
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			invoker := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(traceTestURL(true)), call: func(context.Context, base.Invocation) result.Result {
				go cancel()
				return &result.RPCResult{}
			}}
			interceptor.Invoke(ctx, invoker, invocation.NewRPCInvocation("Call", nil, nil))
		})
	}
	wg.Wait()
	require.Eventually(t, func() bool { return len(exporter.GetSpans()) == 50 }, time.Second, time.Millisecond)
}

func TestClusterLogicalDisabled(t *testing.T) {
	tp, exporter := traceTestProvider(t)
	response := &result.RPCResult{}
	ctx := context.Background()
	inv := invocation.NewRPCInvocation("Call", nil, nil)
	invoker := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(traceTestURL(false)), call: func(got context.Context, gotInv base.Invocation) result.Result {
		if got != ctx || gotInv != inv {
			panic("disabled tracing must pass through unchanged")
		}
		return response
	}}
	interceptor := &clusterInterceptor{TracerProvider: tp}
	require.Same(t, response, interceptor.Invoke(ctx, invoker, inv))
	require.Zero(t, testing.AllocsPerRun(100, func() { interceptor.Invoke(ctx, invoker, inv) }))
	require.Empty(t, exporter.GetSpans())
	require.Nil(t, inv.Attachments())
	for _, filters := range []string{"", "-otelClientTrace", "otherotelClientTrace", "otelClientTraceOther",
		"otelClientTrace,-otelClientTrace", "-otelClientTrace,otelClientTrace",
		"other, otelClientTrace , -otelClientTrace ,more"} {
		url := traceTestURL(false)
		url.SetParam(constant.ReferenceFilterKey, filters)
		require.False(t, clientTracingEnabled(url))
	}
	require.False(t, clientTracingEnabled(nil))
	url := traceTestURL(false)
	url.SetParam(constant.ReferenceFilterKey, "other, otelClientTrace ,more")
	require.True(t, clientTracingEnabled(url))
}

func TestClusterLogicalPanicEndsSpan(t *testing.T) {
	tp, exporter := traceTestProvider(t)
	invoker := &traceTestInvoker{BaseInvoker: base.NewBaseInvoker(traceTestURL(true)), call: func(context.Context, base.Invocation) result.Result { panic("private panic") }}
	require.PanicsWithValue(t, "private panic", func() {
		(&clusterInterceptor{TracerProvider: tp}).Invoke(context.Background(), invoker, invocation.NewRPCInvocation("Call", nil, nil))
	})
	require.Len(t, exporter.GetSpans(), 1)
	require.Equal(t, codes.Error, exporter.GetSpans()[0].Status.Code)
	require.NotContains(t, fmt.Sprint(exporter.GetSpans()), "private panic")
}

func TestTraceErrorTypes(t *testing.T) {
	cause := errors.New("cause")
	tests := []struct {
		err  error
		want string
	}{
		{nil, "unknown"}, {cause, "unknown"},
		{fmt.Errorf("wrapped: %w", context.DeadlineExceeded), "timeout"},
		{context.Canceled, "unknown"},
		{tri.NewError(tri.CodeBizError, context.Canceled), "business_failed"},
		{tri.NewError(tri.CodeResourceExhausted, context.Canceled), "limit"},
		{tri.NewError(tri.CodeUnavailable, context.Canceled), "service_unavailable"},
		{tri.NewError(tri.CodeInternal, context.DeadlineExceeded), "timeout"},
		{tri.NewError(tri.CodePermissionDenied, context.DeadlineExceeded), "service_unavailable"},
		{tri.NewError(tri.CodeResourceExhausted, context.DeadlineExceeded), "limit"},
		{tri.NewError(tri.CodeDeadlineExceeded, rpcerror.Mark(cause, rpcerror.Network)), "timeout"},
		{tri.NewError(tri.CodeResourceExhausted, rpcerror.Mark(cause, rpcerror.Codec)), "limit"},
		{tri.NewError(tri.CodePermissionDenied, rpcerror.Mark(cause, rpcerror.Network)), "service_unavailable"},
		{tri.NewError(tri.CodeUnavailable, cause), "service_unavailable"},
		{tri.NewError(tri.CodeBizError, rpcerror.Mark(cause, rpcerror.Network)), "business_failed"},
		{rpcerror.Mark(cause, rpcerror.Network), "network_failure"},
		{tri.NewError(tri.CodeUnavailable, rpcerror.Mark(cause, rpcerror.Network)), "network_failure"},
		{rpcerror.Mark(cause, rpcerror.Codec), "codec"},
		{tri.NewError(tri.CodeInternal, cause), "unknown"},
		{errors.Join(rpcerror.Mark(cause, rpcerror.Network), rpcerror.Mark(cause, rpcerror.Codec)), "unknown"},
	}
	for _, test := range tests {
		require.Equal(t, test.want, errorType(test.err), "%v", test.err)
	}
}

var _ cluster.Diagnostics = (*logicalSpan)(nil)
