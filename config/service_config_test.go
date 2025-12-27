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
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
)

type HelloService struct {
}

func (hs *HelloService) Say(ctx context.Context, name string) (string, error) {
	return name, nil
}
func (hs *HelloService) Reference() string {
	return "HelloService"
}

func (hs *HelloService) JavaClassName() string {
	return "org.apache.dubbo.HelloService"
}

func TestNewServiceConfigBuilder(t *testing.T) {
	SetProviderService(&HelloService{})
	var serviceConfig = newEmptyServiceConfig()
	t.Run("NewServiceConfigBuilder", func(t *testing.T) {
		registryConfig := NewRegistryConfigWithProtocolDefaultPort("nacos")
		protocolConfig := NewProtocolConfigBuilder().
			SetName("dubbo").
			SetPort("20000").
			Build()
		rc := newEmptyRootConfig()

		serviceConfig = NewServiceConfigBuilder().
			SetRegistryIDs("nacos").
			SetProtocolIDs("dubbo").
			SetInterface("org.apache.dubbo.HelloService").
			SetMetadataType("local").
			SetLoadBalance("random").
			SetWarmUpTie("warmup").
			SetCluster("cluster").
			AddRCRegistry("nacos", registryConfig).
			AddRCProtocol("dubbo", protocolConfig).
			SetGroup("dubbo").
			SetVersion("1.0.0").
			SetProxyFactoryKey("default").
			SetSerialization("serialization").
			SetServiceID("HelloService").
			Build()

		serviceConfig.InitExported()

		serviceConfig.Methods = []*MethodConfig{
			{
				Name:    "Say",
				Retries: "3",
			},
		}

		err := serviceConfig.Init(rc)
		require.NoError(t, err)
		err = serviceConfig.check()
		require.NoError(t, err)

		assert.Equal(t, serviceConfig.Prefix(), strings.Join([]string{constant.ServiceConfigPrefix, serviceConfig.id}, "."))
		assert.False(t, serviceConfig.IsExport())
	})

	t.Run("LoadRegistries&loadProtocol&getRandomPort", func(t *testing.T) {
		registries := LoadRegistries(serviceConfig.RegistryIDs, serviceConfig.RCRegistriesMap, common.PROVIDER)
		assert.Len(t, registries, 1)
		assert.Equal(t, "service-discovery-registry", registries[0].Protocol)
		assert.Equal(t, "8848", registries[0].Port)
		assert.Equal(t, "3", registries[0].GetParam("registry.role", "1"))
		assert.Equal(t, "nacos", registries[0].GetParam("registry", "zk"))

		protocols := loadProtocol(serviceConfig.ProtocolIDs, serviceConfig.RCProtocolsMap)
		assert.Len(t, protocols, 1)
		assert.Equal(t, "dubbo", protocols[0].Name)
		assert.Equal(t, "20000", protocols[0].Port)

		ports := getRandomPort(protocols)
		nextPort := ports.Front()
		assert.Nil(t, nextPort)
	})
	t.Run("getUrlMap", func(t *testing.T) {
		values := serviceConfig.getUrlMap()
		assert.Equal(t, "0", values.Get("methods.Say.weight"))
		assert.Empty(t, values.Get("methods.Say.tps.limit.rate"))
		assert.Equal(t, "echo,token,accesslog,tps,generic_service,execute,pshutdown", values.Get(constant.ServiceFilterKey))
	})

	t.Run("Implement", func(t *testing.T) {
		serviceConfig.Implement(&HelloService{})
		//urls := serviceConfig.GetExportedUrls()
		//err := serviceConfig.Export()
		assert.NotNil(t, serviceConfig.rpcService)
	})
}
