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

package proxy_factory

import (
	"context"
	"reflect"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/proxy"
)

func init() {
	extension.SetProxyFactory(constant.PassThroughProxyFactoryKey, NewPassThroughProxyFactory)
}

// PassThroughProxyFactory is the factory of PassThroughProxyInvoker
type PassThroughProxyFactory struct {
}

// NewPassThroughProxyFactory returns a proxy factory instance
func NewPassThroughProxyFactory(_ ...proxy.Option) proxy.ProxyFactory {
	return &PassThroughProxyFactory{}
}

// GetProxy gets a proxy
func (factory *PassThroughProxyFactory) GetProxy(invoker protocol.Invoker, url *common.URL) *proxy.Proxy {
	return factory.GetAsyncProxy(invoker, nil, url)
}

// GetAsyncProxy gets a async proxy
func (factory *PassThroughProxyFactory) GetAsyncProxy(invoker protocol.Invoker, callBack interface{}, url *common.URL) *proxy.Proxy {
	//create proxy
	attachments := map[string]string{}
	attachments[constant.AsyncKey] = url.GetParam(constant.AsyncKey, "false")
	return proxy.NewProxy(invoker, callBack, attachments)
}

// GetInvoker gets a invoker
func (factory *PassThroughProxyFactory) GetInvoker(url *common.URL) protocol.Invoker {
	return &PassThroughProxyInvoker{
		ProxyInvoker: &ProxyInvoker{
			BaseInvoker: *protocol.NewBaseInvoker(url),
		},
	}
}

// PassThroughProxyInvoker is a invoker struct, it calls service with specific method 'Serivce' and params:
// Service(method string, argsTypes []string, args [][]byte, attachment map[string]interface{})
// PassThroughProxyInvoker pass through raw invocation data and method name to service, which will deal with them.
type PassThroughProxyInvoker struct {
	*ProxyInvoker
}

// Invoke is used to call service method by invocation
func (pi *PassThroughProxyInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	result := &protocol.RPCResult{}
	result.SetAttachments(invocation.Attachments())
	url := getProviderURL(pi.GetURL())

	arguments := invocation.Arguments()
	srv := common.ServiceMap.GetServiceByServiceKey(url.Protocol, url.ServiceKey())

	var args [][]byte
	if len(arguments) > 0 {
		args = make([][]byte, 0, len(arguments))
		for _, arg := range arguments {
			if v, ok := arg.([]byte); ok {
				args = append(args, v)
			} else {
				result.Err = perrors.New("the param type is not []byte")
				return result
			}
		}
	}
	method := srv.Method()["Service"]

	in := make([]reflect.Value, 5)
	in = append(in, srv.Rcvr())
	in = append(in, reflect.ValueOf(invocation.MethodName()))
	in = append(in, reflect.ValueOf(invocation.GetAttachmentInterface(constant.ParamsTypeKey)))
	in = append(in, reflect.ValueOf(args))
	in = append(in, reflect.ValueOf(invocation.Attachments()))

	var replyv reflect.Value
	var retErr interface{}

	returnValues, callErr := callLocalMethod(method.Method(), in)

	if callErr != nil {
		logger.Errorf("Invoke function error: %+v, service: %#v", callErr, url)
		result.SetError(callErr)
		return result
	}

	replyv = returnValues[0]
	retErr = returnValues[1].Interface()

	if retErr != nil {
		result.SetError(retErr.(error))
		return result
	}
	if replyv.IsValid() && (replyv.Kind() != reflect.Ptr || replyv.Kind() == reflect.Ptr && replyv.Elem().IsValid()) {
		result.SetResult(replyv.Interface())
	}

	return result
}
