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
)

var regProtocol protocol.Protocol

func doInitConsumer() {
	consumerConfig = &ConsumerConfig{
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
		References: map[string]*ReferenceConfig{
			"MockService": {
				Params: map[string]string{
					"serviceid": "soa.mock",
					"forks":     "5",
				},
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
						Loadbalance: "random",
					},
					{
						Name:        "GetUser1",
						Retries:     "2",
						Loadbalance: "random",
					},
				},
			},
		},
	}
}

func doInitConsumerWithSingleRegistry() {
	consumerConfig = &ConsumerConfig{
		ApplicationConfig: &ApplicationConfig{
			Organization: "dubbo_org",
			Name:         "dubbo",
			Module:       "module",
			Version:      "2.6.0",
			Owner:        "dubbo",
			Environment:  "test"},
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
						Loadbalance: "random",
					},
					{
						Name:        "GetUser1",
						Retries:     "2",
						Loadbalance: "random",
					},
				},
			},
		},
	}
}

func Test_ReferMultireg(t *testing.T) {
	doInitConsumer()
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
	doInitConsumer()
	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster("registryAware", cluster_impl.NewRegistryAwareCluster)

	for _, reference := range consumerConfig.References {
		reference.Refer()
		assert.Equal(t, "soa.mock", reference.Params["serviceid"])
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}
func Test_ReferP2P(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("dubbo", GetProtocol)
	m := consumerConfig.References["MockService"]
	m.Url = "dubbo://127.0.0.1:20000"

	for _, reference := range consumerConfig.References {
		reference.Refer()
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func Test_ReferMultiP2P(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("dubbo", GetProtocol)
	m := consumerConfig.References["MockService"]
	m.Url = "dubbo://127.0.0.1:20000;dubbo://127.0.0.2:20000"

	for _, reference := range consumerConfig.References {
		reference.Refer()
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func Test_ReferMultiP2PWithReg(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("dubbo", GetProtocol)
	extension.SetProtocol("registry", GetProtocol)
	m := consumerConfig.References["MockService"]
	m.Url = "dubbo://127.0.0.1:20000;registry://127.0.0.2:20000"

	for _, reference := range consumerConfig.References {
		reference.Refer()
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func Test_Implement(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster("registryAware", cluster_impl.NewRegistryAwareCluster)
	for _, reference := range consumerConfig.References {
		reference.Refer()
		reference.Implement(&MockService{})
		assert.NotNil(t, reference.GetRPCService())

	}
	consumerConfig = nil
}

func Test_Forking(t *testing.T) {
	doInitConsumer()
	extension.SetProtocol("dubbo", GetProtocol)
	extension.SetProtocol("registry", GetProtocol)
	m := consumerConfig.References["MockService"]
	m.Url = "dubbo://127.0.0.1:20000;registry://127.0.0.2:20000"

	for _, reference := range consumerConfig.References {
		reference.Refer()
		forks := int(reference.invoker.GetUrl().GetParamInt(constant.FORKS_KEY, constant.DEFAULT_FORKS))
		assert.Equal(t, 5, forks)
		assert.NotNil(t, reference.pxy)
		assert.NotNil(t, reference.Cluster)
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
