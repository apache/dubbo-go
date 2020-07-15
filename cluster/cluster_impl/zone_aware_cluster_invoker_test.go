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

package cluster_impl

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-go/cluster/directory"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_ZoneWareInvokerWithPreferredSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	// In Go versions 1.14+, if you pass a *testing.T into gomock.NewController(t) you no longer need to call ctrl.Finish().
	//defer ctrl.Finish()

	mockResult := &protocol.RPCResult{Attrs: map[string]string{constant.PREFERRED_KEY: "true"}, Rest: rest{tried: 0, success: true}}

	var invokers []protocol.Invoker
	for i := 0; i < 2; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().GetUrl().Return(url).AnyTimes()
		if 0 == i {
			url.SetParam(constant.REGISTRY_KEY+"."+constant.PREFERRED_KEY, "true")
			invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
				func(invocation protocol.Invocation) protocol.Result {
					return mockResult
				})
		} else {
			invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
				func(invocation protocol.Invocation) protocol.Result {
					return &protocol.RPCResult{}
				})
		}

		invokers = append(invokers, invoker)
	}

	zoneAwareCluster := NewZoneAwareCluster()
	staticDir := directory.NewStaticDirectory(invokers)
	clusterInvoker := zoneAwareCluster.Join(staticDir)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})

	assert.Equal(t, mockResult, result)
}

func Test_ZoneWareInvokerWithWeightSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	// In Go versions 1.14+, if you pass a *testing.T into gomock.NewController(t) you no longer need to call ctrl.Finish().
	//defer ctrl.Finish()

	w1 := "50"
	w2 := "200"

	var invokers []protocol.Invoker
	for i := 0; i < 2; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().GetUrl().Return(url).AnyTimes()
		if 1 == i {
			url.SetParam(constant.REGISTRY_KEY+"."+constant.WEIGHT_KEY, w1)
			invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
				func(invocation protocol.Invocation) protocol.Result {
					return &protocol.RPCResult{Attrs: map[string]string{constant.WEIGHT_KEY: w1}, Rest: rest{tried: 0, success: true}}
				}).MaxTimes(100)
		} else {
			url.SetParam(constant.REGISTRY_KEY+"."+constant.WEIGHT_KEY, w2)
			invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
				func(invocation protocol.Invocation) protocol.Result {
					return &protocol.RPCResult{Attrs: map[string]string{constant.WEIGHT_KEY: w2}, Rest: rest{tried: 0, success: true}}
				}).MaxTimes(100)
		}
		invokers = append(invokers, invoker)
	}

	zoneAwareCluster := NewZoneAwareCluster()
	staticDir := directory.NewStaticDirectory(invokers)
	clusterInvoker := zoneAwareCluster.Join(staticDir)

	var w2Count, w1Count int
	loop := 50
	for i := 0; i < loop; i++ {
		result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
		if w2 == result.Attachment(constant.WEIGHT_KEY, "0") {
			w2Count++
		}
		if w1 == result.Attachment(constant.WEIGHT_KEY, "0") {
			w1Count++
		}
		assert.NoError(t, result.Error())
	}
	t.Logf("loop count : %d, w1 value : %s | count : %d, w2 value : %s | count : %d", loop,
		w1, w1Count, w2, w2Count)
}

func Test_ZoneWareInvokerWithZoneSuccess(t *testing.T) {
	var zoneArray = []string{"hangzhou", "shanghai"}

	ctrl := gomock.NewController(t)
	// In Go versions 1.14+, if you pass a *testing.T into gomock.NewController(t) you no longer need to call ctrl.Finish().
	//defer ctrl.Finish()

	var invokers []protocol.Invoker
	for i := 0; i < 2; i++ {
		zoneValue := zoneArray[i]
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		url.SetParam(constant.REGISTRY_KEY+"."+constant.ZONE_KEY, zoneValue)

		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().GetUrl().Return(url).AnyTimes()
		invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
			func(invocation protocol.Invocation) protocol.Result {
				return &protocol.RPCResult{Attrs: map[string]string{constant.ZONE_KEY: zoneValue}, Rest: rest{tried: 0, success: true}}
			})
		invokers = append(invokers, invoker)
	}

	zoneAwareCluster := NewZoneAwareCluster()
	staticDir := directory.NewStaticDirectory(invokers)
	clusterInvoker := zoneAwareCluster.Join(staticDir)

	inv := &invocation.RPCInvocation{}
	// zone hangzhou
	hz := zoneArray[0]
	inv.SetAttachments(constant.REGISTRY_ZONE, hz)

	result := clusterInvoker.Invoke(context.Background(), inv)

	assert.Equal(t, hz, result.Attachment(constant.ZONE_KEY, ""))
}

func Test_ZoneWareInvokerWithZoneForceFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	// In Go versions 1.14+, if you pass a *testing.T into gomock.NewController(t) you no longer need to call ctrl.Finish().
	//defer ctrl.Finish()

	var invokers []protocol.Invoker
	for i := 0; i < 2; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))

		invoker := mock.NewMockInvoker(ctrl)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().GetUrl().Return(url).AnyTimes()
		invokers = append(invokers, invoker)
	}

	zoneAwareCluster := NewZoneAwareCluster()
	staticDir := directory.NewStaticDirectory(invokers)
	clusterInvoker := zoneAwareCluster.Join(staticDir)

	inv := &invocation.RPCInvocation{}
	// zone hangzhou
	inv.SetAttachments(constant.REGISTRY_ZONE, "hangzhou")
	inv.SetAttachments(constant.REGISTRY_ZONE_FORCE, "true")

	result := clusterInvoker.Invoke(context.Background(), inv)

	assert.NotNil(t, result.Error())
}
