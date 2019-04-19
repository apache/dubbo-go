package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"
)

import (
	gxlog "github.com/AlexStocks/goext/log"
	jerrors "github.com/juju/errors"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/dubbo/dubbo-go/common/constant"
	"github.com/dubbo/dubbo-go/registry"
)

var (
	consumerConfig ConsumerConfig
	providerConfig ProviderConfig
)

// loaded comsumer & provider config from xxx.yml
// Namely: dubbo.comsumer.xml & dubbo.provider.xml
func init() {

	var (
		confFile string
	)

	// configure
	confFile = os.Getenv(constant.CONF_CONSUMER_FILE_PATH)
	if confFile == "" {
		panic(fmt.Sprintf("application configure file name is nil"))
	}
	if path.Ext(confFile) != ".yml" {
		panic(fmt.Sprintf("application configure file name{%v} suffix must be .yml", confFile))
	}

	confFileStream, err := ioutil.ReadFile(confFile)
	if err != nil {
		panic(fmt.Sprintf("ioutil.ReadFile(file:%s) = error:%s", confFile, jerrors.ErrorStack(err)))
	}
	err = yaml.Unmarshal(confFileStream, consumerConfig)
	if err != nil {
		panic(fmt.Sprintf("yaml.Unmarshal() = error:%s", jerrors.ErrorStack(err)))
	}
	//动态加载service config  end
	for _, config := range consumerConfig.Registries {
		if config.Timeout, err = time.ParseDuration(config.TimeoutStr); err != nil {
			panic(fmt.Sprintf("time.ParseDuration(Registry_Config.Timeout:%#v) = error:%s", config.TimeoutStr, err))
		}
	}

	gxlog.CInfo("config{%#v}\n", consumerConfig)

	// log
	//confFile = os.Getenv(APP_LOG_CONF_FILE)
	//if confFile == "" {
	//	panic(fmt.Sprintf("log configure file name is nil"))
	//	return nil
	//}
	//if path.Ext(confFile) != ".xml" {
	//	panic(fmt.Sprintf("log configure file name{%v} suffix must be .xml", confFile))
	//	return nil
	//}
	//log.LoadConfiguration(confFile)

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
	//client load balance algorithm
	ClientLoadBalance string `default:"round_robin"  yaml:"client_load_balance" json:"client_load_balance,omitempty"`
	// application
	Application_Config registry.ApplicationConfig `yaml:"application_config" json:"application_config,omitempty"`
	Registries         []RegistryConfig           `yaml:"registries" json:"registries,omitempty"`
	References         []ReferenceConfig          `yaml:"references" json:"references,omitempty"`
}

type ReferenceConfigTmp struct {
	Service    string                    `required:"true"  yaml:"service"  json:"service,omitempty"`
	Registries []referenceConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	URLs       []map[string]string
}

func SetConsumerConfig(c ConsumerConfig) {
	consumerConfig = c
}
func GetConsumerConfig() ConsumerConfig {
	return consumerConfig
}

/////////////////////////
// providerConfig
/////////////////////////

type ProviderConfig struct {
	RegistryConfigs []map[string]string `yaml:"registryConfigs" json:"registryConfigs"`
}

func SetProviderConfig(p ProviderConfig) {
	providerConfig = p
}
func GetProviderConfig() ProviderConfig {
	return providerConfig
}
