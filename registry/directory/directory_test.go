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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/tag"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center/configurator"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func TestSubscribe(t *testing.T) {
	registryDirectory, _ := normalRegistryDir()

	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 3)
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
	assert.Len(t, registryDirectory.cacheInvokers, 3)
	assert.Equal(t, 3, registry.CountSyncMapEntries(registryDirectory.cacheInvokersMap))
	assert.True(t, registryDirectory.IsAvailable())

	registryDirectory.Destroy()
	assert.Empty(t, registryDirectory.cacheInvokers)
	assert.Zero(t, registry.CountSyncMapEntries(registryDirectory.cacheInvokersMap))
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
	assert.Len(t, registryDirectory.cacheInvokers, 1)
	if len(registryDirectory.cacheInvokers) > 0 {
		assert.Equal(t, "mock1", registryDirectory.cacheInvokers[0].GetURL().GetParam(constant.ClusterKey, ""))
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
		if len(registryDirectory.cacheInvokers) > 0 {
			overrideUrl, _ := common.NewURL("override://0.0.0.0:20000/org.apache.dubbo-go.mockService",
				common.WithParamsValue(constant.ClusterKey, "mock1"),
				common.WithParamsValue(constant.GroupKey, "group"),
				common.WithParamsValue(constant.VersionKey, "1.0.0"))
			mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: overrideUrl})
		Loop2:
			for {
				if len(registryDirectory.cacheInvokers) > 0 {
					if registryDirectory.cacheInvokers[0].GetURL().GetParam(constant.ClusterKey, "") == "mock1" {
						assert.Len(t, registryDirectory.cacheInvokers, 1)

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
	assert.Len(t, registryDirectory.cacheInvokers, 3)
	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerUrl})
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 4)
	mockRegistry.MockEvents([]*registry.ServiceEvent{{Action: remoting.EventTypeUpdate, Service: providerUrl}})
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 1)
	mockRegistry.MockEvents([]*registry.ServiceEvent{
		{Action: remoting.EventTypeUpdate, Service: providerUrl},
		{Action: remoting.EventTypeUpdate, Service: providerUrl2},
	})
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 2)
	// clear all address
	mockRegistry.MockEvents([]*registry.ServiceEvent{})
	time.Sleep(1e9)
	assert.Empty(t, registryDirectory.cacheInvokers)
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

func TestRefreshConfiguratorsUseLatestBatch(t *testing.T) {
	realConfigurator := extension.GetDefaultConfiguratorFunc()

	t.Run("single event replaces previous config", func(t *testing.T) {
		extension.SetDefaultConfigurator(configurator.NewMockConfigurator)
		defer extension.SetDefaultConfigurator(realConfigurator)

		registryDirectory, _ := normalRegistryDir(true)

		overrideURL1, _ := common.NewURL(
			"override://0.0.0.0/org.apache.dubbo-go.mockService",
			common.WithParamsValue(constant.ClusterKey, "mock1"),
		)
		registryDirectory.refreshInvokers(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: overrideURL1})
		require.Len(t, registryDirectory.configurators, 1)
		assert.Equal(t, "mock1", registryDirectory.configurators[0].GetUrl().GetParam(constant.ClusterKey, ""))

		overrideURL2, _ := common.NewURL(
			"override://0.0.0.0/org.apache.dubbo-go.mockService",
			common.WithParamsValue(constant.ClusterKey, "mock2"),
		)
		registryDirectory.refreshInvokers(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: overrideURL2})
		require.Len(t, registryDirectory.configurators, 1)
		assert.Equal(t, "mock2", registryDirectory.configurators[0].GetUrl().GetParam(constant.ClusterKey, ""))
	})

	t.Run("same batch keeps param union", func(t *testing.T) {
		extension.SetDefaultConfigurator(realConfigurator)

		registryDirectory, _ := normalRegistryDir(true)
		registryDirectory.refreshAllInvokers([]*registry.ServiceEvent{
			{Action: remoting.EventTypeAdd, Service: mustURL(t,
				"override://0.0.0.0:0/org.apache.dubbo-go.mockService?timeout=2s",
			)},
			{Action: remoting.EventTypeAdd, Service: mustURL(t,
				"override://0.0.0.0:0/org.apache.dubbo-go.mockService?cluster=mock2",
			)},
		}, func() {})

		target := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService")
		registryDirectory.overrideUrl(target)

		assert.Equal(t, "2s", target.GetParam("timeout", ""))
		assert.Equal(t, "mock2", target.GetParam(constant.ClusterKey, ""))
	})

	t.Run("new batch replaces previous batch", func(t *testing.T) {
		extension.SetDefaultConfigurator(realConfigurator)

		registryDirectory, _ := normalRegistryDir(true)
		registryDirectory.refreshAllInvokers([]*registry.ServiceEvent{
			{Action: remoting.EventTypeAdd, Service: mustURL(t,
				"override://0.0.0.0:0/org.apache.dubbo-go.mockService?timeout=2s",
			)},
			{Action: remoting.EventTypeAdd, Service: mustURL(t,
				"override://0.0.0.0:0/org.apache.dubbo-go.mockService?cluster=mock2",
			)},
		}, func() {})

		registryDirectory.refreshAllInvokers([]*registry.ServiceEvent{
			{Action: remoting.EventTypeAdd, Service: mustURL(t,
				"override://0.0.0.0:0/org.apache.dubbo-go.mockService?cluster=mock3",
			)},
		}, func() {})

		target := mustURL(t, "dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService")
		registryDirectory.overrideUrl(target)

		assert.Empty(t, target.GetParam("timeout", ""))
		assert.Equal(t, "mock3", target.GetParam(constant.ClusterKey, ""))
	})
}

func mustURL(t *testing.T, rawURL string) *common.URL {
	t.Helper()
	u, err := common.NewURL(rawURL)
	require.NoError(t, err)
	return u
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
