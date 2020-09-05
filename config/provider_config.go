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
	"bytes"
)

import (
	"github.com/creasty/defaults"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/yaml"
)

/////////////////////////
// providerConfig
/////////////////////////

// ProviderConfig is the default configuration of service provider
type ProviderConfig struct {
	BaseConfig `yaml:",inline"`
	configCenter
	Filter         string                     `yaml:"filter" json:"filter,omitempty" property:"filter"`
	ProxyFactory   string                     `yaml:"proxy_factory" default:"default" json:"proxy_factory,omitempty" property:"proxy_factory"`
	Services       map[string]*ServiceConfig  `yaml:"services" json:"services,omitempty" property:"services"`
	Protocols      map[string]*ProtocolConfig `yaml:"protocols" json:"protocols,omitempty" property:"protocols"`
	ProtocolConf   interface{}                `yaml:"protocol_conf" json:"protocol_conf,omitempty" property:"protocol_conf"`
	FilterConf     interface{}                `yaml:"filter_conf" json:"filter_conf,omitempty" property:"filter_conf"`
	ShutdownConfig *ShutdownConfig            `yaml:"shutdown_conf" json:"shutdown_conf,omitempty" property:"shutdown_conf"`
	ConfigType     map[string]string          `yaml:"config_type" json:"config_type,omitempty" property:"config_type"`

	Registry   *RegistryConfig            `yaml:"registry" json:"registry,omitempty" property:"registry"`
	Registries map[string]*RegistryConfig `default:"{}" yaml:"registries" json:"registries" property:"registries"`
}

// UnmarshalYAML unmarshals the ProviderConfig by @unmarshal function
func (c *ProviderConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain ProviderConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// nolint
func (*ProviderConfig) Prefix() string {
	return constant.ProviderConfigPrefix
}

// SetProviderConfig sets provider config by @p
func SetProviderConfig(p ProviderConfig) {
	providerConfig = &p
}

// ProviderInit loads config file to init provider config
func ProviderInit(confProFile string) error {
	if len(confProFile) == 0 {
		return perrors.Errorf("application configure(provider) file name is nil")
	}
	providerConfig = &ProviderConfig{}
	fileStream, err := yaml.UnmarshalYMLConfig(confProFile, providerConfig)
	if err != nil {
		return perrors.Errorf("unmarshalYmlConfig error %v", perrors.WithStack(err))
	}

	providerConfig.fileStream = bytes.NewBuffer(fileStream)
	// set method interfaceId & interfaceName
	for k, v := range providerConfig.Services {
		// set id for reference
		for _, n := range providerConfig.Services[k].Methods {
			n.InterfaceName = v.InterfaceName
			n.InterfaceId = k
		}
	}

	return nil
}

func configCenterRefreshProvider() error {
	// fresh it
	if providerConfig.ConfigCenterConfig != nil {
		providerConfig.fatherConfig = providerConfig
		if err := providerConfig.startConfigCenter((*providerConfig).BaseConfig); err != nil {
			return perrors.Errorf("start config center error , error message is {%v}", perrors.WithStack(err))
		}
		providerConfig.fresh()
	}
	return nil
}
