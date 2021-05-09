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

package protocol

import (
	"testing"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/stretchr/testify/assert"
)

import (
	cluster "dubbo.apache.org/dubbo-go/v3/cluster/cluster_impl"
	"dubbo.apache.org/dubbo-go/v3/common"
	common_cfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/configurator"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func init() {
	config.SetProviderConfig(config.ProviderConfig{BaseConfig: config.BaseConfig{
		ApplicationConfig: &config.ApplicationConfig{Name: "test-application"},
	}})
}

func referNormal(t *testing.T, regProtocol *registryProtocol) {
	extension.SetProtocol("registry", GetProtocol)
	extension.SetRegistry("mock", registry.NewMockRegistry)
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)
	extension.SetCluster("mock", cluster.NewMockCluster)

	url, _ := common.NewURL("mock://127.0.0.1:1111")
	suburl, _ := common.NewURL(
		"dubbo://127.0.0.1:20000//",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
	)

	url.SubURL = suburl

	invoker := regProtocol.Refer(url)
	assert.IsType(t, &protocol.BaseInvoker{}, invoker)
	assert.Equal(t, invoker.GetURL().String(), url.String())
}

func TestRefer(t *testing.T) {
	config.SetConsumerConfig(
		config.ConsumerConfig{BaseConfig: config.BaseConfig{
			ApplicationConfig: &config.ApplicationConfig{Name: "test-application"},
		}})
	regProtocol := newRegistryProtocol()
	referNormal(t, regProtocol)
}

func TestMultiRegRefer(t *testing.T) {
	regProtocol := newRegistryProtocol()
	referNormal(t, regProtocol)
	url2, _ := common.NewURL("mock://127.0.0.1:2222")
	suburl2, _ := common.NewURL(
		"dubbo://127.0.0.1:20000//",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
	)

	url2.SubURL = suburl2

	regProtocol.Refer(url2)
	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 2)
}

func TestOneRegRefer(t *testing.T) {
	regProtocol := newRegistryProtocol()
	referNormal(t, regProtocol)

	url2, _ := common.NewURL("mock://127.0.0.1:1111")
	suburl2, _ := common.NewURL(
		"dubbo://127.0.0.1:20000//",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
	)

	url2.SubURL = suburl2

	regProtocol.Refer(url2)
	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 1)
}

func exporterNormal(t *testing.T, regProtocol *registryProtocol) *common.URL {
	extension.SetProtocol("registry", GetProtocol)
	extension.SetRegistry("mock", registry.NewMockRegistry)
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)
	url, _ := common.NewURL("mock://127.0.0.1:1111")
	suburl, _ := common.NewURL(
		"dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
		common.WithParamsValue(constant.GROUP_KEY, "group"),
		common.WithParamsValue(constant.VERSION_KEY, "1.0.0"),
	)

	url.SubURL = suburl
	invoker := protocol.NewBaseInvoker(url)
	exporter := regProtocol.Export(invoker)

	assert.IsType(t, &protocol.BaseExporter{}, exporter)
	assert.Equal(t, exporter.GetInvoker().GetURL().String(), suburl.String())
	return url
}

func TestExporter(t *testing.T) {
	regProtocol := newRegistryProtocol()
	exporterNormal(t, regProtocol)
}

func TestMultiRegAndMultiProtoExporter(t *testing.T) {
	regProtocol := newRegistryProtocol()
	exporterNormal(t, regProtocol)

	url2, _ := common.NewURL("mock://127.0.0.1:2222")
	suburl2, _ := common.NewURL(
		"jsonrpc://127.0.0.1:20000//",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
	)

	url2.SubURL = suburl2
	invoker2 := protocol.NewBaseInvoker(url2)
	regProtocol.Export(invoker2)

	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 2)

	var count2 int
	regProtocol.bounds.Range(func(key, value interface{}) bool {
		count2++
		return true
	})
	assert.Equal(t, count2, 2)
}

func TestOneRegAndProtoExporter(t *testing.T) {
	regProtocol := newRegistryProtocol()
	exporterNormal(t, regProtocol)

	url2, _ := common.NewURL("mock://127.0.0.1:1111")
	suburl2, _ := common.NewURL(
		"dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
		common.WithParamsValue(constant.GROUP_KEY, "group"),
		common.WithParamsValue(constant.VERSION_KEY, "1.0.0"),
	)

	url2.SubURL = suburl2
	invoker2 := protocol.NewBaseInvoker(url2)
	regProtocol.Export(invoker2)

	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 1)

	var count2 int
	regProtocol.bounds.Range(func(key, value interface{}) bool {
		count2++
		return true
	})
	assert.Equal(t, count2, 1)
}

func TestDestry(t *testing.T) {
	regProtocol := newRegistryProtocol()
	referNormal(t, regProtocol)
	exporterNormal(t, regProtocol)

	regProtocol.Destroy()
	assert.Equal(t, len(regProtocol.invokers), 0)

	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, count, 0)

	var count2 int
	regProtocol.bounds.Range(func(key, value interface{}) bool {
		count2++
		return true
	})
	assert.Equal(t, count2, 0)
}

func TestExportWithOverrideListener(t *testing.T) {
	extension.SetDefaultConfigurator(configurator.NewMockConfigurator)

	regProtocol := newRegistryProtocol()
	url := exporterNormal(t, regProtocol)
	var reg *registry.MockRegistry
	if regI, loaded := regProtocol.registries.Load(url.Key()); loaded {
		reg = regI.(*registry.MockRegistry)
	} else {
		assert.Fail(t, "regProtocol.registries.Load can not be loaded")
		return
	}
	overrideUrl, _ := common.NewURL(
		"override://0:0:0:0/org.apache.dubbo-go.mockService?cluster=mock1&&group=group&&version=1.0.0",
	)
	event := &registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: overrideUrl}
	reg.MockEvent(event)
	time.Sleep(1e9)
	newUrl := url.SubURL.Clone()
	newUrl.SetParam(constant.CLUSTER_KEY, "mock1")
	delKeys := gxset.NewSet("dynamic", "enabled")
	key := newUrl.CloneExceptParams(delKeys).String()
	v2, _ := regProtocol.bounds.Load(key)
	assert.NotNil(t, v2)
}

func TestExportWithServiceConfig(t *testing.T) {
	extension.SetDefaultConfigurator(configurator.NewMockConfigurator)
	ccUrl, _ := common.NewURL("mock://127.0.0.1:1111")
	dc, _ := (&config_center.MockDynamicConfigurationFactory{}).GetDynamicConfiguration(ccUrl)
	common_cfg.GetEnvInstance().SetDynamicConfiguration(dc)
	regProtocol := newRegistryProtocol()
	url := exporterNormal(t, regProtocol)
	if _, loaded := regProtocol.registries.Load(url.Key()); !loaded {
		assert.Fail(t, "regProtocol.registries.Load can not be loaded")
		return
	}
	dc.(*config_center.MockDynamicConfiguration).MockServiceConfigEvent()

	newUrl := url.SubURL.Clone()
	newUrl.SetParam(constant.CLUSTER_KEY, "mock1")

	delKeys := gxset.NewSet("dynamic", "enabled")
	key := newUrl.CloneExceptParams(delKeys).String()
	v2, _ := regProtocol.bounds.Load(key)

	assert.NotNil(t, v2)
}

func TestExportWithApplicationConfig(t *testing.T) {
	extension.SetDefaultConfigurator(configurator.NewMockConfigurator)
	ccUrl, _ := common.NewURL("mock://127.0.0.1:1111")
	dc, _ := (&config_center.MockDynamicConfigurationFactory{}).GetDynamicConfiguration(ccUrl)
	common_cfg.GetEnvInstance().SetDynamicConfiguration(dc)
	regProtocol := newRegistryProtocol()
	url := exporterNormal(t, regProtocol)
	if _, loaded := regProtocol.registries.Load(url.Key()); !loaded {
		assert.Fail(t, "regProtocol.registries.Load can not be loaded")
		return
	}
	dc.(*config_center.MockDynamicConfiguration).MockApplicationConfigEvent()

	newUrl := url.SubURL.Clone()
	newUrl.SetParam(constant.CLUSTER_KEY, "mock1")
	delKeys := gxset.NewSet("dynamic", "enabled")
	key := newUrl.CloneExceptParams(delKeys).String()
	v2, _ := regProtocol.bounds.Load(key)
	assert.NotNil(t, v2)
}

func TestGetProviderUrlWithHideKey(t *testing.T) {
	url, _ := common.NewURL("dubbo://127.0.0.1:1111?a=a1&b=b1&.c=c1&.d=d1&e=e1&protocol=registry")
	providerUrl := getUrlToRegistry(url, url)
	assert.NotContains(t, providerUrl.GetParams(), ".c")
	assert.NotContains(t, providerUrl.GetParams(), ".d")
	assert.Contains(t, providerUrl.GetParams(), "a")
}
