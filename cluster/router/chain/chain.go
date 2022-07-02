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
	"sort"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
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
}

// Route Loop routers in RouterChain and call Route method to determine the target invokers list.
func (c *RouterChain) Route(url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	finalInvokers := c.invokers
	for _, r := range c.copyRouters() {
		finalInvokers = r.Route(finalInvokers, url, invocation)
	}
	return finalInvokers
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

// SetInvokers receives updated invokers from registry center. If the times of notification exceeds countThreshold and
// time interval exceeds timeThreshold since last cache update, then notify to update the cache.
func (c *RouterChain) SetInvokers(invokers []protocol.Invoker) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.invokers = invokers
	for _, v := range c.routers {
		v.Notify(c.invokers)
	}
}

// copyRouters make a snapshot copy from RouterChain's router list.
func (c *RouterChain) copyRouters() []router.PriorityRouter {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	ret := make([]router.PriorityRouter, 0, len(c.routers))
	ret = append(ret, c.routers...)
	return ret
}

// copyInvokers copies a snapshot of the received invokers.
func (c *RouterChain) copyInvokers() []protocol.Invoker {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.invokers == nil || len(c.invokers) == 0 {
		return nil
	}
	ret := make([]protocol.Invoker, 0, len(c.invokers))
	ret = append(ret, c.invokers...)
	return ret
}

// NewRouterChain init router chain
// Loop routerFactories and call NewRouter method
func NewRouterChain() (*RouterChain, error) {
	routerFactories := extension.GetRouterFactories()
	if len(routerFactories) == 0 {
		return nil, perrors.Errorf("No routerFactory exits , create one please")
	}

	routers := make([]router.PriorityRouter, 0, len(routerFactories))

	for key, routerFactory := range routerFactories {
		r, err := routerFactory().NewPriorityRouter()
		if err != nil {
			logger.Errorf("Build router chain failed with routerFactories key:%s and error:%v", key, err)
			continue
		} else if r == nil {
			continue
		}
		routers = append(routers, r)
	}

	newRouters := make([]router.PriorityRouter, len(routers))
	copy(newRouters, routers)

	sortRouter(newRouters)

	routerNeedsUpdateInit := atomic.Bool{}
	routerNeedsUpdateInit.Store(false)

	chain := &RouterChain{
		routers:        newRouters,
		builtinRouters: routers,
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
