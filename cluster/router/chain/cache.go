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
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

// routerCache implements router.Cache by storing pre-computed AddrPool
// keyed by Poolable.Name(). It is rebuilt in its entirety on each
// SetInvokers call so that bitmap indices stay aligned with the current
// invoker snapshot.
type routerCache struct {
	mu       sync.RWMutex
	invokers []base.Invoker
	pools    map[string]*poolEntry
}

type poolEntry struct {
	pool router.AddrPool
}

func newRouterCache() *routerCache {
	return &routerCache{
		pools: make(map[string]*poolEntry),
	}
}

func (c *routerCache) GetInvokers() []base.Invoker {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ret := make([]base.Invoker, len(c.invokers))
	copy(ret, c.invokers)
	return ret
}

func (c *routerCache) FindAddrPool(p router.Poolable) router.AddrPool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.pools[p.Name()]
	if !ok {
		return nil
	}
	return entry.pool
}

// FindAddrMeta is a no-op, reserved for future use.
func (c *routerCache) FindAddrMeta(p router.Poolable) router.AddrMetadata {
	return nil
}

func (c *routerCache) FindAddrPoolWithInvokers(p router.Poolable) ([]base.Invoker, router.AddrPool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ret := make([]base.Invoker, len(c.invokers))
	copy(ret, c.invokers)
	entry, ok := c.pools[p.Name()]
	if !ok {
		return ret, nil
	}
	return ret, entry.pool
}

// rebuild iterates all Poolable routers whose ShouldPool returns true,
// calls Pool on each with current invokers, and atomically swaps in the
// new pools under write lock.
func (c *routerCache) rebuild(invokers []base.Invoker, routers []router.PriorityRouter) {
	newPools := make(map[string]*poolEntry, len(routers))
	for _, r := range routers {
		p, ok := r.(router.Poolable)
		if !ok || !p.ShouldPool() {
			continue
		}
		pool, _ := p.Pool(invokers)
		newPools[p.Name()] = &poolEntry{pool: pool}
	}

	c.mu.Lock()
	c.invokers = invokers
	c.pools = newPools
	c.mu.Unlock()
}
