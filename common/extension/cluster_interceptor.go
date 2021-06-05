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
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster"
)

var (
	lock         sync.RWMutex
	interceptors = make(map[string]func() cluster.Interceptor)
)

// SetClusterInterceptor sets cluster interceptor so that user has chance to inject extra logics before and after
// cluster invoker
func SetClusterInterceptor(name string, fun func() cluster.Interceptor) {
	lock.Lock()
	defer lock.Unlock()
	interceptors[name] = fun
}

// GetClusterInterceptor returns the cluster interceptor instance with the given name
func GetClusterInterceptor(name string) cluster.Interceptor {
	lock.RLock()
	defer lock.RUnlock()
	if interceptors[name] == nil {
		panic("cluster_interceptor for " + name + " doesn't exist, make sure the corresponding package is imported")
	}
	return interceptors[name]()
}

// GetClusterInterceptors returns all instances of registered cluster interceptors
func GetClusterInterceptors() []cluster.Interceptor {
	lock.RLock()
	defer lock.RUnlock()
	ret := make([]cluster.Interceptor, 0, len(interceptors))
	for _, f := range interceptors {
		ret = append(ret, f())
	}
	return ret
}
