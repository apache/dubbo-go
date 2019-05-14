package config

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
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
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
	consumerConfig = &ConsumerConfig{}
	err = yaml.Unmarshal(confFileStream, consumerConfig)
	if err != nil {
		return fmt.Errorf("yaml.Unmarshal() = error:%s", jerrors.ErrorStack(err))
	}

	if consumerConfig.RequestTimeout, err = time.ParseDuration(consumerConfig.Request_Timeout); err != nil {
		return jerrors.Annotatef(err, "time.ParseDuration(Request_Timeout{%#v})", consumerConfig.Request_Timeout)
	}
	if consumerConfig.ConnectTimeout, err = time.ParseDuration(consumerConfig.Connect_Timeout); err != nil {
		return jerrors.Annotatef(err, "time.ParseDuration(Connect_Timeout{%#v})", consumerConfig.Connect_Timeout)
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
	providerConfig = &ProviderConfig{}
	err = yaml.Unmarshal(confFileStream, providerConfig)
	if err != nil {
		return fmt.Errorf("yaml.Unmarshal() = error:%s", jerrors.ErrorStack(err))
	}

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

	Request_Timeout string `yaml:"request_timeout" default:"5s" json:"request_timeout,omitempty"`
	RequestTimeout  time.Duration

	// codec & selector & transport & registry
	Selector     string `default:"cache"  yaml:"selector" json:"selector,omitempty"`
	Selector_TTL string `default:"10m"  yaml:"selector_ttl" json:"selector_ttl,omitempty"`
	// application
	ApplicationConfig ApplicationConfig `yaml:"application_config" json:"application_config,omitempty"`
	Registries        []RegistryConfig  `yaml:"registries" json:"registries,omitempty"`
	References        []ReferenceConfig `yaml:"references" json:"references,omitempty"`
	ProtocolConf      interface{}       `yaml:"protocol_conf" json:"protocol_conf,omitempty"`
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
	if consumerConfig == nil {
		log.Warn("consumerConfig is nil!")
		return ConsumerConfig{}
	}
	return *consumerConfig
}

/////////////////////////
// providerConfig
/////////////////////////

type ProviderConfig struct {
	// pprof
	Pprof_Enabled bool `default:"false" yaml:"pprof_enabled" json:"pprof_enabled,omitempty"`
	Pprof_Port    int  `default:"10086"  yaml:"pprof_port" json:"pprof_port,omitempty"`

	ApplicationConfig ApplicationConfig `yaml:"application_config" json:"application_config,omitempty"`
	Path              string            `yaml:"path" json:"path,omitempty"`
	Registries        []RegistryConfig  `yaml:"registries" json:"registries,omitempty"`
	Services          []ServiceConfig   `yaml:"services" json:"services,omitempty"`
	Protocols         []ProtocolConfig  `yaml:"protocols" json:"protocols,omitempty"`
	ProtocolConf      interface{}       `yaml:"protocol_conf" json:"protocol_conf,omitempty"`
}

func SetProviderConfig(p ProviderConfig) {
	providerConfig = &p
}
func GetProviderConfig() ProviderConfig {
	if providerConfig == nil {
		log.Warn("providerConfig is nil!")
		return ProviderConfig{}
	}
	return *providerConfig
}

type ProtocolConfig struct {
	Name        string `required:"true" yaml:"name"  json:"name,omitempty"`
	Ip          string `required:"true" yaml:"ip"  json:"ip,omitempty"`
	Port        string `required:"true" yaml:"port"  json:"port,omitempty"`
	ContextPath string `required:"true" yaml:"contextPath"  json:"contextPath,omitempty"`
}

func loadProtocol(protocolsIds string, protocols []ProtocolConfig) []ProtocolConfig {
	returnProtocols := []ProtocolConfig{}
	for _, v := range strings.Split(protocolsIds, ",") {
		for _, prot := range protocols {
			if v == prot.Name {
				returnProtocols = append(returnProtocols, prot)
			}
		}

	}
	return returnProtocols
}

// Dubbo Init
func Load() (map[string]*ReferenceConfig, map[string]*ServiceConfig) {
	var refMap map[string]*ReferenceConfig
	var srvMap map[string]*ServiceConfig

	// reference config
	if consumerConfig == nil {
		log.Warn("consumerConfig is nil!")
	} else {
		refMap = make(map[string]*ReferenceConfig)
		length := len(consumerConfig.References)
		for index := 0; index < length; index++ {
			con := &consumerConfig.References[index]
			rpcService := conServices[con.InterfaceName]
			if rpcService == nil {
				log.Warn("%s is not exsist!", con.InterfaceName)
				continue
			}
			con.Refer()
			con.Implement(rpcService)
			refMap[con.InterfaceName] = con
		}
	}

	// service config
	if providerConfig == nil {
		log.Warn("providerConfig is nil!")
	} else {
		srvMap = make(map[string]*ServiceConfig)
		length := len(providerConfig.Services)
		for index := 0; index < length; index++ {
			pro := &providerConfig.Services[index]
			rpcService := proServices[pro.InterfaceName]
			if rpcService == nil {
				log.Warn("%s is not exsist!", pro.InterfaceName)
				continue
			}
			pro.Implement(rpcService)
			if err := pro.Export(); err != nil {
				panic(fmt.Sprintf("service %s export failed! ", pro.InterfaceName))
			}
			srvMap[pro.InterfaceName] = pro
		}
	}

	return refMap, srvMap
}
