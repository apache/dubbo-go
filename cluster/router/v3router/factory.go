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
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
)

// UniformRouteFactory is uniform router's factory
type UniformRouteFactory struct{}

// NewUniformRouterFactory constructs a new PriorityRouterFactory
func NewUniformRouterFactory() router.PriorityRouterFactory {
	return &UniformRouteFactory{}
}

// NewPriorityRouter construct a new UniformRouteFactory as PriorityRouter
func (f *UniformRouteFactory) NewPriorityRouter(vsConfigBytes, distConfigBytes []byte) (router.PriorityRouter, error) {
	return NewUniformRouterChain(vsConfigBytes, distConfigBytes)
}
