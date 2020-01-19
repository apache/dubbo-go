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
	"strconv"
)

import (
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
	invocation2 "github.com/apache/dubbo-go/protocol/invocation"
)

const (
	active                  = "active"
	dubbo_invoke_start_time = "dubbo_invoke_start_time"
)

func init() {
	extension.SetFilter(active, GetActiveFilter)
}

type ActiveFilter struct {
}

func (ef *ActiveFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("invoking active filter. %v,%v", invocation.MethodName(), len(invocation.Arguments()))

	invocation.(*invocation2.RPCInvocation).SetAttachments(dubbo_invoke_start_time, strconv.FormatInt(protocol.CurrentTimeMillis(), 10))
	protocol.BeginCount(invoker.GetUrl(), invocation.MethodName())
	return invoker.Invoke(invocation)
}

func (ef *ActiveFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {

	startTime, err := strconv.ParseInt(invocation.(*invocation2.RPCInvocation).AttachmentsByKey(dubbo_invoke_start_time, "0"), 10, 64)
	if err != nil {
		panic("parse dubbo_invoke_start_time to int64 failed")
	}
	elapsed := protocol.CurrentTimeMillis() - startTime
	protocol.EndCount(invoker.GetUrl(), invocation.MethodName(), elapsed, result.Error() == nil)
	return result
}

func GetActiveFilter() filter.Filter {
	return &ActiveFilter{}
}
