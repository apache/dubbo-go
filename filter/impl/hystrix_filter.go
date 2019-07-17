package impl

import (
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
	perrors "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	HYSTRIX                   = "hystrix"
	TIMEOUT_KEY               = "timeout"
	MAXCONCURRENTREQUESTS_KEY = "maxconcurrentrequests"
	SLEEPWINDOW_KEY           = "sleepwindow"
	ERRORPERCENTTHRESHOLD_KEY = "errorpercentthreshold"
)

var (
	//Timeout
	//MaxConcurrentRequests
	//RequestVolumeThreshold
	//SleepWindow
	//ErrorPercentThreshold
	isConfigLoaded = false
	conf           = &HystrixFilterConfig{}
	//

)

func init() {
	extension.SetFilter(HYSTRIX, GetHystrixFilter)
}

type HystrixFilter struct {
}

func (hf *HystrixFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {

	cmdName := fmt.Sprintf("%s&method=%s", invoker.GetUrl().Key(), invocation.MethodName())

	_, ifNew, err := hystrix.GetCircuit(cmdName)
	if err != nil {
		logger.Errorf("[Hystrix Filter]Errors occurred getting circuit for %s , will invoke without hystrix, error is: ", cmdName, err)
		return invoker.Invoke(invocation)
	}

	// Do the configuration if the circuit breaker is created for the first time
	if ifNew {
		hystrix.ConfigureCommand(cmdName, getConfig(invoker.GetUrl().Service(), invocation.MethodName()))
	}

	logger.Infof("[Hystrix Filter]Using hystrix filter: %s", cmdName)
	var result protocol.Result
	_ = hystrix.Do(cmdName, func() error {
		result = invoker.Invoke(invocation)
		return nil
	}, func(err error) error {

		//failure logic
		result = &protocol.RPCResult{}
		result.SetError(err)
		return nil
		//failure logic

	})
	return result
}

func (hf *HystrixFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}
func GetHystrixFilter() filter.Filter {
	//When first called, load the config in
	if !isConfigLoaded {
		if err := initHystrixConfig(); err != nil {
			logger.Warnf("[Hystrix Filter]Config load failed, error is: %v , will use default", err)
		}
		isConfigLoaded = true
	}

	return &HystrixFilter{}
}

func getConfig(service string, method string) hystrix.CommandConfig {

	//Find method level config
	getConf := conf.Configs[conf.Services[service].Methods[method]]
	if getConf != nil {
		logger.Infof("[Hystrix Filter]Found method-level config for %s - %s", service, method)
		return *getConf
	}
	//Find service level config
	getConf = conf.Configs[conf.Services[service].ServiceConfig]
	if getConf != nil {
		logger.Infof("[Hystrix Filter]Found service-level config for %s - %s", service, method)
		return *getConf
	}
	//Find default config
	getConf = conf.Configs[conf.Default]
	if getConf != nil {
		logger.Infof("[Hystrix Filter]Found global default config for %s - %s", service, method)
		return *getConf
	}
	getConf = &hystrix.CommandConfig{}
	logger.Infof("[Hystrix Filter]No config found for %s - %s, using default", service, method)
	return *getConf

}

func initHystrixConfig() error {
	filterConfig := config.GetConsumerConfig().FilterConf.(map[interface{}]interface{})[HYSTRIX]
	if filterConfig == nil {
		return perrors.Errorf("no config for hystrix")
	}
	hystrixConfByte, err := yaml.Marshal(filterConfig)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(hystrixConfByte, conf)
	if err != nil {
		panic(err)
	}
	return nil
}

type HystrixFilterConfig struct {
	Configs  map[string]*hystrix.CommandConfig
	Default  string
	Services map[string]ServiceHystrixConfig
}
type ServiceHystrixConfig struct {
	ServiceConfig string `yaml:"service_config,omitempty"`
	Methods       map[string]string
}
