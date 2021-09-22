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
	_ "net/http/pprof"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// RootConfig is the root config
type RootConfig struct {
	// Application applicationConfig config
	Application *ApplicationConfig `validate:"required" yaml:"application" json:"application,omitempty" property:"application"`

	Protocols map[string]*ProtocolConfig `validate:"required" yaml:"protocols" json:"protocols" property:"protocols"`

	// Registries registry config
	Registries map[string]*RegistryConfig `yaml:"registries" json:"registries" property:"registries"`

	// Remotes to be remove in 3.0 config-enhance
	Remotes map[string]*RemoteConfig `yaml:"remote" json:"remote,omitempty" property:"remote"`

	ConfigCenter *CenterConfig `yaml:"config-center" json:"config-center,omitempty"`

	// ServiceDiscoveries to be remove in 3.0 config-enhance
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

	Router []*RouterConfig `yaml:"router" json:"router,omitempty" property:"router"`

	EventDispatcherType string `default:"direct" yaml:"event-dispatcher-type" json:"event-dispatcher-type,omitempty"`

	// cache file used to store the current used configurations.
	CacheFile string `yaml:"cache_file" json:"cache_file,omitempty" property:"cache_file"`
}

func SetRootConfig(r RootConfig) {
	rootConfig = &r
}

// Prefix dubbo
func (RootConfig) Prefix() string {
	return constant.DUBBO
}

func GetRootConfig() *RootConfig {
	return rootConfig
}

func GetProviderConfig() *ProviderConfig {
	if err := check(); err != nil {
		return GetProviderInstance()
	}
	if rootConfig.Provider != nil {
		return rootConfig.Provider
	}
	return GetProviderInstance()
}

func GetConsumerConfig() *ConsumerConfig {
	if err := check(); err != nil {
		return GetConsumerInstance()
	}
	if rootConfig.Consumer != nil {
		return rootConfig.Consumer
	}
	return GetConsumerInstance()
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

// getRegistryIds get registry ids
func (rc *RootConfig) getRegistryIds() []string {
	ids := make([]string, 0)
	for key := range rc.Registries {
		ids = append(ids, key)
	}
	return removeDuplicateElement(ids)
}

// NewRootConfig get root config
func NewRootConfig(opts ...RootConfigOpt) *RootConfig {
	newRootConfig := &RootConfig{
		ConfigCenter:         &CenterConfig{},
		ServiceDiscoveries:   make(map[string]*ServiceDiscoveryConfig),
		MetadataReportConfig: &MetadataReportConfig{},
		Application:          &ApplicationConfig{},
		Registries:           make(map[string]*RegistryConfig),
		Protocols:            make(map[string]*ProtocolConfig),
		Provider:             GetProviderInstance(),
		Consumer:             GetConsumerInstance(),
		MetricConfig:         &MetricConfig{},
	}
	for _, o := range opts {
		o(newRootConfig)
	}
	return newRootConfig
}

type RootConfigOpt func(config *RootConfig)

// WithMetricsConfig set root config with given @metricsConfig
func WithMetricsConfig(metricsConfig *MetricConfig) RootConfigOpt {
	return func(rc *RootConfig) {
		rc.MetricConfig = metricsConfig
	}
}

// WithRootConsumerConfig set root config with given @consumerConfig
func WithRootConsumerConfig(consumerConfig *ConsumerConfig) RootConfigOpt {
	return func(rc *RootConfig) {
		rc.Consumer = consumerConfig
	}
}

// WithRootProviderConfig set root config with given @providerConfig
func WithRootProviderConfig(providerConfig *ProviderConfig) RootConfigOpt {
	return func(rc *RootConfig) {
		rc.Provider = providerConfig
	}
}

// WithRootProtocolConfig set root config with key @protocolName and given @protocolConfig
func WithRootProtocolConfig(protocolName string, protocolConfig *ProtocolConfig) RootConfigOpt {
	return func(rc *RootConfig) {
		rc.Protocols[protocolName] = protocolConfig
	}
}

// WithRootRegistryConfig set root config with key @registryKey and given @regConfig
func WithRootRegistryConfig(registryKey string, regConfig *RegistryConfig) RootConfigOpt {
	return func(rc *RootConfig) {
		rc.Registries[registryKey] = regConfig
	}
}

// WithRootApplicationConfig set root config with given @appConfig
func WithRootApplicationConfig(appConfig *ApplicationConfig) RootConfigOpt {
	return func(rc *RootConfig) {
		rc.Application = appConfig
	}
}

// WithRootMetadataReportConfig set root config with given @metadataReportConfig
func WithRootMetadataReportConfig(metadataReportConfig *MetadataReportConfig) RootConfigOpt {
	return func(rc *RootConfig) {
		rc.MetadataReportConfig = metadataReportConfig
	}
}

// WithRootServiceDiscoverConfig set root config with given @serviceDiscConfig and key @name
func WithRootServiceDiscoverConfig(name string, serviceDiscConfig *ServiceDiscoveryConfig) RootConfigOpt {
	return func(rc *RootConfig) {
		rc.ServiceDiscoveries[name] = serviceDiscConfig
	}
}

// WithRootCenterConfig set root config with given centerConfig
func WithRootCenterConfig(centerConfig *CenterConfig) RootConfigOpt {
	return func(rc *RootConfig) {
		rc.ConfigCenter = centerConfig
	}
}
