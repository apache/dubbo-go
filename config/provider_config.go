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
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"fmt"
	"github.com/creasty/defaults"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// ProviderConfig is the default configuration of service provider
type ProviderConfig struct {
	//base.ShutdownConfig         `yaml:",inline" property:"base"`
	//center.configCenter `yaml:"-"`
	Filter string `yaml:"filter" json:"filter,omitempty" property:"filter"`
	// Register whether registration is required
	Register bool `yaml:"register" json:"register" property:"register"`
	// Registry registry ids
	Registry []string `yaml:"registry" json:"registry" property:"registry"`
	// Services services
	Services map[string]*ServiceConfig `yaml:"services" json:"services,omitempty" property:"services"`

	ProxyFactory   string                     `default:"default" yaml:"proxy-factory" json:"proxy-factory,omitempty" property:"proxy-factory"`
	Protocols      map[string]*ProtocolConfig `yaml:"protocols" json:"protocols,omitempty" property:"protocols"`
	ProtocolConf   interface{}                `yaml:"protocol_conf" json:"protocol_conf,omitempty" property:"protocol_conf"`
	FilterConf     interface{}                `yaml:"filter_conf" json:"filter_conf,omitempty" property:"filter_conf"`
	ShutdownConfig *ShutdownConfig            `yaml:"shutdown_conf" json:"shutdown_conf,omitempty" property:"shutdown_conf"`
	ConfigType     map[string]string          `yaml:"config_type" json:"config_type,omitempty" property:"config_type"`
}

func (c *ProviderConfig) CheckConfig() error {
	// todo check
	defaults.MustSet(c)
	return verify(c)
}

func (c *ProviderConfig) Validate(r *RootConfig) {
	ids := make([]string, 0)
	for key := range r.Registries {
		ids = append(ids, key)
	}
	c.Registry = removeDuplicateElement(ids)
	for k, _ := range c.Services {
		c.Services[k].Validate(r)
	}
	// todo set default application
}

// UnmarshalYAML unmarshals the ProviderConfig by @unmarshal function
func (c *ProviderConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain ProviderConfig
	return unmarshal((*plain)(c))
}

// Prefix dubbo.provider
func (c *ProviderConfig) Prefix() string {
	return constant.ProviderConfigPrefix
}

func (c *ProviderConfig) Load() {
	// todo Write the current configuration to cache file.
	//if c.CacheFile != "" {
	//	if data, err := yaml.MarshalYML(providerConfig); err != nil {
	//		logger.Errorf("Marshal provider config err: %s", err.Error())
	//	} else {
	//		if err := ioutil.WriteFile(provider  CacheFile, data, 0666); err != nil {
	//			logger.Errorf("Write provider config cache file err: %s", err.Error())
	//		}
	//	}
	//}

	for key, svs := range c.Services {
		rpcService := GetProviderService(key)
		if rpcService == nil {
			logger.Warnf("%s does not exist!", key)
			continue
		}
		svs.id = key
		svs.Implement(rpcService)
		if err := svs.Export(); err != nil {
			panic(fmt.Sprintf("service %s export failed! err: %#v", key, err))
		}
	}
}

// SetProviderConfig sets provider config by @p
func SetProviderConfig(p ProviderConfig) {
	rootConfig.Provider = &p
}

//
//// ProviderInit loads config file to init provider config
//func ProviderInit(confProFile string) error {
//	if len(confProFile) == 0 {
//		return perrors.Errorf("applicationConfig configure(provider) file name is nil")
//	}
//	  providerConfig = &ProviderConfig{}
//	fileStream, err := yaml.UnmarshalYMLConfig(confProFile,   providerConfig)
//	if err != nil {
//		return perrors.Errorf("unmarshalYmlConfig error %v", perrors.WithStack(err))
//	}
//
//	  provider  fileStream = bytes.NewBuffer(fileStream)
//	// set method interfaceId & interfaceName
//	for k, v := range   provider  Services {
//		// set id for reference
//		for _, n := range   provider  Services[k].Methods {
//			n.InterfaceName = v.InterfaceName
//			n.InterfaceId = k
//		}
//	}
//
//	return nil
//}
//
//func configCenterRefreshProvider() error {
//	// fresh it
//	if   provider  ConfigCenterConfig != nil {
//		  provider  fatherConfig =   providerConfig
//		if err :=   provider  startConfigCenter((*  providerConfig).BaseConfig); err != nil {
//			return perrors.Errorf("start config center error , error message is {%v}", perrors.WithStack(err))
//		}
//		  provider  fresh()
//	}
//	return nil
//}

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

// WithProviderServices returns ProviderConfig with given serviceNameKey @serviceName and @serviceConfig
func WithProviderServices(serviceName string, serviceConfig *ServiceConfig) ProviderConfigOpt {
	return func(config *ProviderConfig) *ProviderConfig {
		config.Services[serviceName] = serviceConfig
		return config
	}
}

// WithProviderRegistry returns ProviderConfigOpt with given @registryKey and registry @registryConfig
func WithProviderRegistry(registryKey ...string) ProviderConfigOpt {
	return func(config *ProviderConfig) *ProviderConfig {
		config.Registry = append(config.Registry, registryKey...)
		return config
	}
}
