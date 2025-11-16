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
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/configurator"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/registry/directory"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func referNormal(t *testing.T, regProtocol *registryProtocol) {
	extension.SetProtocol("registry", GetProtocol)
	extension.SetRegistry("mock", registry.NewMockRegistry)
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)
	extension.SetCluster("mock", cluster.NewMockCluster)
	extension.SetDirectory("mock", directory.NewRegistryDirectory)

	shutdownConfig := &global.ShutdownConfig{
		StepTimeout:            "4s",
		ConsumerUpdateWaitTime: "4s",
	}

	applicationConfig := &global.ApplicationConfig{
		Name: "test-application",
	}

	url, _ := common.NewURL("mock://127.0.0.1:1111",
		common.WithAttribute(constant.ShutdownConfigPrefix, shutdownConfig),
	)
	suburl, _ := common.NewURL(
		"dubbo://127.0.0.1:20000//",
		common.WithParamsValue(constant.ClusterKey, "mock"),
		common.WithParamsValue(constant.ApplicationKey, applicationConfig.Name),
	)

	url.SubURL = suburl

	invoker := regProtocol.Refer(url)
	assert.IsType(t, &base.BaseInvoker{}, invoker)
	assert.Equal(t, invoker.GetURL().String(), url.String())
}

func TestRefer(t *testing.T) {
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
	regProtocol.registries.Range(func(key, value any) bool {
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
	regProtocol.registries.Range(func(key, value any) bool {
		count++
		return true
	})
	assert.Equal(t, count, 1)
}

func exporterNormal(t *testing.T, regProtocol *registryProtocol) *common.URL {
	extension.SetProtocol("registry", GetProtocol)
	extension.SetRegistry("mock", registry.NewMockRegistry)
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)

	// Clear ServiceMap to avoid "service already defined" errors between tests
	common.ServiceMap.UnRegister("org.apache.dubbo-go.mockService", "dubbo", "group/org.apache.dubbo-go.mockService:1.0.0")

	shutdownConfig := &global.ShutdownConfig{
		StepTimeout:            "4s",
		ConsumerUpdateWaitTime: "4s",
	}

	applicationConfig := &global.ApplicationConfig{
		Name: "test-application",
	}

	// Create service config for registerServiceMap
	serviceConfig := &global.ServiceConfig{
		Interface:   "org.apache.dubbo-go.mockService",
		ProtocolIDs: []string{"dubbo"},
		Group:       "group",
		Version:     "1.0.0",
	}

	providerConfig := &global.ProviderConfig{
		Services: map[string]*global.ServiceConfig{
			"org.apache.dubbo-go.mockService": serviceConfig,
		},
	}

	// Create a mock RPCService with exported methods
	mockRPCService := &MockRPCService{}

	url, _ := common.NewURL("mock://127.0.0.1:1111",
		common.WithAttribute(constant.ShutdownConfigPrefix, shutdownConfig),
	)
	suburl, _ := common.NewURL(
		"dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"),
		common.WithParamsValue(constant.BeanNameKey, "org.apache.dubbo-go.mockService"),
		common.WithParamsValue(constant.ApplicationKey, applicationConfig.Name),
		common.WithAttribute(constant.ProviderConfigPrefix, providerConfig),
		common.WithAttribute(constant.RpcServiceKey, mockRPCService),
	)

	url.SubURL = suburl
	invoker := base.NewBaseInvoker(url)
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
	invoker2 := base.NewBaseInvoker(url2)
	regProtocol.Export(invoker2)

	var count int
	regProtocol.registries.Range(func(key, value any) bool {
		count++
		return true
	})
	assert.Equal(t, count, 2)

	var count2 int
	regProtocol.bounds.Range(func(key, value any) bool {
		count2++
		return true
	})
	assert.Equal(t, count2, 2)
}

func TestOneRegAndProtoExporter(t *testing.T) {
	regProtocol := newRegistryProtocol()
	exporterNormal(t, regProtocol)

	// The test expects that exporting the same service to the same registry
	// should reuse the same bound (not create a new one)
	// So we export the exact same service again
	shutdownConfig := &global.ShutdownConfig{
		StepTimeout:            "4s",
		ConsumerUpdateWaitTime: "4s",
	}

	applicationConfig := &global.ApplicationConfig{
		Name: "test-application",
	}

	serviceConfig := &global.ServiceConfig{
		Interface:   "org.apache.dubbo-go.mockService",
		ProtocolIDs: []string{"dubbo"},
		Group:       "group",
		Version:     "1.0.0",
	}

	providerConfig := &global.ProviderConfig{
		Services: map[string]*global.ServiceConfig{
			"org.apache.dubbo-go.mockService": serviceConfig,
		},
	}

	mockRPCService := &MockRPCService{}

	url2, _ := common.NewURL("mock://127.0.0.1:1111",
		common.WithAttribute(constant.ShutdownConfigPrefix, shutdownConfig),
	)
	suburl2, _ := common.NewURL(
		"dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"),
		common.WithParamsValue(constant.BeanNameKey, "org.apache.dubbo-go.mockService"),
		common.WithParamsValue(constant.ApplicationKey, applicationConfig.Name),
		common.WithAttribute(constant.ProviderConfigPrefix, providerConfig),
		common.WithAttribute(constant.RpcServiceKey, mockRPCService),
	)

	url2.SubURL = suburl2
	invoker2 := base.NewBaseInvoker(url2)
	regProtocol.Export(invoker2)

	var count int
	regProtocol.registries.Range(func(key, value any) bool {
		count++
		return true
	})
	assert.Equal(t, count, 1)

	var count2 int
	regProtocol.bounds.Range(func(key, value any) bool {
		count2++
		return true
	})
	// Should still be 1 because we're exporting the same service (same cache key)
	assert.Equal(t, count2, 1)
}

func TestDestroy(t *testing.T) {
	regProtocol := newRegistryProtocol()
	referNormal(t, regProtocol)
	exporterNormal(t, regProtocol)

	regProtocol.Destroy()

	var count int
	regProtocol.registries.Range(func(key, value any) bool {
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
	// Use common/config (not dubbo.apache.org/dubbo-go/v3/config)
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
	// Use common/config (not dubbo.apache.org/dubbo-go/v3/config)
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

// MockRPCService is a mock RPC service for testing
type MockRPCService struct{}

// MockMethod is a mock method that satisfies RPC method requirements
func (m *MockRPCService) MockMethod(arg1, arg2 string) error {
	return nil
}

// Reference returns the reference path
func (m *MockRPCService) Reference() string {
	return "org.apache.dubbo-go.mockService"
}
