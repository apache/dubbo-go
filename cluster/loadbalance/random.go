// Copyright 2016-2019 hxmhlt
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package loadbalance

import (
	"math/rand"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

const name = "random"

func init() {
	extension.SetLoadbalance(name, NewRandomLoadBalance)
}

type randomLoadBalance struct {
}

func NewRandomLoadBalance() cluster.LoadBalance {
	return &randomLoadBalance{}
}

func (lb *randomLoadBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {
	var length int
	if length = len(invokers); length == 1 {
		return invokers[0]
	}
	sameWeight := true
	weights := make([]int64, length)

	firstWeight := GetWeight(invokers[0], invocation)
	totalWeight := firstWeight
	weights[0] = firstWeight

	for i := 1; i < length; i++ {
		weight := GetWeight(invokers[i], invocation)
		weights[i] = weight

		totalWeight += weight
		if sameWeight && weight != firstWeight {
			sameWeight = false
		}
	}

	if totalWeight > 0 && !sameWeight {
		// If (not every invoker has the same weight & at least one invoker's weight>0), select randomly based on totalWeight.
		offset := rand.Int63n(totalWeight)

		for i := 0; i < length; i++ {
			offset -= weights[i]
			if offset < 0 {
				return invokers[i]
			}
		}
	}
	// If all invokers have the same weight value or totalWeight=0, return evenly.
	return invokers[rand.Intn(length)]
}
