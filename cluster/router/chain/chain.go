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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

// RouterChain Router chain
type RouterChain struct {
	// Full list of addresses from registry, classified by method name.
	invokers []base.Invoker
	// Containing all routers, reconstruct every time 'route://' urls change.
	routers []router.PriorityRouter
	// Fixed router instances: ConfigConditionRouter, TagRouter, e.g., the rule for each instance may change but the
	// instance will never delete or recreate.
	builtinRouters []router.PriorityRouter

	cache *routerCache
	mutex sync.RWMutex
}

// Route Loop routers in RouterChain and call Route method to determine the target invokers list.
func (c *RouterChain) Route(url *common.URL, invocation base.Invocation) []base.Invoker {
	invokers := c.snapshotInvokers()
	finalInvokers := make([]base.Invoker, 0, len(invokers))
	// multiple invoker may include different methods, find correct invoker otherwise
	// will return the invoker without methods
	for _, invoker := range invokers {
		if invoker.GetURL().ServiceKey() == url.ServiceKey() {
			finalInvokers = append(finalInvokers, invoker)
		}
	}

	if len(finalInvokers) == 0 {
		finalInvokers = invokers
	}

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
func (c *RouterChain) SetInvokers(invokers []base.Invoker) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.invokers = invokers
	c.rebuildCache(invokers)
	for _, v := range c.routers {
		v.Notify(c.invokers)
	}
}

func (c *RouterChain) rebuildCache(invokers []base.Invoker) {
	if c.cache == nil {
		c.cache = newRouterCache()
		for _, r := range c.routers {
			if accessor, ok := r.(router.CacheAccessor); ok {
				accessor.SetCache(c.cache)
			}
		}
	}
	c.cache.rebuild(invokers, c.routers)
}

// copyRouters make a snapshot copy from RouterChain's router list.
func (c *RouterChain) copyRouters() []router.PriorityRouter {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	ret := make([]router.PriorityRouter, 0, len(c.routers))
	ret = append(ret, c.routers...)
	return ret
}

// snapshotInvokers returns a copy of current invokers under lock.
func (c *RouterChain) snapshotInvokers() []base.Invoker {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	ret := make([]base.Invoker, len(c.invokers))
	copy(ret, c.invokers)
	return ret
}

// injectStaticRouters injects static router configurations into the router chain.
// Called after all routers are created to ensure they exist.
// The injected static configs act as bootstrap state only during initialization. For the shared
// static and dynamic lifecycle semantics, see router.StaticConfigSetter.
func (c *RouterChain) injectStaticRouters(url *common.URL) {
	staticRoutersAttrAny, ok := url.GetAttribute(constant.RoutersConfigKey)
	if !ok && url.SubURL != nil {
		staticRoutersAttrAny, ok = url.SubURL.GetAttribute(constant.RoutersConfigKey)
	}
	if !ok {
		return
	}
	staticRoutersAttr, ok := staticRoutersAttrAny.([]*global.RouterConfig)
	if !ok {
		logger.Errorf("failed to type assert routers config: expected []*global.RouterConfig, got %T", staticRoutersAttrAny)
		return
	}
	if len(staticRoutersAttr) == 0 {
		return
	}

	for _, routerCfg := range staticRoutersAttr {
		if routerCfg == nil {
			continue
		}
		if routerCfg.Enabled != nil && !*routerCfg.Enabled {
			continue
		}
		if routerCfg.Valid != nil && !*routerCfg.Valid {
			continue
		}
		c.injectRouterConfig(routerCfg)
	}
}

// injectRouterConfig injects router configuration into routers that implement StaticConfigSetter.
// Each router decides whether the config applies to it and no-ops if not.
func (c *RouterChain) injectRouterConfig(routerCfg *global.RouterConfig) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	for _, r := range c.routers {
		if setter, ok := r.(router.StaticConfigSetter); ok {
			setter.SetStaticConfig(routerCfg.Clone())
		}
	}
}

// NewRouterChain init router chain
// Loop routerFactories and call NewRouter method
func NewRouterChain(url *common.URL) (*RouterChain, error) {
	if url.SubURL != nil {
		if appName := url.SubURL.GetParam(constant.ApplicationKey, ""); appName != "" {
			url.CompareAndSwapParam(constant.ApplicationKey, "", appName)
		}
	}

	routerFactories := extension.GetRouterFactories()
	if len(routerFactories) == 0 {
		return nil, perrors.Errorf("No routerFactory exits , create one please")
	}

	routers := make([]router.PriorityRouter, 0, len(routerFactories))

	for key, routerFactory := range routerFactories {
		r, err := routerFactory().NewPriorityRouter(url)
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

	// Inject static router configurations after all routers are created.
	// This happens before the first registry notification triggers dynamic Notify/Process
	// updates on builtin routers.
	chain.injectStaticRouters(url)

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
