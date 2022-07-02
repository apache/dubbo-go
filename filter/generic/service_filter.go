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

package generic

import (
	"context"
	"sync"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation2 "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

var (
	serviceGenericOnce sync.Once
	serviceGeneric     *genericServiceFilter
)

func init() {
	extension.SetFilter(constant.GenericServiceFilterKey, newGenericServiceFilter)
}

// genericServiceFilter is for Server
type genericServiceFilter struct{}

func newGenericServiceFilter() filter.Filter {
	if serviceGeneric == nil {
		serviceGenericOnce.Do(func() {
			serviceGeneric = &genericServiceFilter{}
		})
	}
	return serviceGeneric
}
func (f *genericServiceFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if !invocation.IsGenericInvocation() {
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
	generic := invocation.GetAttachmentWithDefaultValue(constant.GenericKey, constant.GenericSerializationDefault)
	// get generalizer according to value in the `generic`
	g := getGeneralizer(generic)

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

func (f *genericServiceFilter) OnResponse(_ context.Context, result protocol.Result, _ protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if invocation.IsGenericInvocation() && result.Result() != nil {
		// get generic info from attachments of invocation, the default value is "true"
		generic := invocation.GetAttachmentWithDefaultValue(constant.GenericKey, constant.GenericSerializationDefault)
		// get generalizer according to value in the `generic`
		g := getGeneralizer(generic)

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
