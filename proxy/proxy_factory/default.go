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
	"errors"
	"fmt"
	"reflect"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/proxy"
)

func init() {
	extension.SetProxyFactory("default", NewDefaultProxyFactory)
}

// DefaultProxyFactory is the default proxy factory
type DefaultProxyFactory struct { // delegate ProxyFactory
}

// you can rewrite DefaultProxyFactory in extension and delegate the default proxy factory like below

//func WithDelegate(delegateProxyFactory ProxyFactory) Option {
//	return func(proxy ProxyFactory) {
//		proxy.(*DefaultProxyFactory).delegate = delegateProxyFactory
//	}
//}

// NewDefaultProxyFactory returns a proxy factory instance
func NewDefaultProxyFactory(_ ...proxy.Option) proxy.ProxyFactory {
	return &DefaultProxyFactory{}
}

// GetProxy gets a proxy
func (factory *DefaultProxyFactory) GetProxy(invoker base.Invoker, url *common.URL) *proxy.Proxy {
	return factory.GetAsyncProxy(invoker, nil, url)
}

// GetAsyncProxy gets a async proxy
func (factory *DefaultProxyFactory) GetAsyncProxy(invoker base.Invoker, callBack any, url *common.URL) *proxy.Proxy {
	// create proxy
	attachments := map[string]string{}
	attachments[constant.AsyncKey] = url.GetParam(constant.AsyncKey, "false")
	attachments[constant.EnvironmentKey] = url.GetParam(constant.EnvironmentKey, "dev")
	return proxy.NewProxy(invoker, callBack, attachments)
}

// GetInvoker gets a invoker
func (factory *DefaultProxyFactory) GetInvoker(url *common.URL) base.Invoker {
	if url.Protocol == constant.TriProtocol || (url.SubURL != nil && url.SubURL.Protocol == constant.TriProtocol) {
		if info, ok := url.GetAttribute(constant.ServiceInfoKey); ok {
			svc, _ := url.GetAttribute(constant.RpcServiceKey)
			return newInfoInvoker(url, info.(*common.ServiceInfo), svc)
		}
	}
	return &ProxyInvoker{
		BaseInvoker: *base.NewBaseInvoker(url),
	}
}

// ProxyInvoker is a invoker struct
type ProxyInvoker struct {
	base.BaseInvoker
}

// Invoke is used to call service method by invocation
func (pi *ProxyInvoker) Invoke(ctx context.Context, invocation base.Invocation) result.Result {
	result := &result.RPCResult{}
	result.SetAttachments(invocation.Attachments())

	// get providerUrl. The origin url may be is registry URL.
	url := getProviderURL(pi.GetURL())

	methodName := invocation.MethodName()
	proto := url.Protocol
	path := strings.TrimPrefix(url.Path, "/")
	args := invocation.Arguments()

	// get service
	svc := common.ServiceMap.GetServiceByServiceKey(proto, url.ServiceKey())
	if svc == nil {
		logger.Errorf("cannot find service [%s] in %s", path, proto)
		result.SetError(perrors.Errorf("cannot find service [%s] in %s", path, proto))
		return result
	}

	// get method
	method := svc.Method()[methodName]
	if method == nil {
		logger.Errorf("cannot find method [%s] of service [%s] in %s", methodName, path, proto)
		result.SetError(perrors.Errorf("cannot find method [%s] of service [%s] in %s", methodName, path, proto))
		return result
	}

	in := []reflect.Value{svc.Rcvr()}
	if method.CtxType() != nil {
		ctx = context.WithValue(ctx, constant.AttachmentKey, invocation.Attachments())
		in = append(in, method.SuiteContext(ctx))
	}

	// prepare argv
	if (len(method.ArgsType()) == 1 || len(method.ArgsType()) == 2 && method.ReplyType() == nil) && method.ArgsType()[0].String() == "[]interface {}" {
		in = append(in, reflect.ValueOf(args))
	} else {
		for i := 0; i < len(args); i++ {
			t := reflect.ValueOf(args[i])
			if !t.IsValid() {
				at := method.ArgsType()[i]
				if at.Kind() == reflect.Ptr {
					at = at.Elem()
				}
				t = reflect.New(at)
			}
			in = append(in, t)
		}
	}

	// prepare replyv
	var replyv reflect.Value
	var retErr any

	returnValues, callErr := callLocalMethod(method.Method(), in)

	if callErr != nil {
		logger.Errorf("Invoke function error: %+v, service: %#v", callErr, url)
		result.SetError(callErr)
		return result
	}

	if len(returnValues) == 1 {
		retErr = returnValues[0].Interface()
	} else {
		replyv = returnValues[0]
		retErr = returnValues[1].Interface()
	}

	if retErr != nil {
		result.SetError(retErr.(error))
		return result
	}
	if replyv.IsValid() && (replyv.Kind() != reflect.Ptr || replyv.Kind() == reflect.Ptr && replyv.Elem().IsValid()) {
		result.SetResult(replyv.Interface())
	}

	return result
}

func getProviderURL(url *common.URL) *common.URL {
	if url.SubURL == nil {
		return url
	}
	return url.SubURL
}

type infoProxyInvoker struct {
	base.BaseInvoker
	info      *common.ServiceInfo
	svc       common.RPCService
	methodMap map[string]*common.MethodInfo
}

func (tpi *infoProxyInvoker) init() {
	methodMap := make(map[string]*common.MethodInfo)
	for i := range tpi.info.Methods {
		methodMap[tpi.info.Methods[i].Name] = &tpi.info.Methods[i]
	}
	tpi.methodMap = methodMap
}

func (tpi *infoProxyInvoker) Invoke(ctx context.Context, invocation base.Invocation) result.Result {
	name := invocation.MethodName()
	args := invocation.Arguments()
	result := new(result.RPCResult)
	if method, ok := tpi.methodMap[name]; ok {
		res, err := method.MethodFunc(ctx, args, tpi.svc)
		result.SetResult(res)
		if err != nil {
			var proError *triple_protocol.Error
			if !errors.As(err, &proError) {
				err = triple_protocol.NewError(triple_protocol.CodeBizError, err)
			} else if proError.Code() != triple_protocol.CodeBizError {
				err = triple_protocol.NewError(proError.Code(), proError.Unwrap())
			}
			result.SetError(err)
		}
		return result
	}
	result.SetError(fmt.Errorf("no match method for %s", name))

	return result
}

func newInfoInvoker(url *common.URL, info *common.ServiceInfo, svc common.RPCService) base.Invoker {
	invoker := &infoProxyInvoker{
		BaseInvoker: *base.NewBaseInvoker(url),
		info:        info,
		svc:         svc,
	}
	invoker.init()
	return invoker
}
