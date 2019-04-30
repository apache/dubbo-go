package support

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

import (
	"github.com/AlexStocks/goext/log"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/dubbo/dubbo-go/common/constant"
)

var (
	consumerConfig *ConsumerConfig
	providerConfig *ProviderConfig
)

// loaded comsumer & provider config from xxx.yml, and log config from xxx.xml
// Namely: dubbo.comsumer.xml & dubbo.provider.xml in java dubbo
func init() {

	if err := logInit(); err != nil { // log config
		log.Warn("[logInit] %#v", err)
	}

	var (
		confConFile, confProFile string
	)

	confConFile = os.Getenv(constant.CONF_CONSUMER_FILE_PATH)
	confProFile = os.Getenv(constant.CONF_PROVIDER_FILE_PATH)

	if errCon := consumerInit(confConFile); errCon != nil {
		log.Warn("[consumerInit] %#v", errCon)
		consumerConfig = nil
	}
	if errPro := providerInit(confProFile); errPro != nil {
		log.Warn("[providerInit] %#v", errPro)
		providerConfig = nil
	}

}

func logInit() error {
	var (
		confFile string
	)

	confFile = os.Getenv(constant.APP_LOG_CONF_FILE)
	if confFile == "" {
		return fmt.Errorf("log configure file name is nil")
	}
	if path.Ext(confFile) != ".xml" {
		return fmt.Errorf("log configure file name{%v} suffix must be .xml", confFile)
	}

	log.LoadConfiguration(confFile)

	return nil
}

func consumerInit(confConFile string) error {
	if confConFile == "" {
		return fmt.Errorf("application configure(consumer) file name is nil")
	}

	if path.Ext(confConFile) != ".yml" {
		return fmt.Errorf("application configure file name{%v} suffix must be .yml", confConFile)
	}

	confFileStream, err := ioutil.ReadFile(confConFile)
	if err != nil {
		return fmt.Errorf("ioutil.ReadFile(file:%s) = error:%s", confConFile, jerrors.ErrorStack(err))
	}
	err = yaml.Unmarshal(confFileStream, consumerConfig)
	if err != nil {
		return fmt.Errorf("yaml.Unmarshal() = error:%s", jerrors.ErrorStack(err))
	}
	//动态加载service config  end
	for _, config := range consumerConfig.Registries {
		if config.Timeout, err = time.ParseDuration(config.TimeoutStr); err != nil {
			return fmt.Errorf("time.ParseDuration(Registry_Config.Timeout:%#v) = error:%s", config.TimeoutStr, err)
		}
	}

	gxlog.CInfo("consumer config{%#v}\n", consumerConfig)
	return nil
}

func providerInit(confProFile string) error {
	if confProFile == "" {
		return fmt.Errorf("application configure(provider) file name is nil")
	}

	if path.Ext(confProFile) != ".yml" {
		return fmt.Errorf("application configure file name{%v} suffix must be .yml", confProFile)
	}

	confFileStream, err := ioutil.ReadFile(confProFile)
	if err != nil {
		return fmt.Errorf("ioutil.ReadFile(file:%s) = error:%s", confProFile, jerrors.ErrorStack(err))
	}
	err = yaml.Unmarshal(confFileStream, providerConfig)
	if err != nil {
		return fmt.Errorf("yaml.Unmarshal() = error:%s", jerrors.ErrorStack(err))
	}

	//todo: provider config

	gxlog.CInfo("provider config{%#v}\n", providerConfig)
	return nil
}

/////////////////////////
// consumerConfig
/////////////////////////

type ConsumerConfig struct {
	// pprof
	Pprof_Enabled bool `default:"false" yaml:"pprof_enabled" json:"pprof_enabled,omitempty"`
	Pprof_Port    int  `default:"10086"  yaml:"pprof_port" json:"pprof_port,omitempty"`

	// client
	Connect_Timeout string `default:"100ms"  yaml:"connect_timeout" json:"connect_timeout,omitempty"`
	ConnectTimeout  time.Duration

	Request_Timeout string `yaml:"request_timeout" default:"5s" json:"request_timeout,omitempty"` // 500ms, 1m
	RequestTimeout  time.Duration

	// codec & selector & transport & registry
	Selector     string `default:"cache"  yaml:"selector" json:"selector,omitempty"`
	Selector_TTL string `default:"10m"  yaml:"selector_ttl" json:"selector_ttl,omitempty"`
	// application
	ApplicationConfig ApplicationConfig `yaml:"application_config" json:"application_config,omitempty"`
	Registries        []RegistryConfig  `yaml:"registries" json:"registries,omitempty"`
	References        []ReferenceConfig `yaml:"references" json:"references,omitempty"`
}

type ReferenceConfigTmp struct {
	Service    string           `required:"true"  yaml:"service"  json:"service,omitempty"`
	Registries []RegistryConfig `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	URLs       []map[string]string
}

func SetConsumerConfig(c ConsumerConfig) {
	consumerConfig = &c
}
func GetConsumerConfig() ConsumerConfig {
	return *consumerConfig
}

/////////////////////////
// providerConfig
/////////////////////////

type ProviderConfig struct {
	ApplicationConfig ApplicationConfig `yaml:"application_config" json:"application_config,omitempty"`
	Path              string            `yaml:"path" json:"path,omitempty"`
	Registries        []RegistryConfig  `yaml:"registries" json:"registries,omitempty"`
	Services          []ServiceConfig   `yaml:"services" json:"services,omitempty"`
	Protocols         []ProtocolConfig  `yaml:"protocols" json:"protocols,omitempty"`
}

func SetProviderConfig(p ProviderConfig) {
	providerConfig = &p
}
func GetProviderConfig() ProviderConfig {
	return *providerConfig
}


type ProtocolConfig struct {
	name        string `required:"true" yaml:"name"  json:"name,omitempty"`
	ip          string `required:"true" yaml:"ip"  json:"ip,omitempty"`
	port        string `required:"true" yaml:"port"  json:"port,omitempty"`
	contextPath string `required:"true" yaml:"contextPath"  json:"contextPath,omitempty"`
}

func loadProtocol(protocolsIds string, protocols []ProtocolConfig) []ProtocolConfig {
	returnProtocols := []ProtocolConfig{}
	for _, v := range strings.Split(protocolsIds, ",") {
		for _, prot := range protocols {
			if v == prot.name {
				returnProtocols = append(returnProtocols, prot)
			}
		}

	}
	return returnProtocols
}

// Dubbo Init
func Load() (map[string]*ReferenceConfig, map[string]*ServiceConfig) {
	refMap := make(map[string]*ReferenceConfig)
	srvMap := make(map[string]*ServiceConfig)

	// reference config
	length := len(consumerConfig.References)
	for index := 0; index < length; index++ {
		con := &consumerConfig.References[index]
		con.Implement(services[con.Interface])
		con.Refer()
		refMap[con.Interface] = con
	}

	// service config
	length = len(providerConfig.Services)
	for index := 0; index < length; index++ {
		pro := &providerConfig.Services[index]
		pro.Implement(services[pro.Interface])
		pro.Export()
		srvMap[pro.Interface] = pro
	}

	return refMap, srvMap
}
