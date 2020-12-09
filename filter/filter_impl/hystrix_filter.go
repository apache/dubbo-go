/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package filter_impl

import (
	"context"
	"fmt"
	"regexp"
	"sync"
)
import (
	"github.com/afex/hystrix-go/hystrix"
	perrors "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)
import (
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
)

const (
	// nolint
	HYSTRIX_CONSUMER = "hystrix_consumer"
	// nolint
	HYSTRIX_PROVIDER = "hystrix_provider"
	// nolint
	HYSTRIX = "hystrix"
)

var (
	confConsumer       = &HystrixFilterConfig{}
	confProvider       = &HystrixFilterConfig{}
	configLoadMutex    = sync.RWMutex{}
	consumerConfigOnce sync.Once
	providerConfigOnce sync.Once
)

func init() {
	extension.SetFilter(HYSTRIX_CONSUMER, GetHystrixFilterConsumer)
	extension.SetFilter(HYSTRIX_PROVIDER, GetHystrixFilterProvider)
}

// HystrixFilterError implements error interface
type HystrixFilterError struct {
	err           error
	failByHystrix bool
}

func (hfError *HystrixFilterError) Error() string {
	return hfError.err.Error()
}

// FailByHystrix returns whether the fails causing by Hystrix
func (hfError *HystrixFilterError) FailByHystrix() bool {
	return hfError.failByHystrix
}

// NewHystrixFilterError return a HystrixFilterError instance
func NewHystrixFilterError(err error, failByHystrix bool) error {
	return &HystrixFilterError{
		err:           err,
		failByHystrix: failByHystrix,
	}
}

/**
 * HystrixFilter
 * You should add hystrix related configuration in provider or consumer config or both, according to which side you are to apply HystrixFilter.
 * For example:
 * filter_conf:
 * 	hystrix:
 * 	 configs:
 * 	  # =========== Define config here ============
 * 	  "Default":
 * 	    timeout : 1000
 * 	    max_concurrent_requests : 25
 * 	    sleep_window : 5000
 * 	    error_percent_threshold : 50
 * 	    request_volume_threshold: 20
 * 	  "userp":
 * 	    timeout: 2000
 * 	    max_concurrent_requests: 512
 * 	    sleep_window: 4000
 * 	    error_percent_threshold: 35
 * 	    request_volume_threshold: 6
 * 	  "userp_m":
 * 	    timeout : 1200
 * 	    max_concurrent_requests : 512
 * 	    sleep_window : 6000
 * 	    error_percent_threshold : 60
 * 	    request_volume_threshold: 16
 *      # =========== Define error whitelist which will be ignored by Hystrix counter ============
 * 	    error_whitelist: [".*exception.*"]
 *
 * 	 # =========== Apply default config here ===========
 * 	 default: "Default"
 *
 * 	 services:
 * 	  "com.ikurento.user.UserProvider":
 * 	    # =========== Apply service level config ===========
 * 	    service_config: "userp"
 * 	    # =========== Apply method level config ===========
 * 	    methods:
 * 	      "GetUser": "userp_m"
 * 	      "GetUser1": "userp_m"
 */
type HystrixFilter struct {
	COrP     bool //true for consumer
	res      map[string][]*regexp.Regexp
	ifNewMap sync.Map
}

// Invoke is an implementation of filter, provides Hystrix pattern latency and fault tolerance
func (hf *HystrixFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	cmdName := fmt.Sprintf("%s&method=%s", invoker.GetUrl().Key(), invocation.MethodName())

	// Do the configuration if the circuit breaker is created for the first time
	if _, load := hf.ifNewMap.LoadOrStore(cmdName, true); !load {
		configLoadMutex.Lock()
		filterConf := getConfig(invoker.GetUrl().Service(), invocation.MethodName(), hf.COrP)
		for _, ptn := range filterConf.Error {
			reg, err := regexp.Compile(ptn)
			if err != nil {
				logger.Warnf("[Hystrix Filter]Errors occurred parsing error omit regexp: %s, %v", ptn, err)
			} else {
				if hf.res == nil {
					hf.res = make(map[string][]*regexp.Regexp)
				}
				hf.res[invocation.MethodName()] = append(hf.res[invocation.MethodName()], reg)
			}
		}
		hystrix.ConfigureCommand(cmdName, hystrix.CommandConfig{
			Timeout:                filterConf.Timeout,
			MaxConcurrentRequests:  filterConf.MaxConcurrentRequests,
			SleepWindow:            filterConf.SleepWindow,
			ErrorPercentThreshold:  filterConf.ErrorPercentThreshold,
			RequestVolumeThreshold: filterConf.RequestVolumeThreshold,
		})
		configLoadMutex.Unlock()
	}
	configLoadMutex.RLock()
	_, _, err := hystrix.GetCircuit(cmdName)
	configLoadMutex.RUnlock()
	if err != nil {
		logger.Errorf("[Hystrix Filter]Errors occurred getting circuit for %s , will invoke without hystrix, error is: %+v", cmdName, err)
		return invoker.Invoke(ctx, invocation)
	}
	logger.Infof("[Hystrix Filter]Using hystrix filter: %s", cmdName)
	var result protocol.Result
	_ = hystrix.Do(cmdName, func() error {
		result = invoker.Invoke(ctx, invocation)
		err := result.Error()
		if err != nil {
			result.SetError(NewHystrixFilterError(err, false))
			for _, reg := range hf.res[invocation.MethodName()] {
				if reg.MatchString(err.Error()) {
					logger.Debugf("[Hystrix Filter]Error in invocation but omitted in circuit breaker: %v; %s", err, cmdName)
					return nil
				}
			}
		}
		return err
	}, func(err error) error {
		//Return error and if it is caused by hystrix logic, so that it can be handled by previous filters.
		_, ok := err.(hystrix.CircuitError)
		logger.Debugf("[Hystrix Filter]Hystrix health check counted, error is: %v, failed by hystrix: %v; %s", err, ok, cmdName)
		result = &protocol.RPCResult{}
		result.SetResult(nil)
		result.SetError(NewHystrixFilterError(err, ok))
		return err
	})
	return result
}

// OnResponse dummy process, returns the result directly
func (hf *HystrixFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

// GetHystrixFilterConsumer returns HystrixFilter instance for consumer
func GetHystrixFilterConsumer() filter.Filter {
	//When first called, load the config in
	consumerConfigOnce.Do(func() {
		if err := initHystrixConfigConsumer(); err != nil {
			logger.Warnf("[Hystrix Filter]Config load failed for consumer, error is: %v , will use default", err)
		}
	})
	return &HystrixFilter{COrP: true}
}

// GetHystrixFilterProvider returns HystrixFilter instance for provider
func GetHystrixFilterProvider() filter.Filter {
	providerConfigOnce.Do(func() {
		if err := initHystrixConfigProvider(); err != nil {
			logger.Warnf("[Hystrix Filter]Config load failed for provider, error is: %v , will use default", err)
		}
	})
	return &HystrixFilter{COrP: false}
}

func getConfig(service string, method string, cOrP bool) CommandConfigWithError {
	//Find method level config
	var conf *HystrixFilterConfig
	if cOrP {
		conf = confConsumer
	} else {
		conf = confProvider
	}
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
	getConf = &CommandConfigWithError{}
	logger.Infof("[Hystrix Filter]No config found for %s - %s, using default", service, method)
	return *getConf

}

func initHystrixConfigConsumer() error {
	if config.GetConsumerConfig().FilterConf == nil {
		return perrors.Errorf("no config for hystrix_consumer")
	}
	filterConfig := config.GetConsumerConfig().FilterConf.(map[interface{}]interface{})[HYSTRIX]
	if filterConfig == nil {
		return perrors.Errorf("no config for hystrix_consumer")
	}
	hystrixConfByte, err := yaml.Marshal(filterConfig)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(hystrixConfByte, confConsumer)
	if err != nil {
		return err
	}
	return nil
}

func initHystrixConfigProvider() error {
	if config.GetProviderConfig().FilterConf == nil {
		return perrors.Errorf("no config for hystrix_provider")
	}
	filterConfig := config.GetProviderConfig().FilterConf.(map[interface{}]interface{})[HYSTRIX]
	if filterConfig == nil {
		return perrors.Errorf("no config for hystrix_provider")
	}
	hystrixConfByte, err := yaml.Marshal(filterConfig)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(hystrixConfByte, confProvider)
	if err != nil {
		return err
	}
	return nil
}

//For sake of dynamic config
//func RefreshHystrix() error {
//	conf = &HystrixFilterConfig{}
//	hystrix.Flush()
//	return initHystrixConfig()
//}

// nolint
type CommandConfigWithError struct {
	Timeout                int      `yaml:"timeout"`
	MaxConcurrentRequests  int      `yaml:"max_concurrent_requests"`
	RequestVolumeThreshold int      `yaml:"request_volume_threshold"`
	SleepWindow            int      `yaml:"sleep_window"`
	ErrorPercentThreshold  int      `yaml:"error_percent_threshold"`
	Error                  []string `yaml:"error_whitelist"`
}

//Config:
//- Timeout: how long to wait for command to complete, in milliseconds
//- MaxConcurrentRequests: how many commands of the same type can run at the same time
//- RequestVolumeThreshold: the minimum number of requests needed before a circuit can be tripped due to health
//- SleepWindow: how long, in milliseconds, to wait after a circuit opens before testing for recovery
//- ErrorPercentThreshold: it causes circuits to open once the rolling measure of errors exceeds this percent of requests
//See hystrix doc

// nolint
type HystrixFilterConfig struct {
	Configs  map[string]*CommandConfigWithError
	Default  string
	Services map[string]ServiceHystrixConfig
}

// nolint
type ServiceHystrixConfig struct {
	ServiceConfig string `yaml:"service_config"`
	Methods       map[string]string
}
