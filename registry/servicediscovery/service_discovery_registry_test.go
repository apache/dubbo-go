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
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	gxset "github.com/dubbogo/gost/container/set"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/proxy"
	"dubbo.apache.org/dubbo-go/v3/registry"
	gxpage "github.com/dubbogo/gost/hash/page"
	"github.com/stretchr/testify/assert"
)

const (
	testInterface   = "org.apache.dubbo.test.TestService"
	testGroup       = "test-group"
	testApp         = "test-app"
	testRegistryURL = "service-discovery://localhost:12345"
)

// TestServiceDiscoveryRegistryRegister verifies the registration process.
func TestServiceDiscoveryRegistryRegister(t *testing.T) {
	mockSD, mockMapping := setupEnvironment(t)
	regID := fmt.Sprintf("mock-reg-%d", time.Now().UnixNano())

	registryURL, err := common.NewURL(testRegistryURL,
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

	registryURL, _ := common.NewURL(testRegistryURL,
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

	registryURL, _ := common.NewURL(testRegistryURL,
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
	appConfig := global.DefaultApplicationConfig()
	appConfig.Name = testApp

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

	// for Unregister tests
	unregisterCalled  bool
	unregisterIDs     []string
	unregisterErrByID map[string]error
}

func (m *mockServiceDiscovery) String() string { return "mock" }
func (m *mockServiceDiscovery) Destroy() error { return nil }
func (m *mockServiceDiscovery) Register(inst registry.ServiceInstance) error {
	m.registerCalled = true
	m.capturedInstance = inst
	return nil
}
func (m *mockServiceDiscovery) Update(inst registry.ServiceInstance) error { return nil }
func (m *mockServiceDiscovery) Unregister(inst registry.ServiceInstance) error {
	m.unregisterCalled = true
	if inst != nil {
		m.unregisterIDs = append(m.unregisterIDs, inst.GetID())
		if m.unregisterErrByID != nil {
			if err, ok := m.unregisterErrByID[inst.GetID()]; ok {
				return err
			}
		}
	}
	return nil
}
func (m *mockServiceDiscovery) GetDefaultPageSize() int     { return 10 }
func (m *mockServiceDiscovery) GetServices() *gxset.HashSet { return gxset.NewSet("mock-service") }
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

func (m *mockNotifyListener) Notify(*registry.ServiceEvent) {
	// for mocking
}

func (m *mockNotifyListener) NotifyAll([]*registry.ServiceEvent, func()) {
	// for mocking
}

type mockProtocol struct{}

func (m *mockProtocol) Export(invoker protocol.Invoker) protocol.Exporter { return &mockExporter{} }
func (m *mockProtocol) Refer(url *common.URL) protocol.Invoker            { return &mockInvoker{} }
func (m *mockProtocol) Destroy() {
	// for mocking
}

type mockExporter struct{}

func (m *mockExporter) UnExport() {
	// for mocking
}

func (m *mockExporter) Unexport() {
	// for mocking
}
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

func (m *mockInvoker) GetURL() *common.URL { return nil }
func (m *mockInvoker) IsAvailable() bool   { return true }
func (m *mockInvoker) Destroy() {
	// for mocking
}
func (m *mockInvoker) Invoke(context.Context, protocol.Invocation) protocol.Result { return nil }

// TestServiceDiscoveryRegistryUnRegister_AllSuccess verifies UnRegisterService bulk deregister success.
func TestServiceDiscoveryRegistryUnRegister_AllSuccess(t *testing.T) {
	mockSD, _ := setupEnvironment(t)

	registryURL, _ := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"))

	reg, err := newServiceDiscoveryRegistry(registryURL)
	assert.NoError(t, err)

	sdReg, ok := reg.(*serviceDiscoveryRegistry)
	assert.True(t, ok)

	inst1 := &registry.DefaultServiceInstance{
		ID:          "inst-1",
		ServiceName: testApp,
		Host:        "127.0.0.1",
		Port:        20880,
		Enable:      true,
		Healthy:     true,
		Metadata:    map[string]string{"k": "v"},
	}
	inst2 := &registry.DefaultServiceInstance{
		ID:          "inst-2",
		ServiceName: testApp,
		Host:        "127.0.0.1",
		Port:        20881,
		Enable:      true,
		Healthy:     true,
		Metadata:    map[string]string{"k2": "v2"},
	}

	sdReg.instances = []registry.ServiceInstance{inst1, inst2}
	mockSD.unregisterErrByID = nil

	err = sdReg.UnRegisterService()
	assert.NoError(t, err)

	// all instances should be removed after successful deregister
	assert.Empty(t, sdReg.instances)

	// verify Unregister called for each instance
	assert.True(t, mockSD.unregisterCalled)
	assert.ElementsMatch(t, []string{"inst-1", "inst-2"}, mockSD.unregisterIDs)
}

// TestServiceDiscoveryRegistryUnRegister_PartialFail verifies failed instances are kept and errors are returned.
func TestServiceDiscoveryRegistryUnRegister_PartialFail(t *testing.T) {
	mockSD, _ := setupEnvironment(t)

	registryURL, _ := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"))

	reg, err := newServiceDiscoveryRegistry(registryURL)
	assert.NoError(t, err)

	sdReg, ok := reg.(*serviceDiscoveryRegistry)
	assert.True(t, ok)

	inst1 := &registry.DefaultServiceInstance{
		ID:          "inst-1",
		ServiceName: testApp,
		Host:        "127.0.0.1",
		Port:        20880,
		Enable:      true,
		Healthy:     true,
	}
	inst2 := &registry.DefaultServiceInstance{
		ID:          "inst-2",
		ServiceName: testApp,
		Host:        "127.0.0.1",
		Port:        20881,
		Enable:      true,
		Healthy:     true,
	}

	sdReg.instances = []registry.ServiceInstance{inst1, inst2}

	mockSD.unregisterErrByID = map[string]error{
		"inst-1": errors.New("mock unregister failed"),
	}

	err = sdReg.UnRegisterService()
	assert.Error(t, err)

	// failed instance should remain, successful one removed
	assert.Equal(t, 1, len(sdReg.instances))
	assert.Equal(t, "inst-1", sdReg.instances[0].GetID())

	// still called for both
	assert.True(t, mockSD.unregisterCalled)
	assert.ElementsMatch(t, []string{"inst-1", "inst-2"}, mockSD.unregisterIDs)
}

// TestServiceDiscoveryRegistryUnRegister_NoInstances verifies UnRegisterService handles empty list.
func TestServiceDiscoveryRegistryUnRegister_NoInstances(t *testing.T) {
	mockSD, _ := setupEnvironment(t)

	registryURL, _ := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"))

	reg, err := newServiceDiscoveryRegistry(registryURL)
	assert.NoError(t, err)

	sdReg, ok := reg.(*serviceDiscoveryRegistry)
	assert.True(t, ok)

	sdReg.instances = []registry.ServiceInstance{}

	err = sdReg.UnRegisterService()
	assert.NoError(t, err)

	// should stay empty
	assert.Empty(t, sdReg.instances)

	// should not call Unregister
	assert.False(t, mockSD.unregisterCalled)
	assert.Empty(t, mockSD.unregisterIDs)
}

// TestServiceDiscoveryRegistryUnRegister_Concurrent verifies UnRegisterService under concurrent access.
func TestServiceDiscoveryRegistryUnRegister_Concurrent(t *testing.T) {
	mockSD, _ := setupEnvironment(t)

	registryURL, _ := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"))

	reg, err := newServiceDiscoveryRegistry(registryURL)
	assert.NoError(t, err)

	sdReg, ok := reg.(*serviceDiscoveryRegistry)
	assert.True(t, ok)

	// prepare initial instances
	for i := 0; i < 5; i++ {
		inst := &registry.DefaultServiceInstance{
			ID:          fmt.Sprintf("init-%d", i),
			ServiceName: testApp,
			Host:        "127.0.0.1",
			Port:        20000 + i,
			Enable:      true,
			Healthy:     true,
		}
		sdReg.instances = append(sdReg.instances, inst)
	}

	mockSD.unregisterErrByID = map[string]error{
		"init-2": errors.New("mock unregister failed"),
	}

	var wg sync.WaitGroup
	panicCh := make(chan any, 2)

	// goroutine 1: unregister
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()
		_ = sdReg.UnRegisterService()
	}()

	// goroutine 2: simulate concurrent register / instance append
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()

		for i := 0; i < 3; i++ {
			inst := &registry.DefaultServiceInstance{
				ID:          fmt.Sprintf("concurrent-%d", i),
				ServiceName: testApp,
				Host:        "127.0.0.1",
				Port:        21000 + i,
				Enable:      true,
				Healthy:     true,
			}
			// intentionally no lock: simulating real race
			sdReg.instances = append(sdReg.instances, inst)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	wg.Wait()
	close(panicCh)

	// assert no panic happened
	for p := range panicCh {
		t.Fatalf("panic occurred during concurrent unregister: %v", p)
	}

	// at least the failed instance should remain
	foundFailed := false
	for _, inst := range sdReg.instances {
		if inst.GetID() == "init-2" {
			foundFailed = true
			break
		}
	}
	assert.True(t, foundFailed, "failed instance should be kept after UnRegisterService")

	// unregister should be called
	assert.True(t, mockSD.unregisterCalled)
}
