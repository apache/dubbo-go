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

func NewDefaultProtocolConfig() *ProtocolConfig {
	return &ProtocolConfig{
		Name: constant.DEFAULT_PROTOCOL,
		Port: "20000",
		Ip:   "127.0.0.1",
	}
}

// NewProtocolConfig returns ProtocolConfig with given @opts
func NewProtocolConfig(opts ...ProtocolConfigOpt) *ProtocolConfig {
	newConfig := NewDefaultProtocolConfig()
	for _, v := range opts {
		v(newConfig)
	}
	return newConfig
}

type ProtocolConfigOpt func(config *ProtocolConfig) *ProtocolConfig

// WithProtocolIP set ProtocolConfig with given binding @ip
// Deprecated: the param @ip would be used as service listener binding and would be registered to registry center
func WithProtocolIP(ip string) ProtocolConfigOpt {
	return func(config *ProtocolConfig) *ProtocolConfig {
		config.Ip = ip
		return config
	}
}

func WithProtocolName(protcolName string) ProtocolConfigOpt {
	return func(config *ProtocolConfig) *ProtocolConfig {
		config.Name = protcolName
		return config
	}
}

func WithProtocolPort(port string) ProtocolConfigOpt {
	return func(config *ProtocolConfig) *ProtocolConfig {
		config.Port = port
		return config
	}
}

type ProtocolConfigBuilder struct {
	*ProtocolConfig
}

func (pcb *ProtocolConfigBuilder) Name(name string) *ProtocolConfigBuilder {
	pcb.ProtocolConfig.Name = name
	return pcb
}

func (pcb *ProtocolConfigBuilder) Ip(ip string) *ProtocolConfigBuilder {
	pcb.ProtocolConfig.Ip = ip
	return pcb
}

func (pcb *ProtocolConfigBuilder) Port(port string) *ProtocolConfigBuilder {
	pcb.ProtocolConfig.Port = port
	return pcb
}

func (pcb *ProtocolConfigBuilder) Params(params interface{}) *ProtocolConfigBuilder {
	pcb.ProtocolConfig.Params = params
	return pcb
}

func (pcb *ProtocolConfigBuilder) Build() *ProtocolConfig {
	if err := pcb.Init(); err != nil {
		panic(err)
	}
	return pcb.ProtocolConfig
}
