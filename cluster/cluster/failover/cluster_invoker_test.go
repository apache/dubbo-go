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

package failover

import (
	"context"
	"errors"
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
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

var broadcastUrl, _ = common.NewURL(
	fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LocalHostValue, constant.DefaultPort))

func registerFailover(mockInvokers ...*mock.MockInvoker) base.Invoker {
	extension.SetLoadbalance("random", random.NewRandomLoadBalance)

	invokers := []base.Invoker{}
	for _, ivk := range mockInvokers {
		invokers = append(invokers, ivk)
		ivk.EXPECT().GetURL().Return(broadcastUrl).AnyTimes()
	}
	staticDir := static.NewDirectory(invokers)

	cluster := newFailoverClusterInvoker(staticDir)
	return cluster
}

func TestBroadcastInvokeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokers := make([]*mock.MockInvoker, 0)

	mockResult := &result.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
	for i := 0; i < 3; i++ {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockResult).AnyTimes()
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes() // Mock IsAvailable to return true
	}

	clusterInvoker := registerFailover(invokers...)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	assert.Equal(t, mockResult, result)
}

func TestBroadcastInvokeFailedOver(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokers := make([]*mock.MockInvoker, 0)

	mockResult := &result.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
	mockFailedResult := &result.RPCResult{Err: errors.New("just failed")}

	// Create 10 invokers that return a successful result
	for i := 0; i < 10; i++ {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes() // Mock IsAvailable to return true
		invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockResult).AnyTimes()
	}

	// Create 1 invoker that returns a failed result
	{
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().IsAvailable().Return(false).AnyTimes() // Mock IsAvailable to return true
		invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockFailedResult).AnyTimes()
	}

	// Create 10 more invokers that return successful results
	for i := 0; i < 10; i++ {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes() // Mock IsAvailable to return true
		invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockResult).AnyTimes()
	}

	// Register the clusterInvoker
	clusterInvoker := registerFailover(invokers...)

	// Perform the invocation
	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})

	// Assert that the result is the same as the failed result
	assert.NoError(t, result.Error())
}

func TestBroadcastInvokeFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokers := make([]*mock.MockInvoker, 0)

	mockFailedResult := &result.RPCResult{Err: errors.New("just failed")}

	// Create 1 invoker that returns a failed result
	{
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes() // Mock IsAvailable to return true
		invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockFailedResult).AnyTimes()
	}

	// Register the clusterInvoker
	clusterInvoker := registerFailover(invokers...)

	// Perform the invocation
	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})

	// Assert that the result is the same as the failed result
	assert.Equal(t, mockFailedResult.Err, result.Error())
}

func TestBroadcastInvokeFailedNoServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokers := make([]*mock.MockInvoker, 0)

	mockFailedResult := &result.RPCResult{Err: errors.New("just failed")}

	// Create 1 invoker that returns a failed result
	{
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().IsAvailable().Return(false).AnyTimes() // Mock IsAvailable to return true
		invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockFailedResult).AnyTimes()
	}

	// Register the clusterInvoker
	clusterInvoker := registerFailover(invokers...)

	// Perform the invocation
	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})

	// Assert that the result is the same as the failed result
	assert.Equal(t, "Failed to invoke the method  of the service com.ikurento.user.UserProvider .No provider is available because can't connect server.", result.Error().Error())
}
