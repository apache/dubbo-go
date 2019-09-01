/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package impl

import (
	"fmt"
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
	token = "token"
)

func init() {
	extension.SetFilter(token, GetTokenFilter)
}

type TokenFilter struct{}

func (tf *TokenFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	fmt.Printf("token filter with invocation: %s \n", invocation)
	invokerTkn := invoker.GetUrl().GetParam(constant.TOKEN_KEY, "")
	if len(invokerTkn) > 0 {
		attachs := invocation.Attachments()
		if len(attachs) > 0 {
			remoteTkn, exist := attachs[constant.TOKEN_KEY]
			fmt.Printf("url token: %s, invocation token: %s \n", invokerTkn, remoteTkn)
			if exist && strings.EqualFold(invokerTkn, remoteTkn) {
				return invoker.Invoke(invocation)
			}
		}
		return &protocol.RPCResult{Err: perrors.Errorf("Invalid token! Forbid invoke remote service %s method %s ",
			invoker, invocation.MethodName())}
	}

	return invoker.Invoke(invocation)
}

func (tf *TokenFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func GetTokenFilter() filter.Filter {
	return &TokenFilter{}
}
