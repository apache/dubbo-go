package main

import (
	"fmt"
	"github.com/dubbo/dubbo-go/plugins"
	"io/ioutil"
	"os"
	"path"
	"time"
)

import (
	"github.com/AlexStocks/goext/log"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
	yaml "gopkg.in/yaml.v2"
)

import (
	"github.com/dubbo/dubbo-go/registry"
	"github.com/dubbo/dubbo-go/registry/zookeeper"
	"github.com/dubbo/dubbo-go/server"
)

const (
	APP_CONF_FILE     string = "APP_CONF_FILE"
	APP_LOG_CONF_FILE string = "APP_LOG_CONF_FILE"
)

var (
	conf *ServerConfig
)

type (
	ServerConfig struct {
		// pprof
		Pprof_Enabled bool `default:"false" yaml:"pprof_enabled"  json:"pprof_enabled,omitempty"`
		Pprof_Port    int  `default:"10086"  yaml:"pprof_port" json:"pprof_port,omitempty"`

		// transport & registry
		Transport  string `default:"http"  yaml:"transport" json:"transport,omitempty"`
		NetTimeout string `default:"100ms"  yaml:"net_timeout" json:"net_timeout,omitempty"` // in ms
		netTimeout time.Duration
		// application
		Application_Config registry.ApplicationConfig `yaml:"application_config" json:"application_config,omitempty"`
		// Registry_Address  string `default:"192.168.35.3:2181"`
		Registry         string                     `default:"zookeeper"  yaml:"registry" json:"registry,omitempty"`
		ZkRegistryConfig zookeeper.ZkRegistryConfig `yaml:"zk_registry_config" json:"zk_registry_config,omitempty"`

		ServiceConfigType    string                     `default:"default" yaml:"service_config_type" json:"service_config_type,omitempty"`
		ServiceConfigList    []registry.ReferenceConfig `yaml:"-"`
		ServiceConfigMapList []map[string]string        `yaml:"service_list" json:"service_list,omitempty"`
		Server_List          []server.ServerConfig      `yaml:"server_list" json:"server_list,omitempty"`
	}
)

func initServerConf() *ServerConfig {
	var (
		err      error
		confFile string
	)

	confFile = os.Getenv(APP_CONF_FILE)
	if confFile == "" {
		panic(fmt.Sprintf("application configure file name is nil"))
		return nil
	}
	if path.Ext(confFile) != ".yml" {
		panic(fmt.Sprintf("application configure file name{%v} suffix must be .yml", confFile))
		return nil
	}

	conf = &ServerConfig{}
	confFileStream, err := ioutil.ReadFile(confFile)
	if err != nil {
		panic(fmt.Sprintf("ioutil.ReadFile(file:%s) = error:%s", confFile, jerrors.ErrorStack(err)))
		return nil
	}
	err = yaml.Unmarshal(confFileStream, conf)
	if err != nil {
		panic(fmt.Sprintf("yaml.Unmarshal() = error:%s", jerrors.ErrorStack(err)))
		return nil
	}
	if conf.netTimeout, err = time.ParseDuration(conf.NetTimeout); err != nil {
		panic(fmt.Sprintf("time.ParseDuration(NetTimeout:%#v) = error:%s", conf.NetTimeout, err))
		return nil
	}
	if conf.ZkRegistryConfig.Timeout, err = time.ParseDuration(conf.ZkRegistryConfig.TimeoutStr); err != nil {
		panic(fmt.Sprintf("time.ParseDuration(Registry_Config.Timeout:%#v) = error:%s",
			conf.ZkRegistryConfig.TimeoutStr, err))
		return nil
	}

	// set designated service_config_type to default
	plugins.SetDefaultProviderServiceConfig(conf.ServiceConfigType)
	for _, service := range conf.ServiceConfigMapList {

		svc := plugins.DefaultProviderServiceConfig()()
		svc.SetProtocol(service["protocol"])
		svc.SetService(service["service"])
		conf.ServiceConfigList = append(conf.ServiceConfigList, svc)
	}

	gxlog.CInfo("config{%#v}\n", conf)

	return conf
}

func configInit() error {
	var (
		confFile string
	)

	initServerConf()

	confFile = os.Getenv(APP_LOG_CONF_FILE)
	if confFile == "" {
		panic(fmt.Sprintf("log configure file name is nil"))
		return nil
	}
	if path.Ext(confFile) != ".xml" {
		panic(fmt.Sprintf("log configure file name{%v} suffix must be .xml", confFile))
		return nil
	}

	log.LoadConfiguration(confFile)

	return nil
}
