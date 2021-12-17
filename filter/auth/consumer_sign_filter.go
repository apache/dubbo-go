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

package auth

import (
	"context"
	"fmt"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	signOnce sync.Once
	sign     *signFilter
)

func init() {
	extension.SetFilter(constant.AuthConsumerFilterKey, newSignFilter)
}

// signFilter signs the request on consumer side
type signFilter struct{}

func newSignFilter() filter.Filter {
	if sign == nil {
		signOnce.Do(func() {
			sign = &signFilter{}
		})
	}
	return sign
}

// Invoke retrieves the configured Authenticator to add signature to invocation
func (sf *signFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	url := invoker.GetURL()

	err := doAuthWork(url, func(authenticator filter.Authenticator) error {
		return authenticator.Sign(invocation, url)
	})
	if err != nil {
		panic(fmt.Sprintf("Sign for invocation %s # %s failed", url.ServiceKey(), invocation.MethodName()))
	}
	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (sf *signFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}
