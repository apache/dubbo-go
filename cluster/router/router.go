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
// PriorityRouterFactory creates creates priority router with url
type PriorityRouterFactory interface {
	// NewPriorityRouter creates router instance with URL
	NewPriorityRouter(*common.URL) (PriorityRouter, error)
}

// FilePriorityRouterFactory creates priority router with parse config file
type FilePriorityRouterFactory interface {
	// NewFileRouters Create file router with config file
	NewFileRouter([]byte) (PriorityRouter, error)
}

// Router
type router interface {
	// Route Determine the target invokers list.
	Route([]protocol.Invoker, *common.URL, protocol.Invocation) []protocol.Invoker
	// URL Return URL in router
	URL() common.URL
}

// Router
type PriorityRouter interface {
	router
	// Priority Return Priority in router
	// 0 to ^int(0) is better
	Priority() int64
}

// NotifyRouter notify router use the invoker list. Invoker list may change from time to time. This method gives the router a
// chance to prepare before {@link Router#route(List, URL, Invocation)} gets called.
type NotifyRouter interface {
	PriorityRouter
	// Notify notify whenever addresses in registry change
	Notify([]protocol.Invoker)
}
