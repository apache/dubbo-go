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
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/nacos"
)

func TestProviderConfigEmptyRegistry(t *testing.T) {
	err := Load(WithPath("./testdata/config/provider/empty_registry_application.yaml"))
	require.NoError(t, err)
	provider := rootConfig.Provider
	assert.Len(t, provider.RegistryIDs, 1)
	assert.Equal(t, "nacos", provider.RegistryIDs[0])
}

func TestProviderConfigRootRegistry(t *testing.T) {
	err := Load(WithPath("./testdata/config/provider/registry_application.yaml"))
	require.NoError(t, err)
	provider := rootConfig.Provider
	assert.NotNil(t, provider)
	assert.NotNil(t, provider.Services["HelloService"])
	assert.NotNil(t, provider.Services["OrderService"])

	assert.Len(t, provider.Services["HelloService"].RegistryIDs, 2)
	assert.Len(t, provider.Services["OrderService"].RegistryIDs, 1)
}

//
//func TestConsumerInitWithDefaultProtocol(t *testing.T) {
//	conPath, err := filepath.Abs("./testdata/consumer_config_withoutProtocol.yml")
//	require.NoError(t, err)
//	assert.NoError(t, consumer.ConsumerInit(conPath))
//	assert.Equal(t, "dubbo", config.consumerConfig.References["UserProvider"].Protocol)
//}
//
//func TestProviderInitWithDefaultProtocol(t *testing.T) {
//	conPath, err := filepath.Abs("./testdata/provider_config_withoutProtocol.yml")
//	require.NoError(t, err)
//	assert.NoError(t, ProviderInit(conPath))
//	assert.Equal(t, "dubbo", config.referenceConfig.Services["UserProvider"].Protocol)
//}

func TestNewProviderConfigBuilder(t *testing.T) {

	config := NewProviderConfigBuilder().
		SetFilter("echo").
		SetRegister(true).
		SetRegistryIDs("nacos").
		SetServices(map[string]*ServiceConfig{}).
		AddService("HelloService", &ServiceConfig{}).
		SetProxyFactory("default").
		SetFilterConf(nil).
		SetConfigType(map[string]string{}).
		AddConfigType("", "").
		SetRootConfig(nil).
		Build()

	err := config.check()
	require.NoError(t, err)
	assert.Equal(t, constant.ProviderConfigPrefix, config.Prefix())
}
