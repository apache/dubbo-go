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

package aliasmethod

import (
	"fmt"
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestNewAliasMethodPicker(t *testing.T) {
	invokers := createTestInvokers(3, 100)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	picker := NewAliasMethodPicker(invokers, inv)

	assert.NotNil(t, picker)
	assert.Equal(t, 3, len(picker.invokers))
	assert.NotNil(t, picker.alias)
	assert.NotNil(t, picker.prob)
	assert.Greater(t, picker.weightSum, int64(0))
}

func TestAliasMethodPickerWithEqualWeights(t *testing.T) {
	invokers := createTestInvokers(3, 100)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	picker := NewAliasMethodPicker(invokers, inv)

	assert.Equal(t, int64(300), picker.weightSum)
	assert.Equal(t, 3, len(picker.alias))
	assert.Equal(t, 3, len(picker.prob))
}

func TestAliasMethodPickerWithDifferentWeights(t *testing.T) {
	invokers := []base.Invoker{}
	weights := []int{100, 200, 300}

	for i, weight := range weights {
		urlParams := url.Values{}
		urlParams.Set("methods.test."+constant.WeightKey, fmt.Sprintf("%d", weight))
		u, err := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%d:20000/com.test.UserService", i), common.WithParams(urlParams))
		require.NoError(t, err)
		invokers = append(invokers, base.NewBaseInvoker(u))
	}

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))
	picker := NewAliasMethodPicker(invokers, inv)

	assert.Equal(t, int64(600), picker.weightSum)
	assert.Equal(t, 3, len(picker.alias))
	assert.Equal(t, 3, len(picker.prob))
}

func TestAliasMethodPickerWithZeroWeights(t *testing.T) {
	invokers := createTestInvokersWithWeight(3, 0)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	picker := NewAliasMethodPicker(invokers, inv)

	// When all weights are zero, weightSum should be 1
	assert.Equal(t, int64(1), picker.weightSum)
	assert.Equal(t, 3, len(picker.alias))
	assert.Equal(t, 3, len(picker.prob))
}

func TestAliasMethodPickerPick(t *testing.T) {
	invokers := createTestInvokers(3, 100)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	picker := NewAliasMethodPicker(invokers, inv)

	// Pick multiple times to ensure it doesn't panic
	for i := 0; i < 100; i++ {
		invoker := picker.Pick()
		assert.NotNil(t, invoker)
		assert.Contains(t, invokers, invoker)
	}
}

func TestAliasMethodPickerPickSingleInvoker(t *testing.T) {
	invokers := createTestInvokers(1, 100)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	picker := NewAliasMethodPicker(invokers, inv)

	// With single invoker, should always return that invoker
	for i := 0; i < 10; i++ {
		invoker := picker.Pick()
		assert.Equal(t, invokers[0], invoker)
	}
}

func TestAliasMethodPickerPickDistribution(t *testing.T) {
	invokers := []base.Invoker{}
	// Create invokers with weight 1:2:3
	weights := []int{100, 200, 300}

	for i, weight := range weights {
		urlParams := url.Values{}
		urlParams.Set("methods.test."+constant.WeightKey, fmt.Sprintf("%d", weight))
		u, err := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%d:20000/com.test.UserService", i), common.WithParams(urlParams))
		require.NoError(t, err)
		invokers = append(invokers, base.NewBaseInvoker(u))
	}

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))
	picker := NewAliasMethodPicker(invokers, inv)

	counts := make(map[string]int)
	iterations := 6000

	for i := 0; i < iterations; i++ {
		invoker := picker.Pick()
		counts[invoker.GetURL().Location]++
	}

	// Verify all invokers were picked
	assert.Equal(t, 3, len(counts))

	// The distribution should roughly match the weight ratio (1:2:3)
	// Allow some variance due to randomness
	total := float64(iterations)

	// Verify each invoker was selected and check distribution
	count0, ok0 := counts["192.168.1.0:20000"]
	assert.True(t, ok0, "invoker 0 should be selected")
	assert.InDelta(t, 0.167, float64(count0)/total, 0.05) // ~1/6

	count1, ok1 := counts["192.168.1.1:20000"]
	assert.True(t, ok1, "invoker 1 should be selected")
	assert.InDelta(t, 0.333, float64(count1)/total, 0.05) // ~2/6

	count2, ok2 := counts["192.168.1.2:20000"]
	assert.True(t, ok2, "invoker 2 should be selected")
	assert.InDelta(t, 0.500, float64(count2)/total, 0.05) // ~3/6
}

func TestAliasMethodPickerProbInitialization(t *testing.T) {
	invokers := createTestInvokers(3, 100)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	picker := NewAliasMethodPicker(invokers, inv)

	// All probabilities should be between 0 and 1
	for _, prob := range picker.prob {
		assert.GreaterOrEqual(t, prob, 0.0)
		assert.LessOrEqual(t, prob, 1.0)
	}
}

func TestAliasMethodPickerAliasInitialization(t *testing.T) {
	invokers := createTestInvokers(3, 100)
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	picker := NewAliasMethodPicker(invokers, inv)

	// All alias indices should be valid
	for _, alias := range picker.alias {
		assert.GreaterOrEqual(t, alias, 0)
		assert.Less(t, alias, len(invokers))
	}
}

func TestAliasMethodPickerWithMixedWeights(t *testing.T) {
	invokers := []base.Invoker{}
	weights := []int{1, 100, 500}

	for i, weight := range weights {
		urlParams := url.Values{}
		urlParams.Set("methods.test."+constant.WeightKey, fmt.Sprintf("%d", weight))
		u, err := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%d:20000/com.test.UserService", i), common.WithParams(urlParams))
		require.NoError(t, err)
		invokers = append(invokers, base.NewBaseInvoker(u))
	}

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))
	picker := NewAliasMethodPicker(invokers, inv)

	assert.Equal(t, int64(601), picker.weightSum)

	// Pick and verify it doesn't panic
	for i := 0; i < 100; i++ {
		invoker := picker.Pick()
		assert.NotNil(t, invoker)
	}
}

// Helper functions

func createTestInvokers(count int, weight int) []base.Invoker {
	return createTestInvokersWithWeight(count, weight)
}

func createTestInvokersWithWeight(count int, weight int) []base.Invoker {
	invokers := []base.Invoker{}

	for i := 0; i < count; i++ {
		urlParams := url.Values{}
		if weight > 0 {
			urlParams.Set("methods.test."+constant.WeightKey, fmt.Sprintf("%d", weight))
		} else {
			urlParams.Set("methods.test."+constant.WeightKey, "0")
		}
		u, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%d:20000/com.test.UserService", i), common.WithParams(urlParams))
		invokers = append(invokers, base.NewBaseInvoker(u))
	}

	return invokers
}
