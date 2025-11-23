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

package servicediscovery

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/hash/page"
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/proxy"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

const (
	testInterface = "org.apache.dubbo.test.TestService"
	testGroup     = "test-group"
	testApp       = "test-app"
)

// TestServiceDiscoveryRegistryRegister verifies the registration process.
func TestServiceDiscoveryRegistryRegister(t *testing.T) {
	mockSD, mockMapping := setupEnvironment(t)
	regID := fmt.Sprintf("mock-reg-%d", time.Now().UnixNano())

	registryURL, err := common.NewURL("service-discovery://localhost:12345",
		common.WithParamsValue(constant.RegistryKey, "mock"),
		common.WithParamsValue(constant.RegistryIdKey, regID))
	assert.NoError(t, err)

	reg, err := newServiceDiscoveryRegistry(registryURL)
	assert.NoError(t, err)

	providerURL, _ := common.NewURL("dubbo://127.0.0.1:20880/",
		common.WithParamsValue(constant.ApplicationKey, testApp),
		common.WithInterface(testInterface),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)

	// Interface Mapping
	err = reg.Register(providerURL)
	assert.NoError(t, err)
	assert.True(t, mockMapping.mapCalled, "ServiceNameMapping.Map should be called")

	// Instance Registration
	sdReg, ok := reg.(*serviceDiscoveryRegistry)
	assert.True(t, ok)

	err = sdReg.RegisterService()
	assert.NoError(t, err)

	assert.True(t, mockSD.registerCalled, "ServiceDiscovery.Register should be called")

	if mockSD.capturedInstance != nil {
		assert.Equal(t, testApp, mockSD.capturedInstance.GetServiceName())
		assert.Equal(t, "127.0.0.1", mockSD.capturedInstance.GetHost())
		assert.Equal(t, 20880, mockSD.capturedInstance.GetPort())
		assert.Equal(t, "mock", mockSD.capturedInstance.GetMetadata()[constant.MetadataStorageTypePropertyName])
	}
}

// TestServiceDiscoveryRegistrySubscribe verifies the subscription flow.
func TestServiceDiscoveryRegistrySubscribe(t *testing.T) {
	mockSD, mockMapping := setupEnvironment(t)
	mockMapping.data[testInterface] = gxset.NewSet(testApp)

	registryURL, _ := common.NewURL("service-discovery://localhost:12345",
		common.WithParamsValue(constant.RegistryKey, "mock"))

	reg, err := newServiceDiscoveryRegistry(registryURL)
	assert.NoError(t, err)

	consumerURL, _ := common.NewURL("dubbo://127.0.0.1:20000/",
		common.WithInterface(testInterface),
		common.WithParamsValue(constant.GroupKey, testGroup),
		common.WithParamsValue(constant.SideKey, constant.SideConsumer),
	)

	mockSD.wg.Add(1)
	err = reg.Subscribe(consumerURL, &mockNotifyListener{})
	assert.NoError(t, err)

	assert.True(t, mockMapping.getCalled)
	assert.Equal(t, testApp, mockSD.capturedAppName)

	mockSD.wg.Wait()
	assert.True(t, mockSD.listenerAdded)
}

// TestServiceDiscoveryRegistryUnSubscribe verifies the unsubscription logic.
func TestServiceDiscoveryRegistryUnSubscribe(t *testing.T) {
	mockSD, mockMapping := setupEnvironment(t)
	mockMapping.data[testInterface] = gxset.NewSet(testApp)

	registryURL, _ := common.NewURL("service-discovery://localhost:12345",
		common.WithParamsValue(constant.RegistryKey, "mock"))

	reg, err := newServiceDiscoveryRegistry(registryURL)
	assert.NoError(t, err)

	consumerURL, _ := common.NewURL("dubbo://127.0.0.1:20000/",
		common.WithInterface(testInterface),
		common.WithParamsValue(constant.SideKey, constant.SideConsumer),
	)

	mockSD.wg.Add(1)
	_ = reg.Subscribe(consumerURL, &mockNotifyListener{})
	mockSD.wg.Wait()

	err = reg.UnSubscribe(consumerURL, &mockNotifyListener{})
	assert.NoError(t, err)
	assert.True(t, mockMapping.removeCalled)
}

// setupEnvironment initializes the test environment.
func setupEnvironment(t *testing.T) (*mockServiceDiscovery, *mockServiceNameMapping) {
	rootConfig := config.NewRootConfigBuilder().
		SetApplication(config.NewApplicationConfigBuilder().SetName(testApp).Build()).
		Build()
	config.SetRootConfig(*rootConfig)

	mockSD := &mockServiceDiscovery{}
	mockMapping := &mockServiceNameMapping{data: make(map[string]*gxset.HashSet)}

	extension.SetProtocol("dubbo", func() protocol.Protocol { return &mockProtocol{} })
	extension.SetProxyFactory("default", func(options ...proxy.Option) proxy.ProxyFactory { return &mockProxyFactory{} })
	extension.SetRegistry("service-discovery", newServiceDiscoveryRegistry)
	extension.SetServiceDiscovery("mock", func(url *common.URL) (registry.ServiceDiscovery, error) {
		return mockSD, nil
	})
	extension.SetGlobalServiceNameMapping(func() mapping.ServiceNameMapping {
		return mockMapping
	})

	opts := metadata.NewOptions(metadata.WithMetadataType("mock"))
	_ = opts.Init()
	return mockSD, mockMapping
}

type mockServiceDiscovery struct {
	wg               sync.WaitGroup
	registerCalled   bool
	listenerAdded    bool
	capturedAppName  string
	capturedInstance registry.ServiceInstance
}

func (m *mockServiceDiscovery) String() string { return "mock" }
func (m *mockServiceDiscovery) Destroy() error { return nil }
func (m *mockServiceDiscovery) Register(inst registry.ServiceInstance) error {
	m.registerCalled = true
	m.capturedInstance = inst
	return nil
}
func (m *mockServiceDiscovery) Update(inst registry.ServiceInstance) error     { return nil }
func (m *mockServiceDiscovery) Unregister(inst registry.ServiceInstance) error { return nil }
func (m *mockServiceDiscovery) GetDefaultPageSize() int                        { return 10 }
func (m *mockServiceDiscovery) GetServices() *gxset.HashSet                    { return gxset.NewSet("mock-service") }
func (m *mockServiceDiscovery) GetInstances(name string) []registry.ServiceInstance {
	m.capturedAppName = name
	return []registry.ServiceInstance{}
}
func (m *mockServiceDiscovery) GetInstancesByPage(string, int, int) gxpage.Pager { return nil }
func (m *mockServiceDiscovery) GetHealthyInstancesByPage(string, int, int, bool) gxpage.Pager {
	return nil
}
func (m *mockServiceDiscovery) GetRequestInstances([]string, int, int) map[string]gxpage.Pager {
	return nil
}
func (m *mockServiceDiscovery) AddListener(registry.ServiceInstancesChangedListener) error {
	defer m.wg.Done()
	m.listenerAdded = true
	return nil
}

type mockServiceNameMapping struct {
	data          map[string]*gxset.HashSet
	mapCalled     bool
	getCalled     bool
	removeCalled  bool
	capturedGroup string
}

func (m *mockServiceNameMapping) Map(url *common.URL) error {
	m.mapCalled = true
	serviceInterface := url.GetParam(constant.InterfaceKey, "")
	appName := url.GetParam(constant.ApplicationKey, "")
	m.data[serviceInterface] = gxset.NewSet(appName)
	return nil
}
func (m *mockServiceNameMapping) Get(url *common.URL, _ mapping.MappingListener) (*gxset.HashSet, error) {
	m.getCalled = true
	m.capturedGroup = url.GetParam(constant.GroupKey, "")
	serviceInterface := url.GetParam(constant.InterfaceKey, "")
	if s, ok := m.data[serviceInterface]; ok {
		return s, nil
	}
	return gxset.NewSet(), nil
}
func (m *mockServiceNameMapping) Remove(url *common.URL) error { m.removeCalled = true; return nil }

type mockNotifyListener struct{}

func (m *mockNotifyListener) Notify(*registry.ServiceEvent)              {}
func (m *mockNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {}

type mockProtocol struct{}

func (m *mockProtocol) Export(invoker protocol.Invoker) protocol.Exporter { return &mockExporter{} }
func (m *mockProtocol) Refer(url *common.URL) protocol.Invoker            { return &mockInvoker{} }
func (m *mockProtocol) Destroy()                                          {}

type mockExporter struct{}

func (m *mockExporter) UnExport() {}

func (m *mockExporter) Unexport()                    {}
func (m *mockExporter) GetInvoker() protocol.Invoker { return &mockInvoker{} }

type mockProxyFactory struct{}

func (m *mockProxyFactory) GetProxy(invoker base.Invoker, url *common.URL) *proxy.Proxy {
	return &proxy.Proxy{}
}

func (m *mockProxyFactory) GetAsyncProxy(invoker base.Invoker, callBack any, url *common.URL) *proxy.Proxy {
	return &proxy.Proxy{}
}

func (m *mockProxyFactory) GetInvoker(url *common.URL) protocol.Invoker { return &mockInvoker{} }

type mockInvoker struct{}

func (m *mockInvoker) GetURL() *common.URL                                         { return nil }
func (m *mockInvoker) IsAvailable() bool                                           { return true }
func (m *mockInvoker) Destroy()                                                    {}
func (m *mockInvoker) Invoke(context.Context, protocol.Invocation) protocol.Result { return nil }
