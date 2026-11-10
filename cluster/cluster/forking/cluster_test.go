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
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"
)

import (
	clusterpkg "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/roundrobin"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/tag"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

var forkingUrl, _ = common.NewURL(
	fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LocalHostValue, constant.DefaultPort))

func registerForking(mockInvokers ...*mock.MockInvoker) base.Invoker {
	extension.SetLoadbalance(constant.LoadBalanceKeyRoundRobin, roundrobin.NewRRLoadBalance)

	var invokers []base.Invoker
	for _, ivk := range mockInvokers {
		invokers = append(invokers, ivk)
		ivk.EXPECT().GetURL().Return(forkingUrl).AnyTimes()
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

	mockResult := &result.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
	forkingUrl.AddParam(constant.ForksKey, strconv.Itoa(3))
	// forkingUrl.AddParam(constant.TimeoutKey, strconv.Itoa(constant.DefaultTimeout))

	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(
			func(context.Context, base.Invocation) result.Result {
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

	mockResult := &result.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
	forkingUrl.AddParam(constant.ForksKey, strconv.Itoa(3))

	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(
			func(context.Context, base.Invocation) result.Result {
				time.Sleep(2 * time.Second)
				wg.Done()
				return mockResult
			})
	}

	clusterInvoker := registerForking(invokers...)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	assert.NotNil(t, result)
	require.Error(t, result.Error())
	wg.Wait()
}

func TestForkingInvokeHalfTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invokers := make([]*mock.MockInvoker, 0)

	mockResult := &result.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
	forkingUrl.AddParam(constant.ForksKey, strconv.Itoa(3))

	var wg sync.WaitGroup
	wg.Add(2)
	for i := range 2 {
		invoker := mock.NewMockInvoker(ctrl)
		invokers = append(invokers, invoker)
		invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
		if i == 1 {
			invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(
				func(context.Context, base.Invocation) result.Result {
					wg.Done()
					return mockResult
				}).AnyTimes()
		} else {
			invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(
				func(context.Context, base.Invocation) result.Result {
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

// registryURLDirectory reports the registry URL as its own URL, like the
// registry directory does, while listing invokers that carry the reference URL.
type registryURLDirectory struct {
	directory.Directory
	registryURL *common.URL
}

func (d *registryURLDirectory) GetURL() *common.URL {
	return d.registryURL
}

func TestForkingInvokeConfiguredTimeout(t *testing.T) {
	// Longer than the 1s used when no timeout is configured.
	const providerDelay = 1500 * time.Millisecond
	const service = "tri://127.0.0.1:20000/com.ikurento.user.UserProvider"

	tests := []struct {
		name        string
		invokerURL  string
		attachments map[string]any
		viaRegistry bool
		wantErr     bool
	}{
		{name: "reference timeout", invokerURL: service + "?timeout=3s"},
		{name: "reference timeout behind a registry", invokerURL: service + "?timeout=3s", viaRegistry: true},
		{name: "per-call timeout", invokerURL: service, attachments: map[string]any{constant.TimeoutKey: "3s"}},
		{name: "shorter reference timeout", invokerURL: service + "?timeout=200ms", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			url, err := common.NewURL(tt.invokerURL)
			require.NoError(t, err)
			mockResult := &result.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
			done := make(chan struct{})
			ivk := mock.NewMockInvoker(ctrl)
			ivk.EXPECT().GetURL().Return(url).AnyTimes()
			ivk.EXPECT().IsAvailable().Return(true).AnyTimes()
			ivk.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(
				func(context.Context, base.Invocation) result.Result {
					defer close(done)
					time.Sleep(providerDelay)
					return mockResult
				})

			var dir directory.Directory = static.NewDirectory([]base.Invoker{ivk})
			if tt.viaRegistry {
				registryURL, err := common.NewURL("registry://127.0.0.1:2181?registry=zookeeper&registry.timeout=5s")
				require.NoError(t, err)
				registryURL.SubURL = url
				dir = &registryURLDirectory{Directory: dir, registryURL: registryURL}
			}
			clusterInvoker := newForkingCluster().Join(dir)

			start := time.Now()
			res := clusterInvoker.Invoke(context.Background(),
				invocation.NewRPCInvocationWithOptions(invocation.WithAttachments(tt.attachments)))
			elapsed := time.Since(start)
			<-done

			if tt.wantErr {
				require.Error(t, res.Error())
				assert.Less(t, elapsed, time.Second)
				return
			}
			require.NoError(t, res.Error())
			assert.Equal(t, mockResult, res)
		})
	}
}

func TestForkingInvokeForksBehindRegistry(t *testing.T) {
	ctrl := gomock.NewController(t)
	extension.SetLoadbalance(constant.LoadBalanceKeyRoundRobin, roundrobin.NewRRLoadBalance)

	url, err := common.NewURL("tri://127.0.0.1:20000/com.ikurento.user.UserProvider?forks=1&loadbalance=roundrobin")
	require.NoError(t, err)
	registryURL, err := common.NewURL("registry://127.0.0.1:2181?registry=zookeeper&registry.timeout=5s")
	require.NoError(t, err)
	registryURL.SubURL = url

	var calls atomic.Int32
	invokers := make([]base.Invoker, 0, 2)
	for range 2 {
		ivk := mock.NewMockInvoker(ctrl)
		ivk.EXPECT().GetURL().Return(url).AnyTimes()
		ivk.EXPECT().IsAvailable().Return(true).AnyTimes()
		ivk.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(
			func(context.Context, base.Invocation) result.Result {
				calls.Add(1)
				return &result.RPCResult{}
			}).AnyTimes()
		invokers = append(invokers, ivk)
	}
	dir := &registryURLDirectory{Directory: static.NewDirectory(invokers), registryURL: registryURL}

	res := newForkingCluster().Join(dir).Invoke(context.Background(), &invocation.RPCInvocation{})
	require.NoError(t, res.Error())
	// Give an unexpected second fork time to run before counting.
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), calls.Load())
}
