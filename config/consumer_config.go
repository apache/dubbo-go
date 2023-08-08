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
	"dubbo.apache.org/dubbo-go/v3/client"
)

// SetConsumerConfig sets consumerConfig by @c
func SetConsumerConfig(c client.ConsumerConfig) {
	rootConfig.Consumer = &c
}

func newEmptyConsumerConfig() *client.ConsumerConfig {
	newConsumerConfig := &client.ConsumerConfig{
		References:     make(map[string]*client.ReferenceConfig, 8),
		RequestTimeout: "3s",
		Check:          true,
	}
	return newConsumerConfig
}

type ConsumerConfigBuilder struct {
	consumerConfig *client.ConsumerConfig
}

func NewConsumerConfigsBuilder() *ConsumerConfigBuilder {
	return &ConsumerConfigBuilder{consumerConfig: newEmptyConsumerConfig()}
}

func (ccb *ConsumerConfigBuilder) SetFilter(filter string) *ConsumerConfigBuilder {
	ccb.consumerConfig.Filter = filter
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetRegistryIDs(RegistryIDs ...string) *ConsumerConfigBuilder {
	ccb.consumerConfig.RegistryIDs = RegistryIDs
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetRequestTimeout(requestTimeout string) *ConsumerConfigBuilder {
	ccb.consumerConfig.RequestTimeout = requestTimeout
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetMaxWaitTimeForServiceDiscovery(maxWaitTimeForServiceDiscovery string) *ConsumerConfigBuilder {
	ccb.consumerConfig.MaxWaitTimeForServiceDiscovery = maxWaitTimeForServiceDiscovery
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetProxyFactory(proxyFactory string) *ConsumerConfigBuilder {
	ccb.consumerConfig.ProxyFactory = proxyFactory
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetCheck(check bool) *ConsumerConfigBuilder {
	ccb.consumerConfig.Check = check
	return ccb
}

func (ccb *ConsumerConfigBuilder) AddReference(referenceKey string, referenceConfig *client.ReferenceConfig) *ConsumerConfigBuilder {
	ccb.consumerConfig.References[referenceKey] = referenceConfig
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetReferences(references map[string]*client.ReferenceConfig) *ConsumerConfigBuilder {
	ccb.consumerConfig.References = references
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetFilterConf(filterConf interface{}) *ConsumerConfigBuilder {
	ccb.consumerConfig.FilterConf = filterConf
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetMeshEnabled(meshEnabled bool) *ConsumerConfigBuilder {
	ccb.consumerConfig.MeshEnabled = meshEnabled
	return ccb
}

func (ccb *ConsumerConfigBuilder) SetRootConfig(rootConfig *RootConfig) *ConsumerConfigBuilder {
	ccb.consumerConfig.SetRootConfig(rootConfig)
	return ccb
}

func (ccb *ConsumerConfigBuilder) Build() *client.ConsumerConfig {
	return ccb.consumerConfig
}
