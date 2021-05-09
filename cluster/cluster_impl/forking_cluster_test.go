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
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
)

var forkingUrl, _ = common.NewURL(
	fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LOCAL_HOST_VALUE, constant.DEFAULT_PORT))

func registerForking(mockInvokers ...*mock.MockInvoker) protocol.Invoker {
	extension.SetLoadbalance(loadbalance.RoundRobin, loadbalance.NewRoundRobinLoadBalance)

	invokers := []protocol.Invoker{}
	for i, ivk := range mockInvokers {
		invokers = append(invokers, ivk)
		if i == 0 {
			ivk.EXPECT().GetUrl().Return(forkingUrl)
		}
	}
	staticDir := directory.NewStaticDirectory(invokers)

	forkingCluster := NewForkingCluster()
	clusterInvoker := forkingCluster.Join(staticDir)
	return clusterInvoker
}

func TestForkingInvokeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokers := make([]*mock.MockInvoker, 0)

	mockResult := &protocol.RPCResult{Rest: rest{tried: 0, success: true}}
	forkingUrl.AddParam(constant.FORKS_KEY, strconv.Itoa(3))
	// forkingUrl.AddParam(constant.TIMEOUT_KEY, strconv.Itoa(constant.DEFAULT_TIMEOUT))

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

	mockResult := &protocol.RPCResult{Rest: rest{tried: 0, success: true}}
	forkingUrl.AddParam(constant.FORKS_KEY, strconv.Itoa(3))

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

	mockResult := &protocol.RPCResult{Rest: rest{tried: 0, success: true}}
	forkingUrl.AddParam(constant.FORKS_KEY, strconv.Itoa(3))

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
				})
		} else {
			invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
				func(protocol.Invocation) protocol.Result {
					time.Sleep(2 * time.Second)
					wg.Done()
					return mockResult
				})
		}
	}

	clusterInvoker := registerForking(invokers...)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	assert.Equal(t, mockResult, result)
	wg.Wait()
}
