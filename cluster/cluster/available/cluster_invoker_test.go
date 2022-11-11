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

package available

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

var availableUrl, _ = common.NewURL(fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider",
	constant.LocalHostValue, constant.DefaultPort))

func registerAvailable(invoker *mock.MockInvoker) protocol.Invoker {
	extension.SetLoadbalance("random", random.NewRandomLoadBalance)
	availableCluster := NewAvailableCluster()

	invokers := []protocol.Invoker{}
	invokers = append(invokers, invoker)
	invoker.EXPECT().GetURL().Return(availableUrl).AnyTimes()
	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()

	staticDir := static.NewDirectory(invokers)
	clusterInvoker := availableCluster.Join(staticDir)
	return clusterInvoker
}

func TestAvailableClusterInvokerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerAvailable(invoker)

	mockResult := &protocol.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockResult).AnyTimes()

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})

	assert.Equal(t, mockResult, result)
}

func TestAvailableClusterInvokerNoAvail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerAvailable(invoker)

	invoker.EXPECT().IsAvailable().Return(false).AnyTimes()

	res := &protocol.RPCResult{Err: errors.New("no provider available")}
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(res).AnyTimes()

	result := clusterInvoker.Invoke(context.TODO(), &invocation.RPCInvocation{})

	assert.NotNil(t, result.Error())
	assert.True(t, strings.Contains(result.Error().Error(), "no provider available"))
	assert.Nil(t, result.Result())
}
