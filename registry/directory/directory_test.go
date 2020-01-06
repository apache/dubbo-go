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
	"context"
	"net/url"
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/cluster/cluster_impl"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/protocolwrapper"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting"
)

func init() {
	config.SetConsumerConfig(config.ConsumerConfig{ApplicationConfig: &config.ApplicationConfig{Name: "test-application"}})
}
func TestSubscribe(t *testing.T) {
	registryDirectory, _ := normalRegistryDir()

	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 3)
}

////Deprecated! not support delete
//func TestSubscribe_Delete(t *testing.T) {
//	registryDirectory, mockRegistry := normalRegistryDir()
//	time.Sleep(1e9)
//	assert.Len(t, registryDirectory.cacheInvokers, 3)
//	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeDel, Service: *common.NewURLWithOptions(common.WithPath("TEST0"), common.WithProtocol("dubbo"))})
//	time.Sleep(1e9)
//	assert.Len(t, registryDirectory.cacheInvokers, 2)
//}

func TestSubscribe_InvalidUrl(t *testing.T) {
	url, _ := common.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	mockRegistry, _ := registry.NewMockRegistry(&common.URL{})
	_, err := NewRegistryDirectory(&url, mockRegistry)
	assert.Error(t, err)
}

func TestSubscribe_Group(t *testing.T) {
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)
	extension.SetCluster("mock", cluster_impl.NewMockCluster)

	regurl, _ := common.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl, _ := common.NewURL(context.TODO(), "dubbo://127.0.0.1:20000")
	suburl.SetParam(constant.CLUSTER_KEY, "mock")
	regurl.SubURL = &suburl
	mockRegistry, _ := registry.NewMockRegistry(&common.URL{})
	registryDirectory, _ := NewRegistryDirectory(&regurl, mockRegistry)

	go registryDirectory.Subscribe(common.NewURLWithOptions(common.WithPath("testservice")))

	//for group1
	urlmap := url.Values{}
	urlmap.Set(constant.GROUP_KEY, "group1")
	urlmap.Set(constant.CLUSTER_KEY, "failover") //to test merge url
	for i := 0; i < 3; i++ {
		mockRegistry.(*registry.MockRegistry).MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: *common.NewURLWithOptions(common.WithPath("TEST"+strconv.FormatInt(int64(i), 10)), common.WithProtocol("dubbo"),
			common.WithParams(urlmap))})
	}
	//for group2
	urlmap2 := url.Values{}
	urlmap2.Set(constant.GROUP_KEY, "group2")
	urlmap2.Set(constant.CLUSTER_KEY, "failover") //to test merge url
	for i := 0; i < 3; i++ {
		mockRegistry.(*registry.MockRegistry).MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: *common.NewURLWithOptions(common.WithPath("TEST"+strconv.FormatInt(int64(i), 10)), common.WithProtocol("dubbo"),
			common.WithParams(urlmap2))})
	}

	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 2)
}

func Test_Destroy(t *testing.T) {
	registryDirectory, _ := normalRegistryDir()

	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 3)
	assert.Equal(t, true, registryDirectory.IsAvailable())

	registryDirectory.Destroy()
	assert.Len(t, registryDirectory.cacheInvokers, 0)
	assert.Equal(t, false, registryDirectory.IsAvailable())
}

func Test_List(t *testing.T) {
	registryDirectory, _ := normalRegistryDir()

	time.Sleep(1e9)
	assert.Len(t, registryDirectory.List(&invocation.RPCInvocation{}), 3)
	assert.Equal(t, true, registryDirectory.IsAvailable())

}
func Test_MergeProviderUrl(t *testing.T) {
	registryDirectory, mockRegistry := normalRegistryDir(true)
	providerUrl, _ := common.NewURL(context.TODO(), "dubbo://0.0.0.0:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock1"),
		common.WithParamsValue(constant.GROUP_KEY, "group"),
		common.WithParamsValue(constant.VERSION_KEY, "1.0.0"))
	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerUrl})
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 1)
	if len(registryDirectory.cacheInvokers) > 0 {
		assert.Equal(t, "mock", registryDirectory.cacheInvokers[0].GetUrl().GetParam(constant.CLUSTER_KEY, ""))
	}

}

func Test_MergeOverrideUrl(t *testing.T) {
	registryDirectory, mockRegistry := normalRegistryDir(true)
	providerUrl, _ := common.NewURL(context.TODO(), "dubbo://0.0.0.0:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
		common.WithParamsValue(constant.GROUP_KEY, "group"),
		common.WithParamsValue(constant.VERSION_KEY, "1.0.0"))
	mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: providerUrl})
Loop1:
	for {
		if len(registryDirectory.cacheInvokers) > 0 {
			overrideUrl, _ := common.NewURL(context.TODO(), "override://0.0.0.0:20000/org.apache.dubbo-go.mockService",
				common.WithParamsValue(constant.CLUSTER_KEY, "mock1"),
				common.WithParamsValue(constant.GROUP_KEY, "group"),
				common.WithParamsValue(constant.VERSION_KEY, "1.0.0"))
			mockRegistry.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: overrideUrl})
		Loop2:
			for {
				if len(registryDirectory.cacheInvokers) > 0 {
					if "mock1" == registryDirectory.cacheInvokers[0].GetUrl().GetParam(constant.CLUSTER_KEY, "") {
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

func normalRegistryDir(noMockEvent ...bool) (*registryDirectory, *registry.MockRegistry) {
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)

	url, _ := common.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl, _ := common.NewURL(
		context.TODO(),
		"dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
		common.WithParamsValue(constant.GROUP_KEY, "group"),
		common.WithParamsValue(constant.VERSION_KEY, "1.0.0"),
	)
	url.SubURL = &suburl
	mockRegistry, _ := registry.NewMockRegistry(&common.URL{})
	registryDirectory, _ := NewRegistryDirectory(&url, mockRegistry)

	go registryDirectory.Subscribe(&suburl)
	if len(noMockEvent) == 0 {
		for i := 0; i < 3; i++ {
			mockRegistry.(*registry.MockRegistry).MockEvent(
				&registry.ServiceEvent{
					Action: remoting.EventTypeAdd,
					Service: *common.NewURLWithOptions(
						common.WithPath("TEST"+strconv.FormatInt(int64(i), 10)),
						common.WithProtocol("dubbo"),
					),
				},
			)
		}
	}
	return registryDirectory, mockRegistry.(*registry.MockRegistry)
}
