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
	"reflect"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/filter/generic/generalizer"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation2 "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func init() {
	extension.SetFilter(constant.GenericServiceFilterKey, func() filter.Filter {
		return &ServiceFilter{}
	})
}

// ServiceFilter is for Server
type ServiceFilter struct{}

// Invoke is used to call service method by invocation
func (f *ServiceFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if invocation.MethodName() != constant.GENERIC ||
		invocation.Arguments() == nil || len(invocation.Arguments()) != 3 {
		return invoker.Invoke(ctx, invocation)
	}

	// get real invocation info from the generic invocation
	mtdname := invocation.Arguments()[0].(string)
	types := invocation.Arguments()[1]
	args := invocation.Arguments()[2].([]interface{})

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
		logger.Errorf("\"%s\" method is not found, service key: %s", mtdname, ivkUrl.ServiceKey())
		return &protocol.RPCResult{}
	}
	argsType := method.ArgsType()

	// get generic info from attachments of invocation, the default value is "true"
	generic := invocation.AttachmentsByKey(constant.GENERIC_KEY, constant.GenericSerializationDefault)

	// get generalizer according to value in the `generic`
	var g generalizer.Generalizer
	switch strings.ToLower(generic) {
	case constant.GenericSerializationDefault:
		g = generalizer.GetMapGeneralizer()
	case constant.GenericSerializationProtobuf:
		panic("implement me")
	default:
		logger.Infof("\"%s\" is not supported, use the default generalizer(MapGeneralizer)", generic)
		g = generalizer.GetMapGeneralizer()
	}

	if len(args) != len(argsType) {
		logger.Errorf("the number of args(=%d) should be equals to argsType(=%d)", len(args), len(argsType))
		return &protocol.RPCResult{}
	}

	// realize
	newargs := make([]interface{}, len(argsType))
	for i := 0; i < len(argsType); i++ {
		newarg, err := g.Realize(args[i], argsType[i])
		if err != nil {
			logger.Errorf("realization failed, %v", err)
			return &protocol.RPCResult{}
		}
		newargs[i] = newarg
	}

	// build a normal invocation
	newivc := invocation2.NewRPCInvocation(mtdname, newargs, invocation.Attachments())
	newivc.SetReply(invocation.Reply())

	return invoker.Invoke(ctx, newivc)
}

// nolint
func (f *ServiceFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if invocation.MethodName() == constant.GENERIC && len(invocation.Arguments()) == 3 && result.Result() != nil {
		v := reflect.ValueOf(result.Result())
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		g := generalizer.GetMapGeneralizer()
		obj, _ := g.Generalize(v.Interface())
		result.SetResult(obj)
	}
	return result
}
