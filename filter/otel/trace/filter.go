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
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	instrumentationName = "dubbo.apache.org/dubbo-go/v3/oteldubbo"
)

func init() {
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

type otelServerFilter struct {
	Propagators    propagation.TextMapPropagator
	TracerProvider trace.TracerProvider
}

func (f *otelServerFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, protocol protocol.Invocation) protocol.Result {
	return result
}

func (f *otelServerFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	attachments := invocation.Attachments()
	bags, spanCtx := Extract(ctx, attachments, f.Propagators)
	ctx = baggage.ContextWithBaggage(ctx, bags)

	tracer := f.TracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(SemVersion()),
	)

	ctx, span := tracer.Start(
		trace.ContextWithRemoteSpanContext(ctx, spanCtx),
		invocation.ActualMethodName(),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			RPCSystemDubbo,
			semconv.RPCServiceKey.String(invoker.GetURL().ServiceKey()),
			semconv.RPCMethodKey.String(invocation.MethodName()),
		),
	)
	defer span.End()

	result := invoker.Invoke(ctx, invocation)

	if result.Error() != nil {
		span.SetStatus(codes.Error, result.Error().Error())
	} else {
		span.SetStatus(codes.Ok, codes.Ok.String())
	}
	return result
}

type otelClientFilter struct {
	Propagators    propagation.TextMapPropagator
	TracerProvider trace.TracerProvider
}

func (f *otelClientFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, protocol protocol.Invocation) protocol.Result {
	return result
}

func (f *otelClientFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	tracer := f.TracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(SemVersion()),
	)

	var span trace.Span
	ctx, span = tracer.Start(
		ctx,
		invocation.ActualMethodName(),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			RPCSystemDubbo,
			semconv.RPCServiceKey.String(invoker.GetURL().ServiceKey()),
			semconv.RPCMethodKey.String(invocation.MethodName()),
		),
	)
	defer span.End()

	attachments := invocation.Attachments()
	if attachments == nil {
		attachments = map[string]interface{}{}
	}
	Inject(ctx, attachments, f.Propagators)
	for k, v := range attachments {
		invocation.SetAttachment(k, v)
	}
	result := invoker.Invoke(ctx, invocation)

	if result.Error() != nil {
		span.SetStatus(codes.Error, result.Error().Error())
	} else {
		span.SetStatus(codes.Ok, codes.Ok.String())
	}
	return result
}
