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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"go.opentelemetry.io/otel/trace"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

const (
	adminAlignmentInterface = "org.apache.dubbo.samples.OrderService"
	adminAlignmentGroup     = "gray"
	adminAlignmentVersion   = "1.0.0"
	adminAlignmentMethod    = "PlaceOrder"
)

func TestRpcSpanAttributesExcludesGroupAndVersion(t *testing.T) {
	t.Parallel()

	urlWithInterface := common.NewURLWithOptions(
		common.WithInterface(adminAlignmentInterface),
		common.WithParamsValue(constant.GroupKey, adminAlignmentGroup),
		common.WithParamsValue(constant.VersionKey, adminAlignmentVersion),
	)
	pathFallbackURL := common.NewURLWithOptions(
		common.WithPath(adminAlignmentInterface),
		common.WithParamsValue(constant.GroupKey, adminAlignmentGroup),
		common.WithParamsValue(constant.VersionKey, adminAlignmentVersion),
	)

	assert.Equal(t, "gray/org.apache.dubbo.samples.OrderService:1.0.0", urlWithInterface.ServiceKey())

	attrs := attributesAsMap(rpcSpanAttributes(urlWithInterface, adminAlignmentMethod))
	assert.Equal(t, adminAlignmentInterface, attrs[string(semconv.RPCServiceKey)])
	assert.Equal(t, urlWithInterface.Service(), attrs[string(semconv.RPCServiceKey)])
	assert.NotEqual(t, urlWithInterface.ServiceKey(), attrs[string(semconv.RPCServiceKey)])
	assert.Equal(t, adminAlignmentGroup, attrs[constant.DubboGroupKey])
	assert.Equal(t, adminAlignmentVersion, attrs[constant.DubboVersionKey])

	pathAttrs := attributesAsMap(rpcSpanAttributes(pathFallbackURL, adminAlignmentMethod))
	assert.Equal(t, adminAlignmentInterface, pathAttrs[string(semconv.RPCServiceKey)])
}

func TestOtelServerFilterRPCServiceMatchesMetricInterface(t *testing.T) {
	t.Parallel()
	assertRPCServiceOnSpan(t, func(tp trace.TracerProvider) func(context.Context, base.Invoker, base.Invocation) result.Result {
		return (&otelServerFilter{
			Propagators:    getFields().Propagators,
			TracerProvider: tp,
		}).Invoke
	}, trace.SpanKindServer)
}

func TestOtelClientFilterRPCServiceMatchesMetricInterface(t *testing.T) {
	t.Parallel()
	assertRPCServiceOnSpan(t, func(tp trace.TracerProvider) func(context.Context, base.Invoker, base.Invocation) result.Result {
		return (&otelClientFilter{
			Propagators:    getFields().Propagators,
			TracerProvider: tp,
		}).Invoke
	}, trace.SpanKindClient)
}

func assertRPCServiceOnSpan(
	t *testing.T,
	newInvoke func(tp trace.TracerProvider) func(context.Context, base.Invoker, base.Invocation) result.Result,
	wantKind trace.SpanKind,
) {
	t.Helper()

	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})

	urls := []*common.URL{
		common.NewURLWithOptions(
			common.WithProtocol("tri"),
			common.WithInterface(adminAlignmentInterface),
			common.WithParamsValue(constant.GroupKey, adminAlignmentGroup),
			common.WithParamsValue(constant.VersionKey, adminAlignmentVersion),
			common.WithParamsValue(constant.ApplicationKey, "order-service"),
		),
		common.NewURLWithOptions(
			common.WithProtocol("tri"),
			common.WithPath(adminAlignmentInterface),
			common.WithParamsValue(constant.GroupKey, adminAlignmentGroup),
			common.WithParamsValue(constant.VersionKey, adminAlignmentVersion),
		),
	}

	invoke := newInvoke(tp)
	for _, url := range urls {
		invoker := base.NewBaseInvoker(url)
		inv := invocation.NewRPCInvocation(adminAlignmentMethod, nil, map[string]any{})
		res := invoke(context.Background(), invoker, inv)
		require.NoError(t, res.Error())
	}

	ended := spanRecorder.Ended()
	require.Len(t, ended, len(urls))

	for i, span := range ended {
		url := urls[i]
		assert.Equal(t, adminAlignmentMethod, span.Name())
		assert.Equal(t, wantKind, span.SpanKind())

		attrs := attributesAsMap(span.Attributes())
		rpcService := attrs[string(semconv.RPCServiceKey)]
		require.Equal(t, adminAlignmentInterface, rpcService)
		assert.Equal(t, url.Service(), rpcService)
		assert.NotEqual(t, url.ServiceKey(), rpcService)
		assert.NotContains(t, rpcService, adminAlignmentGroup+"/")
		assert.NotContains(t, rpcService, ":"+adminAlignmentVersion)
		assert.Equal(t, "apache_dubbo", attrs[string(semconv.RPCSystemKey)])
		assert.Equal(t, adminAlignmentMethod, attrs[string(semconv.RPCMethodKey)])
	}
}

func attributesAsMap(attrs []attribute.KeyValue) map[string]string {
	out := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		out[string(attr.Key)] = attr.Value.AsString()
	}
	return out
}
