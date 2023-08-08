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

//////////////////////////////////// reference config api

// newEmptyReferenceConfig returns empty ReferenceConfig
func newEmptyReferenceConfig() *client.ReferenceConfig {
	newReferenceConfig := &client.ReferenceConfig{}
	newReferenceConfig.Methods = make([]*client.MethodConfig, 0, 8)
	newReferenceConfig.Params = make(map[string]string, 8)
	return newReferenceConfig
}

type ReferenceConfigBuilder struct {
	referenceConfig *client.ReferenceConfig
}

func NewReferenceConfigBuilder() *ReferenceConfigBuilder {
	return &ReferenceConfigBuilder{referenceConfig: newEmptyReferenceConfig()}
}

func (pcb *ReferenceConfigBuilder) SetInterface(interfaceName string) *ReferenceConfigBuilder {
	pcb.referenceConfig.InterfaceName = interfaceName
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetRegistryIDs(registryIDs ...string) *ReferenceConfigBuilder {
	pcb.referenceConfig.RegistryIDs = registryIDs
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetGeneric(generic bool) *ReferenceConfigBuilder {
	if generic {
		pcb.referenceConfig.Generic = "true"
	} else {
		pcb.referenceConfig.Generic = "false"
	}
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetCluster(cluster string) *ReferenceConfigBuilder {
	pcb.referenceConfig.Cluster = cluster
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetSerialization(serialization string) *ReferenceConfigBuilder {
	pcb.referenceConfig.Serialization = serialization
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetProtocol(protocol string) *ReferenceConfigBuilder {
	pcb.referenceConfig.Protocol = protocol
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetURL(url string) *ReferenceConfigBuilder {
	pcb.referenceConfig.URL = url
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetFilter(filter string) *ReferenceConfigBuilder {
	pcb.referenceConfig.Filter = filter
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetLoadbalance(loadbalance string) *ReferenceConfigBuilder {
	pcb.referenceConfig.Loadbalance = loadbalance
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetRetries(retries string) *ReferenceConfigBuilder {
	pcb.referenceConfig.Retries = retries
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetGroup(group string) *ReferenceConfigBuilder {
	pcb.referenceConfig.Group = group
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetVersion(version string) *ReferenceConfigBuilder {
	pcb.referenceConfig.Version = version
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetProvidedBy(providedBy string) *ReferenceConfigBuilder {
	pcb.referenceConfig.ProvidedBy = providedBy
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetMethodConfig(methodConfigs []*client.MethodConfig) *ReferenceConfigBuilder {
	pcb.referenceConfig.Methods = methodConfigs
	return pcb
}

func (pcb *ReferenceConfigBuilder) AddMethodConfig(methodConfigs *client.MethodConfig) *ReferenceConfigBuilder {
	pcb.referenceConfig.Methods = append(pcb.referenceConfig.Methods, methodConfigs)
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetAsync(async bool) *ReferenceConfigBuilder {
	pcb.referenceConfig.Async = async
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetParams(params map[string]string) *ReferenceConfigBuilder {
	pcb.referenceConfig.Params = params
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetSticky(sticky bool) *ReferenceConfigBuilder {
	pcb.referenceConfig.Sticky = sticky
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetRequestTimeout(requestTimeout string) *ReferenceConfigBuilder {
	pcb.referenceConfig.RequestTimeout = requestTimeout
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetForceTag(forceTag bool) *ReferenceConfigBuilder {
	pcb.referenceConfig.ForceTag = forceTag
	return pcb
}

func (pcb *ReferenceConfigBuilder) SetTracingKey(tracingKey string) *ReferenceConfigBuilder {
	pcb.referenceConfig.TracingKey = tracingKey
	return pcb
}

func (pcb *ReferenceConfigBuilder) Build() *client.ReferenceConfig {
	return pcb.referenceConfig
}
