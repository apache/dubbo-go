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

package istio

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

func init() {
	extension.SetRouterFactory(constant.XdsRouterFactoryKey, NewXdsRouterFactory)
}

// MeshRouterFactory is mesh router's factory
type XdsRouterFactory struct{}

// NewMeshRouterFactory constructs a new PriorityRouterFactory
func NewXdsRouterFactory() router.PriorityRouterFactory {
	return &XdsRouterFactory{}
}

// NewPriorityRouter construct a new UniformRouteFactory as PriorityRouter
func (f *XdsRouterFactory) NewPriorityRouter() (router.PriorityRouter, error) {
	return NewXdsRouter()
}
