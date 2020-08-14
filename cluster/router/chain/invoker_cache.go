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
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/utils"
	"github.com/apache/dubbo-go/protocol"
)

// Cache caches all addresses relevant info for a snapshot of received invokers. It keeps a snapshot of the received
// address list, and also keeps address pools and address metadata from routers based on the same address snapshot, if
// the router implements Poolable.
type InvokerCache struct {
	// The snapshot of invokers
	invokers []protocol.Invoker

	// The bitmap representation for invokers snapshot
	bitmap *roaring.Bitmap

	// Address pool from routers which implement Poolable
	pools map[string]router.AddrPool

	// Address metadata from routers which implement Poolable
	metadatas map[string]router.AddrMetadata
}

// BuildCache builds address cache from the given invokers.
func BuildCache(invokers []protocol.Invoker) *InvokerCache {
	return &InvokerCache{
		invokers:  invokers,
		bitmap:    utils.ToBitmap(invokers),
		pools:     make(map[string]router.AddrPool, 8),
		metadatas: make(map[string]router.AddrMetadata, 8),
	}
}

// GetInvokers get invokers snapshot.
func (c *InvokerCache) GetInvokers() []protocol.Invoker {
	return c.invokers
}

// FindAddrPool finds address pool for a poolable router.
func (c *InvokerCache) FindAddrPool(p router.Poolable) router.AddrPool {
	return c.pools[p.Name()]
}

// FindAddrMeta finds address metadata for a poolable router.
func (c *InvokerCache) FindAddrMeta(p router.Poolable) router.AddrMetadata {
	return c.metadatas[p.Name()]
}

// SetAddrPool sets address pool for a poolable router, for unit test only
func (c *InvokerCache) SetAddrPool(name string, pool router.AddrPool) {
	c.pools[name] = pool
}

// SetAddrMeta sets address metadata for a poolable router, for unit test only
func (c *InvokerCache) SetAddrMeta(name string, meta router.AddrMetadata) {
	c.metadatas[name] = meta
}
