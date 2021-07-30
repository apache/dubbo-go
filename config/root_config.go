package config

import (
	"bytes"
	"strings"
)

import (
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

var (
	applicationConfig *ApplicationConfig

	consumerConfig *ConsumerConfig

	providerConfig *ProviderConfig

	registriesConfig map[string]*RegistryConfig
)

// RootConfig is the root config
type RootConfig struct {
	ConfigCenter *CenterConfig `yaml:"config-center" json:"config-center,omitempty"`

	// since 1.5.0 version
	//Remotes              map[string]*config.RemoteConfig `yaml:"remote" json:"remote,omitempty" property:"remote"`

	ServiceDiscoveries map[string]*ServiceDiscoveryConfig `yaml:"service-discovery" json:"service-discovery,omitempty" property:"service-discovery"`

	MetadataReportConfig *MetadataReportConfig `yaml:"metadata_report" json:"metadata-report,omitempty" property:"metadata-report"`

	// Application applicationConfig config
	Application *ApplicationConfig `validate:"required" yaml:"application" json:"application,omitempty" property:"application"`

	// Registries registry config
	Registries map[string]*RegistryConfig `yaml:"registries" json:"registries" property:"registries"`

	Protocols map[string]*ProtocolConfig `validate:"required" yaml:"protocols" json:"protocols" property:"protocols"`

	// provider config
	Provider *ProviderConfig `yaml:"provider" json:"provider" property:"provider"`

	// prefix              string
	fatherConfig        interface{}
	EventDispatcherType string        `default:"direct" yaml:"event_dispatcher_type" json:"event_dispatcher_type,omitempty"`
	MetricConfig        *MetricConfig `yaml:"metrics" json:"metrics,omitempty"`
	fileStream          *bytes.Buffer

	// cache file used to store the current used configurations.
	CacheFile string `yaml:"cache_file" json:"cache_file,omitempty" property:"cache_file"`
}

// Prefix dubbo
func (RootConfig) Prefix() string {
	return constant.DUBBO
}

func verify(s interface{}) error {

	if err := validate.Struct(s); err != nil {
		errs := err.(validator.ValidationErrors)
		var slice []string
		for _, msg := range errs {
			slice = append(slice, msg.Error())
		}
		return errors.New(strings.Join(slice, ","))
	}
	return nil
}

// removeDuplicateElement remove duplicate element
func removeDuplicateElement(items []string) []string {
	result := make([]string, 0, len(items))
	temp := map[string]struct{}{}
	for _, item := range items {
		if _, ok := temp[item]; !ok && item != "" {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// translateRegistryIds string "nacos,zk" => ["nacos","zk"]
func translateRegistryIds(registryIds []string) []string {
	ids := make([]string, 0)
	for _, id := range registryIds {
		ids = append(ids, strings.Split(id, ",")...)
	}
	return removeDuplicateElement(ids)
}

func (rc *RootConfig) Init() {
	rc.Application = getApplicationConfig(rc.Application)
	rc.Protocols = getProtocolsConfig(rc.Protocols)
	rc.Registries = getRegistriesConfig(rc.Registries)
	rc.ConfigCenter = getConfigCenterConfig(rc.ConfigCenter)
}

// GetApplicationConfig get applicationConfig config
//func GetApplicationConfig() (*ApplicationConfig, error) {
//	if err := check(); err != nil {
//		return nil, err
//	}
//	if applicationConfig != nil {
//		return applicationConfig, nil
//	}
//	applicationConfig = getApplicationConfig(rootConfig.Application)
//	if err := defaults.Set(applicationConfig); err != nil {
//		return nil, err
//	}
//
//	if err := verify(applicationConfig); err != nil {
//		return nil, err
//	}
//
//	return applicationConfig, nil
//}

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

// getRegistryIds get registry keys
func getRegistryIds() []string {
	ids := make([]string, 0)
	for key := range rootConfig.Registries {
		ids = append(ids, key)
	}
	return removeDuplicateElement(ids)
}
