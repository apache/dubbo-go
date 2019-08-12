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

package loadbalance

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
)

const (
	RoundRobin = "roundrobin"

	COMPLETE = 0
	UPDATING = 1
)

var (
	methodWeightMap sync.Map            // [string]invokers
	state           int32    = COMPLETE // update lock acquired ?
	recyclePeriod   int64    = 60 * time.Second.Nanoseconds()
)

func init() {
	extension.SetLoadbalance(RoundRobin, NewRoundRobinLoadBalance)
}

type roundRobinLoadBalance struct{}

func NewRoundRobinLoadBalance() cluster.LoadBalance {
	return &roundRobinLoadBalance{}
}

func (lb *roundRobinLoadBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {
	count := len(invokers)
	if count == 0 {
		return nil
	}
	if count == 1 {
		return invokers[0]
	}

	key := invokers[0].GetUrl().Path + "." + invocation.MethodName()
	cache, _ := methodWeightMap.LoadOrStore(key, &cachedInvokers{})
	cachedInvokers := cache.(*cachedInvokers)

	var (
		clean               = false
		totalWeight         = int64(0)
		maxCurrentWeight    = int64(math.MinInt64)
		now                 = time.Now()
		selectedInvoker     protocol.Invoker
		selectedWeightRobin *weightedRoundRobin
	)

	for _, invoker := range invokers {
		var weight = GetWeight(invoker, invocation)
		if weight < 0 {
			weight = 0
		}

		identifier := invoker.GetUrl().Key()
		loaded, found := cachedInvokers.LoadOrStore(identifier, &weightedRoundRobin{weight: weight})
		weightRobin := loaded.(*weightedRoundRobin)
		if !found {
			clean = true
		}

		if weightRobin.Weight() != weight {
			weightRobin.setWeight(weight)
		}

		currentWeight := weightRobin.increaseCurrent()
		weightRobin.lastUpdate = &now

		if currentWeight > maxCurrentWeight {
			maxCurrentWeight = currentWeight
			selectedInvoker = invoker
			selectedWeightRobin = weightRobin
		}
		totalWeight += weight
	}

	cleanIfRequired(clean, cachedInvokers, &now)

	if selectedWeightRobin != nil {
		selectedWeightRobin.Current(totalWeight)
		return selectedInvoker
	}

	// should never happen
	return invokers[0]
}

func cleanIfRequired(clean bool, invokers *cachedInvokers, now *time.Time) {
	if clean && atomic.CompareAndSwapInt32(&state, COMPLETE, UPDATING) {
		defer atomic.CompareAndSwapInt32(&state, UPDATING, COMPLETE)
		invokers.Range(func(identify, robin interface{}) bool {
			weightedRoundRobin := robin.(*weightedRoundRobin)
			elapsed := now.Sub(*weightedRoundRobin.lastUpdate).Nanoseconds()
			if elapsed > recyclePeriod {
				invokers.Delete(identify)
			}
			return true
		})
	}
}

// Record the weight of the invoker
type weightedRoundRobin struct {
	weight     int64
	current    int64
	lastUpdate *time.Time
}

func (robin *weightedRoundRobin) Weight() int64 {
	return atomic.LoadInt64(&robin.weight)
}

func (robin *weightedRoundRobin) setWeight(weight int64) {
	robin.weight = weight
	robin.current = 0
}

func (robin *weightedRoundRobin) increaseCurrent() int64 {
	return atomic.AddInt64(&robin.current, robin.weight)
}

func (robin *weightedRoundRobin) Current(delta int64) {
	atomic.AddInt64(&robin.current, -1*delta)
}

type cachedInvokers struct {
	sync.Map /*[string]weightedRoundRobin*/
}
