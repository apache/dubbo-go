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

package p2c

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"

	"go.uber.org/mock/gomock"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/metrics"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	protoinvoc "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestDefaultRnd(t *testing.T) {
	t.Run("n <= 1", func(t *testing.T) {
		i, j := defaultRnd(1)
		assert.Equal(t, 0, i)
		assert.Equal(t, 0, j)
	})

	t.Run("n == 2", func(t *testing.T) {
		// Either order must come up, or Select always prefers the same invoker.
		seen := map[[2]int]int{}
		for k := 0; k < 1000; k++ {
			i, j := defaultRnd(2)
			seen[[2]int{i, j}]++
		}
		assert.Len(t, seen, 2)
		assert.Positive(t, seen[[2]int{0, 1}])
		assert.Positive(t, seen[[2]int{1, 0}])
	})

	t.Run("n > 2", func(t *testing.T) {
		n := 5
		seen := map[[2]int]int{}
		for k := 0; k < 5000; k++ {
			i, j := defaultRnd(n)
			assert.True(t, i >= 0 && i < n)
			assert.True(t, j >= 0 && j < n)
			assert.NotEqual(t, i, j)
			seen[[2]int{i, j}]++
		}
		// Every ordered pair of distinct indices comes up.
		assert.Len(t, seen, n*(n-1))
	})
}

// TestSelectTwoInvokersSpreads selects with the default random picker and checks
// that both of two invokers get a fair share when p2c has no metrics to compare
// or the metrics are equal.
func TestSelectTwoInvokersSpreads(t *testing.T) {
	lb := &p2cLoadBalance{randomPicker: defaultRnd}
	invocation := protoinvoc.NewRPCInvocation("TestMethod", []any{}, nil)
	url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
	url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")
	ivkArr := []base.Invoker{base.NewBaseInvoker(url0), base.NewBaseInvoker(url1)}

	for _, tc := range []struct {
		name      string
		remaining any
		err       error
	}{
		{name: "no metrics", err: metrics.ErrMetricsNotFound},
		{name: "equal metrics", remaining: uint64(10)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := metrics.NewMockMetrics(ctrl)
			setLocalMetrics(t, m)
			m.EXPECT().
				GetMethodMetrics(gomock.Any(), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
				AnyTimes().
				Return(tc.remaining, tc.err)

			const total = 2000
			counts := map[string]int{}
			for k := 0; k < total; k++ {
				counts[lb.Select(ivkArr, invocation).GetURL().Location]++
			}
			// Expected 1000 each; the band is more than 13 standard deviations wide.
			for _, ivk := range ivkArr {
				got := counts[ivk.GetURL().Location]
				assert.Greater(t, got, total*35/100, "invoker %s", ivk.GetURL().Location)
				assert.Less(t, got, total*65/100, "invoker %s", ivk.GetURL().Location)
			}
		})
	}
}

func setLocalMetrics(t *testing.T, localMetrics metrics.Metrics) {
	t.Helper()

	originalLocalMetrics := metrics.LocalMetrics
	metrics.LocalMetrics = localMetrics
	t.Cleanup(func() {
		metrics.LocalMetrics = originalLocalMetrics
	})
}

func TestLoadBalance(t *testing.T) {
	// Create P2C load balancer with deterministic randomPicker for repeatable tests.
	// Always returns fixed indices (0,1) except when n <= 1.
	lb := newP2CLoadBalance(func(n int) (i, j int) {
		if n <= 1 {
			return 0, 0
		}
		if n == 2 {
			return 0, 1
		}
		return 0, 1
	})
	invocation := protoinvoc.NewRPCInvocation("TestMethod", []any{}, nil)

	t.Run("no invokers", func(t *testing.T) {
		ivk := lb.Select([]base.Invoker{}, invocation)
		assert.Nil(t, ivk)
	})

	t.Run("one invoker", func(t *testing.T) {
		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")

		ivkArr := []base.Invoker{
			base.NewBaseInvoker(url0),
		}
		ivk := lb.Select(ivkArr, invocation)
		assert.Equal(t, ivkArr[0].GetURL().String(), ivk.GetURL().String())
	})

	t.Run("two invokers", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := metrics.NewMockMetrics(ctrl)
		setLocalMetrics(t, m)

		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
		url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url0), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(10), nil)
		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url1), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(5), nil)

		ivkArr := []base.Invoker{
			base.NewBaseInvoker(url0),
			base.NewBaseInvoker(url1),
		}

		ivk := lb.Select(ivkArr, invocation)

		assert.Equal(t, ivkArr[0].GetURL().String(), ivk.GetURL().String())
	})

	t.Run("multiple invokers", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := metrics.NewMockMetrics(ctrl)
		setLocalMetrics(t, m)

		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
		url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")
		url2, _ := common.NewURL("dubbo://192.168.1.2:20000/com.ikurento.user.UserProvider")

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url0), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(10), nil)
		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url1), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(5), nil)

		ivkArr := []base.Invoker{
			base.NewBaseInvoker(url0),
			base.NewBaseInvoker(url1),
			base.NewBaseInvoker(url2),
		}

		ivk := lb.Select(ivkArr, invocation)

		assert.Equal(t, ivkArr[0].GetURL().String(), ivk.GetURL().String())
	})

	t.Run("metrics i not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := metrics.NewMockMetrics(ctrl)
		setLocalMetrics(t, m)

		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
		url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")
		url2, _ := common.NewURL("dubbo://192.168.1.2:20000/com.ikurento.user.UserProvider")

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url0), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(0, metrics.ErrMetricsNotFound)

		ivkArr := []base.Invoker{
			base.NewBaseInvoker(url0),
			base.NewBaseInvoker(url1),
			base.NewBaseInvoker(url2),
		}

		ivk := lb.Select(ivkArr, invocation)

		assert.Equal(t, ivkArr[0].GetURL().String(), ivk.GetURL().String())
	})

	t.Run("metrics j not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := metrics.NewMockMetrics(ctrl)
		setLocalMetrics(t, m)

		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
		url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")
		url2, _ := common.NewURL("dubbo://192.168.1.2:20000/com.ikurento.user.UserProvider")

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url0), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(0), nil)

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url1), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(0), metrics.ErrMetricsNotFound)

		ivkArr := []base.Invoker{
			base.NewBaseInvoker(url0),
			base.NewBaseInvoker(url1),
			base.NewBaseInvoker(url2),
		}

		ivk := lb.Select(ivkArr, invocation)

		assert.Equal(t, ivkArr[1].GetURL().String(), ivk.GetURL().String())
	})

	t.Run("metrics i nil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := metrics.NewMockMetrics(ctrl)
		setLocalMetrics(t, m)

		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
		url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url0), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(nil, nil)

		ivkArr := []base.Invoker{
			base.NewBaseInvoker(url0),
			base.NewBaseInvoker(url1),
		}

		assert.NotPanics(t, func() {
			ivk := lb.Select(ivkArr, invocation)
			assert.Equal(t, ivkArr[0].GetURL().String(), ivk.GetURL().String())
		})
	})

	t.Run("metrics j nil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := metrics.NewMockMetrics(ctrl)
		setLocalMetrics(t, m)

		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
		url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url0), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(0), nil)

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url1), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(nil, nil)

		ivkArr := []base.Invoker{
			base.NewBaseInvoker(url0),
			base.NewBaseInvoker(url1),
		}

		assert.NotPanics(t, func() {
			ivk := lb.Select(ivkArr, invocation)
			assert.Equal(t, ivkArr[1].GetURL().String(), ivk.GetURL().String())
		})
	})

	t.Run("metrics i wrong type", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := metrics.NewMockMetrics(ctrl)
		setLocalMetrics(t, m)

		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
		url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url0), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return("bad-metrics", nil)

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url1), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(10), nil)

		ivkArr := []base.Invoker{
			base.NewBaseInvoker(url0),
			base.NewBaseInvoker(url1),
		}

		assert.NotPanics(t, func() {
			ivk := lb.Select(ivkArr, invocation)
			assert.Equal(t, ivkArr[0].GetURL().String(), ivk.GetURL().String())
		})
	})

	t.Run("metrics j wrong type", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := metrics.NewMockMetrics(ctrl)
		setLocalMetrics(t, m)

		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
		url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url0), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(10), nil)

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url1), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return("bad-metrics", nil)

		ivkArr := []base.Invoker{
			base.NewBaseInvoker(url0),
			base.NewBaseInvoker(url1),
		}

		assert.NotPanics(t, func() {
			ivk := lb.Select(ivkArr, invocation)
			assert.Equal(t, ivkArr[1].GetURL().String(), ivk.GetURL().String())
		})
	})
}
