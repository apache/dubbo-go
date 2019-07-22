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
	HYSTRIX = "hystrix"
)

type HystrixFallback interface {
	FallbackFunc(err error, invoker protocol.Invoker, invocation protocol.Invocation, cb hystrix.CircuitBreaker) protocol.Result
}

var (
	isConfigLoaded = false
	fallback       = make(map[string]HystrixFallback)
	conf           = &HystrixFilterConfig{}
	//Timeout
	//MaxConcurrentRequests
	//RequestVolumeThreshold
	//SleepWindow
	//ErrorPercentThreshold
)

func init() {
	extension.SetFilter(HYSTRIX, GetHystrixFilter)
}

type HystrixFilter struct {
	fallback HystrixFallback
}

func (hf *HystrixFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {

	cmdName := fmt.Sprintf("%s&method=%s", invoker.GetUrl().Key(), invocation.MethodName())

	cb, ifNew, err := hystrix.GetCircuit(cmdName)
	if err != nil {
		logger.Errorf("[Hystrix Filter]Errors occurred getting circuit for %s , will invoke without hystrix, error is: ", cmdName, err)
		return invoker.Invoke(invocation)
	}

	// Do the configuration if the circuit breaker is created for the first time

	if ifNew || hf.fallback == nil {
		filterConf := getConfig(invoker.GetUrl().Service(), invocation.MethodName())
		if ifNew {
			hystrix.ConfigureCommand(cmdName, hystrix.CommandConfig{
				Timeout:                filterConf.Timeout,
				MaxConcurrentRequests:  filterConf.MaxConcurrentRequests,
				SleepWindow:            filterConf.SleepWindow,
				ErrorPercentThreshold:  filterConf.ErrorPercentThreshold,
				RequestVolumeThreshold: filterConf.RequestVolumeThreshold,
			})
		}
		if hf.fallback == nil {
			hf.fallback = getHystrixFallback(filterConf.Fallback)
		}
	}

	logger.Infof("[Hystrix Filter]Using hystrix filter: %s", cmdName)
	var result protocol.Result
	_ = hystrix.Do(cmdName, func() error {
		result = invoker.Invoke(invocation)
		return result.Error()
	}, func(err error) error {
		//failure logic
		logger.Debugf("[Hystrix Filter]Invoke failed, error is: %v, circuit breaker open: %v", err, cb.IsOpen())
		result = hf.fallback.FallbackFunc(err, invoker, invocation, *cb)

		//If user try to return nil in the customized fallback func, it will cause panic
		//So check here
		if result == nil {
			result = &protocol.RPCResult{}
		}
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

func getConfig(service string, method string) CommandConfigWithFallback {

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
	getConf = &CommandConfigWithFallback{}
	logger.Infof("[Hystrix Filter]No config found for %s - %s, using default", service, method)
	return *getConf

}

func initHystrixConfig() error {
	if config.GetConsumerConfig().FilterConf == nil {
		return perrors.Errorf("no config for hystrix")
	}
	filterConfig := config.GetConsumerConfig().FilterConf.(map[interface{}]interface{})[HYSTRIX]
	if filterConfig == nil {
		return perrors.Errorf("no config for hystrix")
	}
	hystrixConfByte, err := yaml.Marshal(filterConfig)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(hystrixConfByte, conf)
	if err != nil {
		return err
	}
	return nil
}

//For sake of dynamic config
func RefreshHystrix() error {
	conf = &HystrixFilterConfig{}
	hystrix.Flush()
	return initHystrixConfig()
}

func SetHystrixFallback(name string, fallbackImpl HystrixFallback) {
	fallback[name] = fallbackImpl
}

func getHystrixFallback(name string) HystrixFallback {
	fallbackImpl := fallback[name]
	if fallbackImpl == nil {
		logger.Warnf("[Hystrix Filter]Fallback func not found: %s", name)
		fallbackImpl = &DefaultHystrixFallback{}
	}
	return fallbackImpl
}

type CommandConfigWithFallback struct {
	Timeout                int    `yaml:"timeout"`
	MaxConcurrentRequests  int    `yaml:"max_concurrent_requests"`
	RequestVolumeThreshold int    `yaml:"request_volume_threshold"`
	SleepWindow            int    `yaml:"sleep_window"`
	ErrorPercentThreshold  int    `yaml:"error_percent_threshold"`
	Fallback               string `yaml:"fallback"`
}

type HystrixFilterConfig struct {
	Configs  map[string]*CommandConfigWithFallback
	Default  string
	Services map[string]ServiceHystrixConfig
}
type ServiceHystrixConfig struct {
	ServiceConfig string `yaml:"service_config"`
	Methods       map[string]string
}
