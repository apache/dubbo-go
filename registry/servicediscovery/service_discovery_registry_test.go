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
	"testing"
)

import (
	"github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/page"
	"github.com/stretchr/testify/assert"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/observer"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/metadata/mapping"
	"github.com/apache/dubbo-go/metadata/service"
	"github.com/apache/dubbo-go/registry"
)

var (
	serviceInterface = "org.apache.dubbo.metadata.MetadataService"
	group            = "dubbo-provider"
	version          = "1.0.0"
)

func TestServiceDiscoveryRegistry_Register(t *testing.T) {
	config.GetApplicationConfig().MetadataType = "mock"
	extension.SetMetadataService("mock", func() (service service.MetadataService, err error) {
		service = &mockMetadataService{}
		return
	})

	extension.SetServiceDiscovery("mock", func(name string) (discovery registry.ServiceDiscovery, err error) {
		return &mockServiceDiscovery{}, nil
	})

	extension.SetGlobalServiceNameMapping(func() mapping.ServiceNameMapping {
		return &mockServiceNameMapping{}
	})

	extension.SetEventDispatcher("mock", func() observer.EventDispatcher {
		return &mockEventDispatcher{}
	})
	extension.SetAndInitGlobalDispatcher("mock")

	config.GetBaseConfig().ServiceDiscoveries["mock"] = &config.ServiceDiscoveryConfig{
		Protocol: "mock",
	}
	registryURL, _ := common.NewURL("service-discovery://localhost:12345",
		common.WithParamsValue("service_discovery", "mock"),
		common.WithParamsValue("subscribed-services", "a, b , c,d,e ,"))
	url, _ := common.NewURL("dubbo://192.168.0.102:20880/" + serviceInterface +
		"?&application=" + group +
		"&interface=" + serviceInterface +
		"&group=" + group +
		"&version=" + version +
		"&service_discovery=mock" +
		"&methods=getAllServiceKeys,getServiceRestMetadata,getExportedURLs,getAllExportedURLs" +
		"&side=provider")
	registry, err := newServiceDiscoveryRegistry(&registryURL)
	assert.Nil(t, err)
	assert.NotNil(t, registry)
	registry.Register(url)
}

type mockEventDispatcher struct {
}

func (m *mockEventDispatcher) AddEventListener(listener observer.EventListener) {

}

func (m *mockEventDispatcher) AddEventListeners(listenersSlice []observer.EventListener) {

}

func (m *mockEventDispatcher) RemoveEventListener(listener observer.EventListener) {
	panic("implement me")
}

func (m *mockEventDispatcher) RemoveEventListeners(listenersSlice []observer.EventListener) {
	panic("implement me")
}

func (m *mockEventDispatcher) GetAllEventListeners() []observer.EventListener {
	return []observer.EventListener{}
}

func (m *mockEventDispatcher) RemoveAllEventListeners() {
	panic("implement me")
}

func (m *mockEventDispatcher) Dispatch(event observer.Event) {
}

type mockServiceNameMapping struct {
}

func (m *mockServiceNameMapping) Map(serviceInterface string, group string, version string, protocol string) error {
	return nil
}

func (m *mockServiceNameMapping) Get(serviceInterface string, group string, version string, protocol string) (*gxset.HashSet, error) {
	panic("implement me")
}

type mockServiceDiscovery struct {
}

func (m *mockServiceDiscovery) String() string {
	panic("implement me")
}

func (m *mockServiceDiscovery) Destroy() error {
	panic("implement me")
}

func (m *mockServiceDiscovery) Register(instance registry.ServiceInstance) error {
	return nil
}

func (m *mockServiceDiscovery) Update(instance registry.ServiceInstance) error {
	panic("implement me")
}

func (m *mockServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	panic("implement me")
}

func (m *mockServiceDiscovery) GetDefaultPageSize() int {
	panic("implement me")
}

func (m *mockServiceDiscovery) GetServices() *gxset.HashSet {
	panic("implement me")
}

func (m *mockServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	panic("implement me")
}

func (m *mockServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	panic("implement me")
}

func (m *mockServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	panic("implement me")
}

func (m *mockServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	panic("implement me")
}

func (m *mockServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {
	panic("implement me")
}

func (m *mockServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	panic("implement me")
}

func (m *mockServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	panic("implement me")
}

func (m *mockServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	panic("implement me")
}

type mockMetadataService struct {
}

func (m *mockMetadataService) Reference() string {
	panic("implement me")
}

func (m *mockMetadataService) ServiceName() (string, error) {
	panic("implement me")
}

func (m *mockMetadataService) ExportURL(url common.URL) (bool, error) {
	return true, nil
}

func (m *mockMetadataService) UnexportURL(url common.URL) error {
	panic("implement me")
}

func (m *mockMetadataService) SubscribeURL(url common.URL) (bool, error) {
	panic("implement me")
}

func (m *mockMetadataService) UnsubscribeURL(url common.URL) error {
	panic("implement me")
}

func (m *mockMetadataService) PublishServiceDefinition(url common.URL) error {
	return nil
}

func (m *mockMetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]interface{}, error) {
	panic("implement me")
}

func (m *mockMetadataService) MethodMapper() map[string]string {
	panic("implement me")
}

func (m *mockMetadataService) GetSubscribedURLs() ([]common.URL, error) {
	panic("implement me")
}

func (m *mockMetadataService) GetServiceDefinition(interfaceName string, group string, version string) (string, error) {
	panic("implement me")
}

func (m *mockMetadataService) GetServiceDefinitionByServiceKey(serviceKey string) (string, error) {
	panic("implement me")
}

func (m *mockMetadataService) RefreshMetadata(exportedRevision string, subscribedRevision string) (bool, error) {
	panic("implement me")
}

func (m *mockMetadataService) Version() (string, error) {
	panic("implement me")
}
