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
	_ "github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config"
	_ "github.com/apache/dubbo-go/filter/filter_impl"
	"github.com/apache/dubbo-go/metadata/service/inmemory"
	"github.com/apache/dubbo-go/protocol/dubbo"
	_ "github.com/apache/dubbo-go/protocol/dubbo"
)

func TestConfigurableExporter(t *testing.T) {
	dubbo.SetServerConfig(dubbo.ServerConfig{
		SessionNumber:  700,
		SessionTimeout: "20s",
		GettySessionParam: dubbo.GettySessionParam{
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
	assert.Equal(t, false, exported.IsExported())
	assert.NoError(t, exported.Export())
	assert.Equal(t, true, exported.IsExported())
	assert.Regexp(t, "dubbo://:20000/MetadataService*", exported.GetExportedURLs()[0].String())
	exported.Unexport()
	assert.Equal(t, false, exported.IsExported())
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
						Loadbalance: "random",
						Weight:      200,
					},
					{
						Name:        "GetUser1",
						Retries:     "2",
						Loadbalance: "random",
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
