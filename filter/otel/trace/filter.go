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
	"strconv"
)

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// buildSpanName returns the standardized span name for a Dubbo invocation,
// e.g. "dubbo.consumer <service>/<method>" or "dubbo.provider <service>/<method>".
func buildSpanName(side string, url *common.URL, invocation base.Invocation) string {
	return "dubbo." + side + " " + url.ServiceKey() + "/" + invocation.MethodName()
}

// buildSpanAttributes collects the semantic attributes for a Dubbo span.
func buildSpanAttributes(side string, url *common.URL, invocation base.Invocation) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.RPCSystemApacheDubbo,
		semconv.RPCService(url.ServiceKey()),
		semconv.RPCMethod(invocation.MethodName()),
		DubboSideKey.String(side),
	}
	if url.Protocol != "" {
		attrs = append(attrs, DubboProtocolKey.String(url.Protocol))
	}
	if group := url.Group(); group != "" {
		attrs = append(attrs, DubboGroupKey.String(group))
	}
	if version := url.Version(); version != "" {
		attrs = append(attrs, DubboVersionKey.String(version))
	}
	if url.Ip != "" {
		attrs = append(attrs, semconv.ServerAddress(url.Ip))
	}
	if port, err := strconv.Atoi(url.Port); err == nil {
		attrs = append(attrs, semconv.ServerPort(port))
	}
	return attrs
}

func init() {
	// TODO: use single filter to simplify filter field in configuration
	extension.SetFilter(constant.OTELServerTraceKey, func() filter.Filter {
		return &otelServerFilter{
			Propagators:    otel.GetTextMapPropagator(),
			TracerProvider: otel.GetTracerProvider(),
		}
	})
	extension.SetFilter(constant.OTELClientTraceKey, func() filter.Filter {
		return &otelClientFilter{
			Propagators:    otel.GetTextMapPropagator(),
			TracerProvider: otel.GetTracerProvider(),
		}
	})
}

var _ filter.Filter = (*otelServerFilter)(nil)

// otelServerFilter implements server-side tracing for Dubbo requests
// by creating and managing trace spans using the configured propagator
// and tracer provider.
type otelServerFilter struct {
	Propagators    propagation.TextMapPropagator
	TracerProvider trace.TracerProvider
}

func (f *otelServerFilter) OnResponse(ctx context.Context, result result.Result, invoker base.Invoker, protocol base.Invocation) result.Result {
	return result
}

func (f *otelServerFilter) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	attachments := invocation.Attachments()
	bags, spanCtx := Extract(ctx, attachments, f.Propagators)
	ctx = baggage.ContextWithBaggage(ctx, bags)

	tracer := f.TracerProvider.Tracer(
		constant.TraceScopeName,
		trace.WithInstrumentationVersion(constant.Version),
	)

	url := invoker.GetURL()
	ctx, span := tracer.Start(
		trace.ContextWithRemoteSpanContext(ctx, spanCtx),
		buildSpanName(sideProvider, url, invocation),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(buildSpanAttributes(sideProvider, url, invocation)...),
	)
	defer span.End()

	res := invoker.Invoke(ctx, invocation)

	if res.Error() != nil {
		span.SetStatus(codes.Error, res.Error().Error())
	} else {
		span.SetStatus(codes.Ok, codes.Ok.String())
	}
	return res
}

var _ filter.Filter = (*otelClientFilter)(nil)

// otelClientFilter implements client-side tracing for Dubbo requests
// by creating and managing trace spans using the configured propagator
// and tracer provider.
type otelClientFilter struct {
	Propagators    propagation.TextMapPropagator
	TracerProvider trace.TracerProvider
}

func (f *otelClientFilter) OnResponse(ctx context.Context, result result.Result, invoker base.Invoker, protocol base.Invocation) result.Result {
	return result
}

func (f *otelClientFilter) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	tracer := f.TracerProvider.Tracer(
		constant.TraceScopeName,
		trace.WithInstrumentationVersion(constant.Version),
	)

	url := invoker.GetURL()
	var span trace.Span
	ctx, span = tracer.Start(
		ctx,
		buildSpanName(sideConsumer, url, invocation),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(buildSpanAttributes(sideConsumer, url, invocation)...),
	)
	defer span.End()

	attachments := invocation.Attachments()
	if attachments == nil {
		attachments = map[string]any{}
	}
	Inject(ctx, attachments, f.Propagators)
	for k, v := range attachments {
		invocation.SetAttachment(k, v)
	}
	res := invoker.Invoke(ctx, invocation)

	if res.Error() != nil {
		span.SetStatus(codes.Error, res.Error().Error())
	} else {
		span.SetStatus(codes.Ok, codes.Ok.String())
	}
	return res
}
