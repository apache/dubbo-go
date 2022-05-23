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

	"github.com/knadh/koanf"

	perrors "github.com/pkg/errors"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/exporter"
)

var (
	startOnce sync.Once
	exporting = &atomic.Bool{}
)

// RootConfig is the root config
type RootConfig struct {
	Application         *ApplicationConfig         `validate:"required" yaml:"application" json:"application,omitempty" property:"application"`
	Protocols           map[string]*ProtocolConfig `validate:"required" yaml:"protocols" json:"protocols" property:"protocols"`
	Registries          map[string]*RegistryConfig `yaml:"registries" json:"registries" property:"registries"`
	ConfigCenter        *CenterConfig              `yaml:"config-center" json:"config-center,omitempty"` // TODO ConfigCenter and CenterConfig?
	MetadataReport      *MetadataReportConfig      `yaml:"metadata-report" json:"metadata-report,omitempty" property:"metadata-report"`
	Provider            *ProviderConfig            `yaml:"provider" json:"provider" property:"provider"`
	Consumer            *ConsumerConfig            `yaml:"consumer" json:"consumer" property:"consumer"`
	Metric              *MetricConfig              `yaml:"metrics" json:"metrics,omitempty" property:"metrics"`
	Tracing             map[string]*TracingConfig  `yaml:"tracing" json:"tracing,omitempty" property:"tracing"`
	Logger              *LoggerConfig              `yaml:"logger" json:"logger,omitempty" property:"logger"`
	Shutdown            *ShutdownConfig            `yaml:"shutdown" json:"shutdown,omitempty" property:"shutdown"`
	Router              []*RouterConfig            `yaml:"router" json:"router,omitempty" property:"router"`
	EventDispatcherType string                     `default:"direct" yaml:"event-dispatcher-type" json:"event-dispatcher-type,omitempty"`
	CacheFile           string                     `yaml:"cache_file" json:"cache_file,omitempty" property:"cache_file"`
	Custom              *CustomConfig              `yaml:"custom" json:"custom,omitempty" property:"custom"`
	Profiles            *ProfilesConfig            `yaml:"profiles" json:"profiles,omitempty" property:"profiles"`
	RestProvider        *RestProviderConfig        `yaml:"provider" json:"rest-provider" property:"rest-provider"`
	RestConsumer        *RestConsumerConfig        `yaml:"consumer" json:"rest-consumer" property:"rest-consumer"`
}

func SetRootConfig(r RootConfig) {
	rootConfig = &r
}

// Prefix dubbo
func (RootConfig) Prefix() string {
	return constant.Dubbo
}

func GetRootConfig() *RootConfig {
	return rootConfig
}

func GetProviderConfig() *ProviderConfig {
	if err := check(); err == nil && rootConfig.Provider != nil {
		return rootConfig.Provider
	}
	return NewProviderConfigBuilder().Build()
}

func GetConsumerConfig() *ConsumerConfig {
	if err := check(); err == nil && rootConfig.Consumer != nil {
		return rootConfig.Consumer
	}
	return NewConsumerConfigBuilder().Build()
}

func GetApplicationConfig() *ApplicationConfig {
	return rootConfig.Application
}

func GetShutDown() *ShutdownConfig {
	if err := check(); err == nil && rootConfig.Shutdown != nil {
		return rootConfig.Shutdown
	}
	return NewShutDownConfigBuilder().Build()
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

// Init is to start dubbo-go framework, load local configuration, or read configuration from config-center if necessary.
// It's deprecated for user to call rootConfig.Init() manually, try config.Load(config.WithRootConfig(rootConfig)) instead.
func (rc *RootConfig) Init() error {
	registerPOJO()
	if err := rc.Logger.Init(); err != nil { // init default logger
		return err
	}
	if err := rc.ConfigCenter.Init(rc); err != nil {
		logger.Infof("[Config Center] Config center doesn't start")
		logger.Debugf("config center doesn't start because %s", err)
	} else {
		if err := rc.Logger.Init(); err != nil { // init logger using config from config center again
			return err
		}
	}

	if err := rc.Application.Init(); err != nil {
		return err
	}

	// init user define
	if err := rc.Custom.Init(); err != nil {
		return err
	}

	// init protocol
	protocols := rc.Protocols
	if len(protocols) <= 0 {
		protocol := &ProtocolConfig{}
		protocols = make(map[string]*ProtocolConfig, 1)
		protocols[constant.Dubbo] = protocol
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

	if err := rc.MetadataReport.Init(rc); err != nil {
		return err
	}
	if err := rc.Metric.Init(); err != nil {
		return err
	}
	for _, t := range rc.Tracing {
		if err := t.Init(); err != nil {
			return err
		}
	}
	if err := initRouterConfig(rc); err != nil {
		return err
	}

	// pi rest provider/consumer init
	if err := rc.RestProvider.Init(rc); err != nil {
		return err
	}
	if err := rc.RestConsumer.Init(rc); err != nil {
		return err
	}

	// provider、consumer must last init
	if err := rc.Provider.Init(rc); err != nil {
		return err
	}
	if err := rc.Consumer.Init(rc); err != nil {
		return err
	}
	if err := rc.Shutdown.Init(); err != nil {
		return err
	}
	SetRootConfig(*rc)
	// todo if we can remove this from Init in the future?
	rc.Start()
	return nil
}

func (rc *RootConfig) Start() {
	startOnce.Do(func() {
		gracefulShutdownInit()
		rc.Consumer.Load()
		rc.Provider.Load()
		exportMetadataService()
		registerServiceInstance()
	})
}

// newEmptyRootConfig get empty root config
func newEmptyRootConfig() *RootConfig {
	newRootConfig := &RootConfig{
		ConfigCenter:   NewConfigCenterConfigBuilder().Build(),
		MetadataReport: NewMetadataReportConfigBuilder().Build(),
		Application:    NewApplicationConfigBuilder().Build(),
		Registries:     make(map[string]*RegistryConfig),
		Protocols:      make(map[string]*ProtocolConfig),
		Tracing:        make(map[string]*TracingConfig),
		Provider:       NewProviderConfigBuilder().Build(),
		Consumer:       NewConsumerConfigBuilder().Build(),
		// pi rest provider/consumer builder
		RestProvider: NewRestProviderConfigBuilder().Build(),
		RestConsumer: NewRestConsumerConfigBuilder().Build(),
		Metric:       NewMetricConfigBuilder().Build(),
		Logger:       NewLoggerConfigBuilder().Build(),
		Custom:       NewCustomConfigBuilder().Build(),
		Shutdown:     NewShutDownConfigBuilder().Build(),
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

func (rb *RootConfigBuilder) AddProtocol(protocolID string, protocolConfig *ProtocolConfig) *RootConfigBuilder {
	rb.rootConfig.Protocols[protocolID] = protocolConfig
	return rb
}

func (rb *RootConfigBuilder) AddRegistry(registryID string, registryConfig *RegistryConfig) *RootConfigBuilder {
	rb.rootConfig.Registries[registryID] = registryConfig
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

func (rb *RootConfigBuilder) SetCustom(customConfig *CustomConfig) *RootConfigBuilder {
	rb.rootConfig.Custom = customConfig
	return rb
}

func (rb *RootConfigBuilder) SetShutDown(shutDownConfig *ShutdownConfig) *RootConfigBuilder {
	rb.rootConfig.Shutdown = shutDownConfig
	return rb
}

func (rb *RootConfigBuilder) Build() *RootConfig {
	return rb.rootConfig
}

func exportMetadataService() {
	ms, err := extension.GetLocalMetadataService(constant.DefaultKey)
	if err != nil {
		logger.Warnf("could not init metadata service", err)
		return
	}

	if !IsProvider() || exporting.Load() {
		return
	}

	// In theory, we can use sync.Once
	// But sync.Once is not reentrant.
	// Now the invocation chain is createRegistry -> tryInitMetadataService -> metadataServiceExporter.export
	// -> createRegistry -> initMetadataService...
	// So using sync.Once will result in dead lock
	exporting.Store(true)

	expt := extension.GetMetadataServiceExporter(constant.DefaultKey, ms)
	if expt == nil {
		logger.Warnf("get metadata service exporter failed, pls check if you import _ \"dubbo.apache.org/dubbo-go/v3/metadata/service/exporter/configurable\"")
		return
	}

	err = expt.Export(nil)
	if err != nil {
		logger.Errorf("could not export the metadata service, err = %s", err.Error())
		return
	}

	// report interface-app mapping
	err = publishMapping(expt)
	if err != nil {
		logger.Errorf("Publish interface-application mapping failed, got error %#v", err)
	}
}

// OnEvent only handle ServiceConfigExportedEvent
func publishMapping(sc exporter.MetadataServiceExporter) error {
	urls := sc.GetExportedURLs()

	for _, u := range urls {
		err := extension.GetGlobalServiceNameMapping().Map(u)
		if err != nil {
			return perrors.WithMessage(err, "could not map the service: "+u.String())
		}
	}
	return nil
}

// Process receive changing listener's event, dynamic update config
func (rc *RootConfig) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("CenterConfig process event:\n%+v", event)
	config := NewLoaderConf(WithBytes([]byte(event.Value.(string))))
	koan := GetConfigResolver(config)

	updateRootConfig := &RootConfig{}
	if err := koan.UnmarshalWithConf(rc.Prefix(),
		updateRootConfig, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		logger.Errorf("CenterConfig process unmarshalConf failed, got error %#v", err)
		return
	}
	// dynamically update register
	for registerId, updateRegister := range updateRootConfig.Registries {
		register := rc.Registries[registerId]
		register.DynamicUpdateProperties(updateRegister)
	}
	// dynamically update consumer
	rc.Consumer.DynamicUpdateProperties(updateRootConfig.Consumer)
}
