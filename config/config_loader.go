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
	"dubbo.apache.org/dubbo-go/v3/config/application"
	"dubbo.apache.org/dubbo-go/v3/config/root"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

import (
	_ "dubbo.apache.org/dubbo-go/v3/common/observer/dispatcher"
)

var (
	// application config
	applicationConfig *application.Config
	rootConfig        *root.Config
	//consumerConfig *consumer.Config
	//providerConfig *provider.ProviderConfig
	//// baseConfig = providerConfig.BaseConfig or consumerConfig
	//baseConfig *root.Config
	//sslEnabled = false
	//
	//// configAccessMutex is used to make sure that xxxxConfig will only be created once if needed.
	//// it should be used combine with double-check to avoid the race condition
	//configAccessMutex sync.Mutex
	//
	//maxWait                         = 3
	//confRouterFile                  string
	//confBaseFile                    string
	//uniformVirtualServiceConfigPath string
	//uniformDestRuleConfigPath       string
)

type config struct {
	// config file name default application
	name string
	// config file type default yaml
	genre string
	// config file path default ./conf
	path string
	// config file delim default .
	delim string
}

type optionFunc func(*config)

func (fn optionFunc) apply(vc *config) {
	fn(vc)
}

type Option interface {
	apply(vc *config)
}

func Load(opts ...Option) {
	// pares CommandLine
	//parseCommandLine()
	// conf
	conf := &config{
		name:  "application.yaml",
		genre: "yaml",
		path:  "./conf",
		delim: ".",
	}

	for _, opt := range opts {
		opt.apply(conf)
	}
	rootConfig = new(root.Config)

	k := getKoanf(conf)

	if err := k.Unmarshal(rootConfig.Prefix(), &rootConfig); err != nil {
		panic(err)
	}

	rootConfig.Koanf = k
	rootConfig.Validate = validator.New()
}

// GetApplicationConfig get application config
func GetApplicationConfig() (*application.Config, error) {
	if rootConfig == nil {
		return nil, nil
	}
	if applicationConfig != nil {
		return applicationConfig, nil
	}
	conf := rootConfig.Application
	if err := conf.SetDefault(); err != nil {
		return nil, err
	}
	if err := conf.Validate(rootConfig.Validate); err != nil {
		return nil, err
	}
	applicationConfig = conf
	return conf, nil
}

//parseCommandLine parse command line
//func parseCommandLine() {
//	flag.String("delim", ".", "config file delim")
//	flag.String("name", "application.yaml", "config file name")
//	flag.String("genre", "yaml", "config file type")
//	flag.String("path", "./conf", "config file path default")
//
//	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
//	pflag.Parse()
//
//	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
//		panic(err)
//	}
//}

// WithGenre set config genre
func WithGenre(genre string) Option {
	return optionFunc(func(conf *config) {
		conf.genre = strings.ToLower(genre)
	})
}

// WithPath set config path
func WithPath(path string) Option {
	return optionFunc(func(conf *config) {
		conf.path = absolutePath(path)
	})
}

// WithName set config name
func WithName(name string) Option {
	return optionFunc(func(conf *config) {
		conf.name = name
	})
}

func WithDelim(delim string) Option {
	return optionFunc(func(conf *config) {
		conf.delim = delim
	})
}

// absolutePath get absolut path
func absolutePath(inPath string) string {

	if inPath == "$HOME" || strings.HasPrefix(inPath, "$HOME"+string(os.PathSeparator)) {
		inPath = userHomeDir() + inPath[5:]
	}

	if filepath.IsAbs(inPath) {
		return filepath.Clean(inPath)
	}

	p, err := filepath.Abs(inPath)
	if err == nil {
		return filepath.Clean(p)
	}

	return ""
}

//userHomeDir get gopath
func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func getKoanf(conf *config) *koanf.Koanf {
	var (
		k   *koanf.Koanf
		err error
	)

	k = koanf.New(conf.delim)

	switch conf.genre {

	case "yaml":
		err = k.Load(file.Provider(conf.path), yaml.Parser())
	case "json":
		err = k.Load(file.Provider(conf.path), json.Parser())
	case "toml":
		err = k.Load(file.Provider(conf.path), toml.Parser())
	default:
		err = errors.New(fmt.Sprintf("Unsupported %s file type", conf.genre))
	}

	if err != nil {
		panic(err)
	}
	return k
}

//
//// loaded consumer & provider config from xxx.yml, and log config from xxx.xml
//// Namely: dubbo.consumer.xml & dubbo.provider.xml in java dubbo
//func DefaultInit() []LoaderInitOption {
//	var (
//		confConFile string
//		confProFile string
//	)
//
//	fs := flag.NewFlagSet("config", flag.ContinueOnError)
//	fs.StringVar(&confConFile, "conConf", os.Getenv(constant.CONF_CONSUMER_FILE_PATH), "default client config path")
//	fs.StringVar(&confProFile, "proConf", os.Getenv(constant.CONF_PROVIDER_FILE_PATH), "default server config path")
//	fs.StringVar(&confRouterFile, "rouConf", os.Getenv(constant.CONF_ROUTER_FILE_PATH), "default router config path")
//	fs.StringVar(&uniformVirtualServiceConfigPath, "vsConf", os.Getenv(constant.CONF_VIRTUAL_SERVICE_FILE_PATH), "default virtual service of uniform router config path")
//	fs.StringVar(&uniformDestRuleConfigPath, "drConf", os.Getenv(constant.CONF_DEST_RULE_FILE_PATH), "default destination rule of uniform router config path")
//	fs.Parse(os.Args[1:])
//	for len(fs.Args()) != 0 {
//		fs.Parse(fs.Args()[1:])
//	}
//	// If user did not set the environment variables or flags,
//	// we provide default value
//	if confConFile == "" {
//		confConFile = constant.DEFAULT_CONSUMER_CONF_FILE_PATH
//	}
//	if confProFile == "" {
//		confProFile = constant.DEFAULT_PROVIDER_CONF_FILE_PATH
//	}
//	if confRouterFile == "" {
//		confRouterFile = constant.DEFAULT_ROUTER_CONF_FILE_PATH
//	}
//	return []LoaderInitOption{RouterInitOption(confRouterFile), BaseInitOption(""), ConsumerInitOption(confConFile), ProviderInitOption(confProFile)}
//}
//
//// setDefaultValue set default value for providerConfig or consumerConfig if it is null
//func setDefaultValue(target interface{}) {
//	registryConfig := &registry2.RegistryConfig{
//		Protocol:   constant.DEFAULT_REGISTRY_ZK_PROTOCOL,
//		TimeoutStr: constant.DEFAULT_REGISTRY_ZK_TIMEOUT,
//		Address:    constant.DEFAULT_REGISTRY_ZK_ADDRESS,
//	}
//	switch target.(type) {
//	case *provider.ProviderConfig:
//		p := target.(*provider.ProviderConfig)
//		if len(p.Registries) == 0 && p.Registry == nil {
//			p.Registries[constant.DEFAULT_REGISTRY_ZK_ID] = registryConfig
//		}
//		if len(p.Protocols) == 0 {
//			p.Protocols[constant.DEFAULT_PROTOCOL] = &protocol.ProtocolConfig{
//				Name: constant.DEFAULT_PROTOCOL,
//				Port: strconv.Itoa(constant.DEFAULT_PORT),
//			}
//		}
//		if p.ApplicationConfig == nil {
//			p.ApplicationConfig = NewDefaultApplicationConfig()
//		}
//	default:
//		c := target.(*consumer.Config)
//		if len(c.Registries) == 0 && c.Registry == nil {
//			c.Registries[constant.DEFAULT_REGISTRY_ZK_ID] = registryConfig
//		}
//		if c.ApplicationConfig == nil {
//			c.ApplicationConfig = NewDefaultApplicationConfig()
//		}
//	}
//}
//
//func checkRegistries(registries map[string]*registry2.RegistryConfig, singleRegistry *registry2.RegistryConfig) {
//	if len(registries) == 0 && singleRegistry != nil {
//		registries[constant.DEFAULT_KEY] = singleRegistry
//	}
//}
//
//func checkApplicationName(config *application.Config) {
//	if config == nil || len(config.Name) == 0 {
//		errMsg := "application config must not be nil, pls check your configuration"
//		logger.Errorf(errMsg)
//		panic(errMsg)
//	}
//}
//
//func loadConsumerConfig() {
//	if consumerConfig == nil {
//		logger.Warnf("consumerConfig is nil!")
//		return
//	}
//	// init other consumer config
//	conConfigType := consumerConfig.ConfigType
//	for key, value := range extension.GetDefaultConfigReader() {
//		if conConfigType != nil {
//			if v, ok := conConfigType[key]; ok {
//				value = v
//			}
//		}
//		if err := extension.GetConfigReaders(value).ReadConsumerConfig(consumerConfig.fileStream); err != nil {
//			logger.Errorf("ReadConsumerConfig error: %#v for %s", perrors.WithStack(err), value)
//		}
//	}
//
//	checkApplicationName(consumerConfig.ApplicationConfig)
//	if err := consumer.configCenterRefreshConsumer(); err != nil {
//		logger.Errorf("[consumer config center refresh] %#v", err)
//	}
//
//	// start the metadata report if config set
//	if err := report.startMetadataReport(GetApplicationConfig().MetadataType, GetBaseConfig().MetadataReportConfig); err != nil {
//		logger.Errorf("Provider starts metadata report error, and the error is {%#v}", err)
//		return
//	}
//
//	checkRegistries(consumerConfig.Registries, consumerConfig.Registry)
//	for key, ref := range consumerConfig.References {
//		if ref.Generic {
//			genericService := generic.NewGenericService(key)
//			instance.SetConsumerService(genericService)
//		}
//		rpcService := instance.GetConsumerService(key)
//		if rpcService == nil {
//			logger.Warnf("%s does not exist!", key)
//			continue
//		}
//		ref.id = key
//		ref.Refer(rpcService)
//		ref.Implement(rpcService)
//	}
//
//	// Write current configuration to cache file.
//	if consumerConfig.CacheFile != "" {
//		if data, err := yaml.MarshalYML(consumerConfig); err != nil {
//			logger.Errorf("Marshal consumer config err: %s", err.Error())
//		} else {
//			if err := ioutil.WriteFile(consumerConfig.CacheFile, data, 0666); err != nil {
//				logger.Errorf("Write consumer config cache file err: %s", err.Error())
//			}
//		}
//	}
//
//	// wait for invoker is available, if wait over default 3s, then panic
//	var count int
//	for {
//		checkok := true
//		for _, refconfig := range consumerConfig.References {
//			if (refconfig.Check != nil && *refconfig.Check) ||
//				(refconfig.Check == nil && consumerConfig.Check != nil && *consumerConfig.Check) ||
//				(refconfig.Check == nil && consumerConfig.Check == nil) { // default to true
//
//				if refconfig.invoker != nil && !refconfig.invoker.IsAvailable() {
//					checkok = false
//					count++
//					if count > maxWait {
//						errMsg := fmt.Sprintf("Failed to check the status of the service %v. No provider available for the service to the consumer use dubbo version %v", refconfig.InterfaceName, constant.Version)
//						logger.Error(errMsg)
//						panic(errMsg)
//					}
//					time.Sleep(time.Second * 1)
//					break
//				}
//				if refconfig.invoker == nil {
//					logger.Warnf("The interface %s invoker not exist, may you should check your interface config.", refconfig.InterfaceName)
//				}
//			}
//		}
//		if checkok {
//			break
//		}
//	}
//}
//
//func loadProviderConfig() {
//	if providerConfig == nil {
//		logger.Warnf("providerConfig is nil!")
//		return
//	}
//
//	// init other provider config
//	proConfigType := providerConfig.ConfigType
//	for key, value := range extension.GetDefaultConfigReader() {
//		if proConfigType != nil {
//			if v, ok := proConfigType[key]; ok {
//				value = v
//			}
//		}
//		if err := extension.GetConfigReaders(value).ReadProviderConfig(providerConfig.fileStream); err != nil {
//			logger.Errorf("ReadProviderConfig error: %#v for %s", perrors.WithStack(err), value)
//		}
//	}
//
//	checkApplicationName(providerConfig.ApplicationConfig)
//	if err := provider.configCenterRefreshProvider(); err != nil {
//		logger.Errorf("[provider config center refresh] %#v", err)
//	}
//
//	// start the metadata report if config set
//	if err := report.startMetadataReport(GetApplicationConfig().MetadataType, GetBaseConfig().MetadataReportConfig); err != nil {
//		logger.Errorf("Provider starts metadata report error, and the error is {%#v}", err)
//		return
//	}
//
//	checkRegistries(providerConfig.Registries, providerConfig.Registry)
//
//	// Write the current configuration to cache file.
//	if providerConfig.CacheFile != "" {
//		if data, err := yaml.MarshalYML(providerConfig); err != nil {
//			logger.Errorf("Marshal provider config err: %s", err.Error())
//		} else {
//			if err := ioutil.WriteFile(providerConfig.CacheFile, data, 0666); err != nil {
//				logger.Errorf("Write provider config cache file err: %s", err.Error())
//			}
//		}
//	}
//
//	for key, svs := range providerConfig.Services {
//		rpcService := instance.GetProviderService(key)
//		if rpcService == nil {
//			logger.Warnf("%s does not exist!", key)
//			continue
//		}
//		svs.id = key
//		svs.Implement(rpcService)
//		svs.Protocols = providerConfig.Protocols
//		if err := svs.Export(); err != nil {
//			panic(fmt.Sprintf("service %s export failed! err: %#v", key, err))
//		}
//	}
//	registerServiceInstance()
//}
//
//// registerServiceInstance register service instance
//func registerServiceInstance() {
//	url := selectMetadataServiceExportedURL()
//	if url == nil {
//		return
//	}
//	instance, err := createInstance(url)
//	if err != nil {
//		panic(err)
//	}
//	p := extension.GetProtocol(constant.REGISTRY_KEY)
//	var rp registry.RegistryFactory
//	var ok bool
//	if rp, ok = p.(registry.RegistryFactory); !ok {
//		panic("dubbo registry protocol{" + reflect.TypeOf(p).String() + "} is invalid")
//	}
//	rs := rp.GetRegistries()
//	for _, r := range rs {
//		var sdr registry.ServiceDiscoveryHolder
//		if sdr, ok = r.(registry.ServiceDiscoveryHolder); !ok {
//			continue
//		}
//		err := sdr.GetServiceDiscovery().Register(instance)
//		if err != nil {
//			panic(err)
//		}
//	}
//	// todo publish metadata to remote
//	if remoteMetadataService, err := extension.GetRemoteMetadataService(); err == nil {
//		remoteMetadataService.PublishMetadata(GetApplicationConfig().Name)
//	}
//}
//
//// nolint
//func createInstance(url *common.URL) (registry.ServiceInstance, error) {
//	appConfig := GetApplicationConfig()
//	port, err := strconv.ParseInt(url.Port, 10, 32)
//	if err != nil {
//		return nil, perrors.WithMessage(err, "invalid port: "+url.Port)
//	}
//
//	host := url.Ip
//	if len(host) == 0 {
//		host = common.GetLocalIp()
//	}
//
//	// usually we will add more metadata
//	metadata := make(map[string]string, 8)
//	metadata[constant.METADATA_STORAGE_TYPE_PROPERTY_NAME] = appConfig.MetadataType
//
//	return &registry.DefaultServiceInstance{
//		ServiceName: appConfig.Name,
//		Host:        host,
//		Port:        int(port),
//		ID:          host + constant.KEY_SEPARATOR + url.Port,
//		Enable:      true,
//		Healthy:     true,
//		Metadata:    metadata,
//	}, nil
//}
//
//// selectMetadataServiceExportedURL get already be exported url
//func selectMetadataServiceExportedURL() *common.URL {
//	var selectedUrl *common.URL
//	metaDataService, err := extension.GetLocalMetadataService("")
//	if err != nil {
//		logger.Warn(err)
//		return nil
//	}
//	urlList, err := metaDataService.GetExportedURLs(constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE)
//	if err != nil {
//		panic(err)
//	}
//	if len(urlList) == 0 {
//		return nil
//	}
//	for _, url := range urlList {
//		selectedUrl = url
//		// rest first
//		if url.Protocol == "rest" {
//			break
//		}
//	}
//	return selectedUrl
//}
//
//func initRouter() {
//	if uniformDestRuleConfigPath != "" && uniformVirtualServiceConfigPath != "" {
//		if err := router.RouterInit(uniformVirtualServiceConfigPath, uniformDestRuleConfigPath); err != nil {
//			logger.Warnf("[routerConfig init] %#v", err)
//		}
//	}
//}
//
//// Load Dubbo Init
//func Load() {
//	options := DefaultInit()
//	LoadWithOptions(options...)
//}
//
//func LoadWithOptions(options ...LoaderInitOption) {
//	// register metadata info and service info
//	hessian.RegisterPOJO(&common.MetadataInfo{})
//	hessian.RegisterPOJO(&common.ServiceInfo{})
//	hessian.RegisterPOJO(&common.URL{})
//
//	for _, option := range options {
//		option.init()
//	}
//	for _, option := range options {
//		option.apply()
//	}
//	// init router
//	initRouter()
//
//	// init the shutdown callback
//	shutdown.GracefulShutdownInit()
//}
//
//// GetRPCService get rpc service for consumer
//func GetRPCService(name string) common.RPCService {
//	return consumerConfig.References[name].GetRPCService()
//}
//
//// RPCService create rpc service for consumer
//func RPCService(service common.RPCService) {
//	consumerConfig.References[service.Reference()].Implement(service)
//}
//
//// GetMetricConfig find the MetricConfig
//// if it is nil, create a new one
//// we use double-check to reduce race condition
//// In general, it will be locked 0 or 1 time.
//// So you don't need to worry about the race condition
//func GetMetricConfig() *metric.MetricConfig {
//	if GetBaseConfig().MetricConfig == nil {
//		configAccessMutex.Lock()
//		defer configAccessMutex.Unlock()
//		if GetBaseConfig().MetricConfig == nil {
//			GetBaseConfig().MetricConfig = &metric.MetricConfig{}
//		}
//	}
//	return GetBaseConfig().MetricConfig
//}
//
//// GetApplicationConfig find the application config
//// if not, we will create one
//// Usually applicationConfig will be initialized when system start
//// we use double-check to reduce race condition
//// In general, it will be locked 0 or 1 time.
//// So you don't need to worry about the race condition
//func GetApplicationConfig() *application.Config {
//	if GetBaseConfig().ApplicationConfig == nil {
//		configAccessMutex.Lock()
//		defer configAccessMutex.Unlock()
//		if GetBaseConfig().ApplicationConfig == nil {
//			GetBaseConfig().ApplicationConfig = &application.Config{}
//		}
//	}
//	return GetBaseConfig().ApplicationConfig
//}
//
//// GetProviderConfig find the provider config
//// if not found, create new one
//func GetProviderConfig() provider.ProviderConfig {
//	if providerConfig == nil {
//		if providerConfig == nil {
//			return provider.ProviderConfig{}
//		}
//	}
//	return *providerConfig
//}
//
//// GetConsumerConfig find the consumer config
//// if not found, create new one
//// we use double-check to reduce race condition
//// In general, it will be locked 0 or 1 time.
//// So you don't need to worry about the race condition
//func GetConsumerConfig() consumer.Config {
//	if consumerConfig == nil {
//		if consumerConfig == nil {
//			return consumer.Config{}
//		}
//	}
//	return *consumerConfig
//}
//
//func GetBaseConfig() *base.Config {
//	if baseConfig == nil {
//		configAccessMutex.Lock()
//		defer configAccessMutex.Unlock()
//		if baseConfig == nil {
//			baseConfig = &base.Config{
//				metric.MetricConfig: &metric.MetricConfig{},
//				ConfigCenterConfig:  &center.Config{},
//				Remotes:             make(map[string]*RemoteConfig),
//				application.Config:  &application.Config{},
//				ServiceDiscoveries:  make(map[string]*discovery.Config),
//			}
//		}
//	}
//	return baseConfig
//}
//
//func GetSslEnabled() bool {
//	return sslEnabled
//}
//
//func SetSslEnabled(enabled bool) {
//	sslEnabled = enabled
//}
//
//func IsProvider() bool {
//	return providerConfig != nil
//}
