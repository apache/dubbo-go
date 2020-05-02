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

package config

import (
	"fmt"
	"log"
	"os"
	"time"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	_ "github.com/apache/dubbo-go/common/observer/dispatcher"
)

var (
	consumerConfig    *ConsumerConfig
	providerConfig    *ProviderConfig
	metricConfig      *MetricConfig
	applicationConfig *ApplicationConfig
	maxWait           = 3
	confRouterFile    string
)

// loaded consumer & provider config from xxx.yml, and log config from xxx.xml
// Namely: dubbo.consumer.xml & dubbo.provider.xml in java dubbo
func init() {
	var (
		confConFile string
		confProFile string
	)

	confConFile = os.Getenv(constant.CONF_CONSUMER_FILE_PATH)
	confProFile = os.Getenv(constant.CONF_PROVIDER_FILE_PATH)
	confRouterFile = os.Getenv(constant.CONF_ROUTER_FILE_PATH)

	if errCon := ConsumerInit(confConFile); errCon != nil {
		log.Printf("[consumerInit] %#v", errCon)
		consumerConfig = nil
	}

	if errPro := ProviderInit(confProFile); errPro != nil {
		log.Printf("[providerInit] %#v", errPro)
		providerConfig = nil
	}
}

func checkRegistries(registries map[string]*RegistryConfig, singleRegistry *RegistryConfig) {
	if len(registries) == 0 && singleRegistry != nil {
		registries[constant.DEFAULT_KEY] = singleRegistry
	}
}

func checkApplicationName(config *ApplicationConfig) {
	if config == nil || len(config.Name) == 0 {
		errMsg := "application config must not be nil, pls check your configuration"
		logger.Errorf(errMsg)
		panic(errMsg)
	}
}

func loadConsumerConfig() {
	if consumerConfig == nil {
		logger.Warnf("consumerConfig is nil!")
		return
	}
	// init other consumer config
	conConfigType := consumerConfig.ConfigType
	for key, value := range extension.GetDefaultConfigReader() {
		if conConfigType == nil {
			if v, ok := conConfigType[key]; ok {
				value = v
			}
		}
		if err := extension.GetConfigReaders(value).ReadConsumerConfig(consumerConfig.fileStream); err != nil {
			logger.Errorf("ReadConsumerConfig error: %#v for %s", perrors.WithStack(err), value)
		}
	}

	metricConfig = consumerConfig.MetricConfig
	applicationConfig = consumerConfig.ApplicationConfig
	extension.SetAndInitGlobalDispatcher(consumerConfig.eventDispatcherType)

	extension.SetAndInitGlobalDispatcher(consumerConfig.eventDispatcherType)

	checkApplicationName(consumerConfig.ApplicationConfig)
	if err := configCenterRefreshConsumer(); err != nil {
		logger.Errorf("[consumer config center refresh] %#v", err)
	}
	checkRegistries(consumerConfig.Registries, consumerConfig.Registry)
	for key, ref := range consumerConfig.References {
		if ref.Generic {
			genericService := NewGenericService(key)
			SetConsumerService(genericService)
		}
		rpcService := GetConsumerService(key)
		if rpcService == nil {
			logger.Warnf("%s does not exist!", key)
			continue
		}
		ref.id = key
		ref.Refer(rpcService)
		ref.Implement(rpcService)
	}

	//wait for invoker is available, if wait over default 3s, then panic
	var count int
	checkok := true
	for {
		for _, refconfig := range consumerConfig.References {
			if (refconfig.Check != nil && *refconfig.Check) ||
				(refconfig.Check == nil && consumerConfig.Check != nil && *consumerConfig.Check) ||
				(refconfig.Check == nil && consumerConfig.Check == nil) { //default to true

				if refconfig.invoker != nil &&
					!refconfig.invoker.IsAvailable() {
					checkok = false
					count++
					if count > maxWait {
						errMsg := fmt.Sprintf("Failed to check the status of the service %v . No provider available for the service to the consumer use dubbo version %v", refconfig.InterfaceName, constant.Version)
						logger.Error(errMsg)
						panic(errMsg)
					}
					time.Sleep(time.Second * 1)
					break
				}
				if refconfig.invoker == nil {
					logger.Warnf("The interface %s invoker not exist , may you should check your interface config.", refconfig.InterfaceName)
				}
			}
		}
		if checkok {
			break
		}
		checkok = true
	}
}

func loadProviderConfig() {
	if providerConfig == nil {
		logger.Warnf("providerConfig is nil!")
		return
	}

	// init other provider config
	proConfigType := providerConfig.ConfigType
	for key, value := range extension.GetDefaultConfigReader() {
		if proConfigType != nil {
			if v, ok := proConfigType[key]; ok {
				value = v
			}
		}
		if err := extension.GetConfigReaders(value).ReadProviderConfig(providerConfig.fileStream); err != nil {
			logger.Errorf("ReadProviderConfig error: %#v for %s", perrors.WithStack(err), value)
		}
	}

	// so, you should know that the consumer's config will be override
	metricConfig = providerConfig.MetricConfig
	applicationConfig = providerConfig.ApplicationConfig
	extension.SetAndInitGlobalDispatcher(providerConfig.eventDispatcherType)

	extension.SetAndInitGlobalDispatcher(consumerConfig.eventDispatcherType)

	checkApplicationName(providerConfig.ApplicationConfig)
	if err := configCenterRefreshProvider(); err != nil {
		logger.Errorf("[provider config center refresh] %#v", err)
	}
	checkRegistries(providerConfig.Registries, providerConfig.Registry)
	for key, svs := range providerConfig.Services {
		rpcService := GetProviderService(key)
		if rpcService == nil {
			logger.Warnf("%s does not exist!", key)
			continue
		}
		svs.id = key
		svs.Implement(rpcService)
		svs.Protocols = providerConfig.Protocols
		if err := svs.Export(); err != nil {
			panic(fmt.Sprintf("service %s export failed! err: %#v", key, err))
		}
	}
}

func initRouter() {
	if confRouterFile != "" {
		if err := RouterInit(confRouterFile); err != nil {
			log.Printf("[routerConfig init] %#v", err)
		}
	}
}

// Load Dubbo Init
func Load() {

	// init router
	initRouter()

	// reference config
	loadConsumerConfig()

	// service config
	loadProviderConfig()

	// init the shutdown callback
	GracefulShutdownInit()
}

// GetRPCService get rpc service for consumer
func GetRPCService(name string) common.RPCService {
	return consumerConfig.References[name].GetRPCService()
}

// RPCService create rpc service for consumer
func RPCService(service common.RPCService) {
	consumerConfig.References[service.Reference()].Implement(service)
}

// GetMetricConfig find the MetricConfig
// if it is nil, create a new one
func GetMetricConfig() *MetricConfig {
	if metricConfig == nil {
		metricConfig = &MetricConfig{}
	}
	return metricConfig
}

// GetApplicationConfig find the application config
// if not, we will create one
// Usually applicationConfig will be initialized when system start
func GetApplicationConfig() *ApplicationConfig {
	if applicationConfig == nil {
		applicationConfig = &ApplicationConfig{}
	}
	return applicationConfig
}

// GetProviderConfig find the provider config
// if not found, create new one
func GetProviderConfig() ProviderConfig {
	if providerConfig == nil {
		logger.Warnf("providerConfig is nil!")
		return ProviderConfig{}
	}
	return *providerConfig
}

// GetConsumerConfig find the consumer config
// if not found, create new one
func GetConsumerConfig() ConsumerConfig {
	if consumerConfig == nil {
		logger.Warnf("consumerConfig is nil!")
		return ConsumerConfig{}
	}
	return *consumerConfig
}
