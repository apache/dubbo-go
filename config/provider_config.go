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
	// Registry registry ids
	Registry []string `yaml:"registry" json:"registry" property:"registry"`
	// Services services
	Services map[string]*ServiceConfig `yaml:"services" json:"services,omitempty" property:"services"`

	ProxyFactory string `default:"default" yaml:"proxy" json:"proxy,omitempty" property:"proxy"`

	FilterConf interface{} `yaml:"filter_conf" json:"filter_conf,omitempty" property:"filter_conf"`
	// ShutdownConfig *ShutdownConfig            `yaml:"shutdown_conf" json:"shutdown_conf,omitempty" property:"shutdown_conf"`
	ConfigType map[string]string `yaml:"config_type" json:"config_type,omitempty" property:"config_type"`
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
	c.Registry = translateRegistryIds(c.Registry)
	if len(c.Registry) <= 0 {
		c.Registry = rc.getRegistryIds()
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
			logger.Warnf("%s does not exist!", key)
			continue
		}
		svs.id = key
		svs.Implement(rpcService)
		if err := svs.Export(); err != nil {
			logger.Errorf(fmt.Sprintf("service %s export failed! err: %#v", key, err))
		}
	}

}

// SetProviderConfig sets provider config by @p
func SetProviderConfig(p ProviderConfig) {
	rootConfig.Provider = &p
}

///////////////////////////////////// provider config api
// ProviderConfigOpt is the
type ProviderConfigOpt func(config *ProviderConfig) *ProviderConfig

// NewEmptyProviderConfig returns ProviderConfig with default ApplicationConfig
func NewEmptyProviderConfig() *ProviderConfig {
	newProviderConfig := &ProviderConfig{
		Services: make(map[string]*ServiceConfig),
		Registry: make([]string, 8),
	}
	return newProviderConfig
}

// NewProviderConfig returns ProviderConfig with given @opts
func NewProviderConfig(opts ...ProviderConfigOpt) *ProviderConfig {
	newConfig := NewEmptyProviderConfig()
	for _, v := range opts {
		v(newConfig)
	}
	return newConfig
}

// GetProviderInstance returns ProviderConfig with given @opts
func GetProviderInstance(opts ...ProviderConfigOpt) *ProviderConfig {
	newConfig := &ProviderConfig{
		Services: make(map[string]*ServiceConfig),
		Registry: make([]string, 8),
	}
	for _, opt := range opts {
		opt(newConfig)
	}
	return newConfig
}

// WithProviderServices returns ProviderConfig with given serviceNameKey @serviceName and @serviceConfig
func WithProviderService(serviceName string, serviceConfig *ServiceConfig) ProviderConfigOpt {
	return func(config *ProviderConfig) *ProviderConfig {
		config.Services[serviceName] = serviceConfig
		return config
	}
}

// WithProviderRegistryKeys returns ProviderConfigOpt with given @registryKey and registry @registryConfig
func WithProviderRegistryKeys(registryKey ...string) ProviderConfigOpt {
	return func(config *ProviderConfig) *ProviderConfig {
		config.Registry = append(config.Registry, registryKey...)
		return config
	}
}
