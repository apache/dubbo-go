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

// Package aliasmethod implements alias-method algorithm load balance strategy.
package aliasmethod // weighted random with alias-method algorithm

import (
	"math/rand"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type aliasMethodPicker struct {
	invokers []protocol.Invoker // Instance

	weightSum int64
	alias     []int
	prob      []float64
}

func NewAliasMethodPicker(invokers []protocol.Invoker, invocation protocol.Invocation) *aliasMethodPicker {
	am := &aliasMethodPicker{
		invokers: invokers,
	}
	am.init(invocation)
	return am
}

// Alias Method: https://en.wikipedia.org/wiki/Alias_method
func (am *aliasMethodPicker) init(invocation protocol.Invocation) {
	n := len(am.invokers)
	weights := make([]int64, n)
	am.alias = make([]int, n)
	am.prob = make([]float64, n)

	totalWeight := int64(0)

	scaledProb := make([]float64, n)
	small := make([]int, 0, n)
	large := make([]int, 0, n)

	for i, invoker := range am.invokers {
		weight := loadbalance.GetWeight(invoker, invocation)
		weights[i] = weight
		totalWeight += weight
	}
	// when invoker weight all zero
	if totalWeight <= 0 {
		totalWeight = int64(1)
	}
	am.weightSum = totalWeight

	for i, weight := range weights {
		scaledProb[i] = float64(weight) * float64(n) / float64(totalWeight)
		if scaledProb[i] < 1.0 {
			small = append(small, i)
		} else {
			large = append(large, i)
		}
	}

	for len(small) > 0 && len(large) > 0 {
		l := small[len(small)-1]
		small = small[:len(small)-1]
		g := large[len(large)-1]
		large = large[:len(large)-1]

		am.prob[l] = scaledProb[l]
		am.alias[l] = g

		scaledProb[g] -= 1.0 - scaledProb[l]
		if scaledProb[g] < 1.0 {
			small = append(small, g)
		} else {
			large = append(large, g)
		}
	}

	for len(large) > 0 {
		g := large[len(large)-1]
		large = large[:len(large)-1]
		am.prob[g] = 1.0
	}

	for len(small) > 0 {
		l := small[len(small)-1]
		small = small[:len(small)-1]
		am.prob[l] = 1.0
	}
}

func (am *aliasMethodPicker) Pick() protocol.Invoker {
	i := rand.Intn(len(am.invokers))
	if rand.Float64() < am.prob[i] {
		return am.invokers[i]
	}
	return am.invokers[am.alias[i]]
}
