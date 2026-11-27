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
	"fmt"
	"slices"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestIWrrRoundRobinSelect(t *testing.T) {
	loadBalance := newInterleavedWeightedRoundRobinBalance()

	var invokers []base.Invoker
	assert.Nil(t, loadBalance.Select(invokers, &invocation.RPCInvocation{}))

	url, _ := common.NewURL(fmt.Sprintf("dubbo://%s:%d/org.apache.demo.HelloService",
		constant.LocalHostValue, constant.DefaultPort))
	invokers = append(invokers, base.NewBaseInvoker(url))
	i := loadBalance.Select(invokers, &invocation.RPCInvocation{})
	assert.True(t, i.GetURL().URLEqual(url))

	for i := 1; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/org.apache.demo.HelloService", i))
		invokers = append(invokers, base.NewBaseInvoker(url))
	}
	loadBalance.Select(invokers, &invocation.RPCInvocation{})
}

func TestIWrrRoundRobinByWeight(t *testing.T) {
	invokers := testInvokers(t, 900, 100)
	call := invocation.NewRPCInvocation("sayHello", nil, nil)
	selected := make(map[base.Invoker]int)
	for range 100 {
		lb := extension.GetLoadbalance(constant.LoadBalanceKeyInterleavedWeightedRoundRobin)
		selected[lb.Select(invokers, call)]++
	}

	assert.Equal(t, map[base.Invoker]int{invokers[0]: 90, invokers[1]: 10}, selected)
}

func testInvokers(t *testing.T, weights ...int) []base.Invoker {
	t.Helper()
	invokers := make([]base.Invoker, len(weights))
	for i, weight := range weights {
		url, err := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%d:20000/org.apache.demo.HelloService?weight=%d", i+1, weight))
		require.NoError(t, err)
		invokers[i] = base.NewBaseInvoker(url)
	}
	return invokers
}

func TestIWrrWeightChanges(t *testing.T) {
	lb := newInterleavedWeightedRoundRobinBalance()
	invokers := testInvokers(t, 900, 100)
	call := invocation.NewRPCInvocation(t.Name(), nil, nil)
	lb.Select(invokers, call)

	// Changing the weight also changes the GCD, from 100 to 50.
	invokers[0].GetURL().SetParam(constant.WeightKey, "250")
	selected := make(map[base.Invoker]int)
	for range 7 {
		selected[lb.Select(invokers, call)]++
	}
	assert.Equal(t, map[base.Invoker]int{invokers[0]: 5, invokers[1]: 2}, selected)

	invokers[0].GetURL().SetParam(constant.WeightKey, "0")
	for range 10 {
		assert.Same(t, invokers[1], lb.Select(invokers, call))
	}

	invokers[1].GetURL().SetParam(constant.WeightKey, "0")
	clear(selected)
	for range 10 {
		selected[lb.Select(invokers, call)]++
	}
	assert.Equal(t, map[base.Invoker]int{invokers[0]: 5, invokers[1]: 5}, selected)
}

func TestIWrrCandidateSets(t *testing.T) {
	lb := newInterleavedWeightedRoundRobinBalance()
	invokers := testInvokers(t, 900, 100, 100, 300)
	call := invocation.NewRPCInvocation(t.Name(), nil, nil)
	firstSet := invokers[:2]
	reversedFirstSet := []base.Invoker{invokers[1], invokers[0]}
	secondSet := []base.Invoker{invokers[0], invokers[2]}
	firstSelected, secondSelected := make(map[base.Invoker]int), make(map[base.Invoker]int)
	for i := range 100 {
		set := firstSet
		if i%2 == 1 {
			set = reversedFirstSet
		}
		firstSelected[lb.Select(set, call)]++
		secondSelected[lb.Select(secondSet, call)]++
	}
	assert.Equal(t, map[base.Invoker]int{invokers[0]: 90, invokers[1]: 10}, firstSelected)
	assert.Equal(t, map[base.Invoker]int{invokers[0]: 90, invokers[2]: 10}, secondSelected)

	// Same endpoints and length, different middle invoker: hits viewKey hint and exercises view.matches false path.
	hintSetA := []base.Invoker{invokers[0], invokers[1], invokers[2]}
	hintSetB := []base.Invoker{invokers[0], invokers[3], invokers[2]}
	hintSelectedA, hintSelectedB := make(map[base.Invoker]int), make(map[base.Invoker]int)
	for range 143 {
		hintSelectedA[lb.Select(hintSetA, call)]++
		hintSelectedB[lb.Select(hintSetB, call)]++
	}
	assert.Equal(t, map[base.Invoker]int{invokers[0]: 117, invokers[1]: 13, invokers[2]: 13}, hintSelectedA)
	assert.Equal(t, map[base.Invoker]int{invokers[0]: 99, invokers[3]: 33, invokers[2]: 11}, hintSelectedB)
}

func TestIWrrReplacedInvokers(t *testing.T) {
	lb := newInterleavedWeightedRoundRobinBalance()
	original := testInvokers(t, 1, 1)
	replacement := testInvokers(t, 1, 1)
	call := invocation.NewRPCInvocation(t.Name(), nil, nil)

	first := lb.Select(original, call)
	second := lb.Select(replacement, call)
	require.True(t, slices.Contains(replacement, second), "selected invoker must come from the current list")
	assert.NotEqual(t, first.GetURL().Key(), second.GetURL().Key())
}

func TestIWrrMethodIsolation(t *testing.T) {
	lb := newInterleavedWeightedRoundRobinBalance()
	invokers := testInvokers(t, 1, 1)
	first := invocation.NewRPCInvocation(t.Name()+"First", nil, nil)
	second := invocation.NewRPCInvocation(t.Name()+"Second", nil, nil)
	firstSelected, secondSelected := make(map[base.Invoker]int), make(map[base.Invoker]int)
	for range 10 {
		firstSelected[lb.Select(invokers, first)]++
		secondSelected[lb.Select(invokers, second)]++
	}
	want := map[base.Invoker]int{invokers[0]: 5, invokers[1]: 5}
	assert.Equal(t, want, firstSelected)
	assert.Equal(t, want, secondSelected)

	genericFirst := invocation.NewRPCInvocation(constant.Generic, []any{t.Name() + "GenericFirst", []string{}, []any{}}, nil)
	genericSecond := invocation.NewRPCInvocation(constant.Generic, []any{t.Name() + "GenericSecond", []string{}, []any{}}, nil)
	genericFirstSelected, genericSecondSelected := make(map[base.Invoker]int), make(map[base.Invoker]int)
	for range 10 {
		genericFirstSelected[lb.Select(invokers, genericFirst)]++
		genericSecondSelected[lb.Select(invokers, genericSecond)]++
	}
	assert.Equal(t, want, genericFirstSelected)
	assert.Equal(t, want, genericSecondSelected)
}

func TestNewInterleavedWeightedRoundRobinPicker(t *testing.T) {
	call := invocation.NewRPCInvocation(t.Name(), nil, nil)

	emptyPicker := NewInterleavedweightedRoundRobin(nil, call)
	assert.Nil(t, emptyPicker.Pick(call))

	invokers := testInvokers(t, 900, 100)
	first, second := invokers[0], invokers[1]
	picker := NewInterleavedweightedRoundRobin(invokers, call)
	// Mutating the caller's slice after construction must not affect the picker.
	invokers[0], invokers[1] = second, first

	selected := make(map[base.Invoker]int)
	for range 100 {
		selected[picker.Pick(call)]++
	}
	assert.Equal(t, map[base.Invoker]int{first: 90, second: 10}, selected)
}

func TestIWrrConcurrentSelect(t *testing.T) {
	invokers := testInvokers(t, 900, 100)
	call := invocation.NewRPCInvocation(t.Name(), nil, nil)
	const workers, picksPerWorker = 10, 100
	results := make(chan base.Invoker, workers*picksPerWorker)
	for range workers {
		go func() {
			for range picksPerWorker {
				lb := extension.GetLoadbalance(constant.LoadBalanceKeyInterleavedWeightedRoundRobin)
				results <- lb.Select(invokers, call)
			}
		}()
	}

	selected := make(map[base.Invoker]int)
	for range workers * picksPerWorker {
		selected[<-results]++
	}
	assert.Equal(t, map[base.Invoker]int{invokers[0]: 900, invokers[1]: 100}, selected)
}
