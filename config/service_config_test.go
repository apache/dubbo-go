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
	"github.com/apache/dubbo-go/common"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/common/extension"
)

func doInitProvider() {
	providerConfig = &ProviderConfig{
		BaseConfig: BaseConfig{
			ApplicationConfig: &ApplicationConfig{
				Organization: "dubbo_org",
				Name:         "dubbo",
				Module:       "module",
				Version:      "2.6.0",
				Owner:        "dubbo",
				Environment:  "test",
			},
			Remotes: map[string]*RemoteConfig{
				"test1": {
					Address:    "127.0.0.5:2181",
					TimeoutStr: "5s",
					Username:   "user1",
					Password:   "pwd1",
					Params:     nil,
				},
			},
			ServiceDiscoveries: map[string]*ServiceDiscoveryConfig{
				"mock_servicediscovery": {
					Protocol:  "mock",
					RemoteRef: "test1",
				},
			},
			MetadataReportConfig: &MetadataReportConfig{
				Protocol:  "mock",
				RemoteRef: "test1",
			},
		},
		Services: map[string]*ServiceConfig{
			"MockService": {
				InterfaceName: "com.MockService",
				Protocol:      "mock",
				Registry:      "shanghai_reg1,shanghai_reg2,hangzhou_reg1,hangzhou_reg2,hangzhou_service_discovery_reg",
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
			"MockServiceNoRightProtocol": {
				InterfaceName: "com.MockService",
				Protocol:      "mock1",
				Registry:      "shanghai_reg1,shanghai_reg2,hangzhou_reg1,hangzhou_reg2,hangzhou_service_discovery_reg",
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
			"hangzhou_service_discovery_reg": {
				Protocol: "service-discovery",
				Params: map[string]string{
					"service_discovery": "mock_servicediscovery",
					"name_mapping":      "in-memory",
					"metadata":          "default",
				},
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

func TestExport(t *testing.T) {
	doInitProvider()
	extension.SetProtocol("registry", GetProtocol)

	for i := range providerConfig.Services {
		service := providerConfig.Services[i]
		service.Implement(&MockService{})
		service.Protocols = providerConfig.Protocols
		service.Export()
	}
	providerConfig = nil
}

func TestGetRandomPort(t *testing.T) {
	protocolConfigs := make([]*ProtocolConfig, 0, 3)

	ip := common.GetLocalIp()
	protocolConfigs = append(protocolConfigs, &ProtocolConfig{
		Ip: ip,
	})
	protocolConfigs = append(protocolConfigs, &ProtocolConfig{
		Ip: ip,
	})
	protocolConfigs = append(protocolConfigs, &ProtocolConfig{
		Ip: ip,
	})
	//assert.NoError(t, err)
	ports := getRandomPort(protocolConfigs)

	assert.Equal(t, ports.Len(), len(protocolConfigs))

	front := ports.Front()
	for {
		if front == nil {
			break
		}
		t.Logf("port:%v", front.Value)
		front = front.Next()
	}

	protocolConfigs = make([]*ProtocolConfig, 0, 3)
	ports = getRandomPort(protocolConfigs)
	assert.Equal(t, ports.Len(), len(protocolConfigs))
}
