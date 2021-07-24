package config

import (
	"bytes"
	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// RootConfig is the root config
type RootConfig struct {
	ConfigCenter *CenterConfig `yaml:"config-center" json:"config-center,omitempty"`

	// since 1.5.0 version
	//Remotes              map[string]*config.RemoteConfig `yaml:"remote" json:"remote,omitempty" property:"remote"`

	ServiceDiscoveries map[string]*ServiceDiscoveryConfig `yaml:"service-discovery" json:"service-discovery,omitempty" property:"service_discovery"`

	MetadataReportConfig *MetadataReportConfig `yaml:"metadata_report" json:"metadata-report,omitempty" property:"metadata-report"`

	// Application application config
	Application *ApplicationConfig `yaml:"application" json:"application,omitempty" property:"application"`

	// Registries registry config
	Registries map[string]*RegistryConfig `yaml:"registries" json:"registries" property:"registries"`

	Protocols map[string]*ProtocolConfig `yaml:"protocols" json:"protocols" property:"protocols"`

	// prefix              string
	fatherConfig        interface{}
	EventDispatcherType string        `default:"direct" yaml:"event_dispatcher_type" json:"event_dispatcher_type,omitempty"`
	MetricConfig        *MetricConfig `yaml:"metrics" json:"metrics,omitempty"`
	fileStream          *bytes.Buffer

	// validate
	//Validate *validator.Validate
	// cache file used to store the current used configurations.
	CacheFile string `yaml:"cache_file" json:"cache_file,omitempty" property:"cache_file"`
}

// Prefix dubbo
func (RootConfig) Prefix() string {
	return constant.DUBBO
}

func verification(s interface{}) error {

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

// GetApplicationConfig get application config
func GetApplicationConfig() (*ApplicationConfig, error) {
	if err := check(); err != nil {
		return nil, err
	}
	application := getApplicationConfig(rootConfig.Application)
	if err := defaults.Set(application); err != nil {
		return nil, err
	}

	if err := verification(application); err != nil {
		return nil, err
	}
	rootConfig.Application = application
	return application, nil
}

// GetConfigCenterConfig get config center config
func GetConfigCenterConfig() (*CenterConfig, error) {
	if err := check(); err != nil {
		return nil, err
	}
	centerConfig := getConfigCenterConfig(rootConfig.ConfigCenter)
	if centerConfig == nil {
		return nil, errors.New("config center config is null")
	}
	if err := defaults.Set(centerConfig); err != nil {
		return nil, err
	}
	centerConfig.translateConfigAddress()
	if err := verification(centerConfig); err != nil {
		return nil, err
	}
	return centerConfig, nil
}

// GetRegistriesConfig get registry config default zookeeper registry
func GetRegistriesConfig() (map[string]*RegistryConfig, error) {
	if err := check(); err != nil {
		return nil, err
	}

	registries := getRegistriesConfig(rootConfig.Registries)
	for _, reg := range registries {
		if err := defaults.Set(reg); err != nil {
			return nil, err
		}
		reg.TranslateRegistryAddress()
		if err := verification(reg); err != nil {
			return nil, err
		}
	}
	return registries, nil
}

// GetProtocolsConfig get protocols config default dubbo protocol
func GetProtocolsConfig() (map[string]*ProtocolConfig, error) {
	if err := check(); err != nil {
		return nil, err
	}

	protocols := getProtocolsConfig(rootConfig.Protocols)
	for _, protocol := range protocols {
		if err := defaults.Set(protocol); err != nil {
			return nil, err
		}
		if err := verification(protocol); err != nil {
			return nil, err
		}
	}
	return protocols, nil
}
