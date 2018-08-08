package main

import (
	"fmt"
	"os"
	"path"
	"time"
)

import (
	"github.com/AlexStocks/goext/log"
	log "github.com/AlexStocks/log4go"
	config "github.com/koding/multiconfig"
)

import (
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
		Pprof_Enabled bool `default:"false"`
		Pprof_Port    int  `default:"10086"`

		// client
		Connect_Timeout string `default:"100ms"`
		connectTimeout  time.Duration

		Request_Timeout string `default:"5s"` // 500ms, 1m
		requestTimeout  time.Duration

		// codec & selector & transport & registry
		Selector     string `default:"cache"`
		Selector_TTL string `default:"10m"`
		Registry     string `default:"zookeeper"`
		// application
		Application_Config registry.ApplicationConfig
		Registry_Config    registry.RegistryConfig
		// 一个客户端只允许使用一个service的其中一个group和其中一个version
		Service_List []registry.ServiceConfig
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
	if path.Ext(confFile) != ".toml" {
		panic(fmt.Sprintf("application configure file name{%v} suffix must be .toml", confFile))
		return nil
	}
	clientConfig = new(ClientConfig)
	config.MustLoadWithPath(confFile, clientConfig)
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
