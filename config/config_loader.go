package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"time"
)

import (
	"github.com/AlexStocks/goext/log"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/common/constant"
)

var (
	consumerConfig ConsumerConfig
	providerConfig ProviderConfig
)

// loaded comsumer & provider config from xxx.yml
// Namely: dubbo.comsumer.xml & dubbo.provider.xml in java dubbo
func init() {

	var (
		confConFile, confProFile string
	)

	confConFile = os.Getenv(constant.CONF_CONSUMER_FILE_PATH)
	confProFile = os.Getenv(constant.CONF_PROVIDER_FILE_PATH)

	if confConFile == "" && confProFile == "" {
		panic(fmt.Sprintf("application configure(consumer & provider) file name is nil"))
	}

	if confConFile != "" {

		if path.Ext(confConFile) != ".yml" {
			panic(fmt.Sprintf("application configure file name{%v} suffix must be .yml", confConFile))
		}

		confFileStream, err := ioutil.ReadFile(confConFile)
		if err != nil {
			panic(fmt.Sprintf("ioutil.ReadFile(file:%s) = error:%s", confConFile, jerrors.ErrorStack(err)))
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

		gxlog.CInfo("consumer config{%#v}\n", consumerConfig)
	}

	if confProFile != "" {

	}

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
	ApplicationConfig ApplicationConfig `yaml:"application_config" json:"application_config,omitempty"`
	Registries        []RegistryConfig  `yaml:"registries" json:"registries,omitempty"`
	References        []ReferenceConfig `yaml:"references" json:"references,omitempty"`
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
