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
	"fmt"
)

import (
	"github.com/creasty/defaults"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
)

// ProviderConfig is the default configuration of service provider
type ProviderConfig struct {
	Filter string `yaml:"filter" json:"filter,omitempty" property:"filter"`
	// Deprecated Register whether registration is required
	Register bool `yaml:"register" json:"register" property:"register"`
	// RegistryIDs is registry ids list
	RegistryIDs []string `yaml:"registryIDs" json:"registryIDs" property:"registryIDs"`
	// Services services
	Services map[string]*ServiceConfig `yaml:"services" json:"services,omitempty" property:"services"`

	ProxyFactory string `default:"default" yaml:"proxy" json:"proxy,omitempty" property:"proxy"`

	FilterConf interface{}       `yaml:"filter_conf" json:"filter_conf,omitempty" property:"filter_conf"`
	ConfigType map[string]string `yaml:"config_type" json:"config_type,omitempty" property:"config_type"`

	rootConfig *RootConfig
}

func (ProviderConfig) Prefix() string {
	return constant.ProviderConfigPrefix
}

func (c *ProviderConfig) check() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	return verify(c)
}

func (c *ProviderConfig) Init(rc *RootConfig) error {
	if c == nil {
		return nil
	}
	c.RegistryIDs = translateRegistryIds(c.RegistryIDs)
	if len(c.RegistryIDs) <= 0 {
		c.RegistryIDs = rc.getRegistryIds()
	}
	for _, service := range c.Services {
		if err := service.Init(rc); err != nil {
			return err
		}
	}
	if err := c.check(); err != nil {
		return err
	}
	return nil
}

func (c *ProviderConfig) Load() {
	for key, svs := range c.Services {
		rpcService := GetProviderService(key)
		if rpcService == nil {
			logger.Warnf("Service reference key %s does not exist, please check if this key "+
				"matches your provider struct type name, or matches the returned valued of your provider struct's Reference() function."+
				"View https://www.yuque.com/u772707/eqpff0/pqfgz3#zxdw0 for details", key)
			continue
		}
		svs.id = key
		svs.Implement(rpcService)
		if err := svs.Export(); err != nil {
			logger.Errorf(fmt.Sprintf("service %s export failed! err: %#v", key, err))
		}
	}

}

// newEmptyProviderConfig returns ProviderConfig with default ApplicationConfig
func newEmptyProviderConfig() *ProviderConfig {
	newProviderConfig := &ProviderConfig{
		Services:    make(map[string]*ServiceConfig),
		RegistryIDs: make([]string, 8),
	}
	return newProviderConfig
}

type ProviderConfigBuilder struct {
	providerConfig *ProviderConfig
}

func NewProviderConfigBuilder() *ProviderConfigBuilder {
	return &ProviderConfigBuilder{providerConfig: newEmptyProviderConfig()}
}

func (pcb *ProviderConfigBuilder) SetFilter(filter string) *ProviderConfigBuilder {
	pcb.providerConfig.Filter = filter
	return pcb
}

// nolint
func (pcb *ProviderConfigBuilder) SetRegister(register bool) *ProviderConfigBuilder {
	pcb.providerConfig.Register = register
	return pcb
}

// nolint
func (pcb *ProviderConfigBuilder) SetRegistryIDs(RegistryIDs ...string) *ProviderConfigBuilder {
	pcb.providerConfig.RegistryIDs = RegistryIDs
	return pcb
}

// nolint
func (pcb *ProviderConfigBuilder) SetServices(services map[string]*ServiceConfig) *ProviderConfigBuilder {
	pcb.providerConfig.Services = services
	return pcb
}

// nolint
func (pcb *ProviderConfigBuilder) AddService(serviceID string, serviceConfig *ServiceConfig) *ProviderConfigBuilder {
	if pcb.providerConfig.Services == nil {
		pcb.providerConfig.Services = make(map[string]*ServiceConfig)
	}
	pcb.providerConfig.Services[serviceID] = serviceConfig
	return pcb
}

// nolint
func (pcb *ProviderConfigBuilder) SetProxyFactory(proxyFactory string) *ProviderConfigBuilder {
	pcb.providerConfig.ProxyFactory = proxyFactory
	return pcb
}

// nolint
func (pcb *ProviderConfigBuilder) SetFilterConf(filterConf interface{}) *ProviderConfigBuilder {
	pcb.providerConfig.FilterConf = filterConf
	return pcb
}

// nolint
func (pcb *ProviderConfigBuilder) SetConfigType(configType map[string]string) *ProviderConfigBuilder {
	pcb.providerConfig.ConfigType = configType
	return pcb
}

// nolint
func (pcb *ProviderConfigBuilder) AddConfigType(key, value string) *ProviderConfigBuilder {
	if pcb.providerConfig.ConfigType == nil {
		pcb.providerConfig.ConfigType = make(map[string]string)
	}
	pcb.providerConfig.ConfigType[key] = value
	return pcb
}

// nolint
func (pcb *ProviderConfigBuilder) SetRootConfig(rootConfig *RootConfig) *ProviderConfigBuilder {
	pcb.providerConfig.rootConfig = rootConfig
	return pcb
}

// nolint
func (pcb *ProviderConfigBuilder) Build() *ProviderConfig {
	return pcb.providerConfig
}
