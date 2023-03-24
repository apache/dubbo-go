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

package active

import (
	"context"
	"strconv"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation2 "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

const (
	dubboInvokeStartTime = "dubboInvokeStartTime"
)

var (
	once   sync.Once
	active *activeFilter
)

func init() {
	extension.SetFilter(constant.ActiveFilterKey, newActiveFilter)
}

// Filter tracks the requests status
type activeFilter struct{}

func newActiveFilter() filter.Filter {
	if active == nil {
		once.Do(func() {
			active = &activeFilter{}
		})
	}
	return active
}

// Invoke starts to record the requests status
func (f *activeFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	invocation.(*invocation2.RPCInvocation).SetAttachment(dubboInvokeStartTime, strconv.FormatInt(protocol.CurrentTimeMillis(), 10))
	protocol.BeginCount(invoker.GetURL(), invocation.MethodName())
	return invoker.Invoke(ctx, invocation)
}

// OnResponse update the active count base on the request result.
func (f *activeFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	startTime, err := strconv.ParseInt(invocation.(*invocation2.RPCInvocation).GetAttachmentWithDefaultValue(dubboInvokeStartTime, "0"), 10, 64)
	if err != nil {
		result.SetError(err)
		logger.Errorf("parse dubbo_invoke_start_time to int64 failed")
		return result
	}
	elapsed := protocol.CurrentTimeMillis() - startTime
	protocol.EndCount(invoker.GetURL(), invocation.MethodName(), elapsed, result.Error() == nil)
	return result
}
