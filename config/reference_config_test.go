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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
)

//import (
//	"context"
//	"dubbo.apache.org/dubbo-go/v3/config"
//	"dubbo.apache.org/dubbo-go/v3/config/applicationConfig"
//	"dubbo.apache.org/dubbo-go/v3/config/base"
//	"dubbo.apache.org/dubbo-go/v3/config/consumer"
//	"dubbo.apache.org/dubbo-go/v3/config/instance"
//	"dubbo.apache.org/dubbo-go/v3/config/method"
//	registry2 "dubbo.apache.org/dubbo-go/v3/config/registry"
//	"sync"
//	"testing"
//)
//
//import (
//	"github.com/stretchr/testify/assert"
//)
//
//import (
//	"dubbo.apache.org/dubbo-go/v3/cluster/cluster_impl"
//	"dubbo.apache.org/dubbo-go/v3/common"
//	"dubbo.apache.org/dubbo-go/v3/common/constant"
//	"dubbo.apache.org/dubbo-go/v3/common/extension"
//	"dubbo.apache.org/dubbo-go/v3/filter"
//	"dubbo.apache.org/dubbo-go/v3/protocol"
//	"dubbo.apache.org/dubbo-go/v3/registry"
//)
//
//var regProtocol protocol.Protocol
//
//func doInitConsumer() {
//	config.consumerConfig = &consumer.Config{
//		BaseConfig: base.Config{
//			applicationConfig.Config: &applicationConfig.Config{
//				Organization: "dubbo_org",
//				Name:         "dubbo",
//				Module:       "module",
//				Version:      "2.6.0",
//				Owner:        "dubbo",
//				Environment:  "test",
//			},
//		},
//
//		Registries: map[string]*registry2.RegistryConfig{
//			"shanghai_reg1": {
//				Protocol:   "mock",
//				TimeoutStr: "2s",
//				Group:      "shanghai_idc",
//				Address:    "127.0.0.1:2181",
//				Username:   "user1",
//				Password:   "pwd1",
//			},
//			"shanghai_reg2": {
//				Protocol:   "mock",
//				TimeoutStr: "2s",
//				Group:      "shanghai_idc",
//				Address:    "127.0.0.2:2181",
//				Username:   "user1",
//				Password:   "pwd1",
//			},
//			"hangzhou_reg1": {
//				Protocol:   "mock",
//				TimeoutStr: "2s",
//				Group:      "hangzhou_idc",
//				Address:    "127.0.0.3:2181",
//				Username:   "user1",
//				Password:   "pwd1",
//			},
//			"hangzhou_reg2": {
//				Protocol:   "mock",
//				TimeoutStr: "2s",
//				Group:      "hangzhou_idc",
//				Address:    "127.0.0.4:2181",
//				Username:   "user1",
//				Password:   "pwd1",
//			},
//		},
//
//		References: map[string]*ReferenceConfig{
//			"MockService": {
//				id: "MockProvider",
//				Params: map[string]string{
//					"serviceid": "soa.mock",
//					"forks":     "5",
//				},
//				Sticky:        false,
//				Registry:      "shanghai_reg1,shanghai_reg2,hangzhou_reg1,hangzhou_reg2",
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
//					},
//					{
//						Name:        "GetUser1",
//						Retries:     "2",
//						LoadBalance: "random",
//						Sticky:      true,
//					},
//				},
//			},
//		},
//	}
//}
//
//var mockProvider = new(MockProvider)
//
//type MockProvider struct{}
//
//func (m *MockProvider) Reference() string {
//	return "MockProvider"
//}
//
//func (m *MockProvider) CallBack(res common.CallbackResponse) {
//	// CallBack is a mock function. to implement the interface
//}
//
//func doInitConsumerAsync() {
//	doInitConsumer()
//	instance.SetConsumerService(mockProvider)
//	for _, v := range config.consumerConfig.References {
//		v.Async = true
//	}
//}
//
//func doInitConsumerWithSingleRegistry() {
//	config.consumerConfig = &consumer.Config{
//		BaseConfig: base.Config{
//			applicationConfig.Config: &applicationConfig.Config{
//				Organization: "dubbo_org",
//				Name:         "dubbo",
//				Module:       "module",
//				Version:      "2.6.0",
//				Owner:        "dubbo",
//				Environment:  "test",
//			},
//		},
//
//		Registry: &registry2.RegistryConfig{
//			Address:  "mock://27.0.0.1:2181",
//			Username: "user1",
//			Password: "pwd1",
//		},
//		Registries: map[string]*registry2.RegistryConfig{},
//
//		References: map[string]*ReferenceConfig{
//			"MockService": {
//				Params: map[string]string{
//					"serviceid": "soa.mock",
//					"forks":     "5",
//				},
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
//					},
//					{
//						Name:        "GetUser1",
//						Retries:     "2",
//						LoadBalance: "random",
//					},
//				},
//			},
//		},
//	}
//}
//
//func TestReferMultiReg(t *testing.T) {
//	doInitConsumer()
//	extension.SetProtocol("registry", GetProtocol)
//	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)
//	for _, reference := range config.consumerConfig.References {
//		reference.Refer(nil)
//		assert.NotNil(t, reference.invoker)
//		assert.NotNil(t, reference.pxy)
//	}
//	config.consumerConfig = nil
//}
//
//func TestRefer(t *testing.T) {
//	doInitConsumer()
//	extension.SetProtocol("registry", GetProtocol)
//	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)
//
//	for _, reference := range config.consumerConfig.References {
//		reference.Refer(nil)
//		assert.Equal(t, "soa.mock", reference.Params["serviceid"])
//		assert.NotNil(t, reference.invoker)
//		assert.NotNil(t, reference.pxy)
//	}
//	config.consumerConfig = nil
//}
//
//func TestReferAsync(t *testing.T) {
//	doInitConsumerAsync()
//	extension.SetProtocol("registry", GetProtocol)
//	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)
//
//	for _, reference := range config.consumerConfig.References {
//		reference.Refer(nil)
//		assert.Equal(t, "soa.mock", reference.Params["serviceid"])
//		assert.NotNil(t, reference.invoker)
//		assert.NotNil(t, reference.pxy)
//		assert.NotNil(t, reference.pxy.GetCallback())
//	}
//	config.consumerConfig = nil
//}
//
//func TestReferP2P(t *testing.T) {
//	doInitConsumer()
//	extension.SetProtocol("dubbo", GetProtocol)
//	mockFilter()
//	m := config.consumerConfig.References["MockService"]
//	m.URL = "dubbo://127.0.0.1:20000"
//
//	for _, reference := range config.consumerConfig.References {
//		reference.Refer(nil)
//		assert.NotNil(t, reference.invoker)
//		assert.NotNil(t, reference.pxy)
//	}
//	config.consumerConfig = nil
//}
//
//func TestReferMultiP2P(t *testing.T) {
//	doInitConsumer()
//	extension.SetProtocol("dubbo", GetProtocol)
//	mockFilter()
//	m := config.consumerConfig.References["MockService"]
//	m.URL = "dubbo://127.0.0.1:20000;dubbo://127.0.0.2:20000"
//
//	for _, reference := range config.consumerConfig.References {
//		reference.Refer(nil)
//		assert.NotNil(t, reference.invoker)
//		assert.NotNil(t, reference.pxy)
//	}
//	config.consumerConfig = nil
//}
//
//func TestReferMultiP2PWithReg(t *testing.T) {
//	doInitConsumer()
//	extension.SetProtocol("dubbo", GetProtocol)
//	extension.SetProtocol("registry", GetProtocol)
//	mockFilter()
//	m := config.consumerConfig.References["MockService"]
//	m.URL = "dubbo://127.0.0.1:20000;registry://127.0.0.2:20000"
//
//	for _, reference := range config.consumerConfig.References {
//		reference.Refer(nil)
//		assert.NotNil(t, reference.invoker)
//		assert.NotNil(t, reference.pxy)
//	}
//	config.consumerConfig = nil
//}
//
//func TestImplement(t *testing.T) {
//	doInitConsumer()
//	extension.SetProtocol("registry", GetProtocol)
//	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)
//	for _, reference := range config.consumerConfig.References {
//		reference.Refer(nil)
//		reference.Implement(&config.MockService{})
//		assert.NotNil(t, reference.GetRPCService())
//
//	}
//	config.consumerConfig = nil
//}
//
//func TestForking(t *testing.T) {
//	doInitConsumer()
//	extension.SetProtocol("dubbo", GetProtocol)
//	extension.SetProtocol("registry", GetProtocol)
//	mockFilter()
//	m := config.consumerConfig.References["MockService"]
//	m.URL = "dubbo://127.0.0.1:20000;registry://127.0.0.2:20000"
//
//	for _, reference := range config.consumerConfig.References {
//		reference.Refer(nil)
//		forks := int(reference.invoker.GetURL().GetParamInt(constant.FORKS_KEY, constant.DEFAULT_FORKS))
//		assert.Equal(t, 5, forks)
//		assert.NotNil(t, reference.pxy)
//		assert.NotNil(t, reference.Cluster)
//	}
//	config.consumerConfig = nil
//}
//
//func TestSticky(t *testing.T) {
//	doInitConsumer()
//	extension.SetProtocol("dubbo", GetProtocol)
//	extension.SetProtocol("registry", GetProtocol)
//	mockFilter()
//	m := config.consumerConfig.References["MockService"]
//	m.URL = "dubbo://127.0.0.1:20000;registry://127.0.0.2:20000"
//
//	reference := config.consumerConfig.References["MockService"]
//	reference.Refer(nil)
//	referenceSticky := reference.invoker.GetURL().GetParam(constant.STICKY_KEY, "false")
//	assert.Equal(t, "false", referenceSticky)
//
//	method0StickKey := reference.invoker.GetURL().GetMethodParam(reference.Methods[0].Name, constant.STICKY_KEY, "false")
//	assert.Equal(t, "false", method0StickKey)
//	method1StickKey := reference.invoker.GetURL().GetMethodParam(reference.Methods[1].Name, constant.STICKY_KEY, "false")
//	assert.Equal(t, "true", method1StickKey)
//}
//
//func GetProtocol() protocol.Protocol {
//	if regProtocol != nil {
//		return regProtocol
//	}
//	return newRegistryProtocol()
//}
//
//func newRegistryProtocol() protocol.Protocol {
//	return &mockRegistryProtocol{}
//}
//
//type mockRegistryProtocol struct {
//}
//
//func (*mockRegistryProtocol) Refer(url *common.URL) protocol.Invoker {
//	return protocol.NewBaseInvoker(url)
//}
//
//func (*mockRegistryProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
//	registryURL := getRegistryURL(invoker)
//	if registryURL.Protocol == "service-discovery" {
//		metaDataService, err := extension.GetLocalMetadataService("")
//		if err != nil {
//			panic(err)
//		}
//		ok, err := metaDataService.ExportURL(invoker.GetURL().SubURL.Clone())
//		if err != nil {
//			panic(err)
//		}
//		if !ok {
//			panic("The URL has been registry!")
//		}
//	}
//	return protocol.NewBaseExporter("test", invoker, &sync.Map{})
//}
//
//func (*mockRegistryProtocol) Destroy() {
//	// Destroy is a mock function
//}
//
//func getRegistryURL(invoker protocol.Invoker) *common.URL {
//	// here add * for return a new url
//	url := invoker.GetURL()
//	// if the protocol == registry ,set protocol the registry value in url.params
//	if url.Protocol == constant.REGISTRY_PROTOCOL {
//		protocol := url.GetParam(constant.REGISTRY_KEY, "")
//		url.Protocol = protocol
//	}
//	return url
//}
//
//func (p *mockRegistryProtocol) GetRegistries() []registry.Registry {
//	return []registry.Registry{&config.mockServiceDiscoveryRegistry{}}
//}
//
//func mockFilter() {
//	consumerFiler := &mockShutdownFilter{}
//	extension.SetFilter(constant.GracefulShutdownConsumerFilterKey, func() filter.Filter {
//		return consumerFiler
//	})
//}
//
//type mockShutdownFilter struct {
//}
//
//// Invoke adds the requests count and block the new requests if applicationConfig is closing
//func (gf *mockShutdownFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
//	return invoker.Invoke(ctx, invocation)
//}
//
//// OnResponse reduces the number of active processes then return the process result
//func (gf *mockShutdownFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
//	return result
//}
func TestNewReferenceConfigBuilder(t *testing.T) {
	registryConfig := NewRegistryConfigWithProtocolDefaultPort("nacos")
	protocolConfig := NewProtocolConfigBuilder().
		SetName("dubbo").
		SetPort("20000").
		Build()
	config := NewReferenceConfigBuilder().
		SetInterface("org.apache.dubbo.HelloService").
		SetRegistryIDs("nacos").
		SetGeneric(false).
		SetCluster("cluster").
		SetSerialization("serialization").
		SetProtocol("dubbo").
		Build()

	config.rootConfig = NewRootConfigBuilder().
		SetProtocols(map[string]*ProtocolConfig{"dubbo": protocolConfig}).
		SetRegistries(map[string]*RegistryConfig{"nacos": registryConfig}).
		Build()

	assert.Equal(t, config.Prefix(), constant.ReferenceConfigPrefix+config.InterfaceName+".")
	proxy := config.GetProxy()
	assert.Nil(t, proxy)

	values := config.getURLMap()
	assert.Equal(t, values.Get(constant.GroupKey), "")

	invoker := config.GetInvoker()
	assert.Nil(t, invoker)
}
