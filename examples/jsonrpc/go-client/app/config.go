package main

import (
	"fmt"

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
	"github.com/dubbo/dubbo-go/registry/zookeeper"
	"github.com/dubbo/dubbo-go/service"
	"github.com/dubbo/dubbo-go/registry"
)

const (
	APP_CONF_FILE     = "APP_CONF_FILE"
	APP_LOG_CONF_FILE = "APP_LOG_CONF_FILE"
)

var (
	clientConfig *ClientConfig
)

type (
	// Client holds supported types by the multiconfig package
	ClientConfig struct {
		// pprof
		Pprof_Enabled bool `default:"false" yaml:"pprof_enabled" json:"pprof_enabled,omitempty"`
		Pprof_Port    int  `default:"10086"  yaml:"pprof_port" json:"pprof_port,omitempty"`

		// client
		Connect_Timeout string `default:"100ms"  yaml:"connect_timeout" json:"connect_timeout,omitempty"`
		connectTimeout  time.Duration

		Request_Timeout string `yaml:"request_timeout" default:"5s" json:"request_timeout,omitempty"` // 500ms, 1m
		requestTimeout  time.Duration

		// codec & selector & transport & registry
		Selector     string `default:"cache"  yaml:"selector" json:"selector,omitempty"`
		Selector_TTL string `default:"10m"  yaml:"selector_ttl" json:"selector_ttl,omitempty"`
		//client load balance algorithm
		ClientLoadBalance string  `default:"round_robin"  yaml:"client_load_balance" json:"client_load_balance,omitempty"`
		Registry     string `default:"zookeeper"  yaml:"registry" json:"registry,omitempty"`
		// application
		Application_Config registry.ApplicationConfig `yaml:"application_config" json:"application_config,omitempty"`
		ZkRegistryConfig zookeeper.ZkRegistryConfig  `yaml:"zk_registry_config" json:"zk_registry_config,omitempty"`
		// 一个客户端只允许使用一个service的其中一个group和其中一个version
		Service_List []service.ServiceConfig `yaml:"service_list" json:"service_list,omitempty"`
	}
)

func initClientConfig() error {
	var (
		confFile string
	)

	// configure
	confFile = os.Getenv(APP_CONF_FILE)
	if confFile == "" {
		panic(fmt.Sprintf("application configure file name is nil"))
		return nil // I know it is of no usage. Just Err Protection.
	}
	if path.Ext(confFile) != ".yml" {
		panic(fmt.Sprintf("application configure file name{%v} suffix must be .yml", confFile))
		return nil
	}
	clientConfig = new(ClientConfig)

	confFileStream, err := ioutil.ReadFile(confFile)
	if err != nil {
		panic(fmt.Sprintf("ioutil.ReadFile(file:%s) = error:%s", confFile, jerrors.ErrorStack(err)))
		return nil
	}
	err = yaml.Unmarshal(confFileStream, clientConfig)
	if err != nil {
		panic(fmt.Sprintf("yaml.Unmarshal() = error:%s", jerrors.ErrorStack(err)))
		return nil
	}
	if clientConfig.ZkRegistryConfig.Timeout, err = time.ParseDuration(clientConfig.ZkRegistryConfig.TimeoutStr); err != nil {
		panic(fmt.Sprintf("time.ParseDuration(Registry_Config.Timeout:%#v) = error:%s", clientConfig.ZkRegistryConfig.TimeoutStr, err))
		return nil
	}

	gxlog.CInfo("config{%#v}\n", clientConfig)

	// log
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
