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
	"strings"
	"testing"
)

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/cluster/directory"
	"github.com/apache/dubbo-go/cluster/loadbalance"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/mock"
)

var (
	availableUrl, _ = common.NewURL(context.Background(), "dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")
)

func registerAvailable(t *testing.T, invoker *mock.MockInvoker) protocol.Invoker {
	extension.SetLoadbalance("random", loadbalance.NewRandomLoadBalance)
	availableCluster := NewAvailableCluster()

	invokers := []protocol.Invoker{}
	invokers = append(invokers, invoker)
	invoker.EXPECT().GetUrl().Return(availableUrl)

	staticDir := directory.NewStaticDirectory(invokers)
	clusterInvoker := availableCluster.Join(staticDir)
	return clusterInvoker
}

func TestAvailableClusterInvokerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerAvailable(t, invoker)

	mockResult := &protocol.RPCResult{Rest: rest{tried: 0, success: true}}
	invoker.EXPECT().IsAvailable().Return(true)
	invoker.EXPECT().Invoke(gomock.Any()).Return(mockResult)

	result := clusterInvoker.Invoke(&invocation.RPCInvocation{})

	assert.Equal(t, mockResult, result)
}

func TestAvailableClusterInvokerNoAvail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerAvailable(t, invoker)

	invoker.EXPECT().IsAvailable().Return(false)

	result := clusterInvoker.Invoke(&invocation.RPCInvocation{})

	assert.NotNil(t, result.Error())
	assert.True(t, strings.Contains(result.Error().Error(), "no provider available"))
	assert.Nil(t, result.Result())
}
