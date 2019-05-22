// Copyright 2016-2019 Yincheng Fang
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	log "github.com/AlexStocks/log4go"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/version"
)

var (
	consumerConfig *ConsumerConfig
	providerConfig *ProviderConfig
	maxWait        = 3
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
		return errors.Errorf("log configure file name is nil")
	}
	if path.Ext(confFile) != ".xml" {
		return errors.Errorf("log configure file name{%v} suffix must be .xml", confFile)
	}

	log.LoadConfiguration(confFile)

	return nil
}

func consumerInit(confConFile string) error {
	if confConFile == "" {
		return errors.Errorf("application configure(consumer) file name is nil")
	}

	if path.Ext(confConFile) != ".yml" {
		return errors.Errorf("application configure file name{%v} suffix must be .yml", confConFile)
	}

	confFileStream, err := ioutil.ReadFile(confConFile)
	if err != nil {
		return errors.Errorf("ioutil.ReadFile(file:%s) = error:%v", confConFile, errors.Cause(err))
	}
	consumerConfig = &ConsumerConfig{}
	err = yaml.Unmarshal(confFileStream, consumerConfig)
	if err != nil {
		return errors.Errorf("yaml.Unmarshal() = error:%v", errors.Cause(err))
	}

	if consumerConfig.RequestTimeout, err = time.ParseDuration(consumerConfig.Request_Timeout); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(Request_Timeout{%#v})", consumerConfig.Request_Timeout)
	}
	if consumerConfig.ConnectTimeout, err = time.ParseDuration(consumerConfig.Connect_Timeout); err != nil {
		return errors.WithMessagef(err, "time.ParseDuration(Connect_Timeout{%#v})", consumerConfig.Connect_Timeout)
	}

	log.Debug("consumer config{%#v}\n", consumerConfig)
	return nil
}

func providerInit(confProFile string) error {
	if confProFile == "" {
		return errors.Errorf("application configure(provider) file name is nil")
	}

	if path.Ext(confProFile) != ".yml" {
		return errors.Errorf("application configure file name{%v} suffix must be .yml", confProFile)
	}

	confFileStream, err := ioutil.ReadFile(confProFile)
	if err != nil {
		return errors.Errorf("ioutil.ReadFile(file:%s) = error:%v", confProFile, errors.Cause(err))
	}
	providerConfig = &ProviderConfig{}
	err = yaml.Unmarshal(confFileStream, providerConfig)
	if err != nil {
		return errors.Errorf("yaml.Unmarshal() = error:%v", errors.Cause(err))
	}

	log.Debug("provider config{%#v}\n", providerConfig)
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
	ProxyFactory    string `yaml:"proxy_factory" default:"default" json:"proxy_factory,omitempty"`
	Check           *bool  `yaml:"check"  json:"check,omitempty"`
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
	Pprof_Enabled     bool              `default:"false" yaml:"pprof_enabled" json:"pprof_enabled,omitempty"`
	Pprof_Port        int               `default:"10086"  yaml:"pprof_port" json:"pprof_port,omitempty"`
	ProxyFactory      string            `yaml:"proxy_factory" default:"default" json:"proxy_factory,omitempty"`
	ApplicationConfig ApplicationConfig `yaml:"application_config" json:"application_config,omitempty"`
	Registries        []RegistryConfig  `yaml:"registries" json:"registries,omitempty"`
	Services          []ServiceConfig   `yaml:"services" json:"services,omitempty"`
	Protocols         []ProtocolConfig  `yaml:"protocols" json:"protocols,omitempty"`
	ProtocolConf      interface{}       `yaml:"protocol_conf" json:"protocol_conf,omitempty"`
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
			rpcService := GetConService(con.InterfaceName)
			if rpcService == nil {
				log.Warn("%s is not exsist!", con.InterfaceName)
				continue
			}
			con.Refer()
			con.Implement(rpcService)
			refMap[con.InterfaceName] = con
		}

		//wait for invoker is available, if wait over default 3s, then panic
		var count int
		checkok := true
		for {
			for _, refconfig := range consumerConfig.References {
				if ((refconfig.Check != nil && *refconfig.Check) ||
					(refconfig.Check == nil && consumerConfig.Check != nil && *consumerConfig.Check) ||
					(refconfig.Check == nil && consumerConfig.Check == nil)) && //default to true
					!refconfig.invoker.IsAvailable() {
					checkok = false
					count++
					if count > maxWait {
						panic(fmt.Sprintf("Failed to check the status of the service %v . No provider available for the service to the consumer use dubbo version %v", refconfig.InterfaceName, version.Version))
					}
					time.Sleep(time.Second * 1)
					break
				}
			}
			if checkok {
				break
			}
			checkok = true
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
			rpcService := GetProService(pro.InterfaceName)
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
