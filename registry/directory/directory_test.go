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

package directory

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/tag"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
	protocolbase "dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	protocolmock "dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func TestSubscribe(t *testing.T) {
	registryDirectory, _ := normalRegistryDir()

	time.Sleep(1e9)
	assert.Len(t, registryDirectory.snapshotCacheInvokers(), 3)
}

func TestSubscribe_InvalidUrl(t *testing.T) {
	url, _ := common.NewURL("mock://127.0.0.1:1111")
	mockRegistry, _ := registry.NewMockRegistry(&common.URL{})
	_, err := NewRegistryDirectory(url, mockRegistry)
	require.Error(t, err)
}

func Test_Destroy(t *testing.T) {
	registryDirectory, _ := normalRegistryDir()

	time.Sleep(3e9)
	assert.Len(t, registryDirectory.snapshotCacheInvokers(), 3)
	assert.True(t, registryDirectory.IsAvailable())

	registryDirectory.Destroy()
	assert.Empty(t, registryDirectory.snapshotCacheInvokers())
	assert.False(t, registryDirectory.IsAvailable())
}

func Test_List(t *testing.T) {
	registryDirectory, _ := normalRegistryDir()

	time.Sleep(6e9)
	assert.Len(t, registryDirectory.List(&invocation.RPCInvocation{}), 3)
	assert.True(t, registryDirectory.IsAvailable())
}

func Test_MergeProviderUrl(t *testing.T) {
	registryDirectory, mockRegistry := normalRegistryDir(true)
	providerUrl, _ := common.NewURL("dubbo://0.0.0.0:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock1"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"))
	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerUrl})
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.snapshotCacheInvokers(), 1)
	invokers := registryDirectory.snapshotCacheInvokers()
	if len(invokers) > 0 {
		assert.Equal(t, "mock1", invokers[0].GetURL().GetParam(constant.ClusterKey, ""))
	}
}

func Test_MergeOverrideUrl(t *testing.T) {
	registryDirectory, mockRegistry := normalRegistryDir(true)
	providerUrl, _ := common.NewURL("dubbo://0.0.0.0:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"))
	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerUrl})
Loop1:
	for {
		if len(registryDirectory.snapshotCacheInvokers()) > 0 {
			overrideUrl, _ := common.NewURL("override://0.0.0.0:20000/org.apache.dubbo-go.mockService",
				common.WithParamsValue(constant.ClusterKey, "mock1"),
				common.WithParamsValue(constant.GroupKey, "group"),
				common.WithParamsValue(constant.VersionKey, "1.0.0"))
			mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: overrideUrl})
		Loop2:
			for {
				invokers := registryDirectory.snapshotCacheInvokers()
				if len(invokers) > 0 {
					if invokers[0].GetURL().GetParam(constant.ClusterKey, "") == "mock1" {
						assert.Len(t, invokers, 1)

						break Loop2
					} else {
						time.Sleep(500 * time.Millisecond)
					}
				}
			}
			break Loop1
		}
	}
}

func Test_RefreshUrl(t *testing.T) {
	registryDirectory, mockRegistry := normalRegistryDir()
	providerUrl, _ := common.NewURL("dubbo://0.0.0.0:20011/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock1"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"))
	providerUrl2, _ := common.NewURL("dubbo://0.0.0.0:20012/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock1"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"))
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.snapshotCacheInvokers(), 3)
	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerUrl})
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.snapshotCacheInvokers(), 4)
	mockRegistry.MockEvents([]*registry.ServiceEvent{{Action: remoting.EventTypeUpdate, Service: providerUrl}})
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.snapshotCacheInvokers(), 1)
	mockRegistry.MockEvents([]*registry.ServiceEvent{
		{Action: remoting.EventTypeUpdate, Service: providerUrl},
		{Action: remoting.EventTypeUpdate, Service: providerUrl2},
	})
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.snapshotCacheInvokers(), 2)
	// clear all address
	mockRegistry.MockEvents([]*registry.ServiceEvent{})
	time.Sleep(1e9)
	assert.Empty(t, registryDirectory.snapshotCacheInvokers())
}

func TestRemoveClosingInstanceRemovesExactInstanceKey(t *testing.T) {
	registryDirectory, mockRegistry := normalRegistryDir(true)

	providerURL1, _ := common.NewURL("dubbo://0.0.0.0:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock1"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"))
	providerURL2, _ := common.NewURL("dubbo://0.0.0.0:20001/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock1"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"))

	event1 := &registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerURL1}
	event2 := &registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerURL2}
	key1 := registryDirectory.invokerCacheKey(event1)
	key2 := registryDirectory.invokerCacheKey(event2)

	mockRegistry.MockEvent(event1)
	mockRegistry.MockEvent(event2)
	time.Sleep(1e9)

	assert.Len(t, registryDirectory.cacheInvokers, 2)
	assert.NotEqual(t, key1, key2)

	removed := registryDirectory.RemoveClosingInstance(key1)
	require.True(t, removed)

	assert.Len(t, registryDirectory.cacheInvokers, 1)
	assert.Len(t, registryDirectory.List(&invocation.RPCInvocation{}), 1)

	_, stillExists := registryDirectory.cacheInvokersMap.Load(key1)
	assert.False(t, stillExists)

	remaining, ok := registryDirectory.cacheInvokersMap.Load(key2)
	require.True(t, ok)
	assert.Equal(t, key2, remaining.(interface{ GetURL() *common.URL }).GetURL().GetCacheInvokerMapKey())
}

func TestRemoveClosingInstanceReturnsFalseForUnknownKey(t *testing.T) {
	registryDirectory, _ := normalRegistryDir(true)

	removed := registryDirectory.RemoveClosingInstance("missing-instance-key")
	assert.False(t, removed)
	assert.Empty(t, registryDirectory.cacheInvokers)
}

func TestClosingTombstonePreventsRebuildUntilDeleteEvent(t *testing.T) {
	registryDirectory, mockRegistry := normalRegistryDir(true)

	providerURL, _ := common.NewURL("dubbo://0.0.0.0:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock1"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"))
	event := &registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerURL}
	key := registryDirectory.invokerCacheKey(event)

	mockRegistry.MockEvent(event)
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 1)

	removed := registryDirectory.RemoveClosingInstance(key)
	require.True(t, removed)
	assert.Empty(t, registryDirectory.cacheInvokers)
	assert.True(t, registryDirectory.hasActiveClosingTombstone(key))

	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerURL})
	time.Sleep(1e9)
	assert.Empty(t, registryDirectory.cacheInvokers)

	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeDel, Service: providerURL})
	time.Sleep(1e9)
	assert.False(t, registryDirectory.hasActiveClosingTombstone(key))

	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerURL})
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 1)
}

func TestExpiredClosingTombstoneAllowsRebuild(t *testing.T) {
	registryDirectory, mockRegistry := normalRegistryDir(true)
	registryDirectory.closingTombstoneTTL = 20 * time.Millisecond

	providerURL, _ := common.NewURL("dubbo://0.0.0.0:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock1"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"))
	event := &registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerURL}
	key := registryDirectory.invokerCacheKey(event)

	mockRegistry.MockEvent(event)
	time.Sleep(1e9)
	require.Len(t, registryDirectory.cacheInvokers, 1)

	require.True(t, registryDirectory.RemoveClosingInstance(key))
	assert.Empty(t, registryDirectory.cacheInvokers)

	time.Sleep(40 * time.Millisecond)
	assert.False(t, registryDirectory.hasActiveClosingTombstone(key))

	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerURL})
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 1)
}

func normalRegistryDir(noMockEvent ...bool) (*RegistryDirectory, *registry.MockRegistry) {
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)

	applicationConfig := &global.ApplicationConfig{
		Name: "test-application",
	}

	url, _ := common.NewURL("mock://127.0.0.1:1111")
	suburl, _ := common.NewURL(
		"dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"),
		common.WithParamsValue(constant.ApplicationKey, applicationConfig.Name),
	)
	url.SubURL = suburl
	mockRegistry, _ := registry.NewMockRegistry(&common.URL{})
	dir, _ := NewRegistryDirectory(url, mockRegistry)

	go dir.(*RegistryDirectory).Subscribe(suburl)
	if len(noMockEvent) == 0 {
		for i := 0; i < 3; i++ {
			mockRegistry.(*registry.MockRegistry).MockEvent(
				&registry.ServiceEvent{
					Action: remoting.EventTypeAdd,
					Service: common.NewURLWithOptions(
						common.WithPath("TEST"+strconv.FormatInt(int64(i), 10)),
						common.WithProtocol("dubbo"),
					),
				},
			)
		}
	}
	return dir.(*RegistryDirectory), mockRegistry.(*registry.MockRegistry)
}

func TestToGroupInvokers(t *testing.T) {
	t.Run("SameGroup", func(t *testing.T) {
		registryDirectory, mockRegistry := normalRegistryDir(true)
		providerUrl, _ := common.NewURL("dubbo://0.0.0.0:20000/org.apache.dubbo-go.mockService",
			common.WithParamsValue(constant.ClusterKey, "mock1"),
			common.WithParamsValue(constant.GroupKey, "group"),
			common.WithParamsValue(constant.VersionKey, "1.0.0"))
		mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerUrl})
		time.Sleep(1e9)
		assert.Len(t, registryDirectory.toGroupInvokers(), 1)
		providerUrl1, _ := common.NewURL("dubbo://0.0.0.0:20001/org.apache.dubbo-go.mockService",
			common.WithParamsValue(constant.ClusterKey, "mock1"),
			common.WithParamsValue(constant.GroupKey, "group"),
			common.WithParamsValue(constant.VersionKey, "1.0.0"))
		mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerUrl1})
		time.Sleep(1e9)
		assert.Len(t, registryDirectory.toGroupInvokers(), 2)
	})
	t.Run("DifferentGroup", func(t *testing.T) {
		extension.SetCluster("mock", cluster.NewMockCluster)

		registryDirectory, mockRegistry := normalRegistryDir(true)
		providerUrl, _ := common.NewURL("dubbo://0.0.0.0:20000/org.apache.dubbo-go.mockService",
			common.WithParamsValue(constant.ClusterKey, "mock1"),
			common.WithParamsValue(constant.GroupKey, "group"),
			common.WithParamsValue(constant.VersionKey, "1.0.0"))
		mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerUrl})
		time.Sleep(1e9)
		assert.Len(t, registryDirectory.toGroupInvokers(), 1)

		providerUrl1, _ := common.NewURL("dubbo://0.0.0.0:20000/org.apache.dubbo-go.mockService",
			common.WithParamsValue(constant.ClusterKey, "mock1"),
			common.WithParamsValue(constant.GroupKey, "group1"),
			common.WithParamsValue(constant.VersionKey, "1.0.0"))
		mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerUrl1})
		time.Sleep(1e9)
		assert.Len(t, registryDirectory.toGroupInvokers(), 2)
	})
}

func TestRegistryDirectoryIsAvailableConcurrentWithCacheUpdates(t *testing.T) {
	dir := newRegistryDirectoryForConcurrencyTest(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := protocolmock.NewMockInvoker(ctrl)
	invoker.EXPECT().IsAvailable().AnyTimes().Return(true)

	dir.invokersLock.Lock()
	dir.cacheInvokers = []protocolbase.Invoker{invoker}
	dir.invokersLock.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = dir.IsAvailable()
		}()
		go func() {
			defer wg.Done()
			dir.invokersLock.Lock()
			dir.cacheInvokers = []protocolbase.Invoker{invoker}
			dir.invokersLock.Unlock()
		}()
	}
	wg.Wait()
}

func TestRegistryDirectoryConfiguratorConcurrentAccess(t *testing.T) {
	dir := newRegistryDirectoryForConcurrencyTest(t)
	overrideURL, err := common.NewURL(
		"override://127.0.0.1:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.CategoryKey, constant.ConfiguratorsCategory),
	)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			event := &registry.ServiceEvent{
				Action:  remoting.EventTypeAdd,
				Service: overrideURL,
			}
			_ = dir.convertUrl(event)
		}()
		go func() {
			defer wg.Done()
			targetURL, _ := common.NewURL("dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService")
			dir.overrideUrl(targetURL)
		}()
	}
	wg.Wait()
}

func TestRegistryDirectorySubscribedURLConcurrentAccess(t *testing.T) {
	dir := &RegistryDirectory{}
	subscribedURL, err := common.NewURL("consumer://127.0.0.1:20000/org.apache.dubbo-go.mockService")
	require.NoError(t, err)
	registeredURL, err := common.NewURL("registry://127.0.0.1:20000/org.apache.dubbo-go.mockService")
	require.NoError(t, err)
	dir.RegisteredUrl = registeredURL

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			dir.setSubscribedURL(subscribedURL)
		}()
		go func() {
			defer wg.Done()
			gotRegisteredURL, gotSubscribedURL := dir.snapshotRegistryURLs()
			_ = gotRegisteredURL
			_ = gotSubscribedURL
		}()
	}
	wg.Wait()
}

func TestRegistryDirectorySubscribeTimeoutSkipsSubscribeStart(t *testing.T) {
	registryURL, err := common.NewURL(
		"registry://127.0.0.1:20000",
		common.WithParamsValue(constant.RegistryTimeoutKey, "100ms"),
	)
	require.NoError(t, err)
	subscribeURL, err := common.NewURL("consumer://127.0.0.1:20000/org.apache.dubbo-go.mockService")
	require.NoError(t, err)

	blockingRegistry := &blockingRegistryForSubscribeTest{
		url:           registryURL,
		registerBlock: make(chan struct{}),
	}
	defer close(blockingRegistry.registerBlock)

	dir := &RegistryDirectory{
		registry: blockingRegistry,
	}

	err = dir.Subscribe(subscribeURL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timed out")
	assert.Equal(t, int32(0), blockingRegistry.subscribeCalls.Load())
}

func TestRegistryDirectorySubscribeTimeoutLateRegisterRollback(t *testing.T) {
	registryURL, err := common.NewURL(
		"registry://127.0.0.1:20000",
		common.WithParamsValue(constant.RegistryTimeoutKey, "100ms"),
	)
	require.NoError(t, err)
	subscribeURL, err := common.NewURL("consumer://127.0.0.1:20000/org.apache.dubbo-go.mockService")
	require.NoError(t, err)

	blockingRegistry := &blockingRegistryForSubscribeTest{
		url:           registryURL,
		registerBlock: make(chan struct{}),
	}

	dir := &RegistryDirectory{
		registry: blockingRegistry,
	}

	err = dir.Subscribe(subscribeURL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timed out")
	assert.Equal(t, int32(0), blockingRegistry.subscribeCalls.Load())

	close(blockingRegistry.registerBlock)
	assert.Eventually(t, func() bool {
		return blockingRegistry.unregisterCalls.Load() == 1
	}, time.Second, 10*time.Millisecond)
}

type blockingRegistryForSubscribeTest struct {
	url             *common.URL
	registerBlock   chan struct{}
	subscribeCalls  atomic.Int32
	unregisterCalls atomic.Int32
}

func (r *blockingRegistryForSubscribeTest) Register(_ *common.URL) error {
	<-r.registerBlock
	return nil
}

func (r *blockingRegistryForSubscribeTest) UnRegister(_ *common.URL) error {
	r.unregisterCalls.Add(1)
	return nil
}

func (r *blockingRegistryForSubscribeTest) Subscribe(_ *common.URL, _ registry.NotifyListener) error {
	r.subscribeCalls.Add(1)
	return nil
}

func (r *blockingRegistryForSubscribeTest) UnSubscribe(_ *common.URL, _ registry.NotifyListener) error {
	return nil
}

func (r *blockingRegistryForSubscribeTest) LoadSubscribeInstances(_ *common.URL, _ registry.NotifyListener) error {
	return nil
}

func (r *blockingRegistryForSubscribeTest) GetURL() *common.URL {
	return r.url
}

func (r *blockingRegistryForSubscribeTest) IsAvailable() bool {
	return true
}

func (r *blockingRegistryForSubscribeTest) Destroy() {}

func newRegistryDirectoryForConcurrencyTest(t *testing.T) *RegistryDirectory {
	t.Helper()

	applicationConfig := &global.ApplicationConfig{
		Name: "test-application",
	}

	url, err := common.NewURL("mock://127.0.0.1:1111")
	require.NoError(t, err)
	suburl, err := common.NewURL(
		"dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"),
		common.WithParamsValue(constant.ApplicationKey, applicationConfig.Name),
	)
	require.NoError(t, err)
	url.SubURL = suburl

	mockRegistry, err := registry.NewMockRegistry(&common.URL{})
	require.NoError(t, err)

	dir, err := NewRegistryDirectory(url, mockRegistry)
	require.NoError(t, err)
	return dir.(*RegistryDirectory)
}
