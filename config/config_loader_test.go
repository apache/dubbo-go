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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"testing"
)

import "github.com/stretchr/testify/assert"

import "dubbo.apache.org/dubbo-go/v3/config/testdata/config/service"

func init() {
	SetProviderService(new(service.OrderService))
	SetProviderService(new(service.HelloService))
}

const (
	configPath = "./testdata/application.yaml"
)

func TestLoad(t *testing.T) {
	Load(WithPath(configPath))

	t.Run("application", func(t *testing.T) {
		application := rootConfig.Application

		assert.Equal(t, application.Organization, "dubbo.io")
		assert.Equal(t, application.Name, "dubbo-go")
		assert.Equal(t, application.Module, "local")
		assert.Equal(t, application.Version, "1.0.0")
		assert.Equal(t, application.Owner, "zhaoyunxing")
		assert.Equal(t, application.Environment, "dev")
		assert.Equal(t, application.MetadataType, "local")
	})

	t.Run("registries", func(t *testing.T) {
		registries := rootConfig.Registries

		assert.Equal(t, 2, len(registries))
		//address= nacos://127.0.0.1:8848 Translate Registry Address
		assert.Equal(t, "nacos", registries["nacos"].Protocol)
		assert.Equal(t, "10s", registries["zk"].Timeout)
	})

	//config-center
	t.Run("config-center", func(t *testing.T) {
		conf := rootConfig.ConfigCenter

		assert.Equal(t, "nacos", conf.Protocol)
	})

}

//TestLoadConfigCenter test key  config_center、config-center 、configCenter
func TestLoadConfigCenter(t *testing.T) {

	t.Run("config-center", func(t *testing.T) {
		Load(WithPath("./testdata/config/center/conf-application.yaml"))
		conf := rootConfig.ConfigCenter
		assert.Equal(t, "nacos", conf.Protocol)
		assert.Equal(t, "10s", conf.Timeout)
		assert.Equal(t, "./logs", conf.LogDir)
	})
}

func TestGetRegistriesConfig(t *testing.T) {

	t.Run("registry", func(t *testing.T) {
		Load(WithPath("./testdata/config/registry/application.yaml"))

		registries := rootConfig.Registries

		assert.Equal(t, 2, len(registries))
		// nacos
		assert.Equal(t, "nacos", registries["nacos"].Protocol)
		assert.Equal(t, "5s", registries["nacos"].Timeout)
		assert.Equal(t, "127.0.0.1:8848", registries["nacos"].Address)
		assert.Equal(t, "dev", registries["nacos"].Group)
		// zk
		assert.Equal(t, "zookeeper", registries["zk"].Protocol)
		assert.Equal(t, "10s", registries["zk"].Timeout)
		assert.Equal(t, "127.0.0.1:2181", registries["zk"].Address)
		assert.Equal(t, "test", registries["zk"].Group)
	})
}

func TestGetProtocolsConfig(t *testing.T) {

	//t.Run("empty protocols", func(t *testing.T) {
	//	Load(WithPath("./testdata/config/protocol/empty_application.yaml"))
	//
	//	protocols := rootConfig.Protocols
	//	assert.NotNil(t, protocols)
	//	// default
	//	assert.Equal(t, "dubbo", protocols["default"].Name)
	//	assert.Equal(t, "127.0.0.1", protocols["default"].Ip)
	//	assert.Equal(t, 0, protocols["default"].Port)
	//})

	t.Run("protocols", func(t *testing.T) {
		Load(WithPath("./testdata/config/protocol/application.yaml"))

		protocols := rootConfig.Protocols
		assert.NotNil(t, protocols)
		// default
		assert.Equal(t, "dubbo", protocols["dubbo"].Name)
		assert.Equal(t, "127.0.0.1", protocols["dubbo"].Ip)
		assert.Equal(t, string("20000"), protocols["dubbo"].Port)
	})
}

func TestGetProviderConfig(t *testing.T) {
	// empty registry
	t.Run("empty registry", func(t *testing.T) {
		Load(WithPath("./testdata/config/provider/empty_registry_application.yaml"))
		provider := rootConfig.Provider
		assert.NotNil(t, constant.DEFAULT_Key, provider.Registry[0])
	})

	t.Run("root registry", func(t *testing.T) {
		Load(WithPath("./testdata/config/provider/registry_application.yaml"))
		provider := rootConfig.Provider
		assert.NotNil(t, provider)
	})
}

//
//func TestLoadWithSingleReg(t *testing.T) {
//	reference.doInitConsumerWithSingleRegistry()
//	mockInitProviderWithSingleRegistry()
//
//	ms := &MockService{}
//	instance.SetConsumerService(ms)
//	instance.SetProviderService(ms)
//
//	extension.SetProtocol("registry", reference.GetProtocol)
//	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)
//	extension.SetProxyFactory("default", proxy_factory.NewDefaultProxyFactory)
//	var mm *mockMetadataService
//	GetApplicationConfig().MetadataType = "mock"
//	extension.SetLocalMetadataService("mock", func() (metadataService service.MetadataService, err error) {
//		if mm == nil {
//			mm = &mockMetadataService{
//				exportedServiceURLs: new(sync.Map),
//				lock:                new(sync.RWMutex),
//			}
//		}
//		return mm, nil
//	})
//	Load()
//
//	assert.Equal(t, ms, GetRPCService(ms.Reference()))
//	ms2 := &struct {
//		MockService
//	}{}
//	RPCService(ms2)
//	assert.NotEqual(t, ms2, GetRPCService(ms2.Reference()))
//
//	service2.conServices = map[string]common.RPCService{}
//	service2.proServices = map[string]common.RPCService{}
//	common.ServiceMap.UnRegister("com.MockService", "mock", common.ServiceKey("com.MockService", "huadong_idc", "1.0.0"))
//	consumerConfig = nil
//	providerConfig = nil
//}
//
//func TestWithNoRegLoad(t *testing.T) {
//	reference.doInitConsumer()
//	service2.doInitProvider()
//	providerConfig.Services["MockService"].Registry = ""
//	consumerConfig.References["MockService"].Registry = ""
//	ms := &MockService{}
//	instance.SetConsumerService(ms)
//	instance.SetProviderService(ms)
//
//	extension.SetProtocol("registry", reference.GetProtocol)
//	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)
//	extension.SetProxyFactory("default", proxy_factory.NewDefaultProxyFactory)
//	var mm *mockMetadataService
//	GetApplicationConfig().MetadataType = "mock"
//	extension.SetLocalMetadataService("mock", func() (metadataService service.MetadataService, err error) {
//		if mm == nil {
//			mm = &mockMetadataService{
//				exportedServiceURLs: new(sync.Map),
//				lock:                new(sync.RWMutex),
//			}
//		}
//		return mm, nil
//	})
//	Load()
//
//	assert.Equal(t, ms, GetRPCService(ms.Reference()))
//	ms2 := &struct {
//		MockService
//	}{}
//	RPCService(ms2)
//	assert.NotEqual(t, ms2, GetRPCService(ms2.Reference()))
//
//	service2.conServices = map[string]common.RPCService{}
//	service2.proServices = map[string]common.RPCService{}
//	err := common.ServiceMap.UnRegister("com.MockService", "mock",
//		common.ServiceKey("com.MockService", "huadong_idc", "1.0.0"))
//	assert.Nil(t, err)
//	common.ServiceMap.UnRegister("com.MockService", "mock", common.ServiceKey("com.MockService", "huadong_idc", "1.0.0"))
//	consumerConfig = nil
//	providerConfig = nil
//}
//
//func TestSetDefaultValue(t *testing.T) {
//	proConfig := &provider.ProviderConfig{Registries: make(map[string]*registry2.RegistryConfig), Protocols: make(map[string]*protocol2.ProtocolConfig)}
//	assert.Nil(t, proConfig.ApplicationConfig)
//	setDefaultValue(proConfig)
//	assert.Equal(t, proConfig.Registries["demoZK"].Address, "127.0.0.1:2181")
//	assert.Equal(t, proConfig.Registries["demoZK"].TimeoutStr, "3s")
//	assert.Equal(t, proConfig.Registries["demoZK"].Protocol, "zookeeper")
//	assert.Equal(t, proConfig.Protocols["dubbo"].Name, "dubbo")
//	assert.Equal(t, proConfig.Protocols["dubbo"].Port, "20000")
//	assert.NotNil(t, proConfig.ApplicationConfig)
//
//	conConfig := &consumer.ShutdownConfig{Registries: make(map[string]*registry2.RegistryConfig)}
//	assert.Nil(t, conConfig.ApplicationConfig)
//	setDefaultValue(conConfig)
//	assert.Equal(t, conConfig.Registries["demoZK"].Address, "127.0.0.1:2181")
//	assert.Equal(t, conConfig.Registries["demoZK"].TimeoutStr, "3s")
//	assert.Equal(t, conConfig.Registries["demoZK"].Protocol, "zookeeper")
//	assert.NotNil(t, conConfig.ApplicationConfig)
//
//}
//func TestConfigLoaderWithConfigCenter(t *testing.T) {
//	extension.SetConfigCenterFactory("mock", func() config_center.DynamicConfigurationFactory {
//		return &config_center.MockDynamicConfigurationFactory{}
//	})
//
//	conPath, err := filepath.Abs("./testdata/consumer_config_with_configcenter.yml")
//	assert.NoError(t, err)
//	proPath, err := filepath.Abs(mockProviderConfigPath)
//	assert.NoError(t, err)
//
//	assert.Nil(t, consumerConfig)
//	assert.Equal(t, consumer.ShutdownConfig{}, GetConsumerConfig())
//	assert.Nil(t, providerConfig)
//	assert.Equal(t, provider.ProviderConfig{}, GetProviderConfig())
//
//	err = consumer.ConsumerInit(conPath)
//	assert.NoError(t, err)
//	err = consumer.configCenterRefreshConsumer()
//	assert.NoError(t, err)
//	err = provider.ProviderInit(proPath)
//	assert.NoError(t, err)
//	err = provider.configCenterRefreshProvider()
//	assert.NoError(t, err)
//
//	assert.NotNil(t, consumerConfig)
//	assert.NotEqual(t, consumer.ShutdownConfig{}, GetConsumerConfig())
//	assert.NotNil(t, providerConfig)
//	assert.NotEqual(t, provider.ProviderConfig{}, GetProviderConfig())
//
//	assert.Equal(t, "BDTService", consumerConfig.ApplicationConfig.Name)
//	assert.Equal(t, "127.0.0.1:2181", consumerConfig.Registries["hangzhouzk"].Address)
//}
//
//func TestConfigLoaderWithConfigCenterSingleRegistry(t *testing.T) {
//	consumerConfig = nil
//	providerConfig = nil
//	config.NewEnvInstance()
//	extension.SetConfigCenterFactory("mock", func() config_center.DynamicConfigurationFactory {
//		return &config_center.MockDynamicConfigurationFactory{Content: `
//	dubbo.consumer.request_timeout=5s
//	dubbo.consumer.connect_timeout=5s
//	dubbo.applicationConfig.organization=ikurento.com
//	dubbo.applicationConfig.name=BDTService
//	dubbo.applicationConfig.module=dubbogo user-info server
//	dubbo.applicationConfig.version=0.0.1
//	dubbo.applicationConfig.owner=ZX
//	dubbo.applicationConfig.environment=dev
//	dubbo.registry.address=mock://127.0.0.1:2182
//	dubbo.service.com.ikurento.user.UserProvider.protocol=dubbo
//	dubbo.service.com.ikurento.user.UserProvider.interface=com.ikurento.user.UserProvider
//	dubbo.service.com.ikurento.user.UserProvider.loadbalance=random
//	dubbo.service.com.ikurento.user.UserProvider.warmup=100
//	dubbo.service.com.ikurento.user.UserProvider.cluster=failover
//	dubbo.protocols.jsonrpc1.name=jsonrpc
//	dubbo.protocols.jsonrpc1.ip=127.0.0.1
//	dubbo.protocols.jsonrpc1.port=20001
//`}
//	})
//
//	conPath, err := filepath.Abs("./testdata/consumer_config_with_configcenter.yml")
//	assert.NoError(t, err)
//	proPath, err := filepath.Abs(mockProviderConfigPath)
//	assert.NoError(t, err)
//
//	assert.Nil(t, consumerConfig)
//	assert.Equal(t, consumer.ShutdownConfig{}, GetConsumerConfig())
//	assert.Nil(t, providerConfig)
//	assert.Equal(t, provider.ProviderConfig{}, GetProviderConfig())
//
//	err = consumer.ConsumerInit(conPath)
//	assert.NoError(t, err)
//	checkApplicationName(consumerConfig.ApplicationConfig)
//	err = consumer.configCenterRefreshConsumer()
//	checkRegistries(consumerConfig.Registries, consumerConfig.Registry)
//	assert.NoError(t, err)
//	err = provider.ProviderInit(proPath)
//	assert.NoError(t, err)
//	checkApplicationName(providerConfig.ApplicationConfig)
//	err = provider.configCenterRefreshProvider()
//	checkRegistries(providerConfig.Registries, providerConfig.Registry)
//	assert.NoError(t, err)
//
//	assert.NotNil(t, consumerConfig)
//	assert.NotEqual(t, consumer.ShutdownConfig{}, GetConsumerConfig())
//	assert.NotNil(t, providerConfig)
//	assert.NotEqual(t, provider.ProviderConfig{}, GetProviderConfig())
//
//	assert.Equal(t, "BDTService", consumerConfig.ApplicationConfig.Name)
//	assert.Equal(t, "mock://127.0.0.1:2182", consumerConfig.Registries[constant.DEFAULT_KEY].Address)
//}
//
//func TestGetBaseConfig(t *testing.T) {
//	bc := GetBaseConfig()
//	assert.NotNil(t, bc)
//	_, found := bc.GetRemoteConfig("mock")
//	assert.False(t, found)
//}
//
//// mockInitProviderWithSingleRegistry will init a mocked providerConfig
//func mockInitProviderWithSingleRegistry() {
//	providerConfig = &provider.ProviderConfig{
//		BaseConfig: base.ShutdownConfig{
//			applicationConfig.ShutdownConfig: &applicationConfig.ShutdownConfig{
//				Organization: "dubbo_org",
//				Name:         "dubbo",
//				Module:       "module",
//				Version:      "1.0.0",
//				Owner:        "dubbo",
//				Environment:  "test",
//			},
//		},
//
//		Registry: &registry2.RegistryConfig{
//			Address:  "mock://127.0.0.1:2181",
//			Username: "user1",
//			Password: "pwd1",
//		},
//		Registries: map[string]*registry2.RegistryConfig{},
//
//		Services: map[string]*service2.ShutdownConfig{
//			"MockService": {
//				InterfaceName: "com.MockService",
//				Protocol:      "mock",
//				Cluster:       "failover",
//				Loadbalance:   "random",
//				Retries:       "3",
//				Group:         "huadong_idc",
//				Version:       "1.0.0",
//				Methods: []*method.MethodConfig{
//					{
//						Name:        "GetUser",
//						Retries:     "2",
//						LoadBalance: "random",
//						Weight:      200,
//					},
//					{
//						Name:        "GetUser1",
//						Retries:     "2",
//						LoadBalance: "random",
//						Weight:      200,
//					},
//				},
//				exported: new(atomic.Bool),
//			},
//		},
//		Protocols: map[string]*protocol2.ProtocolConfig{
//			"mock": {
//				Name: "mock",
//				Ip:   "127.0.0.1",
//				Port: "20000",
//			},
//		},
//	}
//}
//
//type mockMetadataService struct {
//	exportedServiceURLs *sync.Map
//	lock                *sync.RWMutex
//}
//
//func (m *mockMetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]*common.URL, error) {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) GetMetadataInfo(revision string) (*common.MetadataInfo, error) {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) GetExportedServiceURLs() []*common.URL {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) GetMetadataServiceURL() *common.URL {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) SetMetadataServiceURL(url *common.URL) {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) Reference() string {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) ServiceName() (string, error) {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) ExportURL(url *common.URL) (bool, error) {
//	return m.addURL(m.exportedServiceURLs, url), nil
//}
//
//func (m *mockMetadataService) UnexportURL(*common.URL) error {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) SubscribeURL(*common.URL) (bool, error) {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) UnsubscribeURL(*common.URL) error {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) PublishServiceDefinition(*common.URL) error {
//	return nil
//}
//
//func (m *mockMetadataService) MethodMapper() map[string]string {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) GetSubscribedURLs() ([]*common.URL, error) {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) GetServiceDefinition(string, string, string) (string, error) {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) GetServiceDefinitionByServiceKey(string) (string, error) {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) RefreshMetadata(string, string) (bool, error) {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) Version() (string, error) {
//	panic("implement me")
//}
//
//func (m *mockMetadataService) addURL(targetMap *sync.Map, url *common.URL) bool {
//	var (
//		urlSet interface{}
//		loaded bool
//	)
//	logger.Debug(url.ServiceKey())
//	if urlSet, loaded = targetMap.LoadOrStore(url.ServiceKey(), skip.New(uint64(0))); loaded {
//		m.lock.RLock()
//		wantedUrl := urlSet.(*skip.SkipList).Get(url)
//		if len(wantedUrl) > 0 && wantedUrl[0] != nil {
//			m.lock.RUnlock()
//			return false
//		}
//		m.lock.RUnlock()
//	}
//	m.lock.Lock()
//	// double chk
//	wantedUrl := urlSet.(*skip.SkipList).Get(url)
//	if len(wantedUrl) > 0 && wantedUrl[0] != nil {
//		m.lock.Unlock()
//		return false
//	}
//	urlSet.(*skip.SkipList).Insert(url)
//	m.lock.Unlock()
//	return true
//}
//
//func (m *mockMetadataService) getAllService(services *sync.Map) []*common.URL {
//	// using skip list to dedup and sorting
//	var res []*common.URL
//	services.Range(func(key, value interface{}) bool {
//		urls := value.(*skip.SkipList)
//		for i := uint64(0); i < urls.Len(); i++ {
//			url := urls.ByPosition(i).(*common.URL)
//			if url.GetParam(constant.INTERFACE_KEY, url.Path) != constant.METADATA_SERVICE_NAME {
//				res = append(res, url)
//			}
//		}
//		return true
//	})
//	sort.Sort(common.URLSlice(res))
//	return res
//}
//
//type mockServiceDiscoveryRegistry struct{}
//
//func (mr *mockServiceDiscoveryRegistry) GetURL() *common.URL {
//	panic("implement me")
//}
//
//func (mr *mockServiceDiscoveryRegistry) IsAvailable() bool {
//	panic("implement me")
//}
//
//func (mr *mockServiceDiscoveryRegistry) Destroy() {
//	panic("implement me")
//}
//
//func (mr *mockServiceDiscoveryRegistry) Register(*common.URL) error {
//	panic("implement me")
//}
//
//func (mr *mockServiceDiscoveryRegistry) UnRegister(*common.URL) error {
//	panic("implement me")
//}
//
//func (mr *mockServiceDiscoveryRegistry) Subscribe(*common.URL, registry.NotifyListener) error {
//	panic("implement me")
//}
//
//func (mr *mockServiceDiscoveryRegistry) UnSubscribe(*common.URL, registry.NotifyListener) error {
//	panic("implement me")
//}
//
//func (mr *mockServiceDiscoveryRegistry) GetServiceDiscovery() registry.ServiceDiscovery {
//	return &mockServiceDiscovery{}
//}
//
//type mockServiceDiscovery struct{}
//
//func (m *mockServiceDiscovery) String() string {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) Destroy() error {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) Register(registry.ServiceInstance) error {
//	return nil
//}
//
//func (m *mockServiceDiscovery) Update(registry.ServiceInstance) error {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) Unregister(registry.ServiceInstance) error {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) GetDefaultPageSize() int {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) GetServices() *gxset.HashSet {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) GetInstances(string) []registry.ServiceInstance {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) GetInstancesByPage(string, int, int) gxpage.Pager {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) GetHealthyInstancesByPage(string, int, int, bool) gxpage.Pager {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) GetRequestInstances([]string, int, int) map[string]gxpage.Pager {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) AddListener(registry.ServiceInstancesChangedListener) error {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) DispatchEventByServiceName(string) error {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) DispatchEventForInstances(string, []registry.ServiceInstance) error {
//	panic("implement me")
//}
//
//func (m *mockServiceDiscovery) DispatchEvent(*registry.ServiceInstancesChangedEvent) error {
//	panic("implement me")
//}
//
//func ConvertURLArrToIntfArr(urls []*common.URL) []interface{} {
//	if len(urls) == 0 {
//		return []interface{}{}
//	}
//
//	res := make([]interface{}, 0, len(urls))
//	for _, u := range urls {
//		res = append(res, u.String())
//	}
//	return res
//}
//
//type mockGracefulShutdownFilter struct{}
//
//func (f *mockGracefulShutdownFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
//	panic("implement me")
//}
//
//func (f *mockGracefulShutdownFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
//	panic("implement me")
//}
//
//func (f *mockGracefulShutdownFilter) Set(name string, config interface{}) {
//	return
//}
