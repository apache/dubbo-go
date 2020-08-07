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
	"github.com/RoaringBitmap/roaring"
	"github.com/apache/dubbo-go/common/constant"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
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

const (
	timeInterval   = 5 * time.Second
	timeThreshold  = 2 * time.Second
	countThreshold = 5
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
	url   common.URL

	count int64
	last  time.Time
	ch    chan struct{}
	cache atomic.Value
}

// Route Loop routers in RouterChain and call Route method to determine the target invokers list.
func (c *RouterChain) Route(url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	rs := c.copyRouters()

	v := c.cache.Load()
	if v == nil {
		return c.invokers
	}

	cache := v.(*router.AddrCache)
	bitmap := cache.Bitmap
	for _, r := range rs {
		bitmap = r.Route(bitmap, cache, url, invocation)
	}

	indexes := bitmap.ToArray()
	finalInvokers := make([]protocol.Invoker, len(indexes))
	for _, index := range indexes {
		finalInvokers = append(finalInvokers, cache.Invokers[index])
	}
	return finalInvokers
}

func (c *RouterChain) copyRouters() []router.PriorityRouter {
	l := len(c.routers)
	rs := make([]router.PriorityRouter, l, int(math.Ceil(float64(l)*1.2)))
	c.mutex.RLock()
	copy(rs, c.routers)
	c.mutex.RUnlock()
	return rs
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

func (c *RouterChain) SetInvokers(invokers []protocol.Invoker) {
	c.invokers = invokers

	c.count++
	now := time.Now()
	if c.count >= countThreshold && now.Sub(c.last) >= timeThreshold {
		c.last = now
		c.count = 0
		go func() {
			c.ch <- struct{}{}
		}()
	}
}

func (c *RouterChain) loop() {
	for {
		select {
		case <-time.Tick(timeInterval):
			logger.Debugf("start to build address cache since time interval %d second ticks", int(timeInterval.Seconds()))
			c.buildCache()
		case <-c.ch:
			logger.Debugf("start to build address cache since at least %d times of address notified within %d seconds",
				countThreshold, int(timeThreshold.Seconds()))
			c.buildCache()
		}
	}
}

func (c *RouterChain) buildCache() {
	if c.invokers == nil || len(c.invokers) == 0 {
		return
	}

	// FIXME: should lock here, it is fine with dirty read if no panic happens I believe.
	invokers := make([]protocol.Invoker, len(c.invokers))
	copy(invokers, c.invokers)
	cache := &router.AddrCache{
		Invokers: invokers,
		Bitmap:   ToBitmap(invokers),
		AddrPool: make(map[string]router.RouterAddrPool),
		AddrMeta: make(map[string]router.AddrMetadata),
	}

	var origin *router.AddrCache
	v := c.cache.Load()
	if v != nil {
		origin = v.(*router.AddrCache)
	}

	var mutex sync.Mutex
	var wg sync.WaitGroup

	for _, r := range c.copyRouters() {
		if p, ok := r.(router.Poolable); ok {
			wg.Add(1)
			go func(p router.Poolable) {
				pool, info := poolRouter(p, origin, invokers)
				mutex.Lock()
				cache.AddrPool[p.Name()] = pool
				cache.AddrMeta[p.Name()] = info
				mutex.Unlock()
				wg.Done()
			}(p)
		}
	}
	wg.Wait()

	c.cache.Store(cache)
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
		last:           time.Now(),
		ch:             make(chan struct{}),
	}
	if url != nil {
		chain.url = *url
	}

	go chain.loop()
	return chain, nil
}

func poolRouter(p router.Poolable, origin *router.AddrCache, invokers []protocol.Invoker) (router.RouterAddrPool, router.AddrMetadata) {
	name := p.Name()
	if isCacheMiss(origin, name) || p.ShouldRePool() || IsDiff(origin.Invokers, invokers) {
		logger.Debugf("build address cache for router %q", name)
		return p.Pool(invokers)
	} else {
		logger.Debugf("reuse existing address cache for router %q", name)
		return origin.AddrPool[name], origin.AddrMeta[name]
	}
}

func isCacheMiss(cache *router.AddrCache, key string) bool {
	if cache == nil || cache.AddrPool == nil || cache.Invokers == nil || cache.AddrPool[key] == nil {
		return true
	}
	return false
}

func IsDiff(left []protocol.Invoker, right []protocol.Invoker) bool {
	if len(right) != len(left) {
		return true
	}

	for _, r := range right {
		found := false
		for _, l := range left {
			if IsEquals(l.GetUrl(), r.GetUrl()) {
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

func IsEquals(left common.URL, right common.URL) bool {
	if left.Ip != right.Ip || left.Port != right.Port {
		return false
	}

	leftMap := left.ToMap()
	delete(leftMap, constant.TIMESTAMP_KEY)
	delete(leftMap, constant.REMOTE_TIMESTAMP_KEY)
	rightMap := right.ToMap()
	delete(rightMap, constant.TIMESTAMP_KEY)
	delete(rightMap, constant.REMOTE_TIMESTAMP_KEY)

	if len(leftMap) != len(rightMap) {
		return false
	}

	for lk, lv := range leftMap {
		if rv, ok := rightMap[lk]; !ok {
			return false
		} else if lv != rv {
			return false
		}
	}
	return true
}

func ToBitmap(invokers []protocol.Invoker) *roaring.Bitmap {
	bitmap := roaring.NewBitmap()
	bitmap.AddRange(0, uint64(len(invokers)))
	return bitmap
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
