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
	mu sync.RWMutex
	// generation identifies the invoker snapshot this cache was built from. It mirrors
	// RouterChain.generation and is set under the same write lock as invokers/pools so that
	// FindAddrPool can hand all three out as one consistent snapshot.
	generation uint64
	invokers   []base.Invoker
	pools      map[string]*poolEntry
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

// FindAddrPool returns the address pool, the invoker snapshot, and the generation of that
// snapshot for the given Poolable. The returned invokers slice is shared and must not be
// modified by the caller. The generation is always returned (even on a miss) so callers can
// detect a snapshot rebuilt by a concurrent SetInvokers.
func (c *routerCache) FindAddrPool(p router.Poolable) (router.AddrPool, []base.Invoker, uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.pools[p.Name()]
	if !ok {
		return nil, nil, c.generation
	}
	return entry.pool, c.invokers, c.generation
}

// FindAddrMeta is a no-op, reserved for future use.
func (c *routerCache) FindAddrMeta(p router.Poolable) router.AddrMetadata {
	return nil
}

// rebuild iterates all Poolable routers whose ShouldPool returns true,
// calls Pool on each with current invokers, and atomically swaps in the
// new pools and generation under write lock. The generation must match the
// RouterChain.generation produced by the same SetInvokers call.
func (c *routerCache) rebuild(generation uint64, invokers []base.Invoker, routers []router.PriorityRouter) {
	newPools := make(map[string]*poolEntry, len(routers))
	for _, r := range routers {
		p, ok := r.(router.Poolable)
		if !ok || !p.ShouldPool() {
			continue
		}
		pool, _ := p.Pool(invokers) // TODO: store AddrMetadata when needed
		newPools[p.Name()] = &poolEntry{pool: pool}
	}

	c.mu.Lock()
	c.generation = generation
	c.invokers = invokers
	c.pools = newPools
	c.mu.Unlock()
}
