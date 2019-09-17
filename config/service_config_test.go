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
	"github.com/apache/dubbo-go/common/extension"
)

func doInitProvider() {
	providerConfig = &ProviderConfig{
		ApplicationConfig: &ApplicationConfig{
			Organization: "dubbo_org",
			Name:         "dubbo",
			Module:       "module",
			Version:      "2.6.0",
			Owner:        "dubbo",
			Environment:  "test"},
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
		Services: map[string]*ServiceConfig{
			"MockService": {
				InterfaceName: "com.MockService",
				Protocol:      "mock",
				Registry:      "shanghai_reg1,shanghai_reg2,hangzhou_reg1,hangzhou_reg2",
				Cluster:       "failover",
				Loadbalance:   "random",
				Retries:       "3",
				Group:         "huadong_idc",
				Version:       "1.0.0",
				Methods: []*MethodConfig{
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
		Protocols: map[string]*ProtocolConfig{
			"mock": {
				Name: "mock",
				Ip:   "127.0.0.1",
				Port: "20000",
			},
		},
	}
}

func doInitProviderWithSingleRegistry() {
	providerConfig = &ProviderConfig{
		ApplicationConfig: &ApplicationConfig{
			Organization: "dubbo_org",
			Name:         "dubbo",
			Module:       "module",
			Version:      "2.6.0",
			Owner:        "dubbo",
			Environment:  "test"},
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
		Protocols: map[string]*ProtocolConfig{
			"mock": {
				Name: "mock",
				Ip:   "127.0.0.1",
				Port: "20000",
			},
		},
	}
}

func Test_Export(t *testing.T) {
	doInitProvider()
	extension.SetProtocol("registry", GetProtocol)

	for i := range providerConfig.Services {
		service := providerConfig.Services[i]
		service.Implement(&MockService{})
		service.Export()
	}
	providerConfig = nil
}
