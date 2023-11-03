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
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster"
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
	"dubbo.apache.org/dubbo-go/v3/registry/directory"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func init() {
	config.SetRootConfig(config.RootConfig{
		Application: &config.ApplicationConfig{Name: "test-application"},
		Shutdown:    &config.ShutdownConfig{StepTimeout: "0s"},
	})
}

func referNormal(t *testing.T, regProtocol *registryProtocol) {
	extension.SetProtocol("registry", GetProtocol)
	extension.SetRegistry("mock", registry.NewMockRegistry)
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)
	extension.SetCluster("mock", cluster.NewMockCluster)
	extension.SetDirectory("mock", directory.NewRegistryDirectory)

	url, _ := common.NewURL("mock://127.0.0.1:1111")
	suburl, _ := common.NewURL(
		"dubbo://127.0.0.1:20000//",
		common.WithParamsValue(constant.ClusterKey, "mock"),
	)

	url.SubURL = suburl

	invoker := regProtocol.Refer(url)
	assert.IsType(t, &protocol.BaseInvoker{}, invoker)
	assert.Equal(t, invoker.GetURL().String(), url.String())
}

func TestRefer(t *testing.T) {
	config.SetRootConfig(config.RootConfig{
		Application: &config.ApplicationConfig{Name: "test-application"},
	})
	regProtocol := newRegistryProtocol()
	referNormal(t, regProtocol)
}

func TestMultiRegRefer(t *testing.T) {
	regProtocol := newRegistryProtocol()
	referNormal(t, regProtocol)
	url2, _ := common.NewURL("mock://127.0.0.1:2222")
	suburl2, _ := common.NewURL(
		"dubbo://127.0.0.1:20000//",
		common.WithParamsValue(constant.ClusterKey, "mock"),
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
		common.WithParamsValue(constant.ClusterKey, "mock"),
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
		common.WithParamsValue(constant.ClusterKey, "mock"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"),
	)

	url.SubURL = suburl
	invoker := protocol.NewBaseInvoker(url)
	exporter := regProtocol.Export(invoker)

	assert.IsType(t, &exporterChangeableWrapper{}, exporter)
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
		common.WithParamsValue(constant.ClusterKey, "mock"),
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
		common.WithParamsValue(constant.ClusterKey, "mock"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"),
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

func TestDestroy(t *testing.T) {
	regProtocol := newRegistryProtocol()
	referNormal(t, regProtocol)
	exporterNormal(t, regProtocol)

	regProtocol.Destroy()

	var count int
	regProtocol.registries.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, 0, count)
}

func TestExportWithOverrideListener(t *testing.T) {
	extension.SetDefaultConfigurator(configurator.NewMockConfigurator)

	regProtocol := newRegistryProtocol()
	url := exporterNormal(t, regProtocol)
	var reg *registry.MockRegistry
	if regI, loaded := regProtocol.registries.Load(url.PrimitiveURL); loaded {
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
	newUrl.SetParam(constant.ClusterKey, "mock1")
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
	if _, loaded := regProtocol.registries.Load(url.PrimitiveURL); !loaded {
		assert.Fail(t, "regProtocol.registries.Load can not be loaded")
		return
	}
	dc.(*config_center.MockDynamicConfiguration).MockServiceConfigEvent()

	newUrl := url.SubURL.Clone()
	newUrl.SetParam(constant.ClusterKey, "mock1")

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
	if _, loaded := regProtocol.registries.Load(url.PrimitiveURL); !loaded {
		assert.Fail(t, "regProtocol.registries.Load can not be loaded")
		return
	}
	dc.(*config_center.MockDynamicConfiguration).MockApplicationConfigEvent()

	newUrl := url.SubURL.Clone()
	newUrl.SetParam(constant.ClusterKey, "mock1")
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
