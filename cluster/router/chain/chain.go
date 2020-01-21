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

package chain

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"sort"
)

// RouterChain Router chain
type Chain struct {
	//full list of addresses from registry, classified by method name.
	invokers []protocol.Invoker
	//containing all routers, reconstruct every time 'route://' urls change.
	routers []router.Router
	// Fixed router instances: ConfigConditionRouter, TagRouter, e.g., the rule for each instance may change but the
	// instance will never delete or recreate.
	builtinRouters []router.Router
}

func (c Chain) Route(invoker []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	finalInvokers := invoker
	for _, r := range c.routers {
		finalInvokers = r.Route(invoker, url, invocation)
	}
	return finalInvokers
}
func (c Chain) AddRouters(routers []router.Router) {
	newRouters := make([]router.Router, 0)
	newRouters = append(newRouters, c.builtinRouters...)
	newRouters = append(newRouters, routers...)
	sortRouter(newRouters)
	c.routers = newRouters
}

func NewRouterChain(url common.URL) *Chain {
	routerFactories := extension.GetRouters()
	if len(routerFactories) == 0 {
		return nil
	}
	routers := make([]router.Router, 0)
	for _, routerFactory := range routerFactories {
		r, err := routerFactory().Router(&url)
		if err != nil {
			logger.Errorf("router chain build router fail! error:%v", err)
			continue
		}
		routers = append(routers, r)
	}

	newRouters := make([]router.Router, len(routers))
	copy(newRouters, routers)

	sortRouter(newRouters)

	chain := &Chain{
		builtinRouters: routers,
		routers:        newRouters,
	}

	return chain
}

func sortRouter(routers []router.Router) {
	sort.Stable(ByPriority(routers))
}

type ByPriority []router.Router

func (a ByPriority) Len() int           { return len(a) }
func (a ByPriority) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPriority) Less(i, j int) bool { return a[i].Priority() < a[j].Priority() }
