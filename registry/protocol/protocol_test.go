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
	"errors"
	"sync"
	"testing"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/common"
	common_cfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/configurator"
	configparser "dubbo.apache.org/dubbo-go/v3/config_center/parser"
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
	assert.Equal(t, 2, count)
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
	assert.Equal(t, 1, count)
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
		common.WithAttribute(constant.ApplicationKey, applicationConfig),
		common.WithAttribute(constant.ProviderConfigKey, providerConfig),
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

func TestExportCopiesProviderAttributesToRegistryURL(t *testing.T) {
	extension.SetRegistry("mock", registry.NewMockRegistry)
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)

	regProtocol := newRegistryProtocol()
	shutdownConfig := &global.ShutdownConfig{
		StepTimeout:            "1ms",
		ConsumerUpdateWaitTime: "1ms",
	}
	applicationConfig := &global.ApplicationConfig{Name: "provider-application"}

	registryURL, err := common.NewURL("mock://127.0.0.1:1111")
	require.NoError(t, err)
	providerURL, err := common.NewURL(
		"dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService",
		common.WithAttribute(constant.ShutdownConfigPrefix, shutdownConfig),
		common.WithAttribute(constant.ApplicationKey, applicationConfig),
	)
	require.NoError(t, err)
	registryURL.SubURL = providerURL

	exporter := regProtocol.Export(base.NewBaseInvoker(registryURL))
	require.NotNil(t, exporter)
	t.Cleanup(regProtocol.Destroy)

	shutdownRaw, ok := registryURL.GetAttribute(constant.ShutdownConfigPrefix)
	require.True(t, ok)
	assert.Same(t, shutdownConfig, shutdownRaw)
	applicationRaw, ok := registryURL.GetAttribute(constant.ApplicationKey)
	require.True(t, ok)
	assert.Same(t, applicationConfig, applicationRaw)

	wrapper := exporter.(*exporterChangeableWrapper)
	registerURL, _, _ := wrapper.LifecycleURLs()
	registeredShutdownRaw, ok := registerURL.GetAttribute(constant.ShutdownConfigPrefix)
	require.True(t, ok)
	assert.Same(t, shutdownConfig, registeredShutdownRaw)
}

func TestRegisterServiceMapUsesURLAttributesAndProviderProtocolFallback(t *testing.T) {
	const (
		interfaceName = "org.apache.dubbo-go.attributeOnlyService"
		protocolName  = "attribute-protocol"
		serviceID     = "attribute-service"
	)

	serviceConfig := &global.ServiceConfig{
		Interface: interfaceName,
		Group:     "group",
		Version:   "1.0.0",
	}
	serviceKey := common.ServiceKey(interfaceName, serviceConfig.Group, serviceConfig.Version)
	_ = common.ServiceMap.UnRegister(interfaceName, protocolName, serviceKey)
	t.Cleanup(func() {
		_ = common.ServiceMap.UnRegister(interfaceName, protocolName, serviceKey)
	})

	providerConfig := &global.ProviderConfig{
		Services: map[string]*global.ServiceConfig{
			serviceID: serviceConfig,
		},
	}
	registryURL, err := common.NewURL("mock://127.0.0.1:1111")
	require.NoError(t, err)
	providerURL, err := common.NewURL(
		protocolName+"://127.0.0.1:20000/"+interfaceName,
		common.WithParamsValue(constant.BeanNameKey, serviceID),
		common.WithAttribute(constant.ProviderConfigKey, providerConfig),
		common.WithAttribute(constant.RpcServiceKey, &MockRPCService{}),
	)
	require.NoError(t, err)
	registryURL.SubURL = providerURL

	err = registerServiceMap(base.NewBaseInvoker(registryURL))
	require.NoError(t, err)
	assert.NotNil(t, common.ServiceMap.GetService(protocolName, interfaceName, serviceConfig.Group, serviceConfig.Version))
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
	assert.Equal(t, 2, count)

	var count2 int
	regProtocol.bounds.Range(func(key, value any) bool {
		count2++
		return true
	})
	assert.Equal(t, 2, count2)
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
		common.WithAttribute(constant.ApplicationKey, applicationConfig),
		common.WithAttribute(constant.ProviderConfigKey, providerConfig),
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
	assert.Equal(t, 1, count)

	var count2 int
	regProtocol.bounds.Range(func(key, value any) bool {
		count2++
		return true
	})
	// Should still be 1 because we're exporting the same service (same cache key)
	assert.Equal(t, 1, count2)
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

func TestDestroyCleansConfigurationListeners(t *testing.T) {
	regProtocol := newRegistryProtocol()
	exporterNormal(t, regProtocol)

	assert.Equal(t, 1, registry.CountSyncMapEntries(regProtocol.overrideListeners))
	assert.Equal(t, 1, registry.CountSyncMapEntries(regProtocol.serviceConfigurationListeners))

	regProtocol.Destroy()

	assert.Equal(t, 0, registry.CountSyncMapEntries(regProtocol.overrideListeners))
	assert.Equal(t, 0, registry.CountSyncMapEntries(regProtocol.serviceConfigurationListeners))
}

func TestDestroyUnsubscribesOverrideListener(t *testing.T) {
	regProtocol, recordingRegistry, originInvoker, exporter, listener := newRegistryProtocolWithSubscribedExporter(t)
	regProtocol.bounds.Store(getCacheKey(originInvoker), exporter)

	regProtocol.Destroy()

	assert.Equal(t, 1, recordingRegistry.unSubscribeCount)
	_, subscribeURL, _ := exporter.LifecycleURLs()
	assert.Equal(t, subscribeURL.String(), recordingRegistry.unSubscribeURL.String())
	assert.Same(t, listener, recordingRegistry.unSubscribeListener)
	assert.Equal(t, 0, registry.CountSyncMapEntries(regProtocol.overrideListeners))
	assert.Equal(t, 0, registry.CountSyncMapEntries(regProtocol.serviceConfigurationListeners))
}

func TestDestroyUsesRegisteredURLShutdownAttribute(t *testing.T) {
	extension.SetRegistry("destroy-recording", registry.NewMockRegistry)

	regProtocol := newRegistryProtocol()
	regProtocol.overrideListeners = &sync.Map{}
	regProtocol.serviceConfigurationListeners = &sync.Map{}

	registryURL, err := common.NewURL("destroy-recording://127.0.0.1:1111")
	require.NoError(t, err)
	providerURL, err := common.NewURL("dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService")
	require.NoError(t, err)
	registryURL.SubURL = providerURL
	originInvoker := base.NewBaseInvoker(registryURL)

	unexported := make(chan time.Time, 1)
	exporter := newExporterChangeableWrapper(originInvoker, &recordingExporter{
		invoker:    base.NewBaseInvoker(providerURL),
		unexported: unexported,
	})
	registerURL := providerURL.Clone()
	registerURL.SetAttribute(constant.ShutdownConfigPrefix, &global.ShutdownConfig{
		StepTimeout:            "30ms",
		ConsumerUpdateWaitTime: "30ms",
	})
	exporter.SetRegisterUrl(registerURL)
	exporter.SetSubscribeUrl(getSubscribedOverrideUrl(providerURL))
	regProtocol.bounds.Store(getCacheKey(originInvoker), exporter)

	regProtocol.Destroy()

	select {
	case <-unexported:
		require.Fail(t, "exporter unexported before shutdown wait elapsed")
	case <-time.After(20 * time.Millisecond):
	}

	select {
	case <-unexported:
	case <-time.After(time.Second):
		require.Fail(t, "exporter was not unexported")
	}
	assert.Eventually(t, func() bool {
		return registry.CountSyncMapEntries(regProtocol.bounds) == 0
	}, time.Second, 10*time.Millisecond)
}

func TestNewProviderConfigurationListenerUsesResolvedApplicationName(t *testing.T) {
	env := common_cfg.GetEnvInstance()
	previousDynamicConfiguration := env.GetDynamicConfiguration()
	dynamicConfiguration := newRecordingDynamicConfiguration()
	env.SetDynamicConfiguration(dynamicConfiguration)
	t.Cleanup(func() {
		env.SetDynamicConfiguration(previousDynamicConfiguration)
	})

	providerURL, err := common.NewURL(
		"dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ApplicationKey, "provider-listener-app"),
	)
	require.NoError(t, err)

	newProviderConfigurationListener(&sync.Map{}, providerURL)

	assert.Contains(t, dynamicConfiguration.keys, "provider-listener-app"+constant.ConfiguratorSuffix)
	applicationRaw, ok := providerURL.GetAttribute(constant.ApplicationKey)
	require.True(t, ok)
	application, ok := applicationRaw.(*global.ApplicationConfig)
	require.True(t, ok)
	assert.Equal(t, "provider-listener-app", application.Name)
}

func TestReExportReplacesConfigurationListeners(t *testing.T) {
	extension.SetDefaultConfigurator(configurator.NewMockConfigurator)

	regProtocol := newRegistryProtocol()
	url := exporterNormal(t, regProtocol)

	assert.Equal(t, 1, registry.CountSyncMapEntries(regProtocol.overrideListeners))
	assert.Equal(t, 1, registry.CountSyncMapEntries(regProtocol.serviceConfigurationListeners))

	regI, loaded := regProtocol.registries.Load(url.PrimitiveURL)
	if !loaded {
		assert.Fail(t, "regProtocol.registries.Load can not be loaded")
		return
	}
	reg := regI.(*registry.MockRegistry)

	overrideURL, _ := common.NewURL(
		"override://0:0:0:0/org.apache.dubbo-go.mockService?cluster=mock1&&group=group&&version=1.0.0",
	)
	reg.MockEvent(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: overrideURL})

	assert.Eventually(t, func() bool {
		newURL := url.SubURL.Clone()
		newURL.SetParam(constant.ClusterKey, "mock1")
		delKeys := gxset.NewSet("dynamic", "enabled")
		key := newURL.CloneExceptParams(delKeys).String()
		_, ok := regProtocol.bounds.Load(key)
		return ok
	}, 5*time.Second, 100*time.Millisecond)

	assert.Equal(t, 1, registry.CountSyncMapEntries(regProtocol.overrideListeners))
	assert.Equal(t, 1, registry.CountSyncMapEntries(regProtocol.serviceConfigurationListeners))
}

func TestReExportUnsubscribesOldOverrideListener(t *testing.T) {
	regProtocol, recordingRegistry, originInvoker, exporter, listener := newRegistryProtocolWithSubscribedExporter(t)
	regProtocol.bounds.Store(getCacheKey(originInvoker), exporter)

	newURL := originInvoker.GetURL().Clone()
	newURL.SubURL = originInvoker.GetURL().SubURL.Clone()
	newURL.SubURL.SetParam(constant.ClusterKey, "mock1")
	regProtocol.reExport(originInvoker, newURL)

	assert.Equal(t, 1, recordingRegistry.unSubscribeCount)
	_, subscribeURL, _ := exporter.LifecycleURLs()
	assert.Equal(t, subscribeURL.String(), recordingRegistry.unSubscribeURL.String())
	assert.Same(t, listener, recordingRegistry.unSubscribeListener)
	assert.Equal(t, 1, registry.CountSyncMapEntries(regProtocol.overrideListeners))
	assert.Equal(t, 1, registry.CountSyncMapEntries(regProtocol.serviceConfigurationListeners))
}

func TestUnsubscribeOverrideListenerKeepsUnexpectedListenerType(t *testing.T) {
	regProtocol, recordingRegistry, _, exporter, _ := newRegistryProtocolWithSubscribedExporter(t)
	_, subscribeURL, _ := exporter.LifecycleURLs()
	regProtocol.overrideListeners.Store(subscribeURL.String(), "unexpected-listener")

	regProtocol.unsubscribeOverrideListener(recordingRegistry, subscribeURL)

	assert.Equal(t, 0, recordingRegistry.unSubscribeCount)
	_, ok := regProtocol.overrideListeners.Load(subscribeURL.String())
	assert.True(t, ok)
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
	// Use the common config environment, not the legacy config package.
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
	// Use the common config environment, not the legacy config package.
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

func TestIsMatchedDoesNotAccumulateCategory(t *testing.T) {
	providerURL, _ := common.NewURL("override://127.0.0.1:20000/org.apache.dubbo-go.mockService?interface=org.apache.dubbo-go.mockService")
	consumerURL, _ := common.NewURL("consumer://127.0.0.1/org.apache.dubbo-go.mockService?interface=org.apache.dubbo-go.mockService&category=configurators")

	assert.True(t, isMatched(providerURL, consumerURL))
	assert.True(t, isMatched(providerURL, consumerURL))
	assert.Equal(t, []string{constant.ConfiguratorsCategory}, providerURL.GetParams()[constant.CategoryKey])
}

func TestGetProviderUrlWithHideKey(t *testing.T) {
	url, _ := common.NewURL("dubbo://127.0.0.1:1111?a=a1&b=b1&.c=c1&.d=d1&e=e1&protocol=registry")
	providerUrl := getUrlToRegistry(url, url)
	assert.NotContains(t, providerUrl.GetParams(), ".c")
	assert.NotContains(t, providerUrl.GetParams(), ".d")
	assert.Contains(t, providerUrl.GetParams(), "a")
}

type unsubscribeRecordingRegistry struct {
	*registry.MockRegistry
	unSubscribeURL      *common.URL
	unSubscribeListener registry.NotifyListener
	unSubscribeCount    int
}

type lifecycleRecordingRegistry struct {
	*registry.MockRegistry
	registerStartedOnce sync.Once
	registerStarted     chan struct{}
	allowRegister       <-chan struct{}
	blockRegisterCall   int
	registerErr         error
	mu                  sync.Mutex
	registerCount       int
	registerURL         *common.URL
	unregisterURL       *common.URL
	unregisterCount     int
	registered          bool
}

func (r *lifecycleRecordingRegistry) Register(url *common.URL) error {
	r.mu.Lock()
	r.registerCount++
	registerCall := r.registerCount
	blockRegisterCall := r.blockRegisterCall
	r.mu.Unlock()

	if blockRegisterCall == 0 || registerCall == blockRegisterCall {
		if r.registerStarted != nil {
			r.registerStartedOnce.Do(func() {
				close(r.registerStarted)
			})
		}
		if r.allowRegister != nil {
			<-r.allowRegister
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.registerErr != nil {
		return r.registerErr
	}
	r.registerURL = url
	r.registered = true
	return nil
}

func (r *lifecycleRecordingRegistry) UnRegister(url *common.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unregisterURL = url
	r.unregisterCount++
	r.registered = false
	return nil
}

func (r *lifecycleRecordingRegistry) Subscribe(*common.URL, registry.NotifyListener) error {
	return nil
}

func (r *lifecycleRecordingRegistry) UnSubscribe(*common.URL, registry.NotifyListener) error {
	return nil
}

func (r *lifecycleRecordingRegistry) snapshot() (*common.URL, *common.URL, int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.registerURL, r.unregisterURL, r.unregisterCount, r.registered
}

func newLifecycleTestInvoker(t *testing.T, registryScheme string, service string) base.Invoker {
	t.Helper()

	registryURL, err := common.NewURL(registryScheme + "://127.0.0.1:1111")
	require.NoError(t, err)
	providerURL, err := common.NewURL(
		"dubbo://127.0.0.1:20000/"+service,
		common.WithAttribute(constant.ApplicationKey, &global.ApplicationConfig{Name: "lifecycle-test"}),
		common.WithAttribute(constant.ShutdownConfigPrefix, &global.ShutdownConfig{
			StepTimeout:            "0s",
			ConsumerUpdateWaitTime: "0s",
		}),
	)
	require.NoError(t, err)
	registryURL.SubURL = providerURL
	return base.NewBaseInvoker(registryURL)
}

func installLifecycleRecordingRegistry(
	t *testing.T,
	registryScheme string,
	recordingRegistry *lifecycleRecordingRegistry,
) {
	t.Helper()
	extension.SetRegistry(registryScheme, func(url *common.URL) (registry.Registry, error) {
		mockRegistry, err := registry.NewMockRegistry(url)
		if err != nil {
			return nil, err
		}
		recordingRegistry.MockRegistry = mockRegistry.(*registry.MockRegistry)
		return recordingRegistry, nil
	})
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)
}

type lifecycleRecordingProtocol struct {
	unexported chan<- time.Time
}

func (p *lifecycleRecordingProtocol) Export(invoker base.Invoker) base.Exporter {
	return &recordingExporter{invoker: invoker, unexported: p.unexported}
}

func (*lifecycleRecordingProtocol) Refer(*common.URL) base.Invoker {
	return nil
}

func (*lifecycleRecordingProtocol) Destroy() {}

func TestDestroyWaitsForExporterLifecyclePublication(t *testing.T) {
	const registryScheme = "lifecycle-blocking"
	registerStarted := make(chan struct{})
	allowRegister := make(chan struct{})
	recordingRegistry := &lifecycleRecordingRegistry{
		registerStarted: registerStarted,
		allowRegister:   allowRegister,
	}
	installLifecycleRecordingRegistry(t, registryScheme, recordingRegistry)

	regProtocol := newRegistryProtocol()
	invoker := newLifecycleTestInvoker(t, registryScheme, "org.apache.dubbo-go.lifecycleService")
	exportDone := make(chan base.Exporter, 1)
	go func() {
		exportDone <- regProtocol.Export(invoker)
	}()
	select {
	case <-registerStarted:
	case <-time.After(time.Second):
		require.FailNow(t, "registration did not start")
	}

	destroyDone := make(chan struct{})
	go func() {
		regProtocol.Destroy()
		close(destroyDone)
	}()
	require.Eventually(t, func() bool {
		regProtocol.lifecycleMu.Lock()
		defer regProtocol.lifecycleMu.Unlock()
		return regProtocol.stopping
	}, time.Second, time.Millisecond)
	select {
	case <-destroyDone:
		require.FailNow(t, "destroy observed an exporter before registration completed")
	default:
	}

	close(allowRegister)
	var exporter base.Exporter
	select {
	case exporter = <-exportDone:
	case <-time.After(time.Second):
		require.FailNow(t, "export did not complete")
	}
	require.NotNil(t, exporter)
	select {
	case <-destroyDone:
	case <-time.After(time.Second):
		require.FailNow(t, "destroy did not complete")
	}

	wrapper := exporter.(*exporterChangeableWrapper)
	registerURL, subscribeURL, _ := wrapper.LifecycleURLs()
	require.NotNil(t, registerURL)
	require.NotNil(t, subscribeURL)
	registeredURL, unregisteredURL, unregisterCount, registered := recordingRegistry.snapshot()
	assert.Same(t, registeredURL, registerURL)
	assert.Same(t, registeredURL, unregisteredURL)
	assert.Equal(t, 1, unregisterCount)
	assert.False(t, registered)
}

func TestRegistrationFailureRemovesUnpublishedExporter(t *testing.T) {
	const registryScheme = "lifecycle-failing"
	recordingRegistry := &lifecycleRecordingRegistry{registerErr: errors.New("register failed")}
	installLifecycleRecordingRegistry(t, registryScheme, recordingRegistry)

	regProtocol := newRegistryProtocol()
	exporter := regProtocol.Export(newLifecycleTestInvoker(
		t,
		registryScheme,
		"org.apache.dubbo-go.failedLifecycleService",
	))

	assert.Nil(t, exporter)
	assert.Zero(t, registry.CountSyncMapEntries(regProtocol.bounds))
	assert.Zero(t, registry.CountSyncMapEntries(regProtocol.overrideListeners))
	assert.Zero(t, registry.CountSyncMapEntries(regProtocol.serviceConfigurationListeners))
	regProtocol.Destroy()
}

func TestBlockedExportDoesNotDelayDifferentProvider(t *testing.T) {
	const (
		blockingRegistryScheme = "lifecycle-blocked-provider"
		readyRegistryScheme    = "lifecycle-ready-provider"
	)
	registerStarted := make(chan struct{})
	allowRegister := make(chan struct{})
	blockingRegistry := &lifecycleRecordingRegistry{
		registerStarted: registerStarted,
		allowRegister:   allowRegister,
	}
	readyRegistry := &lifecycleRecordingRegistry{}
	installLifecycleRecordingRegistry(t, blockingRegistryScheme, blockingRegistry)
	installLifecycleRecordingRegistry(t, readyRegistryScheme, readyRegistry)

	regProtocol := newRegistryProtocol()
	blockedInvoker := newLifecycleTestInvoker(
		t,
		blockingRegistryScheme,
		"org.apache.dubbo-go.blockedLifecycleService",
	)
	readyInvoker := newLifecycleTestInvoker(
		t,
		readyRegistryScheme,
		"org.apache.dubbo-go.readyLifecycleService",
	)
	blockedExportDone := make(chan base.Exporter, 1)
	go func() {
		blockedExportDone <- regProtocol.Export(blockedInvoker)
	}()
	select {
	case <-registerStarted:
	case <-time.After(time.Second):
		require.FailNow(t, "blocking registration did not start")
	}

	readyExportDone := make(chan base.Exporter, 1)
	go func() {
		readyExportDone <- regProtocol.Export(readyInvoker)
	}()
	select {
	case exporter := <-readyExportDone:
		require.NotNil(t, exporter)
	case <-time.After(time.Second):
		require.FailNow(t, "an unrelated provider export was blocked")
	}

	close(allowRegister)
	select {
	case exporter := <-blockedExportDone:
		require.NotNil(t, exporter)
	case <-time.After(time.Second):
		require.FailNow(t, "blocked export did not complete")
	}
	regProtocol.Destroy()
}

func TestReExportWaitsForExporterLifecyclePublication(t *testing.T) {
	t.Run("different cache keys", func(t *testing.T) {
		const registryScheme = "lifecycle-reexport-different-key"
		registerStarted := make(chan struct{})
		allowRegister := make(chan struct{})
		var releaseRegister sync.Once
		release := func() {
			releaseRegister.Do(func() {
				close(allowRegister)
			})
		}
		recordingRegistry := &lifecycleRecordingRegistry{
			registerStarted:   registerStarted,
			allowRegister:     allowRegister,
			blockRegisterCall: 2,
		}
		installLifecycleRecordingRegistry(t, registryScheme, recordingRegistry)
		unexported := make(chan time.Time, 4)
		extension.SetProtocol(protocolwrapper.FILTER, func() base.Protocol {
			return &lifecycleRecordingProtocol{unexported: unexported}
		})

		regProtocol := newRegistryProtocol()
		t.Cleanup(func() {
			release()
			regProtocol.Destroy()
		})
		invoker := newLifecycleTestInvoker(
			t,
			registryScheme,
			"org.apache.dubbo-go.reexportLifecycleService",
		)
		oldExporter := regProtocol.Export(invoker)
		require.NotNil(t, oldExporter)
		oldKey := getCacheKey(invoker)

		repeatedExportDone := make(chan base.Exporter, 1)
		go func() {
			repeatedExportDone <- regProtocol.Export(invoker)
		}()
		select {
		case <-registerStarted:
		case <-time.After(time.Second):
			require.FailNow(t, "repeated registration did not start")
		}

		newURL := invoker.GetURL().Clone()
		newURL.SubURL = invoker.GetURL().SubURL.Clone()
		newURL.SubURL.SetParam(constant.ClusterKey, "reconfigured")
		newKey := getCacheKey(newInvokerDelegate(invoker, newURL))
		require.NotEqual(t, oldKey, newKey)

		reExportStarted := make(chan struct{})
		reExportDone := make(chan struct{})
		go func() {
			close(reExportStarted)
			regProtocol.reExport(invoker, newURL)
			close(reExportDone)
		}()
		<-reExportStarted

		select {
		case <-unexported:
			require.FailNow(t, "re-export closed an exporter whose registration was still in progress")
		case <-time.After(100 * time.Millisecond):
		}
		bound, loaded := regProtocol.bounds.Load(oldKey)
		require.True(t, loaded)
		require.Same(t, oldExporter, bound)

		release()
		select {
		case exporter := <-repeatedExportDone:
			require.Same(t, oldExporter, exporter)
		case <-time.After(time.Second):
			require.FailNow(t, "repeated export did not complete")
		}
		select {
		case <-reExportDone:
		case <-time.After(time.Second):
			require.FailNow(t, "re-export did not complete")
		}

		_, oldLoaded := regProtocol.bounds.Load(oldKey)
		assert.False(t, oldLoaded)
		newBound, newLoaded := regProtocol.bounds.Load(newKey)
		require.True(t, newLoaded)
		registerURL, subscribeURL, _ := newBound.(*exporterChangeableWrapper).LifecycleURLs()
		require.NotNil(t, registerURL)
		require.NotNil(t, subscribeURL)
		select {
		case <-unexported:
		case <-time.After(time.Second):
			require.FailNow(t, "old exporter was not closed after its registration completed")
		}
		select {
		case <-unexported:
			require.FailNow(t, "old exporter was closed more than once")
		default:
		}
		regProtocol.exportLocksMu.Lock()
		assert.Empty(t, regProtocol.exportLocks)
		regProtocol.exportLocksMu.Unlock()
	})

	t.Run("same cache key", func(t *testing.T) {
		const registryScheme = "lifecycle-reexport-same-key"
		recordingRegistry := &lifecycleRecordingRegistry{}
		installLifecycleRecordingRegistry(t, registryScheme, recordingRegistry)
		unexported := make(chan time.Time, 4)
		extension.SetProtocol(protocolwrapper.FILTER, func() base.Protocol {
			return &lifecycleRecordingProtocol{unexported: unexported}
		})

		regProtocol := newRegistryProtocol()
		t.Cleanup(regProtocol.Destroy)
		invoker := newLifecycleTestInvoker(
			t,
			registryScheme,
			"org.apache.dubbo-go.sameKeyReexportLifecycleService",
		)
		require.NotNil(t, regProtocol.Export(invoker))
		oldKey := getCacheKey(invoker)

		newURL := invoker.GetURL().Clone()
		newURL.SubURL = invoker.GetURL().SubURL.Clone()
		newURL.SubURL.SetParam("dynamic", "false")
		newKey := getCacheKey(newInvokerDelegate(invoker, newURL))
		require.Equal(t, oldKey, newKey)

		reExportDone := make(chan struct{})
		go func() {
			regProtocol.reExport(invoker, newURL)
			close(reExportDone)
		}()
		select {
		case <-reExportDone:
		case <-time.After(time.Second):
			require.FailNow(t, "same-key re-export deadlocked")
		}

		newBound, loaded := regProtocol.bounds.Load(newKey)
		require.True(t, loaded)
		registerURL, subscribeURL, _ := newBound.(*exporterChangeableWrapper).LifecycleURLs()
		require.NotNil(t, registerURL)
		require.NotNil(t, subscribeURL)
		select {
		case <-unexported:
		case <-time.After(time.Second):
			require.FailNow(t, "same-key re-export did not close the old exporter")
		}
		regProtocol.exportLocksMu.Lock()
		assert.Empty(t, regProtocol.exportLocks)
		regProtocol.exportLocksMu.Unlock()
	})
}

func TestConcurrentUnregisterAndDestroyUnregisterOnlyOnce(t *testing.T) {
	const registryScheme = "lifecycle-unregister-once"
	recordingRegistry := &lifecycleRecordingRegistry{}
	installLifecycleRecordingRegistry(t, registryScheme, recordingRegistry)

	regProtocol := newRegistryProtocol()
	require.NotNil(t, regProtocol.Export(newLifecycleTestInvoker(
		t,
		registryScheme,
		"org.apache.dubbo-go.unregisterLifecycleService",
	)))

	start := make(chan struct{})
	var shutdown sync.WaitGroup
	shutdown.Add(2)
	go func() {
		defer shutdown.Done()
		<-start
		regProtocol.UnregisterRegistries()
	}()
	go func() {
		defer shutdown.Done()
		<-start
		regProtocol.Destroy()
	}()
	close(start)
	shutdown.Wait()

	_, _, unregisterCount, registered := recordingRegistry.snapshot()
	assert.Equal(t, 1, unregisterCount)
	assert.False(t, registered)
}

func (r *unsubscribeRecordingRegistry) UnSubscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	r.unSubscribeCount++
	r.unSubscribeURL = url
	r.unSubscribeListener = notifyListener
	return nil
}

type recordingExporter struct {
	invoker    base.Invoker
	unexported chan<- time.Time
}

func (e *recordingExporter) GetInvoker() base.Invoker {
	return e.invoker
}

func (e *recordingExporter) UnExport() {
	e.unexported <- time.Now()
}

func newRegistryProtocolWithSubscribedExporter(
	t *testing.T,
) (*registryProtocol, *unsubscribeRecordingRegistry, base.Invoker, *exporterChangeableWrapper, *overrideSubscribeListener) {
	t.Helper()

	var recordingRegistry *unsubscribeRecordingRegistry
	extension.SetRegistry("recording", func(url *common.URL) (registry.Registry, error) {
		mockRegistry, err := registry.NewMockRegistry(url)
		if err != nil {
			return nil, err
		}
		recordingRegistry = &unsubscribeRecordingRegistry{MockRegistry: mockRegistry.(*registry.MockRegistry)}
		return recordingRegistry, nil
	})
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)

	registryURL, err := common.NewURL("recording://127.0.0.1:1111")
	require.NoError(t, err)
	providerURL, err := common.NewURL(
		"dubbo://127.0.0.1:20000/org.apache.dubbo-go.mockService",
		common.WithParamsValue(constant.ClusterKey, "mock"),
		common.WithParamsValue(constant.GroupKey, "group"),
		common.WithParamsValue(constant.VersionKey, "1.0.0"),
	)
	require.NoError(t, err)
	registryURL.SubURL = providerURL

	originInvoker := base.NewBaseInvoker(registryURL)
	regProtocol := newRegistryProtocol()
	regProtocol.initConfigurationListeners(providerURL)
	regProtocol.getRegistry(registryURL)

	overrideURL := getSubscribedOverrideUrl(providerURL)
	listener := newOverrideSubscribeListener(overrideURL, originInvoker, regProtocol)
	regProtocol.overrideListeners.Store(overrideURL.String(), listener)
	serviceConfigurationListener := newServiceConfigurationListener(listener, providerURL)
	regProtocol.serviceConfigurationListeners.Store(providerURL.ServiceKey(), serviceConfigurationListener)

	exporter := newExporterChangeableWrapper(originInvoker, base.NewBaseExporter("recording-key", base.NewBaseInvoker(providerURL), &sync.Map{}))
	exporter.SetRegisterUrl(getUrlToRegistry(providerURL, registryURL))
	exporter.SetSubscribeUrl(overrideURL)

	return regProtocol, recordingRegistry, originInvoker, exporter, listener
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

type recordingDynamicConfiguration struct {
	parser configparser.ConfigurationParser
	keys   []string
}

func newRecordingDynamicConfiguration() *recordingDynamicConfiguration {
	return &recordingDynamicConfiguration{parser: &configparser.DefaultConfigurationParser{}}
}

func (c *recordingDynamicConfiguration) Parser() configparser.ConfigurationParser {
	return c.parser
}

func (c *recordingDynamicConfiguration) SetParser(parser configparser.ConfigurationParser) {
	c.parser = parser
}

func (c *recordingDynamicConfiguration) AddListener(key string, _ config_center.ConfigurationListener, _ ...config_center.Option) {
	c.keys = append(c.keys, key)
}

func (c *recordingDynamicConfiguration) RemoveListener(_ string, _ config_center.ConfigurationListener, _ ...config_center.Option) {
}

func (c *recordingDynamicConfiguration) GetProperties(_ string, _ ...config_center.Option) (string, error) {
	return "", nil
}

func (c *recordingDynamicConfiguration) GetRule(_ string, _ ...config_center.Option) (string, error) {
	return "", nil
}

func (c *recordingDynamicConfiguration) GetInternalProperty(_ string, _ ...config_center.Option) (string, error) {
	return "", nil
}

func (c *recordingDynamicConfiguration) PublishConfig(_, _, _ string) error {
	return nil
}

func (c *recordingDynamicConfiguration) RemoveConfig(_, _ string) error {
	return nil
}

func (c *recordingDynamicConfiguration) GetConfigKeysByGroup(_ string) (*gxset.HashSet, error) {
	return gxset.NewSet(), nil
}
