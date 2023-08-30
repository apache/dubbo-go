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

package dubbo

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type RootConfig struct {
	Application         *ApplicationConfig                  `validate:"required" yaml:"application" json:"application,omitempty" property:"application"`
	Protocols           map[string]*protocol.ProtocolConfig `validate:"required" yaml:"protocols" json:"protocols" property:"protocols"`
	Registries          map[string]*registry.RegistryConfig `yaml:"registries" json:"registries" property:"registries"`
	ConfigCenter        *config_center.CenterConfig         `yaml:"config-center" json:"config-center,omitempty"`
	MetadataReport      *report.MetadataReportConfig        `yaml:"metadata-report" json:"metadata-report,omitempty" property:"metadata-report"`
	Provider            *ProviderConfig                     `yaml:"provider" json:"provider" property:"provider"`
	Consumer            *ConsumerConfig                     `yaml:"consumer" json:"consumer" property:"consumer"`
	Metric              *metrics.MetricConfig               `yaml:"metrics" json:"metrics,omitempty" property:"metrics"`
	Tracing             map[string]*TracingConfig           `yaml:"tracing" json:"tracing,omitempty" property:"tracing"`
	Logger              *LoggerConfig                       `yaml:"logger" json:"logger,omitempty" property:"logger"`
	loggerCompat        *config.LoggerConfig
	Shutdown            *ShutdownConfig `yaml:"shutdown" json:"shutdown,omitempty" property:"shutdown"`
	Router              []*RouterConfig `yaml:"router" json:"router,omitempty" property:"router"`
	EventDispatcherType string          `default:"direct" yaml:"event-dispatcher-type" json:"event-dispatcher-type,omitempty"`
	CacheFile           string          `yaml:"cache_file" json:"cache_file,omitempty" property:"cache_file"`
	Custom              *CustomConfig   `yaml:"custom" json:"custom,omitempty" property:"custom"`
	Profiles            *ProfilesConfig `yaml:"profiles" json:"profiles,omitempty" property:"profiles"`
	TLSConfig           *TLSConfig      `yaml:"tls_config" json:"tls_config,omitempty" property:"tls_config"`
}

func (rc *RootConfig) Init(opts ...RootOption) error {
	for _, opt := range opts {
		opt(rc)
	}

	registerPOJO()
	if rc.Logger != nil {

	}
	if err := rc.Logger.Init(); err != nil { // init default logger
		return err
	}
	if err := rc.ConfigCenter.Init(rc); err != nil {
		logger.Infof("[Config Center] Config center doesn't start")
		logger.Debugf("config center doesn't start because %s", err)
	} else {
		if err = rc.Logger.Init(); err != nil { // init logger using config from config center again
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

type RootOption func(*RootConfig)

// ---------- RootOption ----------

func WithApplication(opts ...ApplicationOption) RootOption {
	appCfg := new(ApplicationConfig)
	for _, opt := range opts {
		opt(appCfg)
	}

	return func(cfg *RootConfig) {
		cfg.Application = appCfg
	}
}

func WithProtocol(key string, opts ...protocol.ProtocolOption) RootOption {
	proCfg := new(protocol.ProtocolConfig)
	for _, opt := range opts {
		opt(proCfg)
	}

	return func(cfg *RootConfig) {
		if cfg.Protocols == nil {
			cfg.Protocols = make(map[string]*protocol.ProtocolConfig)
		}
		cfg.Protocols[key] = proCfg
	}
}

func WithRegistry(key string, opts ...registry.RegistryOption) RootOption {
	regCfg := new(registry.RegistryConfig)
	for _, opt := range opts {
		opt(regCfg)
	}

	return func(cfg *RootConfig) {
		if cfg.Registries == nil {
			cfg.Registries = make(map[string]*registry.RegistryConfig)
		}
		cfg.Registries[key] = regCfg
	}
}

func WithConfigCenter(opts ...config_center.CenterOption) RootOption {
	ccCfg := new(config_center.CenterConfig)
	for _, opt := range opts {
		opt(ccCfg)
	}

	return func(cfg *RootConfig) {
		cfg.ConfigCenter = ccCfg
	}
}

func WithMetadataReport(opts ...report.MetadataReportOption) RootOption {
	mrCfg := new(report.MetadataReportConfig)
	for _, opt := range opts {
		opt(mrCfg)
	}

	return func(cfg *RootConfig) {
		cfg.MetadataReport = mrCfg
	}
}

func WithConsumer(opts ...ConsumerOption) RootOption {
	conCfg := new(ConsumerConfig)
	for _, opt := range opts {
		opt(conCfg)
	}

	return func(cfg *RootConfig) {
		cfg.Consumer = conCfg
	}
}

func WithMetric(opts ...metrics.MetricOption) RootOption {
	meCfg := new(metrics.MetricConfig)
	for _, opt := range opts {
		opt(meCfg)
	}

	return func(cfg *RootConfig) {
		cfg.Metric = meCfg
	}
}

func WithTracing(key string, opts ...TracingOption) RootOption {
	traCfg := new(TracingConfig)
	for _, opt := range opts {
		opt(traCfg)
	}

	return func(cfg *RootConfig) {
		if cfg.Tracing == nil {
			cfg.Tracing = make(map[string]*TracingConfig)
		}
		cfg.Tracing[key] = traCfg
	}
}

func WithLogger(opts ...LoggerOption) RootOption {
	logCfg := new(LoggerConfig)
	for _, opt := range opts {
		opt(logCfg)
	}

	return func(cfg *RootConfig) {
		cfg.Logger = logCfg
	}
}

func WithShutdown(opts ...ShutdownOption) RootOption {
	sdCfg := new(ShutdownConfig)
	for _, opt := range opts {
		opt(sdCfg)
	}

	return func(cfg *RootConfig) {
		cfg.Shutdown = sdCfg
	}
}

func WithEventDispatcherType(typ string) RootOption {
	return func(cfg *RootConfig) {
		cfg.EventDispatcherType = typ
	}
}

func WithCacheFile(file string) RootOption {
	return func(cfg *RootConfig) {
		cfg.CacheFile = file
	}
}

func WithCustom(opts ...CustomOption) RootOption {
	cusCfg := new(CustomConfig)
	for _, opt := range opts {
		opt(cusCfg)
	}

	return func(cfg *RootConfig) {
		cfg.Custom = cusCfg
	}
}

func WithProfiles(opts ...ProfilesOption) RootOption {
	proCfg := new(ProfilesConfig)
	for _, opt := range opts {
		opt(proCfg)
	}

	return func(cfg *RootConfig) {
		cfg.Profiles = proCfg
	}
}

func WithTLS(opts ...TLSOption) RootOption {
	tlsCfg := new(TLSConfig)
	for _, opt := range opts {
		opt(tlsCfg)
	}

	return func(cfg *RootConfig) {
		cfg.TLSConfig = tlsCfg
	}
}
