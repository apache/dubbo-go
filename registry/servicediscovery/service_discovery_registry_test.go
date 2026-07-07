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
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/hash/page"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/proxy"
	"dubbo.apache.org/dubbo-go/v3/registry"
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
	regID := fmt.Sprintf("mock-reg-%s-%d", t.Name(), time.Now().UnixNano())

	registryURL, err := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"),
		common.WithParamsValue(constant.RegistryIdKey, regID))
	require.NoError(t, err)

	reg, err := newServiceDiscoveryRegistry(registryURL)
	require.NoError(t, err)

	providerURL, _ := common.NewURL("dubbo://127.0.0.1:20880/",
		common.WithParamsValue(constant.ApplicationKey, testApp),
		common.WithInterface(testInterface),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)

	// Interface Mapping
	err = reg.Register(providerURL)
	require.NoError(t, err)
	assert.True(t, mockMapping.mapCalled, "ServiceNameMapping.Map should be called")

	// Instance Registration
	sdReg, ok := reg.(*serviceDiscoveryRegistry)
	assert.True(t, ok)

	err = sdReg.RegisterService()
	require.NoError(t, err)

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
	require.NoError(t, err)

	consumerURL, _ := common.NewURL("dubbo://127.0.0.1:20000/",
		common.WithInterface(testInterface),
		common.WithParamsValue(constant.GroupKey, testGroup),
		common.WithParamsValue(constant.SideKey, constant.SideConsumer),
	)

	mockSD.wg.Add(1)
	err = reg.Subscribe(consumerURL, &mockNotifyListener{})
	require.NoError(t, err)

	assert.True(t, mockMapping.getCalled)
	assert.Equal(t, testApp, mockSD.capturedAppName)

	mockSD.wg.Wait()
	assert.True(t, mockSD.listenerAdded)
}

// TestServiceDiscoveryRegistryUnSubscribe verifies the unsubscription logic.
func TestServiceDiscoveryRegistryUnSubscribe(t *testing.T) {
	mockSD, mockMapping := setupEnvironment(t)
	mockMapping.data[testInterface] = gxset.NewSet(testApp)
	regID := fmt.Sprintf("mock-reg-%s-%d", t.Name(), time.Now().UnixNano())

	registryURL, _ := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"),
		common.WithParamsValue(constant.RegistryIdKey, regID))

	reg, err := newServiceDiscoveryRegistry(registryURL)
	require.NoError(t, err)

	consumerURL, _ := common.NewURL("dubbo://127.0.0.1:20000/",
		common.WithInterface(testInterface),
		common.WithParamsValue(constant.SideKey, constant.SideConsumer),
	)

	mockSD.wg.Add(1)
	_ = reg.Subscribe(consumerURL, &mockNotifyListener{})
	mockSD.wg.Wait()

	metaInfo := metadata.GetMetadataInfo(regID)
	require.NotNil(t, metaInfo)
	assert.Len(t, metaInfo.GetSubscribedURLs(), 1)

	err = reg.UnSubscribe(consumerURL, &mockNotifyListener{})
	require.NoError(t, err)
	assert.True(t, mockMapping.removeCalled)
	assert.Empty(t, metaInfo.GetSubscribedURLs())
}

func TestServiceDiscoveryRegistryUnSubscribeKeepsMetadataOnRemoveFailure(t *testing.T) {
	mockSD, mockMapping := setupEnvironment(t)
	mockMapping.data[testInterface] = gxset.NewSet(testApp)
	mockMapping.removeErr = errors.New("mock remove failed")
	regID := fmt.Sprintf("mock-reg-%s-%d", t.Name(), time.Now().UnixNano())

	registryURL, _ := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"),
		common.WithParamsValue(constant.RegistryIdKey, regID))

	reg, err := newServiceDiscoveryRegistry(registryURL)
	require.NoError(t, err)

	consumerURL, _ := common.NewURL("dubbo://127.0.0.1:20000/",
		common.WithInterface(testInterface),
		common.WithParamsValue(constant.SideKey, constant.SideConsumer),
	)

	mockSD.wg.Add(1)
	_ = reg.Subscribe(consumerURL, &mockNotifyListener{})
	mockSD.wg.Wait()

	metaInfo := metadata.GetMetadataInfo(regID)
	require.NotNil(t, metaInfo)
	require.Len(t, metaInfo.GetSubscribedURLs(), 1)

	err = reg.UnSubscribe(consumerURL.Clone(), &mockNotifyListener{})
	require.Error(t, err)
	assert.Len(t, metaInfo.GetSubscribedURLs(), 1)
}

func TestServiceDiscoveryRegistryUnRegisterSyncsBulkMetadataCleanup(t *testing.T) {
	mockSD, mockMapping := setupEnvironment(t)
	regID := fmt.Sprintf("mock-reg-%s-%d", t.Name(), time.Now().UnixNano())

	registryURL, err := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"),
		common.WithParamsValue(constant.RegistryIdKey, regID))
	require.NoError(t, err)

	reg, err := newServiceDiscoveryRegistry(registryURL)
	require.NoError(t, err)

	providerURL1, err := common.NewURL("dubbo://127.0.0.1:20880/org.apache.dubbo.test.TestService",
		common.WithParamsValue(constant.ApplicationKey, testApp),
		common.WithInterface(testInterface),
		common.WithMethods([]string{"MethodA"}),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)
	require.NoError(t, err)
	providerURL2, err := common.NewURL("tri://127.0.0.1:20881/org.apache.dubbo.test.TestService",
		common.WithParamsValue(constant.ApplicationKey, testApp),
		common.WithInterface(testInterface),
		common.WithMethods([]string{"MethodB"}),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)
	require.NoError(t, err)

	err = reg.Register(providerURL1)
	require.NoError(t, err)
	err = reg.Register(providerURL2)
	require.NoError(t, err)
	assert.True(t, mockMapping.mapCalled)

	sdReg, ok := reg.(*serviceDiscoveryRegistry)
	require.True(t, ok)

	err = sdReg.RegisterService()
	require.NoError(t, err)

	metaInfo := metadata.GetMetadataInfo(regID)
	require.NotNil(t, metaInfo)
	require.Len(t, metaInfo.GetExportedServiceURLs(), 2)
	oldRevision := metaInfo.Revision
	require.NotEmpty(t, oldRevision)

	err = sdReg.UnRegister(providerURL1.Clone())
	require.NoError(t, err)

	assert.Empty(t, metaInfo.GetExportedServiceURLs())
	assert.Empty(t, metaInfo.Services)
	assert.Equal(t, "0", metaInfo.Revision)
	assert.True(t, mockSD.unregisterCalled)
	assert.ElementsMatch(t, []string{providerURL1.Address(), providerURL2.Address()}, mockSD.unregisterIDs)
	assert.NotEqual(t, oldRevision, metaInfo.Revision)
}

func TestServiceDiscoveryRegistryUnRegisterServicePartialFailSyncsMetadata(t *testing.T) {
	mockSD, mockMapping := setupEnvironment(t)
	regID := fmt.Sprintf("mock-reg-%s-%d", t.Name(), time.Now().UnixNano())

	registryURL, err := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"),
		common.WithParamsValue(constant.RegistryIdKey, regID))
	require.NoError(t, err)

	reg, err := newServiceDiscoveryRegistry(registryURL)
	require.NoError(t, err)

	providerURL1, err := common.NewURL("dubbo://127.0.0.1:20880/org.apache.dubbo.test.TestService",
		common.WithParamsValue(constant.ApplicationKey, testApp),
		common.WithInterface(testInterface),
		common.WithMethods([]string{"MethodA"}),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)
	require.NoError(t, err)
	providerURL2, err := common.NewURL("tri://127.0.0.1:20881/org.apache.dubbo.test.TestService",
		common.WithParamsValue(constant.ApplicationKey, testApp),
		common.WithInterface(testInterface),
		common.WithMethods([]string{"MethodB"}),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)
	require.NoError(t, err)

	err = reg.Register(providerURL1)
	require.NoError(t, err)
	err = reg.Register(providerURL2)
	require.NoError(t, err)
	assert.True(t, mockMapping.mapCalled)

	sdReg, ok := reg.(*serviceDiscoveryRegistry)
	require.True(t, ok)

	err = sdReg.RegisterService()
	require.NoError(t, err)

	mockSD.unregisterErrByID = map[string]error{
		providerURL1.Address(): errors.New("mock unregister failed"),
	}

	err = sdReg.UnRegisterService()
	require.Error(t, err)

	metaInfo := metadata.GetMetadataInfo(regID)
	require.NotNil(t, metaInfo)
	require.Len(t, metaInfo.GetExportedServiceURLs(), 1)
	assert.Equal(t, providerURL1, metaInfo.GetExportedServiceURLs()[0])
	assert.Len(t, metaInfo.Services, 1)

	expectedRevision := createInstance(metaInfo, providerURL1, regID).GetMetadata()[constant.ExportedServicesRevisionPropertyName]
	assert.Equal(t, expectedRevision, metaInfo.Revision)
	assert.True(t, mockSD.updateCalled)
	assert.Contains(t, mockSD.updatedIDs, providerURL1.Address())
}

func TestServiceDiscoveryRegistryUnRegisterWithoutTrackedInstancesReconcilesMetadata(t *testing.T) {
	_, mockMapping := setupEnvironment(t)
	regID := fmt.Sprintf("mock-reg-%s-%d", t.Name(), time.Now().UnixNano())

	registryURL, err := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"),
		common.WithParamsValue(constant.RegistryIdKey, regID))
	require.NoError(t, err)

	reg, err := newServiceDiscoveryRegistry(registryURL)
	require.NoError(t, err)

	providerURL1, err := common.NewURL("dubbo://127.0.0.1:20880/org.apache.dubbo.test.TestService",
		common.WithParamsValue(constant.ApplicationKey, testApp),
		common.WithInterface(testInterface),
		common.WithMethods([]string{"MethodA"}),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)
	require.NoError(t, err)
	providerURL2, err := common.NewURL("tri://127.0.0.1:20881/org.apache.dubbo.test.TestService",
		common.WithParamsValue(constant.ApplicationKey, testApp),
		common.WithInterface(testInterface),
		common.WithMethods([]string{"MethodB"}),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)
	require.NoError(t, err)

	err = reg.Register(providerURL1)
	require.NoError(t, err)
	err = reg.Register(providerURL2)
	require.NoError(t, err)
	assert.True(t, mockMapping.mapCalled)

	metaInfo := metadata.GetMetadataInfo(regID)
	require.NotNil(t, metaInfo)
	require.Len(t, metaInfo.GetExportedServiceURLs(), 2)

	err = reg.UnRegister(providerURL1.Clone())
	require.NoError(t, err)

	require.Len(t, metaInfo.GetExportedServiceURLs(), 1)
	assert.Equal(t, providerURL2, metaInfo.GetExportedServiceURLs()[0])
	assert.Len(t, metaInfo.Services, 1)

	expectedRevision := createInstance(metaInfo, providerURL2, regID).GetMetadata()[constant.ExportedServicesRevisionPropertyName]
	assert.Equal(t, expectedRevision, metaInfo.Revision)
}

// setupEnvironment initializes the test environment.
func setupEnvironment(_ *testing.T) (*mockServiceDiscovery, *mockServiceNameMapping) {
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
	updateCalled      bool
	updatedIDs        []string
}

func (m *mockServiceDiscovery) String() string { return "mock" }
func (m *mockServiceDiscovery) Destroy() error { return nil }
func (m *mockServiceDiscovery) Register(inst registry.ServiceInstance) error {
	m.registerCalled = true
	m.capturedInstance = inst
	return nil
}
func (m *mockServiceDiscovery) Update(inst registry.ServiceInstance) error {
	m.updateCalled = true
	if inst != nil {
		m.updatedIDs = append(m.updatedIDs, inst.GetID())
	}
	return nil
}
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
	removeErr     error
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
func (m *mockServiceNameMapping) Remove(url *common.URL) error {
	m.removeCalled = true
	return m.removeErr
}

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

// ========== Metadata Lifecycle Tests ==========

// mockMetadataReportForGC is a lightweight mock for testing GC logic
type mockMetadataReportForGC struct {
	revisions []report.AppRevision
	deleted   []string // tracks deleted revisions
	published int      // tracks publish calls
	reportURL *common.URL
}

func (m *mockMetadataReportForGC) GetAppMetadata(string, string) (*info.MetadataInfo, error) {
	return nil, nil
}
func (m *mockMetadataReportForGC) PublishAppMetadata(string, string, *info.MetadataInfo) error {
	m.published++
	return nil
}
func (m *mockMetadataReportForGC) RegisterServiceAppMapping(string, string, string) error {
	return nil
}
func (m *mockMetadataReportForGC) GetServiceAppMapping(string, string, mapping.MappingListener) (*gxset.HashSet, error) {
	return gxset.NewSet(), nil
}
func (m *mockMetadataReportForGC) RemoveServiceAppMappingListener(string, string) error {
	return nil
}
func (m *mockMetadataReportForGC) UnPublishAppMetadata(application, revision string) error {
	m.deleted = append(m.deleted, revision)
	return nil
}
func (m *mockMetadataReportForGC) ListAppRevisions(application string) ([]report.AppRevision, error) {
	return m.revisions, nil
}
func (m *mockMetadataReportForGC) URL() *common.URL {
	if m.reportURL != nil {
		return m.reportURL
	}
	u, _ := common.NewURL("mock://127.0.0.1:8848")
	return u
}

// mockServiceDiscoveryWithInstances returns configurable instances for GC tests
type mockServiceDiscoveryWithInstances struct {
	*mockServiceDiscovery
	instances []registry.ServiceInstance
}

func (m *mockServiceDiscoveryWithInstances) GetInstances(name string) []registry.ServiceInstance {
	return m.instances
}

func TestServiceDiscoveryRegistry_DoGarbageCollect_CleansStaleRevisions(t *testing.T) {
	mockReport := &mockMetadataReportForGC{
		revisions: []report.AppRevision{
			{Revision: "stale-rev1", ModifyTime: time.Now().Add(-96 * time.Hour).UnixMilli()}, // 4 days old
			{Revision: "stale-rev2", ModifyTime: time.Now().Add(-48 * time.Hour).UnixMilli()}, // 2 days old, but < 72h window
			{Revision: "fresh-rev", ModifyTime: time.Now().UnixMilli()},                       // current
		},
		reportURL: common.NewURLWithOptions(
			common.WithParamsValue(constant.MetadataGCWindowKey, "3"),
		),
	}

	sd := &mockServiceDiscoveryWithInstances{
		mockServiceDiscovery: &mockServiceDiscovery{},
		instances: []registry.ServiceInstance{
			&registry.DefaultServiceInstance{
				Metadata: map[string]string{
					constant.ExportedServicesRevisionPropertyName: "fresh-rev",
				},
			},
		},
	}

	regID := fmt.Sprintf("gc-reg-%d", time.Now().UnixNano())
	url := common.NewURLWithOptions(
		common.WithParamsValue(constant.RegistryIdKey, regID),
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
	)

	reg := &serviceDiscoveryRegistry{
		url:              url,
		serviceDiscovery: sd,
		metadataReport:   mockReport,
	}

	// Set up metadata info so doGarbageCollect can find the app name
	serviceURL, _ := common.NewURL("dubbo://127.0.0.1:20880/org.test.GCStale",
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
	)
	metadata.AddService(regID, serviceURL)

	reg.doGarbageCollect()

	// Only stale-rev1 should be deleted (stale-rev2 is within 72h window, fresh-rev is alive)
	assert.Equal(t, []string{"stale-rev1"}, mockReport.deleted)
}

func TestServiceDiscoveryRegistry_DoGarbageCollect_SkipsWhenNoStaleRevisions(t *testing.T) {
	mockReport := &mockMetadataReportForGC{
		revisions: []report.AppRevision{
			{Revision: "fresh-rev", ModifyTime: time.Now().UnixMilli()},
		},
		reportURL: common.NewURLWithOptions(
			common.WithParamsValue(constant.MetadataGCWindowKey, "3"),
		),
	}

	sd := &mockServiceDiscoveryWithInstances{
		mockServiceDiscovery: &mockServiceDiscovery{},
		instances:            nil,
	}

	regID := fmt.Sprintf("gc-nostale-reg-%d", time.Now().UnixNano())
	url := common.NewURLWithOptions(
		common.WithParamsValue(constant.RegistryIdKey, regID),
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
	)

	reg := &serviceDiscoveryRegistry{
		url:              url,
		serviceDiscovery: sd,
		metadataReport:   mockReport,
	}

	serviceURL, _ := common.NewURL("dubbo://127.0.0.1:20880/org.test.GCNoStale",
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
	)
	metadata.AddService(regID, serviceURL)

	reg.doGarbageCollect()

	assert.Empty(t, mockReport.deleted)
}

func TestServiceDiscoveryRegistry_DoGarbageCollect_SkipsReferencedStaleRevision(t *testing.T) {
	staleTime := time.Now().Add(-96 * time.Hour).UnixMilli()
	mockReport := &mockMetadataReportForGC{
		revisions: []report.AppRevision{
			{Revision: "stale-but-alive", ModifyTime: staleTime},
		},
		reportURL: common.NewURLWithOptions(
			common.WithParamsValue(constant.MetadataGCWindowKey, "3"),
		),
	}

	sd := &mockServiceDiscoveryWithInstances{
		mockServiceDiscovery: &mockServiceDiscovery{},
		instances: []registry.ServiceInstance{
			&registry.DefaultServiceInstance{
				Metadata: map[string]string{
					constant.ExportedServicesRevisionPropertyName: "stale-but-alive",
				},
			},
		},
	}

	regID := fmt.Sprintf("gc-ref-reg-%d", time.Now().UnixNano())
	url := common.NewURLWithOptions(
		common.WithParamsValue(constant.RegistryIdKey, regID),
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
	)

	reg := &serviceDiscoveryRegistry{
		url:              url,
		serviceDiscovery: sd,
		metadataReport:   mockReport,
	}

	serviceURL, _ := common.NewURL("dubbo://127.0.0.1:20880/org.test.GCRefStale",
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
	)
	metadata.AddService(regID, serviceURL)

	reg.doGarbageCollect()

	// stale-but-alive is referenced by a live instance, should NOT be deleted
	assert.Empty(t, mockReport.deleted)
}

func TestServiceDiscoveryRegistry_DoRenewAppMetadata(t *testing.T) {
	mockReport := &mockMetadataReportForGC{}

	regID := fmt.Sprintf("renew-reg-%d", time.Now().UnixNano())
	url := common.NewURLWithOptions(
		common.WithParamsValue(constant.RegistryIdKey, regID),
	)

	reg := &serviceDiscoveryRegistry{
		url:            url,
		metadataReport: mockReport,
	}

	// Set up metadata info via AddService (which populates registryMetadataInfo)
	serviceURL, _ := common.NewURL("dubbo://127.0.0.1:20880/org.test.RenewAppMetadata",
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)
	metadata.AddService(regID, serviceURL)
	metaInfo := metadata.GetMetadataInfo(regID)
	require.NotNil(t, metaInfo)
	metaInfo.Revision = "abc123"

	reg.doRenewAppMetadata()
	assert.Equal(t, 1, mockReport.published)
}

func TestServiceDiscoveryRegistry_Destroy_StopsTimers(t *testing.T) {
	url := common.NewURLWithOptions()
	sd := &mockServiceDiscovery{}

	reg := &serviceDiscoveryRegistry{
		url:              url,
		serviceDiscovery: sd,
	}

	// Simulate running timer
	reg.renewAppMetadataTimer = time.AfterFunc(1*time.Hour, func() {})

	reg.Destroy()

	assert.Nil(t, reg.renewAppMetadataTimer)
}

func TestServiceDiscoveryRegistry_CalculateRenewAppMetadataDelay(t *testing.T) {
	url := common.NewURLWithOptions()
	reg := &serviceDiscoveryRegistry{url: url}

	delay := reg.calculateRenewAppMetadataDelay()

	// Delay should be between ~14h and ~28h (next day 2AM + 0~4h random offset)
	// Minimum: if now is 23:59, next 2AM is ~2h + 0 = 2h
	// Maximum: if now is 00:01, next 2AM is ~26h + 4h = 30h
	assert.GreaterOrEqual(t, delay, 1*time.Hour, "delay should be at least 1 hour")
	assert.LessOrEqual(t, delay, 32*time.Hour, "delay should be at most 32 hours")
}

// TestServiceDiscoveryRegistry_DoGarbageCollect_SkipsSpecialRevisions verifies that
// special revisions ("0", "N/A", "") and entries with ModifyTime==0 are never GC'd,
// while genuinely stale revisions are cleaned up.
func TestServiceDiscoveryRegistry_DoGarbageCollect_SkipsSpecialRevisions(t *testing.T) {
	staleTime := time.Now().Add(-240 * time.Hour).UnixMilli() // 10 days old
	mockReport := &mockMetadataReportForGC{
		revisions: []report.AppRevision{
			{Revision: "0", ModifyTime: staleTime},               // special — skip
			{Revision: "N/A", ModifyTime: staleTime},             // special — skip
			{Revision: "", ModifyTime: staleTime},                // special — skip
			{Revision: "no-timestamp", ModifyTime: 0},            // ModifyTime==0 — skip
			{Revision: "genuinely-stale", ModifyTime: staleTime}, // stale, no alive ref — delete
		},
		reportURL: common.NewURLWithOptions(
			common.WithParamsValue(constant.MetadataGCWindowKey, "5"),
		),
	}

	sd := &mockServiceDiscoveryWithInstances{
		mockServiceDiscovery: &mockServiceDiscovery{},
		instances:            nil, // no alive instances
	}

	regID := fmt.Sprintf("gc-special-reg-%d", time.Now().UnixNano())
	url := common.NewURLWithOptions(
		common.WithParamsValue(constant.RegistryIdKey, regID),
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
	)

	reg := &serviceDiscoveryRegistry{
		url:              url,
		serviceDiscovery: sd,
		metadataReport:   mockReport,
	}

	serviceURL, _ := common.NewURL("dubbo://127.0.0.1:20880/org.test.GCSpecial",
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
	)
	metadata.AddService(regID, serviceURL)

	reg.doGarbageCollect()

	// Only "genuinely-stale" should be deleted; all special/zero-timestamp revisions skipped
	assert.Equal(t, []string{"genuinely-stale"}, mockReport.deleted)
}

// TestServiceDiscoveryRegistry_DoGarbageCollect_MixedAliveAndStale verifies the core
// GC branch: stale revisions with no alive reference are cleaned, while stale revisions
// still referenced by alive instances are preserved.
func TestServiceDiscoveryRegistry_DoGarbageCollect_MixedAliveAndStale(t *testing.T) {
	staleTime := time.Now().Add(-240 * time.Hour).UnixMilli() // 10 days old, well beyond 5-day window

	mockReport := &mockMetadataReportForGC{
		revisions: []report.AppRevision{
			{Revision: "stale-unreferenced", ModifyTime: staleTime},     // stale, no ref — delete
			{Revision: "stale-but-referenced", ModifyTime: staleTime},   // stale, but alive ref — keep
			{Revision: "fresh-rev", ModifyTime: time.Now().UnixMilli()}, // fresh — keep
		},
		reportURL: common.NewURLWithOptions(
			common.WithParamsValue(constant.MetadataGCWindowKey, "5"),
		),
	}

	sd := &mockServiceDiscoveryWithInstances{
		mockServiceDiscovery: &mockServiceDiscovery{},
		instances: []registry.ServiceInstance{
			&registry.DefaultServiceInstance{
				Metadata: map[string]string{
					constant.ExportedServicesRevisionPropertyName: "stale-but-referenced",
				},
			},
			&registry.DefaultServiceInstance{
				Metadata: map[string]string{
					constant.ExportedServicesRevisionPropertyName: "fresh-rev",
				},
			},
		},
	}

	regID := fmt.Sprintf("gc-mixed-reg-%d", time.Now().UnixNano())
	url := common.NewURLWithOptions(
		common.WithParamsValue(constant.RegistryIdKey, regID),
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
	)

	reg := &serviceDiscoveryRegistry{
		url:              url,
		serviceDiscovery: sd,
		metadataReport:   mockReport,
	}

	serviceURL, _ := common.NewURL("dubbo://127.0.0.1:20880/org.test.GCMixed",
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
	)
	metadata.AddService(regID, serviceURL)

	reg.doGarbageCollect()

	// Only "stale-unreferenced" should be deleted
	assert.Equal(t, []string{"stale-unreferenced"}, mockReport.deleted)
}

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
	require.NoError(t, err)

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
	require.NoError(t, err)

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
	require.NoError(t, err)

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
	require.Error(t, err)

	// failed instance should remain, successful one removed
	assert.Len(t, sdReg.instances, 1)
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
	require.NoError(t, err)

	sdReg, ok := reg.(*serviceDiscoveryRegistry)
	assert.True(t, ok)

	sdReg.instances = []registry.ServiceInstance{}

	err = sdReg.UnRegisterService()
	require.NoError(t, err)

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
	require.NoError(t, err)

	sdReg, ok := reg.(*serviceDiscoveryRegistry)
	assert.True(t, ok)

	// prepare initial instances
	for i := range 5 {
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
	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()
		_ = sdReg.UnRegisterService()
	})

	// goroutine 2: simulate concurrent register / instance append
	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()

		for i := range 3 {
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
	})

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
