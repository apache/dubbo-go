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

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/filter/generic"
	_ "dubbo.apache.org/dubbo-go/v3/filter/otel/trace"
	"dubbo.apache.org/dubbo-go/v3/internal/rpcexemplar"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsprom "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func TestFilterToRPCExemplars(t *testing.T) {
	const child = "DUBBO_EXEMPLAR_INTEGRATION_CHILD"
	if os.Getenv(child) == "" {
		for _, mode := range []string{"enabled", "disabled"} {
			t.Run(mode, func(t *testing.T) { runHistogramChild(t, child, mode) })
		}
		return
	}

	enabled := os.Getenv(child) == "enabled"
	registry := prom.NewRegistry()
	metricsURL := common.NewURLWithOptions(
		common.WithProtocol("exemplar-fixture"),
		common.WithParamsValue(constant.HistogramEnabledKey, "true"),
		common.WithParamsValue(constant.HistogramExemplarsKey, strconv.FormatBool(enabled)),
	)
	metrics.SetRegistry(metricsURL.Protocol, func(u *common.URL) metrics.MetricRegistry {
		return metricsprom.NewPromMetricRegistry(registry, u)
	})
	metrics.Init(metricsURL)
	t.Cleanup(func() {
		metrics.SubscribeRecorder(constant.MetricsRpc, nil)
	})

	spans := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()), sdktrace.WithSpanProcessor(spans))
	otel.SetTracerProvider(provider)
	histogram := func(method string) string {
		t.Helper()
		request := httptest.NewRequest("GET", "/metrics", nil)
		request.Header.Set("Accept", "application/openmetrics-text")
		response := httptest.NewRecorder()
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{EnableOpenMetrics: true}).ServeHTTP(response, request)
		require.Equal(t, 200, response.Code)
		var buckets []string
		for line := range strings.SplitSeq(response.Body.String(), "\n") {
			if strings.Contains(line, "_histogram_seconds_bucket{") && strings.Contains(line, fmt.Sprintf("method=%q", method)) {
				buckets = append(buckets, line)
			}
		}
		return strings.Join(buckets, "\n")
	}

	for _, scenario := range []struct {
		method, chainKey, tracingFilter string
		role                            common.RoleType
	}{
		{"ClientRPC", constant.ReferenceFilterKey, constant.OTELClientTraceKey, common.CONSUMER},
		{"ServerRPC", constant.ServiceFilterKey, constant.OTELServerTraceKey, common.PROVIDER},
	} {
		t.Run(scenario.method, func(t *testing.T) {
			url := common.NewURLWithOptions(
				common.WithPath("example.Exemplar"),
				common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(int(scenario.role))),
				common.WithParamsValue(scenario.chainKey, scenario.tracingFilter+","+constant.MetricsFilterKey),
			)
			endpoint := &fixtureInvoker{BaseInvoker: base.NewBaseInvoker(url), err: errors.New("first attempt failed")}
			chain := protocolwrapper.BuildInvokerChain(endpoint, scenario.chainKey)
			inv := invocation.NewRPCInvocation(scenario.method, nil, nil)
			ctx, parent := provider.Tracer("fixture").Start(context.Background(), "business")
			defer parent.End()
			for _, attemptErr := range []error{endpoint.err, nil} {
				endpoint.err = attemptErr
				require.Equal(t, attemptErr, chain.Invoke(ctx, inv).Error())
				observed := histogram(scenario.method)
				require.NotEmpty(t, observed)
				if enabled {
					current := spans.Ended()[len(spans.Ended())-1].SpanContext()
					require.NotEqual(t, parent.SpanContext().SpanID(), current.SpanID())
					require.Contains(t, observed, fmt.Sprintf("trace_id=%q", current.TraceID().String()))
					require.Contains(t, observed, fmt.Sprintf("span_id=%q", current.SpanID().String()))
				} else {
					require.NotContains(t, observed, "span_id=")
				}
			}
		})
	}
	if !enabled {
		return
	}

	t.Run("skips inherited or unsampled spans", func(t *testing.T) {
		parentCtx, parent := provider.Tracer("fixture").Start(context.Background(), "business")
		defer parent.End()
		for _, scenario := range []struct {
			method, filters string
			provider        trace.TracerProvider
		}{
			{"MetricsFirst", "metrics,otelClientTrace", provider},
			{"NoopTracer", "otelClientTrace,metrics", noop.NewTracerProvider()},
			{"Unsampled", "otelClientTrace,metrics", sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.NeverSample()))},
		} {
			otel.SetTracerProvider(scenario.provider)
			url := common.NewURLWithOptions(
				common.WithPath("example.Exemplar"),
				common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.CONSUMER)),
				common.WithParamsValue(constant.ReferenceFilterKey, scenario.filters),
			)
			endpoint := &fixtureInvoker{BaseInvoker: base.NewBaseInvoker(url)}
			chain := protocolwrapper.BuildInvokerChain(endpoint, constant.ReferenceFilterKey)
			require.NoError(t, chain.Invoke(parentCtx, invocation.NewRPCInvocation(scenario.method, nil, nil)).Error())
			observed := histogram(scenario.method)
			require.NotEmpty(t, observed)
			require.NotContains(t, observed, "span_id=")
		}
		otel.SetTracerProvider(provider)
	})

	t.Run("generic invocation rewrite", func(t *testing.T) {
		service := &ExemplarEchoService{}
		providerURL := common.NewURLWithOptions(
			common.WithProtocol(constant.TriProtocol),
			common.WithParamsValue(constant.InterfaceKey, service.Reference()),
			common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER)),
			common.WithParamsValue(constant.ServiceFilterKey, "otelServerTrace,generic_service,metrics"),
		)
		_, err := common.ServiceMap.Register(service.Reference(), providerURL.Protocol, "", "", service)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = common.ServiceMap.UnRegister(service.Reference(), providerURL.Protocol, providerURL.ServiceKey())
		})
		consumerURL := common.NewURLWithOptions(
			common.WithPath(service.Reference()),
			common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.CONSUMER)),
			common.WithParamsValue(constant.GenericKey, constant.GenericSerializationDefault),
			common.WithParamsValue(constant.ReferenceFilterKey, "otelClientTrace,generic,metrics"),
		)
		for _, scenario := range []struct {
			method, chainKey string
			url              *common.URL
			newInvocation    func(string) *invocation.RPCInvocation
		}{
			{"GenericEcho", constant.ServiceFilterKey, providerURL, genericExemplarInvocation},
			{"GenericConsumer", constant.ReferenceFilterKey, consumerURL, genericExemplarInvocation},
			{"TypedConsumer", constant.ReferenceFilterKey, consumerURL, func(method string) *invocation.RPCInvocation {
				return invocation.NewRPCInvocation(method, []any{"fixture"}, nil)
			}},
		} {
			t.Run(scenario.method, func(t *testing.T) {
				endpoint := &fixtureInvoker{BaseInvoker: base.NewBaseInvoker(scenario.url)}
				chain := protocolwrapper.BuildInvokerChain(endpoint, scenario.chainKey)
				require.NoError(t, chain.Invoke(context.Background(), scenario.newInvocation(scenario.method)).Error())
				ended := spans.Ended()
				current := ended[len(ended)-1].SpanContext()
				methodLabel := scenario.method
				if scenario.chainKey == constant.ReferenceFilterKey {
					methodLabel = constant.Generic
				}
				observed := histogram(methodLabel)
				require.NotEmpty(t, observed)
				require.Contains(t, observed, fmt.Sprintf("trace_id=%q", current.TraceID().String()))
				require.Contains(t, observed, fmt.Sprintf("span_id=%q", current.SpanID().String()))
			})
		}
	})

	t.Run("nested invocation", func(t *testing.T) {
		ctx, parent := provider.Tracer("fixture").Start(context.Background(), "outer RPC")
		defer parent.End()
		outer := invocation.NewRPCInvocation("Outer", nil, nil)
		ctx, release := rpcexemplar.WithSpan(ctx, outer, parent, trace.SpanKindClient)
		defer release()

		url := common.NewURLWithOptions(
			common.WithPath("example.Exemplar"),
			common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.CONSUMER)),
			common.WithParamsValue(constant.ReferenceFilterKey, constant.MetricsFilterKey),
		)
		endpoint := &fixtureInvoker{BaseInvoker: base.NewBaseInvoker(url)}
		chain := protocolwrapper.BuildInvokerChain(endpoint, constant.ReferenceFilterKey)
		nested := invocation.NewRPCInvocation("Nested", nil, nil)
		rewritten := invocation.NewRPCInvocation("Nested", nil, nil)
		ctx = rpcexemplar.Rebind(ctx, nested, rewritten)
		require.NoError(t, chain.Invoke(ctx, rewritten).Error())
		observed := histogram("Nested")
		require.NotEmpty(t, observed)
		require.NotContains(t, observed, "span_id=")
	})

	t.Run("asynchronous completion", func(t *testing.T) {
		integrationApplication = "exemplar-fixture"
		for _, expire := range []bool{false, true} {
			method := fmt.Sprintf("AsyncExpire%v", expire)
			url := fixtureURL(t, "127.0.0.1:1")
			url.SetParam(constant.ReferenceFilterKey, "otelClientTrace,metrics")
			transport := &controlledTransport{sent: make(chan int64, 1)}
			exchange := remoting.NewExchangeClient(url, transport, time.Second, false)
			endpoint := dubbo.NewDubboInvoker(url, exchange)
			chain := protocolwrapper.BuildInvokerChain(endpoint, constant.ReferenceFilterKey)
			called := make(chan common.CallbackResponse, 1)
			inv := invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName(method), invocation.WithReply(new(string)),
				invocation.WithAttachment(constant.AsyncKey, "true"),
				invocation.WithCallBack(func(response common.CallbackResponse) { called <- response }),
			)
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			require.NoError(t, chain.Invoke(ctx, inv).Error())
			id := <-transport.sent
			span := spans.Ended()[len(spans.Ended())-1].SpanContext()
			require.Empty(t, histogram(method))
			if expire {
				cancel()
				await(t, func() bool { return metricValue(t, registry, "requests_processing_total", method) == 0 })
			}
			(&remoting.Response{ID: id, Result: &result.RPCResult{}}).Handle()
			assertCallbackCauseUnchanged(t, called)
			if expire {
				require.Empty(t, histogram(method))
			} else {
				require.Contains(t, histogram(method), fmt.Sprintf("span_id=%q", span.SpanID().String()))
				cancel()
			}
		}
	})
}

type ExemplarEchoService struct{}

func (*ExemplarEchoService) Reference() string { return "example.ExemplarEcho" }

func (*ExemplarEchoService) GenericEcho(value string) (string, error) { return value, nil }

func genericExemplarInvocation(method string) *invocation.RPCInvocation {
	return invocation.NewRPCInvocation(constant.Generic,
		[]any{method, []string{"java.lang.String"}, []hessian.Object{"fixture"}},
		map[string]any{constant.GenericKey: constant.GenericSerializationDefault},
	)
}
