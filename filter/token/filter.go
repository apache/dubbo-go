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

// Package token provides token filter.
package token

import (
	"context"
	"strings"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	once  sync.Once
	token *tokenFilter
)

func init() {
	extension.SetFilter(constant.TokenFilterKey, newTokenFilter)
}

const (
	InValidTokenFormat = "[Token Filter]Invalid token! Forbid invoke remote service %v with method %s"
)

// tokenFilter will verify if the token is valid
type tokenFilter struct{}

func newTokenFilter() filter.Filter {
	if token == nil {
		once.Do(func() {
			token = &tokenFilter{}
		})
	}
	return token
}

// Invoke verifies the incoming token with the service configured token
func (f *tokenFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	invokerTkn := invoker.GetURL().GetParam(constant.TokenKey, "")
	if len(invokerTkn) > 0 {
		attas := invocation.Attachments()
		var remoteTkn string
		remoteTknIface, exist := attas[constant.TokenKey]
		if !exist || remoteTknIface == nil {
			return &protocol.RPCResult{Err: perrors.Errorf(InValidTokenFormat, invoker, invocation.MethodName())}
		}
		switch remoteTknIface.(type) {
		case string:
			// deal with dubbo protocol
			remoteTkn = remoteTknIface.(string)
		case []string:
			// deal with triple protocol
			remoteTkns := remoteTknIface.([]string)
			if len(remoteTkns) != 1 {
				return &protocol.RPCResult{Err: perrors.Errorf(InValidTokenFormat, invoker, invocation.MethodName())}
			}
			remoteTkn = remoteTkns[0]
		default:
			return &protocol.RPCResult{Err: perrors.Errorf(InValidTokenFormat, invoker, invocation.MethodName())}
		}

		if strings.EqualFold(invokerTkn, remoteTkn) {
			return invoker.Invoke(ctx, invocation)
		}
		return &protocol.RPCResult{Err: perrors.Errorf(InValidTokenFormat, invoker, invocation.MethodName())}
	}

	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (f *tokenFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}
