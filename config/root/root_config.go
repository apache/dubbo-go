package root

import (
	"bytes"
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
	"dubbo.apache.org/dubbo-go/v3/config/service/discovery"
)

// Config is the root config
type Config struct {
	ConfigCenterConfig *center.Config `yaml:"config_center" json:"config_center,omitempty"`

	// since 1.5.0 version
	//Remotes              map[string]*config.RemoteConfig `yaml:"remote" json:"remote,omitempty" property:"remote"`

	ServiceDiscoveries map[string]*discovery.Config `yaml:"service_discovery" json:"service_discovery,omitempty" property:"service_discovery"`

	MetadataReportConfig *report.Config `yaml:"metadata_report" json:"metadata_report,omitempty" property:"metadata_report"`

	// Application application config
	Application *application.Config `yaml:"application" json:"application,omitempty" property:"application"`

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
func (Config) Prefix() string {
	return constant.DUBBO
}