/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster_impl

import (
	"context"
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
	"github.com/apache/dubbo-go/cluster/directory"
	"github.com/apache/dubbo-go/cluster/loadbalance"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/mock"
)

var (
	forkingUrl, _ = common.NewURL(context.TODO(), "dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")
)

func registerForking(t *testing.T, mockInvokers ...*mock.MockInvoker) protocol.Invoker {
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

func Test_ForkingInvokeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokers := make([]*mock.MockInvoker, 0)

	mockResult := &protocol.RPCResult{Rest: rest{tried: 0, success: true}}
	forkingUrl.AddParam(constant.FORKS_KEY, strconv.Itoa(3))
	//forkingUrl.AddParam(constant.TIMEOUT_KEY, strconv.Itoa(constant.DEFAULT_TIMEOUT))

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
			func(invocation protocol.Invocation) protocol.Result {
				wg.Done()
				return mockResult
			})
	}

	clusterInvoker := registerForking(t, invokers...)

	result := clusterInvoker.Invoke(&invocation.RPCInvocation{})
	assert.Equal(t, mockResult, result)
	wg.Wait()
}

func Test_ForkingInvokeTimeout(t *testing.T) {
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
			func(invocation protocol.Invocation) protocol.Result {
				time.Sleep(2 * time.Second)
				wg.Done()
				return mockResult
			})
	}

	clusterInvoker := registerForking(t, invokers...)

	result := clusterInvoker.Invoke(&invocation.RPCInvocation{})
	assert.NotNil(t, result)
	assert.NotNil(t, result.Error())
	wg.Wait()
}

func Test_ForkingInvokeHalfTimeout(t *testing.T) {
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
				func(invocation protocol.Invocation) protocol.Result {
					wg.Done()
					return mockResult
				})
		} else {
			invoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
				func(invocation protocol.Invocation) protocol.Result {
					time.Sleep(2 * time.Second)
					wg.Done()
					return mockResult
				})
		}
	}

	clusterInvoker := registerForking(t, invokers...)

	result := clusterInvoker.Invoke(&invocation.RPCInvocation{})
	assert.Equal(t, mockResult, result)
	wg.Wait()
}
