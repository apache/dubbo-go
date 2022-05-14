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
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
	invocation2 "github.com/apache/dubbo-go/protocol/invocation"
)

const (
	// GENERIC_SERVICE defines the filter name
	GENERIC_SERVICE = "generic_service"
	// nolint
	GENERIC_SERIALIZATION_DEFAULT = "true"
)

func init() {
	extension.SetFilter(GENERIC_SERVICE, GetGenericServiceFilter)
}

// nolint
type GenericServiceFilter struct{}

// Invoke is used to call service method by invocation
func (ef *GenericServiceFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if invocation.MethodName() != constant.GENERIC || len(invocation.Arguments()) != 3 {
		return invoker.Invoke(ctx, invocation)
	}

	// get real invocation info from the generic invocation
	mtdname := invocation.Arguments()[0].(string)
	// types are not required in dubbo-go, for dubbo-go client to dubbo-go server, types could be nil
	types := invocation.Arguments()[1]
	args := invocation.Arguments()[2].([]hessian.Object)

	logger.Debugf(`received a generic invocation: 
		MethodName: %s,
		Types: %s,
		Args: %s
	`, mtdname, types, args)

	// get the type of the argument
	ivkUrl := invoker.GetURL()
	svc := common.ServiceMap.GetServiceByServiceKey(ivkUrl.Protocol, ivkUrl.ServiceKey())
	method := svc.Method()[mtdname]
	if method == nil {
		return &protocol.RPCResult{
			Err: perrors.Errorf("\"%s\" method is not found, service key: %s", mtdname, ivkUrl.ServiceKey()),
		}
	}
	argsType := method.ArgsType()

	// get generic info from attachments of invocation, the default value is "true"
	//generic := invocation.AttachmentsByKey(constant.GENERIC_KEY, GENERIC_SERIALIZATION_DEFAULT)
	// get generalizer according to value in the `generic`
	g := GetMapGeneralizer()

	if len(args) != len(argsType) {
		return &protocol.RPCResult{
			Err: perrors.Errorf("the number of args(=%d) is not matched with \"%s\" method", len(args), mtdname),
		}
	}

	// realize
	newargs := make([]interface{}, len(argsType))
	for i := 0; i < len(argsType); i++ {
		newarg, err := g.Realize(args[i], argsType[i])
		if err != nil {
			return &protocol.RPCResult{
				Err: perrors.Errorf("realization failed, %v", err),
			}
		}
		newargs[i] = newarg
	}

	// build a normal invocation
	newivc := invocation2.NewRPCInvocation(mtdname, newargs, invocation.Attachments())
	newivc.SetReply(invocation.Reply())

	return invoker.Invoke(ctx, newivc)
}

// nolint
func (ef *GenericServiceFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if invocation.MethodName() == constant.GENERIC && len(invocation.Arguments()) == 3 && result.Result() != nil {
		// get generic info from attachments of invocation, the default value is "true"
		//generic := invocation.AttachmentsByKey(constant.GENERIC_KEY, constant.GenericSerializationDefault)
		// get generalizer according to value in the `generic`
		g := GetMapGeneralizer()

		obj, err := g.Generalize(result.Result())
		if err != nil {
			err = perrors.Errorf("generalizaion failed, %v", err)
			result.SetError(err)
			result.SetResult(nil)
			return result
		}
		result.SetResult(obj)
	}
	return result
}

// nolint
func GetGenericServiceFilter() filter.Filter {
	return &GenericServiceFilter{}
}
