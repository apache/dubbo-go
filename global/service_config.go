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

// ServiceConfig is the configuration of the service provider
type ServiceConfig struct {
	Filter                      string            `yaml:"filter" json:"filter,omitempty" property:"filter"`
	ProtocolIDs                 []string          `yaml:"protocol-ids"  json:"protocol-ids,omitempty" property:"protocol-ids"` // multi protocolIDs support, split by ','
	Interface                   string            `yaml:"interface"  json:"interface,omitempty" property:"interface"`
	RegistryIDs                 []string          `yaml:"registry-ids"  json:"registry-ids,omitempty"  property:"registry-ids"`
	Cluster                     string            `default:"failover" yaml:"cluster"  json:"cluster,omitempty" property:"cluster"`
	Loadbalance                 string            `default:"random" yaml:"loadbalance"  json:"loadbalance,omitempty"  property:"loadbalance"`
	Group                       string            `yaml:"group"  json:"group,omitempty" property:"group"`
	Version                     string            `yaml:"version"  json:"version,omitempty" property:"version" `
	Methods                     []*MethodConfig   `yaml:"methods"  json:"methods,omitempty" property:"methods"`
	Warmup                      string            `yaml:"warmup"  json:"warmup,omitempty"  property:"warmup"`
	Retries                     string            `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	Serialization               string            `yaml:"serialization" json:"serialization" property:"serialization"`
	Params                      map[string]string `yaml:"params"  json:"params,omitempty" property:"params"`
	Token                       string            `yaml:"token" json:"token,omitempty" property:"token"`
	AccessLog                   string            `yaml:"accesslog" json:"accesslog,omitempty" property:"accesslog"`
	TpsLimiter                  string            `yaml:"tps.limiter" json:"tps.limiter,omitempty" property:"tps.limiter"`
	TpsLimitInterval            string            `yaml:"tps.limit.interval" json:"tps.limit.interval,omitempty" property:"tps.limit.interval"`
	TpsLimitRate                string            `yaml:"tps.limit.rate" json:"tps.limit.rate,omitempty" property:"tps.limit.rate"`
	TpsLimitStrategy            string            `yaml:"tps.limit.strategy" json:"tps.limit.strategy,omitempty" property:"tps.limit.strategy"`
	TpsLimitRejectedHandler     string            `yaml:"tps.limit.rejected.handler" json:"tps.limit.rejected.handler,omitempty" property:"tps.limit.rejected.handler"`
	ExecuteLimit                string            `yaml:"execute.limit" json:"execute.limit,omitempty" property:"execute.limit"`
	ExecuteLimitRejectedHandler string            `yaml:"execute.limit.rejected.handler" json:"execute.limit.rejected.handler,omitempty" property:"execute.limit.rejected.handler"`
	Auth                        string            `yaml:"auth" json:"auth,omitempty" property:"auth"`
	NotRegister                 bool              `yaml:"not_register" json:"not_register,omitempty" property:"not_register"`
	ParamSign                   string            `yaml:"param.sign" json:"param.sign,omitempty" property:"param.sign"`
	Tag                         string            `yaml:"tag" json:"tag,omitempty" property:"tag"`
	TracingKey                  string            `yaml:"tracing-key" json:"tracing-key,omitempty" propertiy:"tracing-key"`

	RCProtocolsMap  map[string]*ProtocolConfig
	RCRegistriesMap map[string]*RegistryConfig
	ProxyFactoryKey string
}

type ServiceOption func(*ServiceConfig)

func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		Methods:         make([]*MethodConfig, 0, 8),
		Params:          make(map[string]string, 8),
		RCProtocolsMap:  make(map[string]*ProtocolConfig),
		RCRegistriesMap: make(map[string]*RegistryConfig),
	}
}

// Clone a new ServiceConfig
func (c *ServiceConfig) Clone() *ServiceConfig {
	if c == nil {
		return nil
	}

	newProtocolIDs := make([]string, len(c.ProtocolIDs))
	copy(newProtocolIDs, c.ProtocolIDs)

	newRegistryIDs := make([]string, len(c.RegistryIDs))
	copy(newRegistryIDs, c.RegistryIDs)

	newMethods := make([]*MethodConfig, len(c.Methods))
	for i, v := range c.Methods {
		newMethods[i] = v.Clone()
	}

	newParams := make(map[string]string, len(c.Params))
	for k, v := range c.Params {
		newParams[k] = v
	}

	newRCProtocolsMap := make(map[string]*ProtocolConfig, len(c.RCProtocolsMap))
	for k, v := range c.RCProtocolsMap {
		newRCProtocolsMap[k] = v.Clone()
	}

	newRCRegistriesMap := make(map[string]*RegistryConfig, len(c.RCRegistriesMap))
	for k, v := range c.RCRegistriesMap {
		newRCRegistriesMap[k] = v.Clone()
	}

	return &ServiceConfig{
		Filter:                      c.Filter,
		ProtocolIDs:                 newProtocolIDs,
		Interface:                   c.Interface,
		RegistryIDs:                 newRegistryIDs,
		Cluster:                     c.Cluster,
		Loadbalance:                 c.Loadbalance,
		Group:                       c.Group,
		Version:                     c.Version,
		Methods:                     newMethods,
		Warmup:                      c.Warmup,
		Retries:                     c.Retries,
		Serialization:               c.Serialization,
		Params:                      newParams,
		Token:                       c.Token,
		AccessLog:                   c.AccessLog,
		TpsLimiter:                  c.TpsLimiter,
		TpsLimitInterval:            c.TpsLimitInterval,
		TpsLimitRate:                c.TpsLimitRate,
		TpsLimitStrategy:            c.TpsLimitStrategy,
		TpsLimitRejectedHandler:     c.TpsLimitRejectedHandler,
		ExecuteLimit:                c.ExecuteLimit,
		ExecuteLimitRejectedHandler: c.ExecuteLimitRejectedHandler,
		Auth:                        c.Auth,
		RCProtocolsMap:              newRCProtocolsMap,
		RCRegistriesMap:             newRCRegistriesMap,
		ProxyFactoryKey:             c.ProxyFactoryKey,
		NotRegister:                 c.NotRegister,
		ParamSign:                   c.ParamSign,
		Tag:                         c.Tag,
		TracingKey:                  c.TracingKey,
	}
}
