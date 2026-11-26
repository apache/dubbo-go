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
package failsafe

import (
	"context"
	"errors"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/random"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// nilURLDirectory lists its invokers but has no URL of its own.
type nilURLDirectory struct {
	directory.Directory
}

func (nilURLDirectory) GetURL() *common.URL { return nil }

// TestFailSafeInvokeFailWithoutDirectoryURL checks that a failed call is still
// ignored when the directory has no URL.
func TestFailSafeInvokeFailWithoutDirectoryURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	invoker := mock.NewMockInvoker(ctrl)
	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
	invoker.EXPECT().GetURL().Return(failsafeUrl).AnyTimes()
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(&result.RPCResult{Err: errors.New("error")})

	extension.SetLoadbalance("random", random.NewRandomLoadBalance)
	clusterInvoker := newFailsafeCluster().Join(nilURLDirectory{static.NewDirectory([]base.Invoker{invoker})})

	var res result.Result
	require.NotPanics(t, func() {
		res = clusterInvoker.Invoke(context.Background(), invocation.NewRPCInvocation("GetUser", nil, nil))
	})
	require.NoError(t, res.Error())
	assert.Nil(t, res.Result())
}
