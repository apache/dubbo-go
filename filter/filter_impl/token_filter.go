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
	"strings"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
)

const (
	// nolint
	TOKEN = "token"
)

func init() {
	extension.SetFilter(TOKEN, GetTokenFilter)
}

// TokenFilter will verify if the token is valid
type TokenFilter struct{}

// Invoke verifies the incoming token with the service configured token
func (tf *TokenFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	invokerTkn := invoker.GetUrl().GetParam(constant.TOKEN_KEY, "")
	if len(invokerTkn) > 0 {
		attachs := invocation.Attachments()
		remoteTkn, exist := attachs[constant.TOKEN_KEY]
		if exist && remoteTkn != nil && strings.EqualFold(invokerTkn, remoteTkn.(string)) {
			return invoker.Invoke(ctx, invocation)
		}
		return &protocol.RPCResult{Err: perrors.Errorf("Invalid token! Forbid invoke remote service %v method %s ",
			invoker, invocation.MethodName())}
	}

	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (tf *TokenFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

// nolint
func GetTokenFilter() filter.Filter {
	return &TokenFilter{}
}
