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
	"math"
	"sort"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

// RouterChain Router chain
type RouterChain struct {
	// Full list of addresses from registry, classified by method name.
	invokers []protocol.Invoker
	// Containing all routers, reconstruct every time 'route://' urls change.
	routers []router.PriorityRouter
	// Fixed router instances: ConfigConditionRouter, TagRouter, e.g., the rule for each instance may change but the
	// instance will never delete or recreate.
	builtinRouters []router.PriorityRouter

	mutex sync.RWMutex

	url common.URL
}

// Route Loop routers in RouterChain and call Route method to determine the target invokers list.
func (c *RouterChain) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	finalInvokers := invokers
	l := len(c.routers)
	rs := make([]router.PriorityRouter, l, int(math.Ceil(float64(l)*1.2)))
	c.mutex.RLock()
	copy(rs, c.routers)
	c.mutex.RUnlock()

	for _, r := range rs {
		finalInvokers = r.Route(finalInvokers, url, invocation)
	}
	return finalInvokers
}

// Notify router chain of the initial addresses from registry at the first time. Notify whenever addresses in registry change.
func (c *RouterChain) SetInvokers(invokers []protocol.Invoker) {
	for _, r := range c.routers {
		if notifyRouter, ok := r.(router.NotifyRouter); ok {
			notifyRouter.Notify(invokers)
		}
	}
}

// AddRouters Add routers to router chain
// New a array add builtinRouters which is not sorted in RouterChain and routers
// Sort the array
// Replace router array in RouterChain
func (c *RouterChain) AddRouters(routers []router.PriorityRouter) {
	newRouters := make([]router.PriorityRouter, 0, len(c.builtinRouters)+len(routers))
	newRouters = append(newRouters, c.builtinRouters...)
	newRouters = append(newRouters, routers...)
	sortRouter(newRouters)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.routers = newRouters
}

// URL Return URL in RouterChain
func (c *RouterChain) URL() common.URL {
	return c.url
}

// NewRouterChain Use url to init router chain
// Loop routerFactories and call NewRouter method
func NewRouterChain(url *common.URL) (*RouterChain, error) {
	routerFactories := extension.GetRouterFactories()
	if len(routerFactories) == 0 {
		return nil, perrors.Errorf("No routerFactory exits , create one please")
	}
	routers := make([]router.PriorityRouter, 0, len(routerFactories))
	for key, routerFactory := range routerFactories {
		r, err := routerFactory().NewPriorityRouter(url)
		if r == nil || err != nil {
			logger.Errorf("router chain build router fail! routerFactories key:%s  error:%s", key, err.Error())
			continue
		}
		routers = append(routers, r)
	}

	newRouters := make([]router.PriorityRouter, len(routers))
	copy(newRouters, routers)

	sortRouter(newRouters)

	chain := &RouterChain{
		builtinRouters: routers,
		routers:        newRouters,
	}
	if url != nil {
		chain.url = *url
	}

	return chain, nil
}

// sortRouter Sort router instance by priority with stable algorithm
func sortRouter(routers []router.PriorityRouter) {
	sort.Stable(byPriority(routers))
}

// byPriority Sort by priority
type byPriority []router.PriorityRouter

func (a byPriority) Len() int           { return len(a) }
func (a byPriority) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPriority) Less(i, j int) bool { return a[i].Priority() < a[j].Priority() }
