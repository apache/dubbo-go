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

package cluster_impl

import (
	"context"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// InterceptorInvoker mocks cluster interceptor as an invoker
type InterceptorInvoker struct {
	next        protocol.Invoker
	interceptor cluster.Interceptor
}

// GetURL is used to get url from InterceptorInvoker
func (i *InterceptorInvoker) GetURL() *common.URL {
	return i.next.GetURL()
}

// IsAvailable is used to get available status
func (i *InterceptorInvoker) IsAvailable() bool {
	return i.next.IsAvailable()
}

// Invoke is used to call service method by invocation
func (i *InterceptorInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	return i.interceptor.Invoke(ctx, i.next, invocation)
}

// Destroy will destroy invoker
func (i *InterceptorInvoker) Destroy() {
	i.next.Destroy()
}

func buildInterceptorChain(invoker protocol.Invoker, builtins ...cluster.Interceptor) protocol.Invoker {
	// The order of interceptors is from left to right, so loading from right to left
	next := invoker
	interceptors := extension.GetClusterInterceptors()
	if len(interceptors) != 0 {
		for i := len(interceptors) - 1; i >= 0; i-- {
			v := &InterceptorInvoker{next: next, interceptor: interceptors[i]}
			next = v
		}
	}

	if builtins != nil && len(builtins) > 0 {
		for i := len(builtins) - 1; i >= 0; i-- {
			v := &InterceptorInvoker{next: next, interceptor: builtins[i]}
			next = v
		}
	}

	return next
}
