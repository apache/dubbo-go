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

	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
)

const (
	tracingFilterName = "tracing"
)

func init() {
	extension.SetFilter(tracingFilterName, NewTracingFilter)
}

type TracingFilter struct {
}

func (tf *TracingFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {

	// invoker.GetUrl().Context()

	operationName := invoker.GetUrl().ServiceKey() + invocation.MethodName()

	span, ctx := opentracing.StartSpanFromContext(invocation.Context(), operationName)

	defer span.Finish()
}

func (tf *TracingFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	panic("implement me")
}

var (
	tracingFilterInstance *TracingFilter
)

func NewTracingFilter() filter.Filter {
	if tracingFilterInstance == nil {
		tracingFilterInstance = &TracingFilter{}
	}
	return tracingFilterInstance
}
