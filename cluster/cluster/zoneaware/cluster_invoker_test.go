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

package zoneaware

import (
	"context"
	"fmt"
	"testing"
)

import (
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

import (
	clusterpkg "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/random"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
)

func TestZoneWareInvokerWithPreferredSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	// In Go versions 1.14+, if you pass a *testing.T
	// into gomock.NewController(t) you no longer need to call ctrl.Finish().
	// defer ctrl.Finish()

	mockResult := &protocol.RPCResult{
		Attrs: map[string]interface{}{constant.PreferredKey: "true"},
		Rest:  clusterpkg.Rest{Tried: 0, Success: true},
	}

	var invokers []protocol.Invoker
	for i := 0; i < 2; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().GetUrl().Return(url).AnyTimes()
		if 0 == i {
			url.SetParam(constant.RegistryKey+"."+constant.PreferredKey, "true")
			invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
				func(invocation protocol.Invocation) protocol.Result {
					return mockResult
				}).AnyTimes()
		} else {
			invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
				func(invocation protocol.Invocation) protocol.Result {
					return &protocol.RPCResult{}
				}).AnyTimes()
		}

		invokers = append(invokers, invoker)
	}

	zoneAwareCluster := newZoneawareCluster()
	staticDir := static.NewDirectory(invokers)
	clusterInvoker := zoneAwareCluster.Join(staticDir)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})

	assert.Equal(t, mockResult, result)
}

func TestZoneWareInvokerWithWeightSuccess(t *testing.T) {
	extension.SetLoadbalance(constant.LoadBalanceKeyRandom, random.NewRandomLoadBalance)

	ctrl := gomock.NewController(t)
	// In Go versions 1.14+, if you pass a *testing.T
	// into gomock.NewController(t) you no longer need to call ctrl.Finish().
	// defer ctrl.Finish()

	w1 := "50"
	w2 := "200"

	var invokers []protocol.Invoker
	for i := 0; i < 2; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().GetUrl().Return(url).AnyTimes()
		url.SetParam(constant.RegistryKey+"."+constant.RegistryLabelKey, "true")
		if 1 == i {
			url.SetParam(constant.RegistryKey+"."+constant.WeightKey, w1)
			invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
				func(invocation protocol.Invocation) protocol.Result {
					return &protocol.RPCResult{
						Attrs: map[string]interface{}{constant.WeightKey: w1},
						Rest:  clusterpkg.Rest{Tried: 0, Success: true},
					}
				}).MaxTimes(100)
		} else {
			url.SetParam(constant.RegistryKey+"."+constant.WeightKey, w2)
			invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
				func(invocation protocol.Invocation) protocol.Result {
					return &protocol.RPCResult{
						Attrs: map[string]interface{}{constant.WeightKey: w2},
						Rest:  clusterpkg.Rest{Tried: 0, Success: true},
					}
				}).MaxTimes(100)
		}
		invokers = append(invokers, invoker)
	}

	zoneAwareCluster := newZoneawareCluster()
	staticDir := static.NewDirectory(invokers)
	clusterInvoker := zoneAwareCluster.Join(staticDir)

	var w2Count, w1Count int
	loop := 50
	for i := 0; i < loop; i++ {
		result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
		if w2 == result.Attachment(constant.WeightKey, "0") {
			w2Count++
		}
		if w1 == result.Attachment(constant.WeightKey, "0") {
			w1Count++
		}
		assert.NoError(t, result.Error())
	}
	t.Logf("loop count : %d, w1 height : %s | count : %d, w2 height : %s | count : %d", loop,
		w1, w1Count, w2, w2Count)
}

func TestZoneWareInvokerWithZoneSuccess(t *testing.T) {
	zoneArray := []string{"hangzhou", "shanghai"}

	ctrl := gomock.NewController(t)
	// In Go versions 1.14+, if you pass a *testing.T
	// into gomock.NewController(t) you no longer need to call ctrl.Finish().
	// defer ctrl.Finish()

	var invokers []protocol.Invoker
	for i := 0; i < 2; i++ {
		zoneValue := zoneArray[i]
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		url.SetParam(constant.RegistryKey+"."+constant.RegistryZoneKey, zoneValue)

		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().GetUrl().Return(url).AnyTimes()
		invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
			func(invocation protocol.Invocation) protocol.Result {
				return &protocol.RPCResult{
					Attrs: map[string]interface{}{constant.RegistryZoneKey: zoneValue},
					Rest:  clusterpkg.Rest{Tried: 0, Success: true},
				}
			}).AnyTimes()
		invokers = append(invokers, invoker)
	}

	zoneAwareCluster := newZoneawareCluster()
	staticDir := static.NewDirectory(invokers)
	clusterInvoker := zoneAwareCluster.Join(staticDir)

	inv := &invocation.RPCInvocation{}
	// zone hangzhou
	hz := zoneArray[0]
	inv.SetAttachment(constant.RegistryKey+"."+constant.RegistryZoneKey, hz)

	result := clusterInvoker.Invoke(context.Background(), inv)

	assert.Equal(t, hz, result.Attachment(constant.RegistryZoneKey, ""))
}

func TestZoneWareInvokerWithZoneForceFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	// In Go versions 1.14+, if you pass a *testing.T
	// into gomock.NewController(t) you no longer need to call ctrl.Finish().
	// defer ctrl.Finish()

	var invokers []protocol.Invoker
	for i := 0; i < 2; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))

		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().GetUrl().Return(url).AnyTimes()
		invokers = append(invokers, invoker)
	}

	zoneAwareCluster := newZoneawareCluster()
	staticDir := static.NewDirectory(invokers)
	clusterInvoker := zoneAwareCluster.Join(staticDir)

	inv := &invocation.RPCInvocation{}
	// zone hangzhou
	inv.SetAttachment(constant.RegistryKey+"."+constant.RegistryZoneKey, "hangzhou")
	// zone force
	inv.SetAttachment(constant.RegistryKey+"."+constant.RegistryZoneForceKey, "true")

	result := clusterInvoker.Invoke(context.Background(), inv)

	assert.NotNil(t, result.Error())
}
