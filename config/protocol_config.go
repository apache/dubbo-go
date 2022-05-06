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
	"github.com/creasty/defaults"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// ProtocolConfig is protocol configuration
type ProtocolConfig struct {
	Name   string      `default:"dubbo" validate:"required" yaml:"name" json:"name,omitempty" property:"name"`
	Ip     string      `yaml:"ip"  json:"ip,omitempty" property:"ip"`
	Port   string      `default:"20000" yaml:"port" json:"port,omitempty" property:"port"`
	Params interface{} `yaml:"params" json:"params,omitempty" property:"params"`
}

// Prefix dubbo.config-center
func (ProtocolConfig) Prefix() string {
	return constant.ConfigCenterPrefix
}

func GetProtocolsInstance() map[string]*ProtocolConfig {
	return make(map[string]*ProtocolConfig, 1)
}

func (p *ProtocolConfig) Init() error {
	if err := defaults.Set(p); err != nil {
		return err
	}
	return verify(p)
}

func NewProtocolConfigBuilder() *ProtocolConfigBuilder {
	return &ProtocolConfigBuilder{protocolConfig: &ProtocolConfig{}}
}

type ProtocolConfigBuilder struct {
	protocolConfig *ProtocolConfig
}

func (pcb *ProtocolConfigBuilder) SetName(name string) *ProtocolConfigBuilder {
	pcb.protocolConfig.Name = name
	return pcb
}

func (pcb *ProtocolConfigBuilder) SetIp(ip string) *ProtocolConfigBuilder {
	pcb.protocolConfig.Ip = ip
	return pcb
}

func (pcb *ProtocolConfigBuilder) SetPort(port string) *ProtocolConfigBuilder {
	pcb.protocolConfig.Port = port
	return pcb
}

func (pcb *ProtocolConfigBuilder) SetParams(params interface{}) *ProtocolConfigBuilder {
	pcb.protocolConfig.Params = params
	return pcb
}

func (pcb *ProtocolConfigBuilder) Build() *ProtocolConfig {
	return pcb.protocolConfig
}
