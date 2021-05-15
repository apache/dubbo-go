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
	"errors"
	"fmt"
	"testing"
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

var broadcastUrl, _ = common.NewURL(
	fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LOCAL_HOST_VALUE, constant.DEFAULT_PORT))

func registerBroadcast(mockInvokers ...*mock.MockInvoker) protocol.Invoker {
	extension.SetLoadbalance("random", loadbalance.NewRandomLoadBalance)

	invokers := []protocol.Invoker{}
	for i, ivk := range mockInvokers {
		invokers = append(invokers, ivk)
		if i == 0 {
			ivk.EXPECT().GetUrl().Return(broadcastUrl)
		}
	}
	staticDir := directory.NewStaticDirectory(invokers)

	broadcastCluster := NewBroadcastCluster()
	clusterInvoker := broadcastCluster.Join(staticDir)
	return clusterInvoker
}

func TestBroadcastInvokeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokers := make([]*mock.MockInvoker, 0)

	mockResult := &protocol.RPCResult{Rest: rest{tried: 0, success: true}}
	for i := 0; i < 3; i++ {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().Invoke(gomock.Any()).Return(mockResult)
	}

	clusterInvoker := registerBroadcast(invokers...)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	assert.Equal(t, mockResult, result)
}

func TestBroadcastInvokeFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokers := make([]*mock.MockInvoker, 0)

	mockResult := &protocol.RPCResult{Rest: rest{tried: 0, success: true}}
	mockFailedResult := &protocol.RPCResult{Err: errors.New("just failed")}
	for i := 0; i < 10; i++ {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().Invoke(gomock.Any()).Return(mockResult)
	}
	{
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().Invoke(gomock.Any()).Return(mockFailedResult)
	}
	for i := 0; i < 10; i++ {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().Invoke(gomock.Any()).Return(mockResult)
	}

	clusterInvoker := registerBroadcast(invokers...)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	assert.Equal(t, mockFailedResult.Err, result.Error())
}
