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
	"path/filepath"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/cluster/cluster_impl"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config_center"
)

func TestConfigLoader(t *testing.T) {
	conPath, err := filepath.Abs("./testdata/consumer_config.yml")
	assert.NoError(t, err)
	proPath, err := filepath.Abs("./testdata/provider_config.yml")
	assert.NoError(t, err)

	assert.Nil(t, consumerConfig)
	assert.Equal(t, ConsumerConfig{}, GetConsumerConfig())
	assert.Nil(t, providerConfig)
	assert.Equal(t, ProviderConfig{}, GetProviderConfig())

	err = consumerInit(conPath)
	assert.NoError(t, err)
	err = providerInit(proPath)
	assert.NoError(t, err)

	assert.NotNil(t, consumerConfig)
	assert.NotEqual(t, ConsumerConfig{}, GetConsumerConfig())
	assert.NotNil(t, providerConfig)
	assert.NotEqual(t, ProviderConfig{}, GetProviderConfig())
}

func TestLoad(t *testing.T) {
	doInit()
	doinit()

	ms := &MockService{}
	SetConsumerService(ms)
	SetProviderService(ms)

	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster("registryAware", cluster_impl.NewRegistryAwareCluster)
	extension.SetProxyFactory("default", proxy_factory.NewDefaultProxyFactory)

	Load()

	assert.Equal(t, ms, GetRPCService(ms.Service()))
	ms2 := &struct {
		MockService
	}{}
	RPCService(ms2)
	assert.NotEqual(t, ms2, GetRPCService(ms2.Service()))

	conServices = map[string]common.RPCService{}
	proServices = map[string]common.RPCService{}
	common.ServiceMap.UnRegister("mock", "MockService")
	consumerConfig = nil
	providerConfig = nil
}

func TestConfigLoaderWithConfigCenter(t *testing.T) {
	extension.SetConfigCenterFactory("mock", func() config_center.DynamicConfigurationFactory {
		return &config_center.MockDynamicConfigurationFactory{}
	})

	conPath, err := filepath.Abs("./testdata/consumer_config_with_configcenter.yml")
	assert.NoError(t, err)
	proPath, err := filepath.Abs("./testdata/provider_config.yml")
	assert.NoError(t, err)

	assert.Nil(t, consumerConfig)
	assert.Equal(t, ConsumerConfig{}, GetConsumerConfig())
	assert.Nil(t, providerConfig)
	assert.Equal(t, ProviderConfig{}, GetProviderConfig())

	err = consumerInit(conPath)
	configCenterRefreshConsumer()
	assert.NoError(t, err)
	err = providerInit(proPath)
	configCenterRefreshProvider()
	assert.NoError(t, err)

	assert.NotNil(t, consumerConfig)
	assert.NotEqual(t, ConsumerConfig{}, GetConsumerConfig())
	assert.NotNil(t, providerConfig)
	assert.NotEqual(t, ProviderConfig{}, GetProviderConfig())

	assert.Equal(t, "BDTService", consumerConfig.ApplicationConfig.Name)
	assert.Equal(t, "127.0.0.1:2181", consumerConfig.Registries["hangzhouzk"].Address)

}
