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
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/proxy"
	"github.com/apache/dubbo-go/protocol"
)

func init() {
	extension.SetProxyFactory("default", NewDefaultProxyFactory)
}

type DefaultProxyFactory struct {
	//delegate ProxyFactory
}

//you can rewrite DefaultProxyFactory in extension and delegate the default proxy factory like below

//func WithDelegate(delegateProxyFactory ProxyFactory) Option {
//	return func(proxy ProxyFactory) {
//		proxy.(*DefaultProxyFactory).delegate = delegateProxyFactory
//	}
//}

func NewDefaultProxyFactory(options ...proxy.Option) proxy.ProxyFactory {
	return &DefaultProxyFactory{}
}
func (factory *DefaultProxyFactory) GetProxy(invoker protocol.Invoker, url *common.URL) *proxy.Proxy {
	//create proxy
	attachments := map[string]string{}
	attachments[constant.ASYNC_KEY] = url.GetParam(constant.ASYNC_KEY, "false")
	return proxy.NewProxy(invoker, nil, attachments)
}
func (factory *DefaultProxyFactory) GetInvoker(url common.URL) protocol.Invoker {
	//TODO:yincheng need to do the service invoker refactor
	return protocol.NewBaseInvoker(url)
}
