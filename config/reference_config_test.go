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
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/cluster/cluster_impl"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/registry"
)

var regProtocol protocol.Protocol

func doInitConsumer() {
	consumerConfig = &ConsumerConfig{
		BaseConfig: BaseConfig{
			ApplicationConfig: &ApplicationConfig{
				Organization: "dubbo_org",
				Name:         "dubbo",
				Module:       "module",
				Version:      "2.6.0",
				Owner:        "dubbo",
				Environment:  "test"},
		},

		Registries: map[string]*RegistryConfig{
			"shanghai_reg1": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "shanghai_idc",
				Address:    "127.0.0.1:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			"shanghai_reg2": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "shanghai_idc",
				Address:    "127.0.0.2:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			"hangzhou_reg1": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.3:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			"hangzhou_reg2": {
				Protocol:   "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.4:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
		},

		References: map[string]*ReferenceConfig{
			"MockService": {
				id: "MockProvider",
				Params: map[string]string{
					"serviceid": "soa.mock",
					"forks":     "5",
				},
				Sticky:        false,
				Registry:      "shanghai_reg1,shanghai_reg2,hangzhou_reg1,hangzhou_reg2",
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
					},
					{
						Name:        "GetUser1",
						Retries:     "2",
						LoadBalance: "random",
						Sticky:      true,
					},
				},
			},
		},
	}
}

var mockProvider = new(MockProvider)

type MockProvider struct {
}

func (m *MockProvider) Reference() string {
	return "MockProvider"
}

func (m *MockProvider) CallBack(res common.CallbackResponse) {
	// CallBack is a mock function. to implement the interface
}

func doInitConsumerAsync() {
	doInitConsumer()
	SetConsumerService(mockProvider)
	for _, v := range consumerConfig.References {
		v.Async = true
	}
}

func doInitConsumerWithSingleRegistry() {
	consumerConfig = &ConsumerConfig{
		BaseConfig: BaseConfig{
			ApplicationConfig: &ApplicationConfig{
				Organization: "dubbo_org",
				Name:         "dubbo",
				Module:       "module",
				Version:      "2.6.0",
				Owner:        "dubbo",
				Environment:  "test"},
		},

		Registry: &RegistryConfig{
			Address:  "mock://27.0.0.1:2181",
			Username: "user1",
			Password: "pwd1",
		},
		Registries: map[string]*RegistryConfig{},

		References: map[string]*ReferenceConfig{
			"MockService": {
				Params: map[string]string{
					"serviceid": "soa.mock",
					"forks":     "5",
				},
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
					},
					{
						Name:        "GetUser1",
						Retries:     "2",
						LoadBalance: "random",
					},
				},
			},
		},
	}
}

func TestReferMultiReg(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)

	for _, reference := range consumerConfig.References {
		reference.Refer(nil)
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func TestRefer(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)

	for _, reference := range consumerConfig.References {
		reference.Refer(nil)
		assert.Equal(t, "soa.mock", reference.Params["serviceid"])
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func TestReferAsync(t *testing.T) {
	doInitConsumerAsync()
	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)

	for _, reference := range consumerConfig.References {
		reference.Refer(nil)
		assert.Equal(t, "soa.mock", reference.Params["serviceid"])
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
		assert.NotNil(t, reference.pxy.GetCallback())
	}
	consumerConfig = nil
}

func TestReferP2P(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("dubbo", GetProtocol)
	m := consumerConfig.References["MockService"]
	m.Url = "dubbo://127.0.0.1:20000"

	for _, reference := range consumerConfig.References {
		reference.Refer(nil)
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func TestReferMultiP2P(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("dubbo", GetProtocol)
	m := consumerConfig.References["MockService"]
	m.Url = "dubbo://127.0.0.1:20000;dubbo://127.0.0.2:20000"

	for _, reference := range consumerConfig.References {
		reference.Refer(nil)
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func TestReferMultiP2PWithReg(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("dubbo", GetProtocol)
	extension.SetProtocol("registry", GetProtocol)
	m := consumerConfig.References["MockService"]
	m.Url = "dubbo://127.0.0.1:20000;registry://127.0.0.2:20000"

	for _, reference := range consumerConfig.References {
		reference.Refer(nil)
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func TestImplement(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster(constant.ZONEAWARE_CLUSTER_NAME, cluster_impl.NewZoneAwareCluster)
	for _, reference := range consumerConfig.References {
		reference.Refer(nil)
		reference.Implement(&MockService{})
		assert.NotNil(t, reference.GetRPCService())

	}
	consumerConfig = nil
}

func TestForking(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("dubbo", GetProtocol)
	extension.SetProtocol("registry", GetProtocol)
	m := consumerConfig.References["MockService"]
	m.Url = "dubbo://127.0.0.1:20000;registry://127.0.0.2:20000"

	for _, reference := range consumerConfig.References {
		reference.Refer(nil)
		forks := int(reference.invoker.GetUrl().GetParamInt(constant.FORKS_KEY, constant.DEFAULT_FORKS))
		assert.Equal(t, 5, forks)
		assert.NotNil(t, reference.pxy)
		assert.NotNil(t, reference.Cluster)
	}
	consumerConfig = nil
}

func TestSticky(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("dubbo", GetProtocol)
	extension.SetProtocol("registry", GetProtocol)
	m := consumerConfig.References["MockService"]
	m.Url = "dubbo://127.0.0.1:20000;registry://127.0.0.2:20000"

	reference := consumerConfig.References["MockService"]
	reference.Refer(nil)
	referenceSticky := reference.invoker.GetUrl().GetParam(constant.STICKY_KEY, "false")
	assert.Equal(t, "false", referenceSticky)

	method0StickKey := reference.invoker.GetUrl().GetMethodParam(reference.Methods[0].Name, constant.STICKY_KEY, "false")
	assert.Equal(t, "false", method0StickKey)
	method1StickKey := reference.invoker.GetUrl().GetMethodParam(reference.Methods[1].Name, constant.STICKY_KEY, "false")
	assert.Equal(t, "true", method1StickKey)
}

func GetProtocol() protocol.Protocol {
	if regProtocol != nil {
		return regProtocol
	}
	return newRegistryProtocol()
}

func newRegistryProtocol() protocol.Protocol {
	return &mockRegistryProtocol{}
}

type mockRegistryProtocol struct{}

func (*mockRegistryProtocol) Refer(url *common.URL) protocol.Invoker {
	return protocol.NewBaseInvoker(url)
}

func (*mockRegistryProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	registryUrl := getRegistryUrl(invoker)
	if registryUrl.Protocol == "service-discovery" {
		metaDataService, err := extension.GetMetadataService(GetApplicationConfig().MetadataType)
		if err != nil {
			panic(err)
		}
		ok, err := metaDataService.ExportURL(invoker.GetUrl().SubURL.Clone())
		if err != nil {
			panic(err)
		}
		if !ok {
			panic("The URL has been registry!")
		}
	}
	return protocol.NewBaseExporter("test", invoker, &sync.Map{})
}

func (*mockRegistryProtocol) Destroy() {
	// Destroy is a mock function
}
func getRegistryUrl(invoker protocol.Invoker) *common.URL {
	// here add * for return a new url
	url := invoker.GetUrl()
	// if the protocol == registry ,set protocol the registry value in url.params
	if url.Protocol == constant.REGISTRY_PROTOCOL {
		protocol := url.GetParam(constant.REGISTRY_KEY, "")
		url.Protocol = protocol
	}
	return url
}

func (p *mockRegistryProtocol) GetRegistries() []registry.Registry {
	return []registry.Registry{&mockServiceDiscoveryRegistry{}}
}
