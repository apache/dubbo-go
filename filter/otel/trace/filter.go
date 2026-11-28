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
	"dubbo.apache.org/dubbo-go/v3/internal/rpcexemplar"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

func init() {
	// TODO: use single filter to simplify filter field in configuration
	extension.SetFilter(constant.OTELServerTraceKey, func() filter.Filter {
		return &otelServerFilter{
			Propagators:      otel.GetTextMapPropagator(),
			TracerProvider:   otel.GetTracerProvider(),
			captureExemplars: rpcexemplar.Enabled(),
		}
	})
	extension.SetFilter(constant.OTELClientTraceKey, func() filter.Filter {
		return &otelClientFilter{
			Propagators:      otel.GetTextMapPropagator(),
			TracerProvider:   otel.GetTracerProvider(),
			captureExemplars: rpcexemplar.Enabled(),
		}
	})
}

func rpcSpanAttributes(serviceURL *common.URL, methodName string) []attribute.KeyValue {
	attributes := []attribute.KeyValue{
		semconv.RPCSystemApacheDubbo,
		semconv.RPCService(serviceURL.Service()),
		semconv.RPCMethod(methodName),
	}
	if group := serviceURL.Group(); group != "" {
		attributes = append(attributes, attribute.Key(constant.DubboGroupKey).String(group))
	}
	if version := serviceURL.Version(); version != "" {
		attributes = append(attributes, attribute.Key(constant.DubboVersionKey).String(version))
	}
	return attributes
}

var _ filter.Filter = (*otelServerFilter)(nil)

// otelServerFilter implements server-side tracing for Dubbo requests
// by creating and managing trace spans using the configured propagator
// and tracer provider.
type otelServerFilter struct {
	Propagators      propagation.TextMapPropagator
	TracerProvider   trace.TracerProvider
	captureExemplars bool
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

	// The filter runs before the generic service filter resolves a $invoke
	// call, so name the span and set rpc.method from the actual method.
	methodName := invocation.ActualMethodName()
	ctx, span := tracer.Start(
		trace.ContextWithRemoteSpanContext(ctx, spanCtx),
		methodName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(rpcSpanAttributes(invoker.GetURL(), methodName)...),
	)
	defer span.End()

	if f.captureExemplars {
		var release func()
		ctx, release = rpcexemplar.WithSpan(ctx, invocation, span, trace.SpanKindServer)
		defer release()
	}
	res := invoker.Invoke(ctx, invocation)

	if res.Error() != nil {
		recordSpanError(span, res.Error(), errorType(res.Error()))
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
	Propagators      propagation.TextMapPropagator
	TracerProvider   trace.TracerProvider
	captureExemplars bool
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
		trace.WithAttributes(rpcSpanAttributes(invoker.GetURL(), invocation.MethodName())...),
	)
	if f.captureExemplars {
		var release func()
		ctx, release = rpcexemplar.WithSpan(ctx, invocation, span, trace.SpanKindClient)
		defer release()
	}
	if base.IsStreamingCall(invocation) {
		return f.invokeStream(ctx, span, invoker, invocation)
	}
	defer span.End()

	f.inject(ctx, invocation)
	res := invoker.Invoke(ctx, invocation)
	setClientSpanStatus(span, res.Error())
	return res
}

// invokeStream ends the span of an open stream when the stream ends.
func (f *otelClientFilter) invokeStream(ctx context.Context, span trace.Span, invoker base.Invoker, invocation base.Invocation) result.Result {
	ctx, streamClaimed := base.WatchStream(ctx, invocation, func(err error) {
		setClientSpanStatus(span, err)
		span.End()
	})
	defer func() {
		if !streamClaimed() {
			span.End()
		}
	}()

	f.inject(ctx, invocation)
	res := invoker.Invoke(ctx, invocation)
	if !streamClaimed() {
		setClientSpanStatus(span, res.Error())
	}
	return res
}

func (f *otelClientFilter) inject(ctx context.Context, invocation base.Invocation) {
	attachments := invocation.Attachments()
	if attachments == nil {
		attachments = map[string]any{}
	}
	Inject(ctx, attachments, f.Propagators)
	for k, v := range attachments {
		invocation.SetAttachment(k, v)
	}
}

func setClientSpanStatus(span trace.Span, err error) {
	if err != nil {
		recordSpanError(span, err, errorType(err))
	} else {
		span.SetStatus(codes.Ok, codes.Ok.String())
	}
}
