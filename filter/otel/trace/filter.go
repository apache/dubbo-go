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
)

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

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

	ctx, span := tracer.Start(
		trace.ContextWithRemoteSpanContext(ctx, spanCtx),
		invocation.ActualMethodName(),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.RPCSystemApacheDubbo,
			semconv.RPCService(invoker.GetURL().ServiceKey()),
			semconv.RPCMethod(invocation.MethodName()),
		),
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

	var span trace.Span
	ctx, span = tracer.Start(
		ctx,
		invocation.ActualMethodName(),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.RPCSystemApacheDubbo,
			semconv.RPCService(invoker.GetURL().ServiceKey()),
			semconv.RPCMethod(invocation.MethodName()),
		),
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
