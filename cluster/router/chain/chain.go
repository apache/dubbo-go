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
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	perrors "github.com/pkg/errors"

	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

const (
	timeInterval = 5 * time.Second
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

	url *common.URL

	// The times of address notification since last update for address cache
	count int64
	// The timestamp of last update for address cache
	last time.Time
	// Channel for notify to update the address cache
	notify chan struct{}
	// Address cache
	cache atomic.Value

	routerStatus atomic.Int32
}

const (
	NoRouter = iota
	HasRouter
)

func (c *RouterChain) GetNotifyChan() chan struct{} {
	return c.notify
}

// Route Loop routers in RouterChain and call Route method to determine the target invokers list.
func (c *RouterChain) Route(url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	cache := c.loadCache()
	if cache == nil {
		logger.Warnf("there is no cache invoker.")
		c.mutex.RLock()
		defer c.mutex.RUnlock()
		return c.invokers
	}

	bitmap := cache.bitmap
	for _, r := range c.copyRouters() {
		bitmap = r.Route(bitmap, cache, url, invocation)
	}

	indexes := bitmap.ToArray()
	// if the indexes is empty, print the routeSnapshot
	if len(indexes) == 0 {
		go c.printRouteSnapshot(cache, url, invocation)
	}
	finalInvokers := make([]protocol.Invoker, len(indexes))
	for i, index := range indexes {
		finalInvokers[i] = cache.invokers[index]
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
	if c.routerStatus.Load() == int32(NoRouter) {
		return
	}

	go func() {
		c.notify <- struct{}{}
	}()
}

// SetInvokers receives updated invokers from registry center. If the times of notification exceeds countThreshold and
// time interval exceeds timeThreshold since last cache update, then notify to update the cache.
func (c *RouterChain) SetInvokers(invokers []protocol.Invoker) {
	c.mutex.Lock()
	c.invokers = invokers
	c.mutex.Unlock()
	if c.routerStatus.Load() == int32(NoRouter) {
		return
	}

	go func() {
		c.notify <- struct{}{}
	}()
}

// Detect Route State
func (c *RouterChain) DetectRoute() (router.RouteSnapshot, error) {
	defer func() {
		if err := recover(); err != nil {
			logger.Warnf("Detect Route fail. %+v", err)
		}
	}()
	cache := c.loadCache()
	if cache == nil {
		c.mutex.RLock()
		defer c.mutex.RUnlock()
		return router.RouteSnapshot{Invokers: c.invokers}, nil
	}
	routers := c.copyRouters()
	routeSnapshots := make([]string, 0, len(routers))
	for _, r := range routers {
		if v, ok := r.(router.PriorityRouterDetecter); ok {
			routeSnapshots = append(routeSnapshots, v.RouteSnapshot(cache))
		}
	}

	return router.RouteSnapshot{Invokers: cache.invokers, RouteSnapshots: routeSnapshots}, nil
}

// nolint
func (c *RouterChain) printRouteSnapshot(cache *InvokerCache, url *common.URL, invocation protocol.Invocation) {
	defer func() {
		if err := recover(); err != nil {
			logger.Warnf("print Route Snapshot fail. %+v", err)
		}
	}()

	bitmap := cache.bitmap

	logger.Warnf("start:print the route info:%s", url.ServiceKey())
	for _, r := range c.copyRouters() {
		bitmap = r.Route(bitmap, cache, url, invocation)
		routeSnapshotSb := strings.Builder{}
		routeSnapshotSb.WriteString(reflect.TypeOf(r).String())
		routeSnapshotSb.WriteString(", count:")
		routeSnapshotSb.WriteString(strconv.FormatUint(bitmap.GetCardinality(), 10))
		routeSnapshotSb.WriteString(bitmap.String())
		logger.Warn(routeSnapshotSb.String())
	}

	if routerSnapshot, err := c.DetectRoute(); err == nil {
		logger.Warnf("the size of invokers:%d", len(routerSnapshot.Invokers))
		for _, item := range routerSnapshot.RouteSnapshots {
			logger.Warn(item)
		}
	}
	logger.Warnf("end: print the route info:%s", url.ServiceKey())
}

// loop listens on events to update the address cache  when it receives notification
// from address update,
func (c *RouterChain) Loop() {
	ticker := time.NewTicker(timeInterval)
	for {
		select {
		case <-ticker.C:
			if protocol.GetAndRefreshState(c.url) {
				logger.Infof("start to build route cache because the invokers in black list is changed [%s]", c.url.ServiceKey())
				c.buildCache()
			}
		case <-c.notify:
			c.buildCache()
		}
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

// loadCache loads cache from sync.Value to guarantee the visibility
func (c *RouterChain) loadCache() *InvokerCache {
	v := c.cache.Load()
	if v == nil {
		return nil
	}

	return v.(*InvokerCache)
}

// copyInvokerIfNecessary compares chain's invokers copy and cache's invokers copy, to avoid copy as much as possible
func (c *RouterChain) copyInvokerIfNecessary(cache *InvokerCache) []protocol.Invoker {
	var invokers []protocol.Invoker
	if cache != nil {
		invokers = cache.invokers
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if isInvokersChanged(invokers, c.invokers) {
		invokers = c.copyInvokers()
	}
	return invokers
}

// buildCache builds address cache with the new invokers for all poolable routers.
func (c *RouterChain) buildCache() {
	origin := c.loadCache()
	invokers := c.copyInvokerIfNecessary(origin)
	if len(invokers) == 0 {
		return
	}

	var (
		mutex sync.Mutex
		wg    sync.WaitGroup
	)

	cache := BuildCache(invokers)
	for _, r := range c.copyRouters() {
		if p, ok := r.(router.Poolable); ok {
			wg.Add(1)
			go func(p router.Poolable) {
				defer wg.Done()
				pool, info := poolRouter(p, origin, invokers)
				mutex.Lock()
				defer mutex.Unlock()
				cache.pools[p.Name()] = pool
				cache.metadatas[p.Name()] = info
			}(p)
		}
	}
	wg.Wait()

	c.cache.Store(cache)
}

// URL Return URL in RouterChain
func (c *RouterChain) URL() *common.URL {
	return c.url
}

// NewRouterChain Use url to init router chain
// Loop routerFactories and call NewRouter method
func NewRouterChain(url *common.URL) (*RouterChain, error) {
	routerFactories := extension.GetRouterFactories()
	if len(routerFactories) == 0 {
		return nil, perrors.Errorf("No routerFactory exits , create one please")
	}

	chain := &RouterChain{
		last:   time.Now(),
		notify: make(chan struct{}),
	}
	chain.routerStatus.Store(int32(HasRouter))

	routers := make([]router.PriorityRouter, 0, len(routerFactories))
	for key, routerFactory := range routerFactories {
		r, err := routerFactory().NewPriorityRouter(url, chain.notify)
		if r == nil || err != nil {
			logger.Errorf("router chain build router fail! routerFactories key:%s  error:%s", key, err.Error())
			continue
		}
		routers = append(routers, r)
	}

	newRouters := make([]router.PriorityRouter, len(routers))
	copy(newRouters, routers)

	sortRouter(newRouters)

	routerNeedsUpdateInit := atomic.Bool{}
	routerNeedsUpdateInit.Store(false)
	chain.routers = newRouters
	chain.builtinRouters = routers
	if url != nil {
		chain.url = url
	}

	go chain.Loop()
	return chain, nil
}

// poolRouter calls poolable router's Pool() to create new address pool and address metadata if necessary.
// If the corresponding cache entry exists, and the poolable router answers no need to re-pool (possibly because its
// rule doesn't change), and the address list doesn't change, then the existing data will be re-used.
func poolRouter(p router.Poolable, origin *InvokerCache, invokers []protocol.Invoker) (router.AddrPool, router.AddrMetadata) {
	name := p.Name()
	if isCacheMiss(origin, name) || p.ShouldPool() || &(origin.invokers) != &invokers {
		logger.Debugf("build address cache for router %q", name)
		return p.Pool(invokers)
	}

	logger.Debugf("reuse existing address cache for router %q", name)
	return origin.pools[name], origin.metadatas[name]
}

// isCacheMiss checks if the corresponding cache entry for a poolable router has already existed.
// False returns when the cache is nil, or cache's pool is nil, or cache's invokers snapshot is nil, or the entry
// doesn't exist.
func isCacheMiss(cache *InvokerCache, key string) bool {
	if cache == nil || cache.pools == nil || cache.invokers == nil || cache.pools[key] == nil {
		return true
	}
	return false
}

// isInvokersChanged compares new invokers on the right changes, compared with the old invokers on the left.
func isInvokersChanged(left []protocol.Invoker, right []protocol.Invoker) bool {
	if len(right) != len(left) {
		return true
	}

	for _, r := range right {
		found := false
		rurl := r.GetURL()
		for _, l := range left {
			lurl := l.GetURL()
			if common.GetCompareURLEqualFunc()(lurl, rurl, constant.TIMESTAMP_KEY, constant.REMOTE_TIMESTAMP_KEY) {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}
	return false
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
