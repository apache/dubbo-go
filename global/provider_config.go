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

package global

// ProviderConfig is the default configuration of service provider
type ProviderConfig struct {
	ServiceConfig
	Filter string `yaml:"filter" json:"filter,omitempty" property:"filter"`
	// Deprecated Register whether registration is required
	Register bool `yaml:"register" json:"register" property:"register"`
	// RegistryIDs is registry ids list
	RegistryIDs []string `yaml:"registry-ids" json:"registry-ids" property:"registry-ids"`
	// protocol
	ProtocolIDs []string `yaml:"protocol-ids" json:"protocol-ids" property:"protocol-ids"`
	// TracingKey is tracing ids list
	TracingKey string `yaml:"tracing-key" json:"tracing-key" property:"tracing-key"`
	// there is no need to configure Services
	// Services services
	Services     map[string]*ServiceConfig `yaml:"services" json:"services,omitempty" property:"services"`
	ProxyFactory string                    `default:"default" yaml:"proxy" json:"proxy,omitempty" property:"proxy"`
	FilterConf   interface{}               `yaml:"filter_conf" json:"filter_conf,omitempty" property:"filter_conf"`
	ConfigType   map[string]string         `yaml:"config_type" json:"config_type,omitempty" property:"config_type"`
	// adaptive service
	AdaptiveService        bool `yaml:"adaptive-service" json:"adaptive-service" property:"adaptive-service"`
	AdaptiveServiceVerbose bool `yaml:"adaptive-service-verbose" json:"adaptive-service-verbose" property:"adaptive-service-verbose"`
}

func DefaultProviderConfig() *ProviderConfig {
	return &ProviderConfig{
		RegistryIDs: make([]string, 8),
		ProtocolIDs: make([]string, 8),
		Services:    make(map[string]*ServiceConfig),
	}
}

// Clone a new ProviderConfig
func (c *ProviderConfig) Clone() *ProviderConfig {
	newRegistryIDs := make([]string, len(c.RegistryIDs))
	copy(newRegistryIDs, c.RegistryIDs)

	newProtocolIDs := make([]string, len(c.ProtocolIDs))
	copy(newProtocolIDs, c.ProtocolIDs)

	newServices := make(map[string]*ServiceConfig, len(c.Services))
	for k, v := range c.Services {
		newServices[k] = v.Clone()
	}

	newConfigType := make(map[string]string, len(c.ConfigType))
	for k, v := range c.ConfigType {
		newConfigType[k] = v
	}

	return &ProviderConfig{
		ServiceConfig:          *c.ServiceConfig.Clone(),
		Filter:                 c.Filter,
		Register:               c.Register,
		RegistryIDs:            newRegistryIDs,
		ProtocolIDs:            newProtocolIDs,
		TracingKey:             c.TracingKey,
		Services:               newServices,
		ProxyFactory:           c.ProxyFactory,
		FilterConf:             c.FilterConf,
		ConfigType:             c.ConfigType,
		AdaptiveService:        c.AdaptiveService,
		AdaptiveServiceVerbose: c.AdaptiveServiceVerbose,
	}
}
