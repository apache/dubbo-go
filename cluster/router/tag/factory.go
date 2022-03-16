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

package tag

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

func init() {
	/*
		Tag router is not imported in dubbo-go/imports/imports.go, because it relies on config center,
		and cause warning if config center is empty.
		User can import this package and config config center to use tag router.
	*/
	extension.SetRouterFactory(constant.TagRouterFactoryKey, NewTagRouterFactory)
}

// RouteFactory router factory
type RouteFactory struct{}

// NewTagRouterFactory constructs a new PriorityRouterFactory
func NewTagRouterFactory() router.PriorityRouterFactory {
	return &RouteFactory{}
}

// NewPriorityRouter construct a new PriorityRouter
func (f *RouteFactory) NewPriorityRouter() (router.PriorityRouter, error) {
	return NewTagPriorityRouter()
}
