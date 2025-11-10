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
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
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
		i, j := defaultRnd(2)
		assert.Equal(t, 0, i)
		assert.Equal(t, 1, j)
	})

	t.Run("n > 2", func(t *testing.T) {
		n := 5
		i, j := defaultRnd(n)
		assert.True(t, i >= 0 && i < n)
		assert.True(t, j >= 0 && j < n)
		assert.NotEqual(t, i, j)
	})
}

func TestLoadBalance(t *testing.T) {
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
		metrics.LocalMetrics = m

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
		metrics.LocalMetrics = m

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
		metrics.LocalMetrics = m

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
		metrics.LocalMetrics = m

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

}
