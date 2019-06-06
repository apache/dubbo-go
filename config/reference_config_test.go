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
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
)

var regProtocol protocol.Protocol

func doInit() {
	consumerConfig = &ConsumerConfig{
		ApplicationConfig: ApplicationConfig{
			Organization: "dubbo_org",
			Name:         "dubbo",
			Module:       "module",
			Version:      "2.6.0",
			Owner:        "dubbo",
			Environment:  "test"},
		Registries: []RegistryConfig{
			{
				Id:         "shanghai_reg1",
				Type:       "mock",
				TimeoutStr: "2s",
				Group:      "shanghai_idc",
				Address:    "127.0.0.1:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			{
				Id:         "shanghai_reg2",
				Type:       "mock",
				TimeoutStr: "2s",
				Group:      "shanghai_idc",
				Address:    "127.0.0.2:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			{
				Id:         "hangzhou_reg1",
				Type:       "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.3:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			{
				Id:         "hangzhou_reg2",
				Type:       "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.4:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
		},
		References: []ReferenceConfig{
			{
				InterfaceName: "MockService",
				Protocol:      "mock",
				Registries:    []ConfigRegistry{"shanghai_reg1", "shanghai_reg2", "hangzhou_reg1", "hangzhou_reg2"},
				Cluster:       "failover",
				Loadbalance:   "random",
				Retries:       3,
				Group:         "huadong_idc",
				Version:       "1.0.0",
				Methods: []struct {
					Name        string `yaml:"name"  json:"name,omitempty"`
					Retries     int64  `yaml:"retries"  json:"retries,omitempty"`
					Loadbalance string `yaml:"loadbalance"  json:"loadbalance,omitempty"`
				}{
					{
						Name:        "GetUser",
						Retries:     2,
						Loadbalance: "random",
					},
					{
						Name:        "GetUser1",
						Retries:     2,
						Loadbalance: "random",
					},
				},
			},
		},
	}
}

func Test_ReferMultireg(t *testing.T) {
	doInit()
	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster("registryAware", cluster_impl.NewRegistryAwareCluster)

	for _, reference := range consumerConfig.References {
		reference.Refer()
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func Test_Refer(t *testing.T) {
	doInit()
	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster("registryAware", cluster_impl.NewRegistryAwareCluster)
	consumerConfig.References[0].Registries = []ConfigRegistry{"shanghai_reg1"}

	for _, reference := range consumerConfig.References {
		reference.Refer()
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}
func Test_ReferP2P(t *testing.T) {
	doInit()
	extension.SetProtocol("dubbo", GetProtocol)
	consumerConfig.References[0].Url = "dubbo://127.0.0.1:20000"

	for _, reference := range consumerConfig.References {
		reference.Refer()
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}
func Test_ReferMultiP2P(t *testing.T) {
	doInit()
	extension.SetProtocol("dubbo", GetProtocol)
	consumerConfig.References[0].Url = "dubbo://127.0.0.1:20000;dubbo://127.0.0.2:20000"

	for _, reference := range consumerConfig.References {
		reference.Refer()
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func Test_ReferMultiP2PWithReg(t *testing.T) {
	doInit()
	extension.SetProtocol("dubbo", GetProtocol)
	extension.SetProtocol("registry", GetProtocol)
	consumerConfig.References[0].Url = "dubbo://127.0.0.1:20000;registry://127.0.0.2:20000"

	for _, reference := range consumerConfig.References {
		reference.Refer()
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func Test_Implement(t *testing.T) {
	doInit()
	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster("registryAware", cluster_impl.NewRegistryAwareCluster)
	for _, reference := range consumerConfig.References {
		reference.Refer()
		reference.Implement(&MockService{})
		assert.NotNil(t, reference.GetRPCService())

	}
	consumerConfig = nil
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

func (*mockRegistryProtocol) Refer(url common.URL) protocol.Invoker {
	return protocol.NewBaseInvoker(url)
}

func (*mockRegistryProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	return protocol.NewBaseExporter("test", invoker, &sync.Map{})
}

func (*mockRegistryProtocol) Destroy() {}
