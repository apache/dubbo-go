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

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
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

type tracingFilter struct {
}

func (tf *tracingFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {

	operationName := invoker.GetUrl().ServiceKey() + invocation.MethodName()
	// withTimeout after we support the timeout between different ends.
	invCtx, cancel := context.WithCancel(invocation.Context())
	span, spanCtx := opentracing.StartSpanFromContext(invCtx, operationName)
	invocation.SetContext(common.NewContextWith(spanCtx))
	defer func() {
		span.Finish()
		cancel()
	}()

	result := invoker.Invoke(invocation)
	span.SetTag(successKey, result.Error() != nil)
	if result.Error() != nil {
		span.LogFields(log.String(errorKey, result.Error().Error()))
	}
	return result
}

func (tf *tracingFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

var (
	tracingFilterInstance *tracingFilter
)

func newTracingFilter() filter.Filter {
	if tracingFilterInstance == nil {
		tracingFilterInstance = &tracingFilter{}
	}
	return tracingFilterInstance
}
