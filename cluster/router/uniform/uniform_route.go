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

package uniform

import (
	"github.com/apache/dubbo-go/config_center"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

const (
	name = "uniform-router"
)

// UniformRouter provides a health-first routing mechanism through ConnChecker
type UniformRouter struct {
	dubboRouter *DubboRouterRule
	notify      chan struct{}
}

func mapCombine(dist map[string]map[string]string, from map[string]map[string]string) {
	for k, v := range from {
		dist[k] = v
	}
}

// NewUniformRouter construct an NewConnCheckRouter via url
func NewUniformRouter(dubboRouter *DubboRouterRule, notify chan struct{}) (*UniformRouter, error) {
	r := &UniformRouter{
		dubboRouter: dubboRouter,
		notify:      notify,
	}
	return r, nil
}

// Route gets a list of routed invoker
func (r *UniformRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	return r.dubboRouter.route(invokers, url, invocation)
}

func (r *UniformRouter) Process(event *config_center.ConfigChangeEvent) {
	// todo deal with router change

}

// Pool separates healthy invokers from others.
func (r *UniformRouter) Pool(invokers []protocol.Invoker) (router.AddrPool, router.AddrMetadata) {
	rb := make(router.AddrPool, 8)
	//rb[uniformSelected] = roaring.NewBitmap()
	//for i, invoker := range invokers {
	//	if r.checker.IsConnHealthy(invoker) {
	//		rb[connHealthy].Add(uint32(i))
	//	}
	//}
	return rb, nil
}

// ShouldPool will always return true to make sure healthy check constantly.
func (r *UniformRouter) ShouldPool() bool {
	return true
}

// Name get name of ConnCheckerRouter
func (r *UniformRouter) Name() string {
	return name
}

// Priority get Router priority level
func (r *UniformRouter) Priority() int64 {
	return 0
}

// URL Return URL in router
func (r *UniformRouter) URL() *common.URL {
	return nil
}
