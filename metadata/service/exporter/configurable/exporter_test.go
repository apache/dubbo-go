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

package configurable

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	_ "github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config"
	_ "github.com/apache/dubbo-go/filter/filter_impl"
	"github.com/apache/dubbo-go/metadata/service/inmemory"
	_ "github.com/apache/dubbo-go/protocol/dubbo"
	"github.com/apache/dubbo-go/remoting/getty"
)

func TestConfigurableExporter(t *testing.T) {
	getty.SetServerConfig(getty.ServerConfig{
		SessionNumber:  700,
		SessionTimeout: "20s",
		GettySessionParam: getty.GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePeriod:  "120s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
			PkgWQSize:        512,
			TcpReadTimeout:   "1s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        10240000000,
			SessionName:      "server",
		}})
	mockInitProviderWithSingleRegistry()
	metadataService, _ := inmemory.NewMetadataService()
	exported := NewMetadataServiceExporter(metadataService)

	t.Run("configurableExporterUrlNil", func(t *testing.T) {
		assert.Equal(t, false, exported.IsExported())
		assert.Error(t, exported.Export(nil), "metadata server url is nil, pls check your configuration")
	})

	t.Run("configurableExporter", func(t *testing.T) {
		registryURL, _ := common.NewURL("service-discovery://localhost:12345")
		subURL, _ := common.NewURL("dubbo://localhost:20003")
		registryURL.SubURL = subURL
		assert.Equal(t, false, exported.IsExported())
		assert.NoError(t, exported.Export(registryURL))
		assert.Equal(t, true, exported.IsExported())
		assert.Regexp(t, "dubbo://:20003/org.apache.dubbo.metadata.MetadataService*", exported.GetExportedURLs()[0].String())
		exported.Unexport()
		assert.Equal(t, false, exported.IsExported())
	})
}

// mockInitProviderWithSingleRegistry will init a mocked providerConfig
func mockInitProviderWithSingleRegistry() {
	providerConfig := &config.ProviderConfig{

		BaseConfig: config.BaseConfig{
			ApplicationConfig: &config.ApplicationConfig{
				Organization: "dubbo_org",
				Name:         "dubbo",
				Module:       "module",
				Version:      "1.0.0",
				Owner:        "dubbo",
				Environment:  "test"},
		},

		Registry: &config.RegistryConfig{
			Address:  "mock://127.0.0.1:2181",
			Username: "user1",
			Password: "pwd1",
		},
		Registries: map[string]*config.RegistryConfig{},

		Services: map[string]*config.ServiceConfig{
			"MockService": {
				InterfaceName: "com.MockService",
				Protocol:      "mock",
				Cluster:       "failover",
				Loadbalance:   "random",
				Retries:       "3",
				Group:         "huadong_idc",
				Version:       "1.0.0",
				Methods: []*config.MethodConfig{
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
			},
		},
		Protocols: map[string]*config.ProtocolConfig{
			"mock": {
				Name: "mock",
				Ip:   "127.0.0.1",
				Port: "20000",
			},
		},
	}
	providerConfig.Services["MockService"].InitExported()
	config.SetProviderConfig(*providerConfig)
}
