package config

import (
	"bytes"
	"dubbo.apache.org/dubbo-go/v3/config/protocol"
	"dubbo.apache.org/dubbo-go/v3/config/registry"
)

import (
	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config/application"
	"dubbo.apache.org/dubbo-go/v3/config/center"
	"dubbo.apache.org/dubbo-go/v3/config/medadata/report"
	"dubbo.apache.org/dubbo-go/v3/config/metric"
)

// RootConfig is the root config
type RootConfig struct {
	ConfigCenter *center.Config `yaml:"config-center" json:"config-center,omitempty"`

	// since 1.5.0 version
	//Remotes              map[string]*config.RemoteConfig `yaml:"remote" json:"remote,omitempty" property:"remote"`

	ServiceDiscoveries map[string]*ServiceDiscoveryConfig `yaml:"service-discovery" json:"service-discovery,omitempty" property:"service_discovery"`

	MetadataReportConfig *report.Config `yaml:"metadata_report" json:"metadata-report,omitempty" property:"metadata-report"`

	// Application application config
	Application *application.Config `yaml:"application" json:"application,omitempty" property:"application"`

	// Registries registry config
	Registries map[string]*registry.Config `yaml:"registries" json:"registries" property:"registries"`

	Protocols map[string]*protocol.Config `yaml:"protocols" json:"protocols" property:"protocols"`

	// prefix              string
	fatherConfig        interface{}
	EventDispatcherType string               `default:"direct" yaml:"event_dispatcher_type" json:"event_dispatcher_type,omitempty"`
	MetricConfig        *metric.MetricConfig `yaml:"metrics" json:"metrics,omitempty"`
	fileStream          *bytes.Buffer

	Koanf *koanf.Koanf
	// validate
	Validate *validator.Validate
	// cache file used to store the current used configurations.
	CacheFile string `yaml:"cache_file" json:"cache_file,omitempty" property:"cache_file"`
}

// Prefix dubbo
func (RootConfig) Prefix() string {
	return constant.DUBBO
}
