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

// Package hystrix provides hystrix filter.
package hystrix

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sync"
)

import (
	"github.com/afex/hystrix-go/hystrix"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	// nolint
	HYSTRIX = "hystrix"
)

var (
	confConsumer       = &FilterConfig{}
	confProvider       = &FilterConfig{}
	configLoadMutex    = sync.RWMutex{}
	consumerConfigOnce sync.Once
	providerConfigOnce sync.Once
)

func init() {
	extension.SetFilter(constant.HystrixConsumerFilterKey, newFilterConsumer)
	extension.SetFilter(constant.HystrixProviderFilterKey, newFilterProvider)
}

// FilterError implements error interface
type FilterError struct {
	err           error
	failByHystrix bool
}

func (hfError *FilterError) Error() string {
	return hfError.err.Error()
}

// FailByHystrix returns whether the fails causing by Hystrix
func (hfError *FilterError) FailByHystrix() bool {
	return hfError.failByHystrix
}

// NewHystrixFilterError return a FilterError instance
func NewHystrixFilterError(err error, failByHystrix bool) error {
	return &FilterError{
		err:           err,
		failByHystrix: failByHystrix,
	}
}

// Filter for Hystrix
/**
 * You should add hystrix related configuration in provider or consumer config or both, according to which side you are to apply Filter.
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
type Filter struct {
	COrP     bool // true for consumer
	res      map[string][]*regexp.Regexp
	ifNewMap sync.Map
}

// Invoke is an implementation of filter, provides Hystrix pattern latency and fault tolerance
func (f *Filter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	cmdName := fmt.Sprintf("%s&method=%s", invoker.GetURL().Key(), invocation.MethodName())

	// Do the configuration if the circuit breaker is created for the first time
	if _, load := f.ifNewMap.LoadOrStore(cmdName, true); !load {
		configLoadMutex.Lock()
		filterConf := getConfig(invoker.GetURL().Service(), invocation.MethodName(), f.COrP)
		for _, ptn := range filterConf.Error {
			reg, err := regexp.Compile(ptn)
			if err != nil {
				logger.Warnf("[Hystrix Filter]Errors occurred parsing error omit regexp: %s, %v", ptn, err)
			} else {
				if f.res == nil {
					f.res = make(map[string][]*regexp.Regexp)
				}
				f.res[invocation.MethodName()] = append(f.res[invocation.MethodName()], reg)
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
			for _, reg := range f.res[invocation.MethodName()] {
				if reg.MatchString(err.Error()) {
					logger.Debugf("[Hystrix Filter]Error in invocation but omitted in circuit breaker: %v; %s", err, cmdName)
					return nil
				}
			}
		}
		return err
	}, func(err error) error {
		// Return error and if it is caused by hystrix logic, so that it can be handled by previous filters.
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
func (f *Filter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

// newFilterConsumer returns Filter instance for consumer
func newFilterConsumer() filter.Filter {
	// When first called, load the config in
	consumerConfigOnce.Do(func() {
		if err := initConfigConsumer(); err != nil {
			logger.Warnf("[Hystrix Filter]ShutdownConfig load failed for consumer, error is: %v , will use default", err)
		}
	})
	return &Filter{COrP: true}
}

// newFilterProvider returns Filter instance for provider
func newFilterProvider() filter.Filter {
	providerConfigOnce.Do(func() {
		if err := initConfigProvider(); err != nil {
			logger.Warnf("[Hystrix Filter]ShutdownConfig load failed for provider, error is: %v , will use default", err)
		}
	})
	return &Filter{COrP: false}
}

func getConfig(service string, method string, cOrP bool) CommandConfigWithError {
	// Find method level config
	var conf *FilterConfig
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
	// Find service level config
	getConf = conf.Configs[conf.Services[service].ServiceConfig]
	if getConf != nil {
		logger.Infof("[Hystrix Filter]Found service-level config for %s - %s", service, method)
		return *getConf
	}
	// Find default config
	getConf = conf.Configs[conf.Default]
	if getConf != nil {
		logger.Infof("[Hystrix Filter]Found global default config for %s - %s", service, method)
		return *getConf
	}
	getConf = &CommandConfigWithError{}
	logger.Infof("[Hystrix Filter]No config found for %s - %s, using default", service, method)
	return *getConf
}

func initConfigConsumer() error {
	if config.GetConsumerConfig().FilterConf == nil {
		return perrors.Errorf("no config for hystrix_consumer")
	}
	filterConf := config.GetConsumerConfig().FilterConf
	var filterConfig interface{}
	switch reflect.ValueOf(filterConf).Interface().(type) {
	case map[interface{}]interface{}:
		filterConfig = config.GetConsumerConfig().FilterConf.(map[interface{}]interface{})[HYSTRIX]
	case map[string]interface{}:
		filterConfig = config.GetConsumerConfig().FilterConf.(map[string]interface{})[HYSTRIX]
	}
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

func initConfigProvider() error {
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
//	conf = &FilterConfig{}
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

//ShutdownConfig:
//- Timeout: how long to wait for command to complete, in milliseconds
//- MaxConcurrentRequests: how many commands of the same type can run at the same time
//- RequestVolumeThreshold: the minimum number of requests needed before a circuit can be tripped due to health
//- SleepWindow: how long, in milliseconds, to wait after a circuit opens before testing for recovery
//- ErrorPercentThreshold: it causes circuits to open once the rolling measure of errors exceeds this percent of requests
//See hystrix doc

// nolint
type FilterConfig struct {
	Configs  map[string]*CommandConfigWithError
	Default  string
	Services map[string]ServiceHystrixConfig
}

// nolint
type ServiceHystrixConfig struct {
	ServiceConfig string `yaml:"service_config"`
	Methods       map[string]string
}
