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

package generic_test

import (
	"context"
	"errors"
	"testing"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	prom "github.com/prometheus/client_golang/prometheus"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/filter"
	_ "dubbo.apache.org/dubbo-go/v3/filter/generic"
	_ "dubbo.apache.org/dubbo-go/v3/filter/metrics"
	_ "dubbo.apache.org/dubbo-go/v3/filter/otel/trace"
	metricapi "dubbo.apache.org/dubbo-go/v3/metrics"
	metricsProm "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	_ "dubbo.apache.org/dubbo-go/v3/metrics/rpc"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type chainInvoker struct {
	base.Invoker
	fn func(context.Context, base.Invocation) result.Result
}

func (c chainInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	return c.fn(ctx, inv)
}

func wrapFilter(f filter.Filter, next base.Invoker) base.Invoker {
	return chainInvoker{
		Invoker: next,
		fn: func(ctx context.Context, inv base.Invocation) result.Result {
			res := f.Invoke(ctx, next, inv)
			return f.OnResponse(ctx, res, next, inv)
		},
	}
}

func assertSpanCodecAttribution(t *testing.T, recorder *tracetest.SpanRecorder) {
	t.Helper()
	spans := recorder.Ended()
	if len(spans) != 1 || len(spans[0].Events()) != 1 {
		t.Fatalf("expected 1 span with 1 exception event, got %d spans", len(spans))
	}
	attrs := make(map[string]string, len(spans[0].Attributes()))
	for _, attr := range spans[0].Attributes() {
		attrs[string(attr.Key)] = attr.Value.AsString()
	}
	if attrs["error.type"] != "codec" || attrs["rpc.error.origin"] != "codec" {
		t.Fatalf("expected trace error.type=codec and rpc.error.origin=codec, got %v", attrs)
	}
}

func gatherFailureCounts(t *testing.T, registry *prom.Registry) (codecTotal, timeoutTotal float64) {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		switch family.GetName() {
		case "dubbo_consumer_requests_failed_codec_total":
			for _, m := range family.Metric {
				codecTotal += m.GetCounter().GetValue()
			}
		case "dubbo_consumer_requests_failed_timeout_total":
			for _, m := range family.Metric {
				timeoutTotal += m.GetCounter().GetValue()
			}
		}
	}
	return codecTotal, timeoutTotal
}

func runGenericTelemetryChainCase(t *testing.T, registry *prom.Registry, inv base.Invocation) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	prevTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() {
		otel.SetTracerProvider(prevTP)
		_ = tp.Shutdown(context.Background())
	}()

	beforeCodec, beforeTimeout := gatherFailureCounts(t, registry)

	invokeURL, err := common.NewURL("dubbo://127.0.0.1:20000/GenericFixture?registry.role=0&generic=true")
	if err != nil {
		t.Fatal(err)
	}
	leaf := chainInvoker{
		Invoker: base.NewBaseInvoker(invokeURL),
		fn: func(ctx context.Context, forwarded base.Invocation) result.Result {
			if forwarded == inv {
				t.Fatal("generic filter did not replace invocation")
			}
			obs := rpcerror.ClaimObservation(ctx, forwarded)
			if obs == nil {
				t.Fatal("failed to claim rebound observation")
			}
			obs.BindSession("session")
			obs.Record("session", rpcerror.Mark(errors.New("decode response failed"), rpcerror.Codec))
			return &result.RPCResult{Err: context.DeadlineExceeded}
		},
	}

	metricsFilter, ok := extension.GetFilter(constant.MetricsFilterKey)
	if !ok {
		t.Fatal("metrics filter not registered")
	}
	genericFilter, ok := extension.GetFilter(constant.GenericFilterKey)
	if !ok {
		t.Fatal("generic filter not registered")
	}
	traceFilter, ok := extension.GetFilter(constant.OTELClientTraceKey)
	if !ok {
		t.Fatal("otelClientTrace filter not registered")
	}

	// Real filter chain: otelClientTrace -> generic -> metrics -> leaf
	chain := wrapFilter(traceFilter, wrapFilter(genericFilter, wrapFilter(metricsFilter, leaf)))
	chain.Invoke(context.Background(), inv)

	assertSpanCodecAttribution(t, recorder)
	afterCodec, afterTimeout := gatherFailureCounts(t, registry)
	if afterCodec-beforeCodec != 1 || afterTimeout-beforeTimeout != 0 {
		t.Fatalf("expected codec delta=1 timeout delta=0, got codec=%v timeout=%v", afterCodec-beforeCodec, afterTimeout-beforeTimeout)
	}
}

func TestGenericFilterRebindsObservationAcrossTraceAndMetricsChain(t *testing.T) {
	registry := prom.NewRegistry()
	metricapi.SetRegistry("generic-chain-test", func(u *common.URL) metricapi.MetricRegistry {
		return metricsProm.NewPromMetricRegistry(registry, u)
	})
	metricsURL, err := common.NewURL("generic-chain-test://127.0.0.1?application=generic-chain-consumer")
	if err != nil {
		t.Fatal(err)
	}
	metricapi.Init(metricsURL)

	cases := []struct {
		name string
		inv  base.Invocation
	}{
		{
			name: "typed method",
			inv:  invocation.NewRPCInvocation("Hello", []any{"arg1"}, make(map[string]any)),
		},
		{
			name: "generic method",
			inv: invocation.NewRPCInvocation(
				constant.Generic,
				[]any{"hello", []string{"java.lang.String"}, []hessian.Object{"arg1"}},
				make(map[string]any),
			),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runGenericTelemetryChainCase(t, registry, tc.inv)
		})
	}
}
