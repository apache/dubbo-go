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

package random

import (
	"math/rand"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

func init() {
	extension.SetLoadbalance(constant.LoadBalanceKeyRandom, NewRandomLoadBalance)
}

type randomLoadBalance struct{}

// NewRandomLoadBalance returns a random load balance instance.
//
// Set random probabilities by weight, and the request will be sent to provider randomly.
func NewRandomLoadBalance() loadbalance.LoadBalance {
	return &randomLoadBalance{}
}

func (lb *randomLoadBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {

	// Number of invokers
	var length int
	if length = len(invokers); length == 1 {
		return invokers[0]
	}

	// Every invoker has the same weight?
	sameWeight := true
	// the maxWeight of every invokers, the minWeight = 0 or the maxWeight of the last invoker
	weights := make([]int64, length)
	// The sum of weights
	var totalWeight int64 = 0

	for i := 0; i < length; i++ {
		weight := loadbalance.GetWeight(invokers[i], invocation)
		//Sum
		totalWeight += weight
		// save for later use
		weights[i] = totalWeight
		if sameWeight && totalWeight != weight*(int64(i+1)) {
			sameWeight = false
		}
	}

	if totalWeight > 0 && !sameWeight {
		// If (not every invoker has the same weight & at least one invoker's weight>0), select randomly based on totalWeight.
		offset := rand.Int63n(totalWeight)

		for i := 0; i < length; i++ {
			if offset < weights[i] {
				return invokers[i]
			}
		}
	}
	// If all invokers have the same weight value or totalWeight=0, return evenly.
	return invokers[rand.Intn(length)]
}
