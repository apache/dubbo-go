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

package ringhash

import (
	"fmt"
	"math"
	"math/bits"
	"sort"
	"strconv"
)

import (
	"github.com/cespare/xxhash/v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
	"dubbo.apache.org/dubbo-go/v3/xds/utils/grpcrand"
)

type invokerWithWeight struct {
	invoker protocol.Invoker
	weight  float64
}

type ringEntry struct {
	idx     int
	hash    uint64
	invoker protocol.Invoker
}

func (lb *ringhashLoadBalance) generateRing(invokers []invokerWrapper, minRingSize, maxRingSize uint64) ([]*ringEntry, error) {
	normalizedWeights, minWeight, err := normalizeWeights(invokers)
	if err != nil {
		return nil, err
	}
	// Normalized weights for {3,3,4} is {0.3,0.3,0.4}.

	// Scale up the size of the ring such that the least-weighted host gets a
	// whole number of hashes on the ring.
	//
	// Note that size is limited by the input max/min.
	scale := math.Min(math.Ceil(minWeight*float64(minRingSize))/minWeight, float64(maxRingSize))
	ringSize := math.Ceil(scale)
	items := make([]*ringEntry, 0, int(ringSize))

	// For each entry, scale*weight nodes are generated in the ring.
	//
	// Not all of these are whole numbers. E.g. for weights {a:3,b:3,c:4}, if
	// ring size is 7, scale is 6.66. The numbers of nodes will be
	// {a,a,b,b,c,c,c}.
	//
	// A hash is generated for each item, and later the results will be sorted
	// based on the hash.
	var (
		idx       int
		targetIdx float64
	)
	for _, inw := range normalizedWeights {
		targetIdx += scale * inw.weight
		for float64(idx) < targetIdx {
			h := xxhash.Sum64String(inw.invoker.GetURL().String() + strconv.Itoa(len(items)))
			items = append(items, &ringEntry{idx: idx, hash: h, invoker: inw.invoker})
			idx++
		}
	}

	// Sort items based on hash, to prepare for binary search.
	sort.Slice(items, func(i, j int) bool { return items[i].hash < items[j].hash })
	for i, ii := range items {
		ii.idx = i
	}
	return items, nil
}

// normalizeWeights divides all the weights by the sum, so that the total weight
// is 1.
func normalizeWeights(invokers []invokerWrapper) ([]invokerWithWeight, float64, error) {
	var weightSum int
	for _, v := range invokers {
		weightSum += v.weight
	}
	if weightSum == 0 {
		return nil, 0, fmt.Errorf("total weight of all endpoints is 0")
	}
	weightSumF := float64(weightSum)
	ret := make([]invokerWithWeight, 0, len(invokers))
	min := math.MaxFloat64
	for _, invoker := range invokers {
		nw := float64(invoker.weight) / weightSumF
		ret = append(ret, invokerWithWeight{invoker: invoker.invoker, weight: nw})
		if nw < min {
			min = nw
		}
	}
	return ret, min, nil
}

func (lb *ringhashLoadBalance) pick(h uint64, items []*ringEntry) *ringEntry {
	i := sort.Search(len(items), func(i int) bool { return items[i].hash >= h })
	if i == len(items) {
		// If not found, and h is greater than the largest hash, return the
		// first item.
		i = 0
	}
	return items[i]
}

func (lb *ringhashLoadBalance) generateHash(invocation protocol.Invocation, hashPolicies []*resource.HashPolicy) uint64 {
	var (
		hash          uint64
		generatedHash bool
	)
	for _, policy := range hashPolicies {
		var (
			policyHash          uint64
			generatedPolicyHash bool
		)
		switch policy.HashPolicyType {
		case resource.HashPolicyTypeHeader:
			value, ok := invocation.GetAttachment(policy.HeaderName)
			if !ok || len(value) == 0 {
				continue
			}
			policyHash = xxhash.Sum64String(value)
			generatedHash = true
			generatedPolicyHash = true
		case resource.HashPolicyTypeChannelID:
			// Hash the ClientConn pointer which logically uniquely
			// identifies the client.
			policyHash = xxhash.Sum64String(fmt.Sprintf("%p", &lb.client))
			generatedHash = true
			generatedPolicyHash = true
		}

		// Deterministically combine the hash policies. Rotating prevents
		// duplicate hash policies from canceling each other out and preserves
		// the 64 bits of entropy.
		if generatedPolicyHash {
			hash = bits.RotateLeft64(hash, 1)
			hash = hash ^ policyHash
		}

		// If terminal policy and a hash has already been generated, ignore the
		// rest of the policies and use that hash already generated.
		if policy.Terminal && generatedHash {
			break
		}
	}

	if generatedHash {
		return hash
	}
	// If no generated hash return a random long. In the grand scheme of things
	// this logically will map to choosing a random backend to route request to.
	return grpcrand.Uint64()
}
