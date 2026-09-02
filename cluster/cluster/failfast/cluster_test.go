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

package failfast

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"
)

import (
	clusterpkg "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/random"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/tag"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

var failfastUrl, _ = common.NewURL(
	fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LocalHostValue, constant.DefaultPort))

// registerFailfast register failfastCluster to failfastCluster extension.
func registerFailfast(invoker *mock.MockInvoker) base.Invoker {
	extension.SetLoadbalance("random", random.NewRandomLoadBalance)
	failfastCluster := newFailfastCluster()

	var invokers []base.Invoker
	invokers = append(invokers, invoker)

	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
	invoker.EXPECT().GetURL().Return(failfastUrl).AnyTimes()

	staticDir := static.NewDirectory(invokers)
	clusterInvoker := failfastCluster.Join(staticDir)
	return clusterInvoker
}

func TestFailfastInvokeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailfast(invoker)

	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
	invoker.EXPECT().GetURL().Return(failfastUrl).AnyTimes()

	mockResult := &result.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}

	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockResult).AnyTimes()
	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})

	require.NoError(t, result.Error())
	res := result.Result().(clusterpkg.Rest)
	assert.True(t, res.Success)
	assert.Equal(t, 0, res.Tried)
}

func TestFailfastInvokeFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailfast(invoker)

	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
	invoker.EXPECT().GetURL().Return(failfastUrl).AnyTimes()

	mockResult := &result.RPCResult{Err: errors.New("error")}

	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockResult).AnyTimes()
	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})

	require.Error(t, result.Error())
	assert.Equal(t, "error", result.Error().Error())
	assert.Nil(t, result.Result())
}

// TestFailfastInvokeWithNoAvailableProvider verifies that invoking with
// no available providers returns an error instead of panicking.
func TestFailfastInvokeWithNoAvailableProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	extension.SetLoadbalance("random", random.NewRandomLoadBalance)

	// Simulate unavailable invoker causing DoSelect to return nil.
	invoker := mock.NewMockInvoker(ctrl)
	invoker.EXPECT().IsAvailable().Return(false).AnyTimes()
	invoker.EXPECT().GetURL().Return(failfastUrl).AnyTimes()

	staticDir := static.NewDirectory([]base.Invoker{invoker})
	clusterInvoker := newFailfastCluster().Join(staticDir)

	require.NotPanics(t, func() {
		res := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
		// Must return an error, not panic
		require.NotNil(t, res)
		require.Error(t, res.Error())
	})
}
