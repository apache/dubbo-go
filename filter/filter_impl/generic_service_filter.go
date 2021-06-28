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
	"reflect"
)

import (
	"github.com/mitchellh/mapstructure"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/filter/filter_impl/generic"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation2 "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

const (
	// GenericService defines the filter name
	GenericService = "generic_service"
)

func init() {
	extension.SetFilter(GenericService, GetGenericServiceFilter)
}

// nolint
type GenericServiceFilter struct{}

// Invoke is used to call service method by invocation
func (ef *GenericServiceFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("invoking generic service filter.")
	logger.Debugf("generic service filter methodName:%v,args:%v", invocation.MethodName(), len(invocation.Arguments()))

	if invocation.MethodName() != constant.GENERIC || len(invocation.Arguments()) != 3 {
		return invoker.Invoke(ctx, invocation)
	}

	var (
		methodName     string
		genericKey     string
		invocationArgs []interface{}
		argsType       []reflect.Type
	)

	genericKey = invocation.AttachmentsByKey(constant.GENERIC_KEY, constant.GENERIC_SERIALIZATION_DEFAULT)
	processor := extension.GetGenericProcessor(genericKey)
	if processor == nil {
		logger.Errorf("[Generic Service Filter] Don't support this generic: %s", genericKey)
		return &protocol.RPCResult{}
	}
	newArgs, err := processor.Deserialize(invocation.Arguments()[2])
	if err != nil {
		logger.Errorf("[Generic Service Filter] Deserialization error")
		return &protocol.RPCResult{}
	}
	// check method and arguments
	url := invoker.GetURL()
	methodName = invocation.Arguments()[0].(string)
	svc := common.ServiceMap.GetServiceByServiceKey(url.Protocol, url.ServiceKey())
	method := svc.Method()[methodName]
	if method == nil {
		logger.Errorf("[Generic Service Filter] Don't have this method: %s, "+
			"make sure you have import the package and register it by invoking extension.SetGenericProcessor.", methodName)
		return &protocol.RPCResult{}
	}
	argsType = method.ArgsType()
	if len(newArgs) != len(argsType) {
		logger.Errorf("[Generic Service Filter] method:%s invocation arguments number was wrong", methodName)
		return &protocol.RPCResult{}
	}
	// newArgs convert to invocationArgs
	invocationArgs = make([]interface{}, len(newArgs))
	for i := range argsType {
		newArgument := reflect.New(argsType[i]).Interface()
		err = mapstructure.Decode(newArgs[i], newArgument)
		newArgument = reflect.ValueOf(newArgument).Elem().Interface()
		if err != nil {
			logger.Errorf("[Generic Service Filter] decode arguments map to struct wrong: error{%v}", perrors.WithStack(err))
			return &protocol.RPCResult{}
		}
		invocationArgs[i] = newArgument
	}
	logger.Debugf("[Generic Service Filter] invocationArgs: %v", invocationArgs)
	newInvocation := invocation2.NewRPCInvocation(methodName, invocationArgs, invocation.Attachments())
	newInvocation.SetReply(invocation.Reply())
	return invoker.Invoke(ctx, newInvocation)
}

// nolint
func (ef *GenericServiceFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if invocation.MethodName() == constant.GENERIC && len(invocation.Arguments()) == 3 && result.Result() != nil {
		v := reflect.ValueOf(result.Result())
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		result.SetResult(generic.Struct2MapAll(v.Interface()))
	}
	return result
}

// nolint
func GetGenericServiceFilter() filter.Filter {
	return &GenericServiceFilter{}
}
