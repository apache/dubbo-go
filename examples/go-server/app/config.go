/******************************************************
# DESC    : env var & configure
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-07-21 16:41
# FILE    : config.go
******************************************************/

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/AlexStocks/goext/log"

	log "github.com/AlexStocks/log4go"

	jerrors "github.com/juju/errors"

	"github.com/dubbo/dubbo-go/registry"
	yaml "gopkg.in/yaml.v2"
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
		Registry_Config registry.RegistryConfig  `yaml:"registry_config" json:"registry_config,omitempty"`
		Service_List    []registry.ServiceConfig `yaml:"service_list" json:"service_list,omitempty"`
		Server_List     []registry.ServerConfig  `yaml:"server_list" json:"server_list,omitempty"`
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
