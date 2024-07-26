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

package script

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
)

func init() {
	/*
		Tag router is not imported in dubbo-go/imports/imports.go, because it relies on config center,
		and cause warning if config center is empty.
		User can import this package and config config center to use Script router.
	*/
	// TODO(finalt) Temporarily removed until fixed (https://github.com/apache/dubbo-go/pull/2716)
	//extension.SetRouterFactory(constant.ScriptRouterFactoryKey, NewScriptRouterFactory)
}

// ScriptRouteFactory router factory
type ScriptRouteFactory struct{}

// NewScriptRouterFactory constructs a new PriorityRouterFactory
func NewScriptRouterFactory() router.PriorityRouterFactory {
	return &ScriptRouteFactory{}
}

// NewPriorityRouter construct a new PriorityRouter
func (f *ScriptRouteFactory) NewPriorityRouter() (router.PriorityRouter, error) {
	return NewScriptRouter(), nil
}
