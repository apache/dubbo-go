package config

import (
	"bytes"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"github.com/creasty/defaults"
)

// RootConfig is the root config
type RootConfig struct {
	ConfigCenter *CenterConfig `yaml:"config-center" json:"config-center,omitempty"`

	// since 1.5.0 version
	Remotes map[string]*RemoteConfig `yaml:"remote" json:"remote,omitempty" property:"remote"`

	ServiceDiscoveries map[string]*ServiceDiscoveryConfig `yaml:"service-discovery" json:"service-discovery,omitempty" property:"service-discovery"`

	MetadataReportConfig *MetadataReportConfig `yaml:"metadata_report" json:"metadata-report,omitempty" property:"metadata-report"`

	// Application applicationConfig config
	Application *ApplicationConfig `validate:"required" yaml:"application" json:"application,omitempty" property:"application"`

	// Registries registry config
	Registries map[string]*RegistryConfig `yaml:"registries" json:"registries" property:"registries"`

	Protocols map[string]*ProtocolConfig `validate:"required" yaml:"protocols" json:"protocols" property:"protocols"`

	// provider config
	Provider *ProviderConfig `yaml:"provider" json:"provider" property:"provider"`

	// consumer config
	Consumer *ConsumerConfig `yaml:"consumer" json:"consumer" property:"consumer"`

	// prefix              string
	fatherConfig        interface{}
	EventDispatcherType string        `default:"direct" yaml:"event_dispatcher_type" json:"event_dispatcher_type,omitempty"`
	MetricConfig        *MetricConfig `yaml:"metrics" json:"metrics,omitempty"`
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

// Prefix dubbo
func (RootConfig) Prefix() string {
	return constant.DUBBO
}

func (rc *RootConfig) CheckConfig() error {
	defaults.MustSet(rc)

	if err := rc.Application.CheckConfig(); err != nil {
		return err
	}

	for k, _ := range rc.Registries {
		if err := rc.Registries[k].CheckConfig(); err != nil {
			return err
		}
	}

	for k, _ := range rc.Protocols {
		if err := rc.Protocols[k].CheckConfig(); err != nil {
			return err
		}
	}

	if err := rc.ConfigCenter.CheckConfig(); err != nil {
		return err
	}

	if err := rc.MetadataReportConfig.CheckConfig(); err != nil {
		return err
	}

	if err := rc.Provider.CheckConfig(); err != nil {
		return err
	}

	if err := rc.Consumer.CheckConfig(); err != nil {
		return err
	}

	return verify(rootConfig)
}

func (rc *RootConfig) Validate() {
	// 2. validate config
	rc.Application.Validate()

	for k, _ := range rc.Registries {
		rc.Registries[k].Validate()
	}

	for k, _ := range rc.Protocols {
		rc.Protocols[k].Validate()
	}

	for k, _ := range rc.Registries {
		rc.Registries[k].Validate()
	}

	rc.ConfigCenter.Validate()
	rc.MetadataReportConfig.Validate()
	rc.Provider.Validate(rc)
	rc.Consumer.Validate(rc)
}

//GetApplicationConfig get applicationConfig config
func GetApplicationConfig() *ApplicationConfig {
	if err := check(); err != nil {
		return NewApplicationConfig()
	}
	if rootConfig.Application != nil {
		return rootConfig.Application
	}
	return NewApplicationConfig()
}

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
//	registriesConfig = getRegistriesConfig(rootConfig.Registries)
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
