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

package v3router

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	name = "uniform-router"
)

// UniformRouter have
type UniformRouter struct {
	dubboRouter *DubboRouterRule
}

// NewUniformRouter construct an NewConnCheckRouter via url
func NewUniformRouter(dubboRouter *DubboRouterRule) (*UniformRouter, error) {
	r := &UniformRouter{
		dubboRouter: dubboRouter,
	}
	return r, nil
}

// Route gets a list of routed invoker
func (r *UniformRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	return r.dubboRouter.route(invokers, url, invocation)
}

// Process there is no process needs for uniform Router, as it upper struct RouterChain has done it
func (r *UniformRouter) Process(event *config_center.ConfigChangeEvent) {
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
