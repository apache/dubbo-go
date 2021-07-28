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

package config

import (
	"context"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

import (
	"github.com/Workiva/go-datastructures/slice/skip"
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/hash/page"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster_impl"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/common/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

const (
	mockConsumerConfigPath = "./testdata/consumer_config.yml"
	mockProviderConfigPath = "./testdata/provider_config.yml"
)

func TestConfigLoader(t *testing.T) {
	conPath, err := filepath.Abs(mockConsumerConfigPath)
	assert.NoError(t, err)
	proPath, err := filepath.Abs(mockProviderConfigPath)
	assert.NoError(t, err)

	assert.Nil(t, consumerConfig)
	assert.Equal(t, ConsumerConfig{}, GetConsumerConfig())
	assert.Nil(t, providerConfig)
	assert.Equal(t, ProviderConfig{}, GetProviderConfig())

	err = ConsumerInit(conPath)
	assert.NoError(t, err)
	err = ProviderInit(proPath)
	assert.NoError(t, err)

	assert.NotNil(t, consumerConfig)
	assert.NotEqual(t, ConsumerConfig{}, GetConsumerConfig())
	assert.NotNil(t, providerConfig)
	assert.NotEqual(t, ProviderConfig{}, GetProviderConfig())
	assert.Equal(t, "soa.com.ikurento.user.UserProvider", GetConsumerConfig().References["UserProvider"].Params["serviceid"])
}

func TestLoad(t *testing.T) {
	extension.SetFilter(constant.GracefulShutdownConsumerFilterKey, func() filter.Filter {
		return &mockGracefulShutdownFilter{}
	})
	extension.SetFilter(constant.GracefulShutdownProviderFilterKey, func() filter.Filter {
		return &mockGracefulShutdownFilter{}
	})

	doInitConsumer()
	doInitProvider()

	ms := &MockService{}
	SetConsumerService(ms)
	SetProviderService(ms)

	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)
	extension.SetProxyFactory("default", proxy_factory.NewDefaultProxyFactory)
	GetApplicationConfig().MetadataType = "mock"
	var mm *mockMetadataService
	extension.SetLocalMetadataService("mock", func() (metadataService service.MetadataService, err error) {
		if mm == nil {
			mm = &mockMetadataService{
				exportedServiceURLs: new(sync.Map),
				lock:                new(sync.RWMutex),
			}
		}
		return mm, nil
	})
	Load()

	assert.Equal(t, ms, GetRPCService(ms.Reference()))
	ms2 := &struct {
		MockService
	}{}
	RPCService(ms2)
	assert.NotEqual(t, ms2, GetRPCService(ms2.Reference()))

	conServices = map[string]common.RPCService{}
	proServices = map[string]common.RPCService{}
	err := common.ServiceMap.UnRegister("com.MockService", "mock",
		common.ServiceKey("com.MockService", "huadong_idc", "1.0.0"))
	assert.Nil(t, err)
	consumerConfig = nil
	providerConfig = nil
}

func TestLoadWithSingleReg(t *testing.T) {
	doInitConsumerWithSingleRegistry()
	mockInitProviderWithSingleRegistry()

	ms := &MockService{}
	SetConsumerService(ms)
	SetProviderService(ms)

	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)
	extension.SetProxyFactory("default", proxy_factory.NewDefaultProxyFactory)
	var mm *mockMetadataService
	GetApplicationConfig().MetadataType = "mock"
	extension.SetLocalMetadataService("mock", func() (metadataService service.MetadataService, err error) {
		if mm == nil {
			mm = &mockMetadataService{
				exportedServiceURLs: new(sync.Map),
				lock:                new(sync.RWMutex),
			}
		}
		return mm, nil
	})
	Load()

	assert.Equal(t, ms, GetRPCService(ms.Reference()))
	ms2 := &struct {
		MockService
	}{}
	RPCService(ms2)
	assert.NotEqual(t, ms2, GetRPCService(ms2.Reference()))

	conServices = map[string]common.RPCService{}
	proServices = map[string]common.RPCService{}
	common.ServiceMap.UnRegister("com.MockService", "mock", common.ServiceKey("com.MockService", "huadong_idc", "1.0.0"))
	consumerConfig = nil
	providerConfig = nil
}

func TestWithNoRegLoad(t *testing.T) {
	doInitConsumer()
	doInitProvider()
	providerConfig.Services["MockService"].Registry = ""
	consumerConfig.References["MockService"].Registry = ""
	ms := &MockService{}
	SetConsumerService(ms)
	SetProviderService(ms)

	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)
	extension.SetProxyFactory("default", proxy_factory.NewDefaultProxyFactory)
	var mm *mockMetadataService
	GetApplicationConfig().MetadataType = "mock"
	extension.SetLocalMetadataService("mock", func() (metadataService service.MetadataService, err error) {
		if mm == nil {
			mm = &mockMetadataService{
				exportedServiceURLs: new(sync.Map),
				lock:                new(sync.RWMutex),
			}
		}
		return mm, nil
	})
	Load()

	assert.Equal(t, ms, GetRPCService(ms.Reference()))
	ms2 := &struct {
		MockService
	}{}
	RPCService(ms2)
	assert.NotEqual(t, ms2, GetRPCService(ms2.Reference()))

	conServices = map[string]common.RPCService{}
	proServices = map[string]common.RPCService{}
	err := common.ServiceMap.UnRegister("com.MockService", "mock",
		common.ServiceKey("com.MockService", "huadong_idc", "1.0.0"))
	assert.Nil(t, err)
	common.ServiceMap.UnRegister("com.MockService", "mock", common.ServiceKey("com.MockService", "huadong_idc", "1.0.0"))
	consumerConfig = nil
	providerConfig = nil
}

func TestSetDefaultValue(t *testing.T) {
	proConfig := &ProviderConfig{Registries: make(map[string]*RegistryConfig), Protocols: make(map[string]*ProtocolConfig)}
	assert.Nil(t, proConfig.ApplicationConfig)
	setDefaultValue(proConfig)
	assert.Equal(t, proConfig.Protocols["dubbo"].Name, "dubbo")
	assert.Equal(t, proConfig.Protocols["dubbo"].Port, "20000")
	assert.NotNil(t, proConfig.ApplicationConfig)

	conConfig := &ConsumerConfig{Registries: make(map[string]*RegistryConfig)}
	assert.Nil(t, conConfig.ApplicationConfig)
	setDefaultValue(conConfig)
	assert.NotNil(t, conConfig.ApplicationConfig)

}
func TestConfigLoaderWithConfigCenter(t *testing.T) {
	extension.SetConfigCenterFactory("mock", func() config_center.DynamicConfigurationFactory {
		return &config_center.MockDynamicConfigurationFactory{}
	})

	conPath, err := filepath.Abs("./testdata/consumer_config_with_configcenter.yml")
	assert.NoError(t, err)
	proPath, err := filepath.Abs(mockProviderConfigPath)
	assert.NoError(t, err)

	assert.Nil(t, consumerConfig)
	assert.Equal(t, ConsumerConfig{}, GetConsumerConfig())
	assert.Nil(t, providerConfig)
	assert.Equal(t, ProviderConfig{}, GetProviderConfig())

	err = ConsumerInit(conPath)
	assert.NoError(t, err)
	err = configCenterRefreshConsumer()
	assert.NoError(t, err)
	err = ProviderInit(proPath)
	assert.NoError(t, err)
	err = configCenterRefreshProvider()
	assert.NoError(t, err)

	assert.NotNil(t, consumerConfig)
	assert.NotEqual(t, ConsumerConfig{}, GetConsumerConfig())
	assert.NotNil(t, providerConfig)
	assert.NotEqual(t, ProviderConfig{}, GetProviderConfig())

	assert.Equal(t, "BDTService", consumerConfig.ApplicationConfig.Name)
	assert.Equal(t, "127.0.0.1:2181", consumerConfig.Registries["hangzhouzk"].Address)
}

func TestConfigLoaderWithConfigCenterSingleRegistry(t *testing.T) {
	consumerConfig = nil
	providerConfig = nil
	config.NewEnvInstance()
	extension.SetConfigCenterFactory("mock", func() config_center.DynamicConfigurationFactory {
		return &config_center.MockDynamicConfigurationFactory{Content: `
	dubbo.consumer.request_timeout=5s
	dubbo.consumer.connect_timeout=5s
	dubbo.application.organization=ikurento.com
	dubbo.application.name=BDTService
	dubbo.application.module=dubbogo user-info server
	dubbo.application.version=0.0.1
	dubbo.application.owner=ZX
	dubbo.application.environment=dev
	dubbo.registry.address=mock://127.0.0.1:2182
	dubbo.service.com.ikurento.user.UserProvider.protocol=dubbo
	dubbo.service.com.ikurento.user.UserProvider.interface=com.ikurento.user.UserProvider
	dubbo.service.com.ikurento.user.UserProvider.loadbalance=random
	dubbo.service.com.ikurento.user.UserProvider.warmup=100
	dubbo.service.com.ikurento.user.UserProvider.cluster=failover
	dubbo.protocols.jsonrpc1.name=jsonrpc
	dubbo.protocols.jsonrpc1.ip=127.0.0.1
	dubbo.protocols.jsonrpc1.port=20001
`}
	})

	conPath, err := filepath.Abs("./testdata/consumer_config_with_configcenter.yml")
	assert.NoError(t, err)
	proPath, err := filepath.Abs(mockProviderConfigPath)
	assert.NoError(t, err)

	assert.Nil(t, consumerConfig)
	assert.Equal(t, ConsumerConfig{}, GetConsumerConfig())
	assert.Nil(t, providerConfig)
	assert.Equal(t, ProviderConfig{}, GetProviderConfig())

	err = ConsumerInit(conPath)
	assert.NoError(t, err)
	checkApplicationName(consumerConfig.ApplicationConfig)
	err = configCenterRefreshConsumer()
	checkRegistries(consumerConfig.Registries, consumerConfig.Registry)
	assert.NoError(t, err)
	err = ProviderInit(proPath)
	assert.NoError(t, err)
	checkApplicationName(providerConfig.ApplicationConfig)
	err = configCenterRefreshProvider()
	checkRegistries(providerConfig.Registries, providerConfig.Registry)
	assert.NoError(t, err)

	assert.NotNil(t, consumerConfig)
	assert.NotEqual(t, ConsumerConfig{}, GetConsumerConfig())
	assert.NotNil(t, providerConfig)
	assert.NotEqual(t, ProviderConfig{}, GetProviderConfig())

	assert.Equal(t, "BDTService", consumerConfig.ApplicationConfig.Name)
	assert.Equal(t, "mock://127.0.0.1:2182", consumerConfig.Registries[constant.DEFAULT_KEY].Address)
}

func TestGetBaseConfig(t *testing.T) {
	bc := GetBaseConfig()
	assert.NotNil(t, bc)
	_, found := bc.GetRemoteConfig("mock")
	assert.False(t, found)
}

// mockInitProviderWithSingleRegistry will init a mocked providerConfig
func mockInitProviderWithSingleRegistry() {
	providerConfig = &ProviderConfig{
		BaseConfig: BaseConfig{
			ApplicationConfig: &ApplicationConfig{
				Organization: "dubbo_org",
				Name:         "dubbo",
				Module:       "module",
				Version:      "1.0.0",
				Owner:        "dubbo",
				Environment:  "test",
			},
		},

		Registry: &RegistryConfig{
			Address:  "mock://127.0.0.1:2181",
			Username: "user1",
			Password: "pwd1",
		},
		Registries: map[string]*RegistryConfig{},

		Services: map[string]*ServiceConfig{
			"MockService": {
				InterfaceName: "com.MockService",
				Protocol:      "mock",
				Cluster:       "failover",
				Loadbalance:   "random",
				Retries:       "3",
				Group:         "huadong_idc",
				Version:       "1.0.0",
				Methods: []*MethodConfig{
					{
						Name:        "GetUser",
						Retries:     "2",
						LoadBalance: "random",
						Weight:      200,
					},
					{
						Name:        "GetUser1",
						Retries:     "2",
						LoadBalance: "random",
						Weight:      200,
					},
				},
				exported: new(atomic.Bool),
			},
		},
		Protocols: map[string]*ProtocolConfig{
			"mock": {
				Name: "mock",
				Ip:   "127.0.0.1",
				Port: "20000",
			},
		},
	}
}

type mockMetadataService struct {
	exportedServiceURLs *sync.Map
	lock                *sync.RWMutex
}

func (m *mockMetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]*common.URL, error) {
	panic("implement me")
}

func (m *mockMetadataService) GetMetadataInfo(revision string) (*common.MetadataInfo, error) {
	panic("implement me")
}

func (m *mockMetadataService) GetExportedServiceURLs() []*common.URL {
	panic("implement me")
}

func (m *mockMetadataService) GetMetadataServiceURL() *common.URL {
	panic("implement me")
}

func (m *mockMetadataService) SetMetadataServiceURL(url *common.URL) {
	panic("implement me")
}

func (m *mockMetadataService) Reference() string {
	panic("implement me")
}

func (m *mockMetadataService) ServiceName() (string, error) {
	panic("implement me")
}

func (m *mockMetadataService) ExportURL(url *common.URL) (bool, error) {
	return m.addURL(m.exportedServiceURLs, url), nil
}

func (m *mockMetadataService) UnexportURL(*common.URL) error {
	panic("implement me")
}

func (m *mockMetadataService) SubscribeURL(*common.URL) (bool, error) {
	panic("implement me")
}

func (m *mockMetadataService) UnsubscribeURL(*common.URL) error {
	panic("implement me")
}

func (m *mockMetadataService) PublishServiceDefinition(*common.URL) error {
	return nil
}

func (m *mockMetadataService) MethodMapper() map[string]string {
	panic("implement me")
}

func (m *mockMetadataService) GetSubscribedURLs() ([]*common.URL, error) {
	panic("implement me")
}

func (m *mockMetadataService) GetServiceDefinition(string, string, string) (string, error) {
	panic("implement me")
}

func (m *mockMetadataService) GetServiceDefinitionByServiceKey(string) (string, error) {
	panic("implement me")
}

func (m *mockMetadataService) RefreshMetadata(string, string) (bool, error) {
	panic("implement me")
}

func (m *mockMetadataService) Version() (string, error) {
	panic("implement me")
}

func (m *mockMetadataService) addURL(targetMap *sync.Map, url *common.URL) bool {
	var (
		urlSet interface{}
		loaded bool
	)
	logger.Debug(url.ServiceKey())
	if urlSet, loaded = targetMap.LoadOrStore(url.ServiceKey(), skip.New(uint64(0))); loaded {
		m.lock.RLock()
		wantedUrl := urlSet.(*skip.SkipList).Get(url)
		if len(wantedUrl) > 0 && wantedUrl[0] != nil {
			m.lock.RUnlock()
			return false
		}
		m.lock.RUnlock()
	}
	m.lock.Lock()
	// double chk
	wantedUrl := urlSet.(*skip.SkipList).Get(url)
	if len(wantedUrl) > 0 && wantedUrl[0] != nil {
		m.lock.Unlock()
		return false
	}
	urlSet.(*skip.SkipList).Insert(url)
	m.lock.Unlock()
	return true
}

func (m *mockMetadataService) getAllService(services *sync.Map) []*common.URL {
	// using skip list to dedup and sorting
	var res []*common.URL
	services.Range(func(key, value interface{}) bool {
		urls := value.(*skip.SkipList)
		for i := uint64(0); i < urls.Len(); i++ {
			url := urls.ByPosition(i).(*common.URL)
			if url.GetParam(constant.INTERFACE_KEY, url.Path) != constant.METADATA_SERVICE_NAME {
				res = append(res, url)
			}
		}
		return true
	})
	sort.Sort(common.URLSlice(res))
	return res
}

type mockServiceDiscoveryRegistry struct{}

func (mr *mockServiceDiscoveryRegistry) GetURL() *common.URL {
	panic("implement me")
}

func (mr *mockServiceDiscoveryRegistry) IsAvailable() bool {
	panic("implement me")
}

func (mr *mockServiceDiscoveryRegistry) Destroy() {
	panic("implement me")
}

func (mr *mockServiceDiscoveryRegistry) Register(*common.URL) error {
	panic("implement me")
}

func (mr *mockServiceDiscoveryRegistry) UnRegister(*common.URL) error {
	panic("implement me")
}

func (mr *mockServiceDiscoveryRegistry) Subscribe(*common.URL, registry.NotifyListener) error {
	panic("implement me")
}

func (mr *mockServiceDiscoveryRegistry) UnSubscribe(*common.URL, registry.NotifyListener) error {
	panic("implement me")
}

func (mr *mockServiceDiscoveryRegistry) GetServiceDiscovery() registry.ServiceDiscovery {
	return &mockServiceDiscovery{}
}

type mockServiceDiscovery struct{}

func (m *mockServiceDiscovery) String() string {
	panic("implement me")
}

func (m *mockServiceDiscovery) Destroy() error {
	panic("implement me")
}

func (m *mockServiceDiscovery) Register(registry.ServiceInstance) error {
	return nil
}

func (m *mockServiceDiscovery) Update(registry.ServiceInstance) error {
	panic("implement me")
}

func (m *mockServiceDiscovery) Unregister(registry.ServiceInstance) error {
	panic("implement me")
}

func (m *mockServiceDiscovery) GetDefaultPageSize() int {
	panic("implement me")
}

func (m *mockServiceDiscovery) GetServices() *gxset.HashSet {
	panic("implement me")
}

func (m *mockServiceDiscovery) GetInstances(string) []registry.ServiceInstance {
	panic("implement me")
}

func (m *mockServiceDiscovery) GetInstancesByPage(string, int, int) gxpage.Pager {
	panic("implement me")
}

func (m *mockServiceDiscovery) GetHealthyInstancesByPage(string, int, int, bool) gxpage.Pager {
	panic("implement me")
}

func (m *mockServiceDiscovery) GetRequestInstances([]string, int, int) map[string]gxpage.Pager {
	panic("implement me")
}

func (m *mockServiceDiscovery) AddListener(registry.ServiceInstancesChangedListener) error {
	panic("implement me")
}

func (m *mockServiceDiscovery) DispatchEventByServiceName(string) error {
	panic("implement me")
}

func (m *mockServiceDiscovery) DispatchEventForInstances(string, []registry.ServiceInstance) error {
	panic("implement me")
}

func (m *mockServiceDiscovery) DispatchEvent(*registry.ServiceInstancesChangedEvent) error {
	panic("implement me")
}

func ConvertURLArrToIntfArr(urls []*common.URL) []interface{} {
	if len(urls) == 0 {
		return []interface{}{}
	}

	res := make([]interface{}, 0, len(urls))
	for _, u := range urls {
		res = append(res, u.String())
	}
	return res
}

type mockGracefulShutdownFilter struct{}

func (f *mockGracefulShutdownFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	panic("implement me")
}

func (f *mockGracefulShutdownFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	panic("implement me")
}

func (f *mockGracefulShutdownFilter) Set(name string, config interface{}) {
	return
}
