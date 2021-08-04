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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// RootConfig is the root config
type RootConfig struct {
	// Application applicationConfig config
	Application *ApplicationConfig `validate:"required" yaml:"application" json:"application,omitempty" property:"application"`

	Protocols map[string]*ProtocolConfig `validate:"required" yaml:"protocols" json:"protocols" property:"protocols"`

	// Registries registry config
	Registries map[string]*RegistryConfig `yaml:"registries" json:"registries" property:"registries"`

	// Deprecated since 1.5.0 version
	Remotes map[string]*RemoteConfig `yaml:"remote" json:"remote,omitempty" property:"remote"`

	ConfigCenter *CenterConfig `yaml:"config-center" json:"config-center,omitempty"`

	ServiceDiscoveries map[string]*ServiceDiscoveryConfig `yaml:"service-discovery" json:"service-discovery,omitempty" property:"service-discovery"`

	MetadataReportConfig *MetadataReportConfig `yaml:"metadata-report" json:"metadata-report,omitempty" property:"metadata-report"`

	// provider config
	Provider *ProviderConfig `yaml:"provider" json:"provider" property:"provider"`

	// consumer config
	Consumer *ConsumerConfig `yaml:"consumer" json:"consumer" property:"consumer"`

	MetricConfig *MetricConfig `yaml:"metrics" json:"metrics,omitempty" property:"metrics"`

	// Logger log
	Logger *LoggerConfig `yaml:"logger" json:"logger,omitempty" property:"logger"`

	// Shutdown config
	Shutdown *ShutdownConfig `yaml:"shutdown" json:"shutdown,omitempty" property:"shutdown"`

	// Deprecated
	Network map[interface{}]interface{} `yaml:"network" json:"network,omitempty" property:"network"`

	Router []*RouterConfig `yaml:"router" json:"router,omitempty" property:"router"`
	// is refresh action
	refresh bool
	// prefix              string
	fatherConfig        interface{}
	EventDispatcherType string `default:"direct" yaml:"event_dispatcher_type" json:"event_dispatcher_type,omitempty"`
	fileStream          *bytes.Buffer

	// cache file used to store the current used configurations.
	CacheFile string `yaml:"cache_file" json:"cache_file,omitempty" property:"cache_file"`
}

func init() {
	rootConfig = NewRootConfig()
}

func SetRootConfig(r RootConfig) {
	rootConfig = &r
}

func NewRootConfig() *RootConfig {
	return &RootConfig{
		ConfigCenter:         &CenterConfig{},
		ServiceDiscoveries:   make(map[string]*ServiceDiscoveryConfig),
		MetadataReportConfig: &MetadataReportConfig{},
		Application:          &ApplicationConfig{},
		Registries:           make(map[string]*RegistryConfig),
		Protocols:            make(map[string]*ProtocolConfig),
		Provider:             NewProviderConfig(),
		Consumer:             NewConsumerConfig(),
		MetricConfig:         &MetricConfig{},
	}
}

type rootConfOption interface {
	apply(vc *RootConfig)
}

type RootConfFunc func(*RootConfig)

func (fn RootConfFunc) apply(vc *RootConfig) {
	fn(vc)
}

// Prefix dubbo
func (RootConfig) Prefix() string {
	return constant.DUBBO
}

// InitConfig init config
func (rc *RootConfig) InitConfig(opts ...rootConfOption) error {
	for _, opt := range opts {
		opt.apply(rc)
	}
	if rc.ConfigCenter != nil && !rc.refresh {
		if err := startConfigCenter(rc); err != nil {
			return err
		}
	}
	if err := initApplicationConfig(rc); err != nil {
		return err
	}
	if err := initProtocolsConfig(rc); err != nil {
		return err
	}
	if err := initRegistriesConfig(rc); err != nil {
		return err
	}
	if err := initLoggerConfig(rc); err != nil {
		return err
	}
	if err := initServiceDiscoveryConfig(rc); err != nil {
		return err
	}
	if err := initMetadataReportConfig(rc); err != nil {
		return err
	}
	if err := initMetricConfig(rc); err != nil {
		return err
	}
	if err := initNetworkConfig(rc); err != nil {
		return err
	}
	if err := initRouterConfig(rc); err != nil {
		return err
	}
	// provider、consumer must last init
	if err := initProviderConfig(rc); err != nil {
		return err
	}
	if err := initConsumerConfig(rc); err != nil {
		return err
	}
	return nil
}

func WithApplication(ac *ApplicationConfig) RootConfFunc {
	return RootConfFunc(func(conf *RootConfig) {
		conf.Application = ac
	})
}

func WithProtocols(protocols map[string]*ProtocolConfig) RootConfFunc {
	return RootConfFunc(func(conf *RootConfig) {
		conf.Protocols = protocols
	})
}

//func (rc *RootConfig) CheckConfig() error {
//	defaults.MustSet(rc)
//
//	if err := rc.Application.CheckConfig(); err != nil {
//		return err
//	}
//
//	for k, _ := range rc.Registries {
//		if err := rc.Registries[k].CheckConfig(); err != nil {
//			return err
//		}
//	}
//
//	for k, _ := range rc.Protocols {
//		if err := rc.Protocols[k].CheckConfig(); err != nil {
//			return err
//		}
//	}
//
//	if err := rc.ConfigCenter.CheckConfig(); err != nil {
//		return err
//	}
//
//	if err := rc.MetadataReportConfig.CheckConfig(); err != nil {
//		return err
//	}
//
//	if err := rc.Provider.CheckConfig(); err != nil {
//		return err
//	}
//
//	if err := rc.Consumer.CheckConfig(); err != nil {
//		return err
//	}
//
//	return verify(rootConfig)
//}

//func (rc *RootConfig) Validate() {
//	// 2. validate config
//	rc.Application.Validate()
//
//	for k, _ := range rc.Registries {
//		rc.Registries[k].Validate()
//	}
//
//	for k, _ := range rc.Protocols {
//		rc.Protocols[k].Validate()
//	}
//
//	for k, _ := range rc.Registries {
//		rc.Registries[k].Validate()
//	}
//
//	rc.ConfigCenter.Validate()
//	rc.MetadataReportConfig.Validate()
//	rc.Provider.Validate(rc)
//	rc.Consumer.Validate(rc)
//}

//GetApplicationConfig get applicationConfig config

func GetRootConfig() *RootConfig {
	return rootConfig
}

func GetProviderConfig() *ProviderConfig {
	if err := check(); err != nil {
		return NewProviderConfig()
	}
	if rootConfig.Provider != nil {
		return rootConfig.Provider
	}
	return NewProviderConfig()
}

func GetConsumerConfig() *ConsumerConfig {
	if err := check(); err != nil {
		return NewConsumerConfig()
	}
	if rootConfig.Consumer != nil {
		return rootConfig.Consumer
	}
	return NewConsumerConfig()
}

func GetApplicationConfig() *ApplicationConfig {
	return rootConfig.Application
}

// GetConfigCenterConfig get config center config
//func GetConfigCenterConfig() (*CenterConfig, error) {
//	if err := check(); err != nil {
//		return nil, err
//	}
//	conf := rootConfig.ConfigCenter
//	if conf == nil {
//		return nil, errors.New("config center config is null")
//	}
//	if err := defaults.Set(conf); err != nil {
//		return nil, err
//	}
//	conf.translateConfigAddress()
//	if err := verify(conf); err != nil {
//		return nil, err
//	}
//	return conf, nil
//}

// GetRegistriesConfig get registry config default zookeeper registry
//func GetRegistriesConfig() (map[string]*RegistryConfig, error) {
//	if err := check(); err != nil {
//		return nil, err
//	}
//
//	if registriesConfig != nil {
//		return registriesConfig, nil
//	}
//	registriesConfig = initRegistriesConfig(rootConfig.Registries)
//	for _, reg := range registriesConfig {
//		if err := defaults.Set(reg); err != nil {
//			return nil, err
//		}
//		reg.translateRegistryAddress()
//		if err := verify(reg); err != nil {
//			return nil, err
//		}
//	}
//
//	return registriesConfig, nil
//}

// GetProtocolsConfig get protocols config default dubbo protocol
//func GetProtocolsConfig() (map[string]*ProtocolConfig, error) {
//	if err := check(); err != nil {
//		return nil, err
//	}
//
//	protocols := getProtocolsConfig(rootConfig.Protocols)
//	for _, protocol := range protocols {
//		if err := defaults.Set(protocol); err != nil {
//			return nil, err
//		}
//		if err := verify(protocol); err != nil {
//			return nil, err
//		}
//	}
//	return protocols, nil
//}

// GetProviderConfig get provider config
//func GetProviderConfig() (*ProviderConfig, error) {
//	if err := check(); err != nil {
//		return nil, err
//	}
//
//	if providerConfig != nil {
//		return providerConfig, nil
//	}
//	provider := getProviderConfig(rootConfig.Provider)
//	if err := defaults.Set(provider); err != nil {
//		return nil, err
//	}
//	if err := verify(provider); err != nil {
//		return nil, err
//	}
//
//	provider.Services = getRegistryServices(common.PROVIDER, provider.Services, provider.Registry)
//	providerConfig = provider
//	return provider, nil
//}

//// getRegistryIds get registry keys
//func getRegistryIds() []string {
//	ids := make([]string, 0)
//	for key := range rootConfig.Registries {
//		ids = append(ids, key)
//	}
//	return removeDuplicateElement(ids)
//}
