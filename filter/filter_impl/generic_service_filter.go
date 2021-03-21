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
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/mitchellh/mapstructure"
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
	logger.Infof("invoking generic service filter.")
	logger.Debugf("generic service filter methodName:%v,args:%v", invocation.MethodName(), len(invocation.Arguments()))

	if invocation.MethodName() != constant.GENERIC || len(invocation.Arguments()) != 3 {
		return invoker.Invoke(ctx, invocation)
	}

	var (
		ok         bool
		err        error
		methodName string
		newParams  []interface{}
		genericKey string
		argsType   []reflect.Type
		oldParams  []hessian.Object
	)

	url := invoker.GetUrl()
	methodName = invocation.Arguments()[0].(string)
	// get service
	svc := common.ServiceMap.GetServiceByServiceKey(url.Protocol, url.ServiceKey())
	// get method
	method := svc.Method()[methodName]
	if method == nil {
		logger.Errorf("[Generic Service Filter] Don't have this method: %s", methodName)
		return &protocol.RPCResult{}
	}
	argsType = method.ArgsType()
	genericKey = invocation.AttachmentsByKey(constant.GENERIC_KEY, GENERIC_SERIALIZATION_DEFAULT)
	if genericKey == GENERIC_SERIALIZATION_DEFAULT {
		oldParams, ok = invocation.Arguments()[2].([]hessian.Object)
	} else {
		logger.Errorf("[Generic Service Filter] Don't support this generic: %s", genericKey)
		return &protocol.RPCResult{}
	}
	if !ok {
		logger.Errorf("[Generic Service Filter] wrong serialization")
		return &protocol.RPCResult{}
	}
	if len(oldParams) != len(argsType) {
		logger.Errorf("[Generic Service Filter] method:%s invocation arguments number was wrong", methodName)
		return &protocol.RPCResult{}
	}
	// oldParams convert to newParams
	newParams = make([]interface{}, len(oldParams))
	for i := range argsType {
		newParam := reflect.New(argsType[i]).Interface()
		err = mapstructure.Decode(oldParams[i], newParam)
		newParam = reflect.ValueOf(newParam).Elem().Interface()
		if err != nil {
			logger.Errorf("[Generic Service Filter] decode arguments map to struct wrong: error{%v}", perrors.WithStack(err))
			return &protocol.RPCResult{}
		}
		newParams[i] = newParam
	}
	newInvocation := invocation2.NewRPCInvocation(methodName, newParams, invocation.Attachments())
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
		result.SetResult(struct2MapAll(v.Interface()))
	}
	return result
}

// nolint
func GetGenericServiceFilter() filter.Filter {
	return &GenericServiceFilter{}
}
