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

package filter_impl

import (
	"context"
)

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	tracingFilterName = "tracing"
)

// this should be executed before users set their own Tracer
func init() {
	extension.SetFilter(tracingFilterName, newTracingFilter)
	opentracing.SetGlobalTracer(opentracing.NoopTracer{})
}

var (
	errorKey   = "ErrorMsg"
	successKey = "Success"
)

// if you wish to using opentracing, please add the this filter into your filter attribute in your configure file.
// notice that this could be used in both client-side and server-side.
type tracingFilter struct{}

func (tf *tracingFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	var (
		spanCtx context.Context
		span    opentracing.Span
	)
	operationName := invoker.GetURL().ServiceKey() + "#" + invocation.MethodName()

	wiredCtx := ctx.Value(constant.TRACING_REMOTE_SPAN_CTX)
	preSpan := opentracing.SpanFromContext(ctx)

	if preSpan != nil {
		// it means that someone already create a span to trace, so we use the span to be the parent span
		span = opentracing.StartSpan(operationName, opentracing.ChildOf(preSpan.Context()))
		spanCtx = opentracing.ContextWithSpan(ctx, span)

	} else if wiredCtx != nil {

		// it means that there has a remote span, usually from client side. so we use this as the parent
		span = opentracing.StartSpan(operationName, opentracing.ChildOf(wiredCtx.(opentracing.SpanContext)))
		spanCtx = opentracing.ContextWithSpan(ctx, span)
	} else {
		// it means that there is not any span, so we create a span as the root span.
		span, spanCtx = opentracing.StartSpanFromContext(ctx, operationName)
	}

	defer func() {
		span.Finish()
	}()

	result := invoker.Invoke(spanCtx, invocation)
	span.SetTag(successKey, result.Error() == nil)
	if result.Error() != nil {
		span.LogFields(log.String(errorKey, result.Error().Error()))
	}
	return result
}

func (tf *tracingFilter) OnResponse(ctx context.Context, result protocol.Result,
	invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

var tracingFilterInstance *tracingFilter

func newTracingFilter() filter.Filter {
	if tracingFilterInstance == nil {
		tracingFilterInstance = &tracingFilter{}
	}
	return tracingFilterInstance
}
