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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	_ "dubbo.apache.org/dubbo-go/v3/common/observer/dispatcher"
	"dubbo.apache.org/dubbo-go/v3/common/yaml"
	"dubbo.apache.org/dubbo-go/v3/registry"
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

	maxWait                         = 3
	confRouterFile                  string
	confBaseFile                    string
	uniformVirtualServiceConfigPath string
	uniformDestRuleConfigPath       string
)

// loaded consumer & provider config from xxx.yml, and log config from xxx.xml
// Namely: dubbo.consumer.xml & dubbo.provider.xml in java dubbo
func DefaultInit() []LoaderInitOption {
	var (
		confConFile string
		confProFile string
	)

	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.StringVar(&confConFile, "conConf", os.Getenv(constant.CONF_CONSUMER_FILE_PATH), "default client config path")
	fs.StringVar(&confProFile, "proConf", os.Getenv(constant.CONF_PROVIDER_FILE_PATH), "default server config path")
	fs.StringVar(&confRouterFile, "rouConf", os.Getenv(constant.CONF_ROUTER_FILE_PATH), "default router config path")
	fs.StringVar(&uniformVirtualServiceConfigPath, "vsConf", os.Getenv(constant.CONF_VIRTUAL_SERVICE_FILE_PATH), "default virtual service of uniform router config path")
	fs.StringVar(&uniformDestRuleConfigPath, "drConf", os.Getenv(constant.CONF_DEST_RULE_FILE_PATH), "default destination rule of uniform router config path")
	fs.Parse(os.Args[1:])
	for len(fs.Args()) != 0 {
		fs.Parse(fs.Args()[1:])
	}
	// If user did not set the environment variables or flags,
	// we provide default value
	if confConFile == "" {
		confConFile = constant.DEFAULT_CONSUMER_CONF_FILE_PATH
	}
	if confProFile == "" {
		confProFile = constant.DEFAULT_PROVIDER_CONF_FILE_PATH
	}
	if confRouterFile == "" {
		confRouterFile = constant.DEFAULT_ROUTER_CONF_FILE_PATH
	}
	return []LoaderInitOption{RouterInitOption(confRouterFile), BaseInitOption(""), ConsumerInitOption(confConFile), ProviderInitOption(confProFile)}
}

// setDefaultValue set default value for providerConfig or consumerConfig if it is null
func setDefaultValue(target interface{}) {
	switch target.(type) {
	case *ProviderConfig:
		p := target.(*ProviderConfig)
		if len(p.Protocols) == 0 {
			p.Protocols[constant.DEFAULT_PROTOCOL] = &ProtocolConfig{
				Name: constant.DEFAULT_PROTOCOL,
				Port: strconv.Itoa(constant.DEFAULT_PORT),
			}
		}
		if p.ApplicationConfig == nil {
			p.ApplicationConfig = NewDefaultApplicationConfig()
		}
	default:
		c := target.(*ConsumerConfig)
		if c.ApplicationConfig == nil {
			c.ApplicationConfig = NewDefaultApplicationConfig()
		}
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
		logger.Debugf("Consumer will not be launched, because consumerConfig is not specified.")
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

	// start the metadata report if config set
	if err := startMetadataReport(GetApplicationConfig().MetadataType, GetBaseConfig().MetadataReportConfig); err != nil {
		logger.Errorf("Provider starts metadata report error, and the error is {%#v}", err)
		return
	}

	checkRegistries(consumerConfig.Registries, consumerConfig.Registry)
	for key, ref := range consumerConfig.References {
		if ref.Generic != "" {
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

	// Write current configuration to cache file.
	if consumerConfig.CacheFile != "" {
		if data, err := yaml.MarshalYML(consumerConfig); err != nil {
			logger.Errorf("Marshal consumer config err: %s", err.Error())
		} else {
			if err := ioutil.WriteFile(consumerConfig.CacheFile, data, 0666); err != nil {
				logger.Errorf("Write consumer config cache file err: %s", err.Error())
			}
		}
	}

	// wait for invoker is available, if wait over default 3s, then panic
	var count int
	for {
		checkok := true
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
	}
}

func loadProviderConfig() {
	if providerConfig == nil {
		logger.Debugf("Provider will not be launched, because providerConfig is not specified.")
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

	// start the metadata report if config set
	if err := startMetadataReport(GetApplicationConfig().MetadataType, GetBaseConfig().MetadataReportConfig); err != nil {
		logger.Errorf("Provider starts metadata report error, and the error is {%#v}", err)
		return
	}

	checkRegistries(providerConfig.Registries, providerConfig.Registry)

	// Write the current configuration to cache file.
	if providerConfig.CacheFile != "" {
		if data, err := yaml.MarshalYML(providerConfig); err != nil {
			logger.Errorf("Marshal provider config err: %s", err.Error())
		} else {
			if err := ioutil.WriteFile(providerConfig.CacheFile, data, 0666); err != nil {
				logger.Errorf("Write provider config cache file err: %s", err.Error())
			}
		}
	}

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
	registerServiceInstance()
}

// registerServiceInstance register service instance
func registerServiceInstance() {
	url := selectMetadataServiceExportedURL()
	if url == nil {
		return
	}
	instance, err := createInstance(url)
	if err != nil {
		panic(err)
	}
	p := extension.GetProtocol(constant.REGISTRY_KEY)
	var rp registry.RegistryFactory
	var ok bool
	if rp, ok = p.(registry.RegistryFactory); !ok {
		panic("dubbo registry protocol{" + reflect.TypeOf(p).String() + "} is invalid")
	}
	rs := rp.GetRegistries()
	for _, r := range rs {
		var sdr registry.ServiceDiscoveryHolder
		if sdr, ok = r.(registry.ServiceDiscoveryHolder); !ok {
			continue
		}
		err := sdr.GetServiceDiscovery().Register(instance)
		if err != nil {
			panic(err)
		}
	}
	// todo publish metadata to remote
	if remoteMetadataService, err := extension.GetRemoteMetadataService(); err == nil {
		remoteMetadataService.PublishMetadata(GetApplicationConfig().Name)
	}
}

// nolint
func createInstance(url *common.URL) (registry.ServiceInstance, error) {
	appConfig := GetApplicationConfig()
	port, err := strconv.ParseInt(url.Port, 10, 32)
	if err != nil {
		return nil, perrors.WithMessage(err, "invalid port: "+url.Port)
	}

	host := url.Ip
	if len(host) == 0 {
		host = common.GetLocalIp()
	}

	// usually we will add more metadata
	metadata := make(map[string]string, 8)
	metadata[constant.METADATA_STORAGE_TYPE_PROPERTY_NAME] = appConfig.MetadataType

	return &registry.DefaultServiceInstance{
		ServiceName: appConfig.Name,
		Host:        host,
		Port:        int(port),
		ID:          host + constant.KEY_SEPARATOR + url.Port,
		Enable:      true,
		Healthy:     true,
		Metadata:    metadata,
	}, nil
}

// selectMetadataServiceExportedURL get already be exported url
func selectMetadataServiceExportedURL() *common.URL {
	var selectedUrl *common.URL
	metaDataService, err := extension.GetLocalMetadataService("")
	if err != nil {
		logger.Warn(err)
		return nil
	}
	urlList, err := metaDataService.GetExportedURLs(constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE)
	if err != nil {
		panic(err)
	}
	if len(urlList) == 0 {
		return nil
	}
	for _, url := range urlList {
		selectedUrl = url
		// rest first
		if url.Protocol == "rest" {
			break
		}
	}
	return selectedUrl
}

func initRouter() {
	if uniformDestRuleConfigPath != "" && uniformVirtualServiceConfigPath != "" {
		if err := RouterInit(uniformVirtualServiceConfigPath, uniformDestRuleConfigPath); err != nil {
			logger.Warnf("[routerConfig init] %#v", err)
		}
	}
}

// Load Dubbo Init
func Load() {
	options := DefaultInit()
	LoadWithOptions(options...)
}

func LoadWithOptions(options ...LoaderInitOption) {
	// register metadata info and service info
	hessian.RegisterPOJO(&common.MetadataInfo{})
	hessian.RegisterPOJO(&common.ServiceInfo{})
	hessian.RegisterPOJO(&common.URL{})

	for _, option := range options {
		option.init()
	}
	for _, option := range options {
		option.apply()
	}
	// init router
	initRouter()

	// init the shutdown callback
	GracefulShutdownInit()
}

// GetRPCService get rpc service for consumer
func GetRPCService(name string) common.RPCService {
	return consumerConfig.References[name].GetRPCService()
}

// RPCService create rpc service for consumer
func RPCService(service common.RPCService) {
	ref := common.GetReference(service)
	consumerConfig.References[ref].Implement(service)
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
				Remotes:            make(map[string]*RemoteConfig),
				ApplicationConfig:  &ApplicationConfig{},
				ServiceDiscoveries: make(map[string]*ServiceDiscoveryConfig),
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
