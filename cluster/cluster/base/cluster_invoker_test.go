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

package base

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	ivkmock "dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"fmt"
	"github.com/golang/mock/gomock"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	clusterpkg "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/p2c"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/random"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

const (
	baseClusterInvokerMethodName = "getUser"
	baseClusterInvokerFormat     = "dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider"
)

func TestStickyNormal(t *testing.T) {
	var invokers []protocol.Invoker
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf(baseClusterInvokerFormat, i))
		url.SetParam("sticky", "true")
		invokers = append(invokers, clusterpkg.NewMockInvoker(url, 1))
	}
	base := &BaseClusterInvoker{}
	base.AvailableCheck = true
	var invoked []protocol.Invoker

	tmpRandomBalance := random.NewRandomLoadBalance()
	tmpInvocation := invocation.NewRPCInvocation(baseClusterInvokerMethodName, nil, nil)
	result := base.DoSelect(tmpRandomBalance, tmpInvocation, invokers, invoked)
	result1 := base.DoSelect(tmpRandomBalance, tmpInvocation, invokers, invoked)
	assert.Equal(t, result, result1)
}

func TestStickyNormalWhenError(t *testing.T) {
	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf(baseClusterInvokerFormat, i))
		url.SetParam("sticky", "true")
		invokers = append(invokers, clusterpkg.NewMockInvoker(url, 1))
	}
	base := &BaseClusterInvoker{}
	base.AvailableCheck = true

	var invoked []protocol.Invoker
	result := base.DoSelect(random.NewRandomLoadBalance(), invocation.NewRPCInvocation(baseClusterInvokerMethodName, nil, nil), invokers, invoked)
	invoked = append(invoked, result)
	result1 := base.DoSelect(random.NewRandomLoadBalance(), invocation.NewRPCInvocation(baseClusterInvokerMethodName, nil, nil), invokers, invoked)
	assert.NotEqual(t, result, result1)
}

func TestGetLoadBalance(t *testing.T) {

	t.Run("Default", func(t *testing.T) {
		ivk := genInvoker(t, 0)
		lb := GetLoadBalance(ivk, "")
		_, ok := lb.(*random.RandomLoadBalance)
		assert.True(t, ok)
	})
	
	t.Run("Method Param", func(t *testing.T) {
		method := "TestMethod"
		ivk := genInvoker(t, 0)
		ivk.GetURL().SetParam(fmt.Sprintf("methods.%s.%s", method, constant.LoadbalanceKey),
			constant.LoadBalanceKeyP2C)
		lb := GetLoadBalance(ivk, method)
		_, ok := lb.(*p2c.P2CLoadBalance)
		assert.True(t, ok)
	})

}

func TestDoSelectInvoker(t *testing.T) {

	t.Run("one healthy invoker", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		url, err := common.NewURL(fmt.Sprintf(baseClusterInvokerFormat, 0))
		assert.Nil(t, err)

		m := ivkmock.NewMockInvoker(ctrl)
		m.EXPECT().IsAvailable().Return(true)
		m.EXPECT().GetUrl().Return(url)

		defaultLb := extension.GetLoadbalance(constant.DefaultLoadBalance)
		clusterInvoker := &BaseClusterInvoker{}
		invoc := &invocation.RPCInvocation{}
		ivk := clusterInvoker.doSelectInvoker(defaultLb, invoc, []protocol.Invoker{m}, []protocol.Invoker{})

		assert.Equal(t, url.String(), ivk.GetURL().String())
	})
	
	t.Run("one unhealthy invoker", func(t *testing.T) {
		
	})

}

func genInvoker(t *testing.T, index int) *protocol.BaseInvoker {
	url, err := common.NewURL(fmt.Sprintf(baseClusterInvokerFormat, index))
	assert.Nil(t, err)
	return protocol.NewBaseInvoker(url)
}
