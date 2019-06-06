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

package extension

import (
	"github.com/apache/dubbo-go/common/proxy"
)

var (
	proxy_factories = make(map[string]func(...proxy.Option) proxy.ProxyFactory)
)

func SetProxyFactory(name string, f func(...proxy.Option) proxy.ProxyFactory) {
	proxy_factories[name] = f
}
func GetProxyFactory(name string) proxy.ProxyFactory {
	if name == "" {
		name = "default"
	}
	if proxy_factories[name] == nil {
		panic("proxy factory for " + name + " is not existing, make sure you have import the package.")
	}
	return proxy_factories[name]()
}
