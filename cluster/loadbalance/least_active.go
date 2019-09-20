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
	"math/rand"
)

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
)

const (
	LeastActive = "leastactive"
)

func init() {
	extension.SetLoadbalance(LeastActive, NewLeastActiveLoadBalance)
}

type leastActiveLoadBalance struct {
}

func NewLeastActiveLoadBalance() cluster.LoadBalance {
	return &leastActiveLoadBalance{}
}

func (lb *leastActiveLoadBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {
	count := len(invokers)
	if count == 0 {
		return nil
	}
	if count == 1 {
		return invokers[0]
	}

	var (
		leastActive  int32 = -1                 // The least active value of all invokers
		totalWeight  int64 = 0                  // The number of invokers having the same least active value (LEAST_ACTIVE)
		firstWeight  int64 = 0                  // Initial value, used for comparison
		leastIndexes       = make([]int, count) // The index of invokers having the same least active value (LEAST_ACTIVE)
		leastCount         = 0                  // The number of invokers having the same least active value (LEAST_ACTIVE)
		sameWeight         = true               // Every invoker has the same weight value?
	)

	for i := 0; i < count; i++ {
		invoker := invokers[i]
		// Active number
		active := protocol.GetStatus(invoker.GetUrl(), invocation.MethodName()).GetActive()
		// current weight (maybe in warmUp)
		weight := GetWeight(invoker, invocation)
		// There are smaller active services
		if leastActive == -1 || active < leastActive {
			leastActive = active
			leastIndexes[0] = i
			leastCount = 1 // next available leastIndex offset
			totalWeight = weight
			firstWeight = weight
			sameWeight = true
		} else if active == leastActive {
			leastIndexes[leastCount] = i
			totalWeight += weight
			leastCount++

			if sameWeight && (i > 0) && weight != firstWeight {
				sameWeight = false
			}
		}
	}

	if leastCount == 1 {
		return invokers[0]
	}

	if !sameWeight && totalWeight > 0 {
		offsetWeight := rand.Int63n(totalWeight) + 1
		for i := 0; i < leastCount; i++ {
			leastIndex := leastIndexes[i]
			offsetWeight -= GetWeight(invokers[i], invocation)
			if offsetWeight <= 0 {
				return invokers[leastIndex]
			}
		}
	}

	index := leastIndexes[rand.Intn(leastCount)]
	return invokers[index]
}
