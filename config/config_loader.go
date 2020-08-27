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
	"sync"
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
	consumerConfig *ConsumerConfig
	providerConfig *ProviderConfig
	// baseConfig = providerConfig.BaseConfig or consumerConfig
	baseConfig *BaseConfig
	sslEnabled = false

	// configAccessMutex is used to make sure that xxxxConfig will only be created once if needed.
	// it should be used combine with double-check to avoid the race condition
	configAccessMutex sync.Mutex

	maxWait        = 3
	confRouterFile string
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
	} else {
		// Even though baseConfig has been initialized, we override it
		// because we think read from config file is correct config
		baseConfig = &consumerConfig.BaseConfig
	}

	if errPro := ProviderInit(confProFile); errPro != nil {
		log.Printf("[providerInit] %#v", errPro)
		providerConfig = nil
	} else {
		// Even though baseConfig has been initialized, we override it
		// because we think read from config file is correct config
		baseConfig = &providerConfig.BaseConfig
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
		if conConfigType != nil {
			if v, ok := conConfigType[key]; ok {
				value = v
			}
		}
		if err := extension.GetConfigReaders(value).ReadConsumerConfig(consumerConfig.fileStream); err != nil {
			logger.Errorf("ReadConsumerConfig error: %#v for %s", perrors.WithStack(err), value)
		}
	}

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

	// wait for invoker is available, if wait over default 3s, then panic
	var count int
	checkok := true
	for {
		for _, refconfig := range consumerConfig.References {
			if (refconfig.Check != nil && *refconfig.Check) ||
				(refconfig.Check == nil && consumerConfig.Check != nil && *consumerConfig.Check) ||
				(refconfig.Check == nil && consumerConfig.Check == nil) { // default to true

				if refconfig.invoker != nil && !refconfig.invoker.IsAvailable() {
					checkok = false
					count++
					if count > maxWait {
						errMsg := fmt.Sprintf("Failed to check the status of the service %v. No provider available for the service to the consumer use dubbo version %v", refconfig.InterfaceName, constant.Version)
						logger.Error(errMsg)
						panic(errMsg)
					}
					time.Sleep(time.Second * 1)
					break
				}
				if refconfig.invoker == nil {
					logger.Warnf("The interface %s invoker not exist, may you should check your interface config.", refconfig.InterfaceName)
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

	// init the global event dispatcher
	extension.SetAndInitGlobalDispatcher(GetBaseConfig().EventDispatcherType)

	// start the metadata report if config set
	if err := startMetadataReport(GetApplicationConfig().MetadataType, GetBaseConfig().MetadataReportConfig); err != nil {
		logger.Errorf("Provider starts metadata report error, and the error is {%#v}", err)
		return
	}

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
// we use double-check to reduce race condition
// In general, it will be locked 0 or 1 time.
// So you don't need to worry about the race condition
func GetMetricConfig() *MetricConfig {
	if GetBaseConfig().MetricConfig == nil {
		configAccessMutex.Lock()
		defer configAccessMutex.Unlock()
		if GetBaseConfig().MetricConfig == nil {
			GetBaseConfig().MetricConfig = &MetricConfig{}
		}
	}
	return GetBaseConfig().MetricConfig
}

// GetApplicationConfig find the application config
// if not, we will create one
// Usually applicationConfig will be initialized when system start
// we use double-check to reduce race condition
// In general, it will be locked 0 or 1 time.
// So you don't need to worry about the race condition
func GetApplicationConfig() *ApplicationConfig {
	if GetBaseConfig().ApplicationConfig == nil {
		configAccessMutex.Lock()
		defer configAccessMutex.Unlock()
		if GetBaseConfig().ApplicationConfig == nil {
			GetBaseConfig().ApplicationConfig = &ApplicationConfig{}
		}
	}
	return GetBaseConfig().ApplicationConfig
}

// GetProviderConfig find the provider config
// if not found, create new one
func GetProviderConfig() ProviderConfig {
	if providerConfig == nil {
		if providerConfig == nil {
			return ProviderConfig{}
		}
	}
	return *providerConfig
}

// GetConsumerConfig find the consumer config
// if not found, create new one
// we use double-check to reduce race condition
// In general, it will be locked 0 or 1 time.
// So you don't need to worry about the race condition
func GetConsumerConfig() ConsumerConfig {
	if consumerConfig == nil {
		if consumerConfig == nil {
			return ConsumerConfig{}
		}
	}
	return *consumerConfig
}

func GetBaseConfig() *BaseConfig {
	if baseConfig == nil {
		configAccessMutex.Lock()
		defer configAccessMutex.Unlock()
		if baseConfig == nil {
			baseConfig = &BaseConfig{
				MetricConfig:       &MetricConfig{},
				ConfigCenterConfig: &ConfigCenterConfig{},
				Remotes:            make(map[string]*RemoteConfig, 0),
				ApplicationConfig:  &ApplicationConfig{},
				ServiceDiscoveries: make(map[string]*ServiceDiscoveryConfig, 0),
			}
		}
	}
	return baseConfig
}

func GetSslEnabled() bool {
	return sslEnabled
}
func SetSslEnabled(enabled bool) {
	sslEnabled = enabled
}
func IsProvider() bool {
	return providerConfig != nil
}
