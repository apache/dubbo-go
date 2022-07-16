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
)

var (
	authOnce sync.Once
	auth     *authFilter
)

func init() {
	extension.SetFilter(constant.AuthProviderFilterKey, newAuthFilter)
}

// authFilter verifies the correctness of the signature on provider side
type authFilter struct{}

func newAuthFilter() filter.Filter {
	if auth == nil {
		authOnce.Do(func() {
			auth = &authFilter{}
		})
	}
	return auth
}

// Invoke retrieves the configured Authenticator to verify the signature in an invocation
func (paf *authFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	url := invoker.GetURL()

	err := doAuthWork(url, func(authenticator filter.Authenticator) error {
		return authenticator.Authenticate(invocation, url)
	})
	if err != nil {
		logger.Errorf("auth the request: %v occur exception, cause: %s", invocation, err.Error())
		return &protocol.RPCResult{
			Err: err,
		}
	}

	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (paf *authFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func newProviderAuthFilter() filter.Filter {
	return &authFilter{}
}
