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

package iwrr

import (
	"slices"
	"strconv"
	"strings"
	"sync"
)

import (
	lru "github.com/hashicorp/golang-lru"
	"github.com/hashicorp/golang-lru/simplelru"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

func init() {
	extension.SetLoadbalance(constant.LoadBalanceKeyInterleavedWeightedRoundRobin, newInterleavedWeightedRoundRobinBalance)
}

const maxCachedSchedulers = 4096

// The extension factory may be called for every request. Sharing the cache
// lets selections across factory calls advance the same rotation.
var (
	schedulersMu  sync.Mutex
	schedulers, _ = simplelru.NewLRU(maxCachedSchedulers, nil)
	views, _      = lru.New(maxCachedSchedulers)
)

type interleavedWeightedRoundRobinBalance struct{}

// newInterleavedWeightedRoundRobinBalance returns an interleaved weighted round robin load balance.
func newInterleavedWeightedRoundRobinBalance() loadbalance.LoadBalance {
	return &interleavedWeightedRoundRobinBalance{}
}

// Select gets invoker based on interleaved weighted round robin load balancing strategy.
func (lb *interleavedWeightedRoundRobinBalance) Select(invokers []base.Invoker, invocation base.Invocation) base.Invoker {
	count := len(invokers)
	if count == 0 {
		return nil
	}
	if count == 1 {
		return invokers[0]
	}

	view := viewFor(invokers, invocation.MethodName(), invocation.ActualMethodName())
	scheduler := schedulerFor(view.key)
	return scheduler.pick(count, func(index int) base.Invoker {
		return invokers[view.order[index]]
	}, invocation)
}

type viewKey struct {
	firstURL     *common.URL
	lastURL      *common.URL
	count        int
	method       string
	actualMethod string
}

// A view maps canonical scheduler positions back to the current input order.
// It is immutable after publication and holds only URL snapshots.
type candidateView struct {
	urls  []*common.URL
	order []int
	key   string
}

func (view *candidateView) matches(invokers []base.Invoker) bool {
	if len(view.urls) != len(invokers) {
		return false
	}
	for i, url := range view.urls {
		if url != invokers[i].GetURL() {
			return false
		}
	}
	return true
}

func viewFor(invokers []base.Invoker, method, actualMethod string) *candidateView {
	hint := viewKey{
		firstURL:     invokers[0].GetURL(),
		lastURL:      invokers[len(invokers)-1].GetURL(),
		count:        len(invokers),
		method:       method,
		actualMethod: actualMethod,
	}
	if cached, ok := views.Get(hint); ok {
		view := cached.(*candidateView)
		if view.matches(invokers) {
			return view
		}
	}

	// Reordered or replaced URL snapshots still share a scheduler if their
	// provider identities match. Sort indices so the caller's slice stays intact.
	view := &candidateView{
		urls:  make([]*common.URL, len(invokers)),
		order: make([]int, len(invokers)),
	}
	keys := make([]string, len(invokers))
	const maxPartOverhead = 21 // Decimal length and separator.
	size := len(method) + len(actualMethod) + (len(invokers)+2)*maxPartOverhead
	for i, invoker := range invokers {
		url := invoker.GetURL()
		view.urls[i], view.order[i] = url, i
		keys[i] = url.GetCacheInvokerMapKey()
		size += len(keys[i])
	}
	slices.SortStableFunc(view.order, func(a, b int) int {
		return strings.Compare(keys[a], keys[b])
	})
	var key strings.Builder
	key.Grow(size)
	appendKeyPart(&key, method)
	appendKeyPart(&key, actualMethod)
	for _, i := range view.order {
		appendKeyPart(&key, keys[i])
	}
	view.key = key.String()
	views.Add(hint, view)
	return view
}

func appendKeyPart(key *strings.Builder, part string) {
	var length [20]byte
	key.Write(strconv.AppendInt(length[:0], int64(len(part)), 10))
	key.WriteByte(':')
	key.WriteString(part)
}

func schedulerFor(key string) *iwrrScheduler {
	schedulersMu.Lock()
	defer schedulersMu.Unlock()

	if cached, ok := schedulers.Get(key); ok {
		return cached.(*iwrrScheduler)
	}
	scheduler := &iwrrScheduler{}
	schedulers.Add(key, scheduler)
	return scheduler
}
