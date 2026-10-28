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
	"testing"
)

import (
	"github.com/RoaringBitmap/roaring"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

type cacheTestRouter struct {
	name       string
	shouldPool bool
	poolCalls  int
}

func (r *cacheTestRouter) Name() string          { return r.name }
func (r *cacheTestRouter) ShouldPool() bool      { return r.shouldPool }
func (r *cacheTestRouter) URL() *common.URL      { return nil }
func (r *cacheTestRouter) Priority() int64       { return 0 }
func (r *cacheTestRouter) Notify([]base.Invoker) {}

func (r *cacheTestRouter) Route(invokers []base.Invoker, _ *common.URL, _ base.Invocation) []base.Invoker {
	return invokers
}

func (r *cacheTestRouter) Pool(invokers []base.Invoker) (router.AddrPool, router.AddrMetadata) {
	r.poolCalls++
	all := roaring.New()
	for i := range invokers {
		all.Add(uint32(i))
	}
	return router.AddrPool{"all": all}, nil
}

// To test if the router loads its invokers into cache so that next time we could get invokers from cache
func TestRouterCacheHit(t *testing.T) {
	invoker := buildInvoker(t, "dubbo://127.0.0.1:20000/com.demo.Service")
	poolable := &cacheTestRouter{name: "test", shouldPool: true}
	cache := newRouterCache()

	// A poolable router should populate an address pool during cache rebuild.
	cache.rebuild(1, []base.Invoker{invoker}, []router.PriorityRouter{poolable})
	pool, cachedInvokers, generation := cache.FindAddrPool(&cacheTestRouter{name: "test"})
	require.NotNil(t, pool)
	assert.True(t, pool["all"].Contains(0))
	assert.Equal(t, []base.Invoker{invoker}, cachedInvokers)
	assert.Equal(t, uint64(1), generation)
	assert.Equal(t, 1, poolable.poolCalls)
}

// To test if the cache is invalid after the cache is terminated and rebuilt.
func TestRouterCacheInvalidation(t *testing.T) {
	invoker := buildInvoker(t, "dubbo://127.0.0.1:20000/com.demo.Service")
	poolable := &cacheTestRouter{name: "test", shouldPool: true}
	cache := newRouterCache()
	cache.rebuild(1, []base.Invoker{invoker}, []router.PriorityRouter{poolable})
	pool, _, _ := cache.FindAddrPool(poolable)
	require.NotNil(t, pool)

	// Rebuilding with pooling disabled should remove the previous cache entry.
	poolable.shouldPool = false
	cache.rebuild(2, []base.Invoker{invoker}, []router.PriorityRouter{poolable})
	pool, cachedInvokers, generation := cache.FindAddrPool(poolable)
	assert.Nil(t, pool)
	assert.Nil(t, cachedInvokers)
	assert.Equal(t, uint64(2), generation)
	assert.Equal(t, 1, poolable.poolCalls)
}
