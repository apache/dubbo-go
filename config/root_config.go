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
	_ "net/http/pprof"
	"sync"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
)

var (
	startOnce sync.Once
)

// RootConfig is the root config
type RootConfig struct {
	// Application applicationConfig config
	Application *ApplicationConfig `validate:"required" yaml:"application" json:"application,omitempty" property:"application"`

	Protocols map[string]*ProtocolConfig `validate:"required" yaml:"protocols" json:"protocols" property:"protocols"`

	// Registries registry config
	Registries map[string]*RegistryConfig `yaml:"registries" json:"registries" property:"registries"`

	// Remotes to be remove in 3.0 config-enhance
	Remotes map[string]*RemoteConfig `yaml:"remote" json:"remote,omitempty" property:"remote"`

	// TODO ConfigCenter and CenterConfig?
	ConfigCenter *CenterConfig `yaml:"config-center" json:"config-center,omitempty"`

	// ServiceDiscoveries to be remove in 3.0 config-enhance
	ServiceDiscoveries map[string]*ServiceDiscoveryConfig `yaml:"service-discovery" json:"service-discovery,omitempty" property:"service-discovery"`

	MetadataReport *MetadataReportConfig `yaml:"metadata-report" json:"metadata-report,omitempty" property:"metadata-report"`

	// provider config
	Provider *ProviderConfig `yaml:"provider" json:"provider" property:"provider"`

	// consumer config
	Consumer *ConsumerConfig `yaml:"consumer" json:"consumer" property:"consumer"`

	Metric *MetricConfig `yaml:"metrics" json:"metrics,omitempty" property:"metrics"`

	// Logger log
	Logger *LoggerConfig `yaml:"logger" json:"logger,omitempty" property:"logger"`

	// Shutdown config
	Shutdown *ShutdownConfig `yaml:"shutdown" json:"shutdown,omitempty" property:"shutdown"`

	Router []*RouterConfig `yaml:"router" json:"router,omitempty" property:"router"`

	EventDispatcherType string `default:"direct" yaml:"event-dispatcher-type" json:"event-dispatcher-type,omitempty"`

	// cache file used to store the current used configurations.
	CacheFile string `yaml:"cache_file" json:"cache_file,omitempty" property:"cache_file"`
}

func SetRootConfig(r RootConfig) {
	rootConfig = &r
}

// Prefix dubbo
func (RootConfig) Prefix() string {
	return constant.DUBBO
}

func GetRootConfig() *RootConfig {
	return rootConfig
}

func GetProviderConfig() *ProviderConfig {
	if err := check(); err != nil {
		return NewProviderConfigBuilder().Build()
	}
	if rootConfig.Provider != nil {
		return rootConfig.Provider
	}
	return NewProviderConfigBuilder().Build()
}

func GetConsumerConfig() *ConsumerConfig {
	if err := check(); err != nil {
		return NewConsumerConfigBuilder().Build()
	}
	if rootConfig.Consumer != nil {
		return rootConfig.Consumer
	}
	return NewConsumerConfigBuilder().Build()
}

func GetApplicationConfig() *ApplicationConfig {
	return rootConfig.Application
}

// getRegistryIds get registry ids
func (rc *RootConfig) getRegistryIds() []string {
	ids := make([]string, 0)
	for key := range rc.Registries {
		ids = append(ids, key)
	}
	return removeDuplicateElement(ids)
}
func registerPOJO() {
	hessian.RegisterPOJO(&common.MetadataInfo{})
	hessian.RegisterPOJO(&common.ServiceInfo{})
	hessian.RegisterPOJO(&common.URL{})
}

func (rc *RootConfig) Init() error {
	registerPOJO()
	if err := rc.Logger.Init(); err != nil {
		return err
	}
	if err := rc.ConfigCenter.Init(rc); err != nil {
		logger.Warnf("config center doesn't start. error is %s", err)
	}
	if err := rc.Application.Init(); err != nil {
		return err
	}

	// init protocol
	protocols := rc.Protocols
	if len(protocols) <= 0 {
		protocol := &ProtocolConfig{}
		protocols = make(map[string]*ProtocolConfig, 1)
		protocols[constant.DUBBO] = protocol
		rc.Protocols = protocols
	}
	for _, protocol := range protocols {
		if err := protocol.Init(); err != nil {
			return err
		}
	}

	// init registry
	registries := rc.Registries
	if registries != nil {
		for _, reg := range registries {
			if err := reg.Init(); err != nil {
				return err
			}
		}
	}

	// init serviceDiscoveries
	serviceDiscoveries := rc.ServiceDiscoveries
	if serviceDiscoveries != nil {
		for _, sd := range serviceDiscoveries {
			if err := sd.Init(); err != nil {
				return err
			}
		}
	}

	if err := rc.MetadataReport.Init(rc); err != nil {
		return err
	}
	if err := rc.Metric.Init(); err != nil {
		return err
	}
	if err := initRouterConfig(rc); err != nil {
		return err
	}
	// provider、consumer must last init
	if err := rc.Provider.Init(rc); err != nil {
		return err
	}
	if err := rc.Consumer.Init(rc); err != nil {
		return err
	}

	rc.Start()
	return nil
}

func (rc *RootConfig) Start() {
	startOnce.Do(func() {
		rc.Provider.Load()
		rc.Consumer.Load()
		registerServiceInstance()
	})
}

// newEmptyRootConfig get empty root config
func newEmptyRootConfig() *RootConfig {
	newRootConfig := &RootConfig{
		ConfigCenter:       NewConfigCenterConfigBuilder().Build(),
		ServiceDiscoveries: make(map[string]*ServiceDiscoveryConfig),
		MetadataReport:     NewMetadataReportConfigBuilder().Build(),
		Application:        NewApplicationConfigBuilder().Build(),
		Registries:         make(map[string]*RegistryConfig),
		Protocols:          make(map[string]*ProtocolConfig),
		Provider:           NewProviderConfigBuilder().Build(),
		Consumer:           NewConsumerConfigBuilder().Build(),
		Metric:             NewMetricConfigBuilder().Build(),
		Logger:             NewLoggerConfigBuilder().Build(),
	}
	return newRootConfig
}

func NewRootConfigBuilder() *RootConfigBuilder {
	return &RootConfigBuilder{rootConfig: newEmptyRootConfig()}
}

type RootConfigBuilder struct {
	rootConfig *RootConfig
}

func (rb *RootConfigBuilder) SetApplication(application *ApplicationConfig) *RootConfigBuilder {
	rb.rootConfig.Application = application
	return rb
}

func (rb *RootConfigBuilder) AddProtocol(protocolKey string, protocolConfig *ProtocolConfig) *RootConfigBuilder {
	rb.rootConfig.Protocols[protocolKey] = protocolConfig
	return rb
}

func (rb *RootConfigBuilder) AddRegistry(registryKey string, registryConfig *RegistryConfig) *RootConfigBuilder {
	rb.rootConfig.Registries[registryKey] = registryConfig
	return rb
}

func (rb *RootConfigBuilder) SetProtocols(protocols map[string]*ProtocolConfig) *RootConfigBuilder {
	rb.rootConfig.Protocols = protocols
	return rb
}

func (rb *RootConfigBuilder) SetRegistries(registries map[string]*RegistryConfig) *RootConfigBuilder {
	rb.rootConfig.Registries = registries
	return rb
}

func (rb *RootConfigBuilder) SetRemotes(remotes map[string]*RemoteConfig) *RootConfigBuilder {
	rb.rootConfig.Remotes = remotes
	return rb
}

func (rb *RootConfigBuilder) SetServiceDiscoveries(serviceDiscoveries map[string]*ServiceDiscoveryConfig) *RootConfigBuilder {
	rb.rootConfig.ServiceDiscoveries = serviceDiscoveries
	return rb
}

func (rb *RootConfigBuilder) SetMetadataReport(metadataReport *MetadataReportConfig) *RootConfigBuilder {
	rb.rootConfig.MetadataReport = metadataReport
	return rb
}

func (rb *RootConfigBuilder) SetProvider(provider *ProviderConfig) *RootConfigBuilder {
	rb.rootConfig.Provider = provider
	return rb
}

func (rb *RootConfigBuilder) SetConsumer(consumer *ConsumerConfig) *RootConfigBuilder {
	rb.rootConfig.Consumer = consumer
	return rb
}

func (rb *RootConfigBuilder) SetMetric(metric *MetricConfig) *RootConfigBuilder {
	rb.rootConfig.Metric = metric
	return rb
}

func (rb *RootConfigBuilder) SetLogger(logger *LoggerConfig) *RootConfigBuilder {
	rb.rootConfig.Logger = logger
	return rb
}

func (rb *RootConfigBuilder) SetShutdown(shutdown *ShutdownConfig) *RootConfigBuilder {
	rb.rootConfig.Shutdown = shutdown
	return rb
}

func (rb *RootConfigBuilder) SetRouter(router []*RouterConfig) *RootConfigBuilder {
	rb.rootConfig.Router = router
	return rb
}

func (rb *RootConfigBuilder) SetEventDispatcherType(eventDispatcherType string) *RootConfigBuilder {
	rb.rootConfig.EventDispatcherType = eventDispatcherType
	return rb
}

func (rb *RootConfigBuilder) SetCacheFile(cacheFile string) *RootConfigBuilder {
	rb.rootConfig.CacheFile = cacheFile
	return rb
}

func (rb *RootConfigBuilder) SetConfigCenter(configCenterConfig *CenterConfig) *RootConfigBuilder {
	rb.rootConfig.ConfigCenter = configCenterConfig
	return rb
}

func (rb *RootConfigBuilder) Build() *RootConfig {
	return rb.rootConfig
}
