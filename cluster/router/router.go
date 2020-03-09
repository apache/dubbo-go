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

package router

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

// Extension - Router

// RouterFactory Router create factory
type RouterFactory interface {
	// NewRouter Create router instance with URL
	NewRouter(*common.URL) (Router, error)
}

// RouterFactory Router create factory use for parse config file
type FIleRouterFactory interface {
	// NewFileRouters Create file router with config file
	NewFileRouter([]byte) (Router, error)
}

// Router
type Router interface {
	// Route Determine the target invokers list.
	Route([]protocol.Invoker, *common.URL, protocol.Invocation) []protocol.Invoker
	// Priority Return Priority in router
	// 0 to ^int(0) is better
	Priority() int64
	// URL Return URL in router
	URL() common.URL
}

// Chain
type Chain interface {
	// Route Determine the target invokers list with chain.
	Route([]protocol.Invoker, *common.URL, protocol.Invocation) []protocol.Invoker
	// AddRouters Add routers
	AddRouters([]Router)
}
