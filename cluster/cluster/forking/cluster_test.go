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

package forking

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

import (
	clusterpkg "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/roundrobin"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
)

var forkingUrl, _ = common.NewURL(
	fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LocalHostValue, constant.DefaultPort))

func registerForking(mockInvokers ...*mock.MockInvoker) protocol.Invoker {
	extension.SetLoadbalance(constant.LoadBalanceKeyRoundRobin, roundrobin.NewRRLoadBalance)

	var invokers []protocol.Invoker
	for _, ivk := range mockInvokers {
		invokers = append(invokers, ivk)
		ivk.EXPECT().GetUrl().Return(forkingUrl).AnyTimes()
	}
	staticDir := static.NewDirectory(invokers)

	forkingCluster := newForkingCluster()
	clusterInvoker := forkingCluster.Join(staticDir)
	return clusterInvoker
}

func TestForkingInvokeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokers := make([]*mock.MockInvoker, 0)

	mockResult := &protocol.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
	forkingUrl.AddParam(constant.ForksKey, strconv.Itoa(3))
	// forkingUrl.AddParam(constant.TimeoutKey, strconv.Itoa(constant.DefaultTimeout))

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
			func(protocol.Invocation) protocol.Result {
				wg.Done()
				return mockResult
			})
	}

	clusterInvoker := registerForking(invokers...)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	assert.Equal(t, mockResult, result)
	wg.Wait()
}

func TestForkingInvokeTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokers := make([]*mock.MockInvoker, 0)

	mockResult := &protocol.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
	forkingUrl.AddParam(constant.ForksKey, strconv.Itoa(3))

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
			func(protocol.Invocation) protocol.Result {
				time.Sleep(2 * time.Second)
				wg.Done()
				return mockResult
			})
	}

	clusterInvoker := registerForking(invokers...)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	assert.NotNil(t, result)
	assert.NotNil(t, result.Error())
	wg.Wait()
}

func TestForkingInvokeHalfTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokers := make([]*mock.MockInvoker, 0)

	mockResult := &protocol.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
	forkingUrl.AddParam(constant.ForksKey, strconv.Itoa(3))

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		if i == 1 {
			invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
				func(protocol.Invocation) protocol.Result {
					wg.Done()
					return mockResult
				}).AnyTimes()
		} else {
			invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
				func(protocol.Invocation) protocol.Result {
					time.Sleep(2 * time.Second)
					wg.Done()
					return mockResult
				}).AnyTimes()
		}
	}

	clusterInvoker := registerForking(invokers...)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	assert.Equal(t, mockResult, result)
	wg.Wait()
}
