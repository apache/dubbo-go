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
)

import (
	_ "dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func init() {
	config.SetConsumerConfig(config.ConsumerConfig{
		BaseConfig: config.BaseConfig{
			ApplicationConfig: &config.ApplicationConfig{Name: "test-application"},
		},
	})
}

func TestSubscribe(t *testing.T) {
	registryDirectory, _ := normalRegistryDir()

	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 3)
}

func TestSubscribe_InvalidUrl(t *testing.T) {
	url, _ := common.NewURL("mock://127.0.0.1:1111")
	mockRegistry, _ := registry.NewMockRegistry(&common.URL{})
	_, err := NewRegistryDirectory(url, mockRegistry)
	assert.Error(t, err)
}

func Test_Destroy(t *testing.T) {
	registryDirectory, _ := normalRegistryDir()

	time.Sleep(3e9)
	assert.Len(t, registryDirectory.cacheInvokers, 3)
	assert.Equal(t, true, registryDirectory.IsAvailable())

	registryDirectory.Destroy()
	assert.Len(t, registryDirectory.cacheInvokers, 0)
	assert.Equal(t, false, registryDirectory.IsAvailable())
}

func Test_List(t *testing.T) {
	registryDirectory, _ := normalRegistryDir()

	time.Sleep(6e9)
	assert.Len(t, registryDirectory.List(&invocation.RPCInvocation{}), 3)
	assert.Equal(t, true, registryDirectory.IsAvailable())
}

func Test_MergeProviderUrl(t *testing.T) {
	registryDirectory, mockRegistry := normalRegistryDir(true)
	providerUrl, _ := common.NewURL("dubbo://0.0.0.0:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock1"),
		common.WithParamsValue(constant.GROUP_KEY, "group"),
		common.WithParamsValue(constant.VERSION_KEY, "1.0.0"))
	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerUrl})
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 1)
	if len(registryDirectory.cacheInvokers) > 0 {
		assert.Equal(t, "mock", registryDirectory.cacheInvokers[0].GetURL().GetParam(constant.CLUSTER_KEY, ""))
	}
}

func Test_MergeOverrideUrl(t *testing.T) {
	registryDirectory, mockRegistry := normalRegistryDir(true)
	providerUrl, _ := common.NewURL("dubbo://0.0.0.0:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
		common.WithParamsValue(constant.GROUP_KEY, "group"),
		common.WithParamsValue(constant.VERSION_KEY, "1.0.0"))
	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerUrl})
Loop1:
	for {
		if len(registryDirectory.cacheInvokers) > 0 {
			overrideUrl, _ := common.NewURL("override://0.0.0.0:20000/org.apache.dubbo-go.mockService",
				common.WithParamsValue(constant.CLUSTER_KEY, "mock1"),
				common.WithParamsValue(constant.GROUP_KEY, "group"),
				common.WithParamsValue(constant.VERSION_KEY, "1.0.0"))
			mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: overrideUrl})
		Loop2:
			for {
				if len(registryDirectory.cacheInvokers) > 0 {
					if "mock1" == registryDirectory.cacheInvokers[0].GetURL().GetParam(constant.CLUSTER_KEY, "") {
						assert.Len(t, registryDirectory.cacheInvokers, 1)
						assert.True(t, true)
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
		common.WithParamsValue(constant.CLUSTER_KEY, "mock1"),
		common.WithParamsValue(constant.GROUP_KEY, "group"),
		common.WithParamsValue(constant.VERSION_KEY, "1.0.0"))
	providerUrl2, _ := common.NewURL("dubbo://0.0.0.0:20012/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock1"),
		common.WithParamsValue(constant.GROUP_KEY, "group"),
		common.WithParamsValue(constant.VERSION_KEY, "1.0.0"))
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
	assert.Len(t, registryDirectory.cacheInvokers, 0)
}

func normalRegistryDir(noMockEvent ...bool) (*RegistryDirectory, *registry.MockRegistry) {
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)

	url, _ := common.NewURL("mock://127.0.0.1:1111")
	suburl, _ := common.NewURL(
		"dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
		common.WithParamsValue(constant.GROUP_KEY, "group"),
		common.WithParamsValue(constant.VERSION_KEY, "1.0.0"),
	)
	url.SubURL = suburl
	mockRegistry, _ := registry.NewMockRegistry(&common.URL{})
	dir, _ := NewRegistryDirectory(url, mockRegistry)

	go dir.(*RegistryDirectory).subscribe(suburl)
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
