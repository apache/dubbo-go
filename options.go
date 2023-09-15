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
	"dubbo.apache.org/dubbo-go/v3/global"
)

type InstanceOptions struct {
	Application    *global.ApplicationConfig         `validate:"required" yaml:"application" json:"application,omitempty" property:"application"`
	Protocols      map[string]*global.ProtocolConfig `validate:"required" yaml:"protocols" json:"protocols" property:"protocols"`
	Registries     map[string]*global.RegistryConfig `yaml:"registries" json:"registries" property:"registries"`
	ConfigCenter   *global.CenterConfig              `yaml:"config-center" json:"config-center,omitempty"`
	MetadataReport *global.MetadataReportConfig      `yaml:"metadata-report" json:"metadata-report,omitempty" property:"metadata-report"`
	Provider       *global.ProviderConfig            `yaml:"provider" json:"provider" property:"provider"`
	Consumer       *global.ConsumerConfig            `yaml:"consumer" json:"consumer" property:"consumer"`
	Metric         *global.MetricConfig              `yaml:"metrics" json:"metrics,omitempty" property:"metrics"`
	Tracing        map[string]*global.TracingConfig  `yaml:"tracing" json:"tracing,omitempty" property:"tracing"`
	Logger         *global.LoggerConfig              `yaml:"logger" json:"logger,omitempty" property:"logger"`
	Shutdown       *global.ShutdownConfig            `yaml:"shutdown" json:"shutdown,omitempty" property:"shutdown"`
	// todo(DMwangnima): router feature would be supported in the future
	//Router              []*RouterConfig                   `yaml:"router" json:"router,omitempty" property:"router"`
	EventDispatcherType string                 `default:"direct" yaml:"event-dispatcher-type" json:"event-dispatcher-type,omitempty"`
	CacheFile           string                 `yaml:"cache_file" json:"cache_file,omitempty" property:"cache_file"`
	Custom              *global.CustomConfig   `yaml:"custom" json:"custom,omitempty" property:"custom"`
	Profiles            *global.ProfilesConfig `yaml:"profiles" json:"profiles,omitempty" property:"profiles"`
	TLSConfig           *global.TLSConfig      `yaml:"tls_config" json:"tls_config,omitempty" property:"tls_config"`
}

func defaultInstanceOptions() *InstanceOptions {
	return &InstanceOptions{
		Application:    global.DefaultApplicationConfig(),
		Protocols:      make(map[string]*global.ProtocolConfig),
		Registries:     make(map[string]*global.RegistryConfig),
		ConfigCenter:   global.DefaultCenterConfig(),
		MetadataReport: global.DefaultMetadataReportConfig(),
		Provider:       global.DefaultProviderConfig(),
		Consumer:       global.DefaultConsumerConfig(),
		Metric:         global.DefaultMetricConfig(),
		Tracing:        make(map[string]*global.TracingConfig),
		Logger:         global.DefaultLoggerConfig(),
		Shutdown:       global.DefaultShutdownConfig(),
		Custom:         global.DefaultCustomConfig(),
		Profiles:       global.DefaultProfilesConfig(),
		TLSConfig:      global.DefaultTLSConfig(),
	}
}

func (rc *InstanceOptions) init(opts ...InstanceOption) error {
	for _, opt := range opts {
		opt(rc)
	}

	rcCompat := compatRootConfig(rc)
	if err := rcCompat.Init(); err != nil {
		return err
	}

	return nil
}

func (rc *InstanceOptions) Prefix() string {
	return constant.Dubbo
}

type InstanceOption func(*InstanceOptions)

func WithApplication(opts ...global.ApplicationOption) InstanceOption {
	appCfg := new(global.ApplicationConfig)
	for _, opt := range opts {
		opt(appCfg)
	}

	return func(cfg *InstanceOptions) {
		cfg.Application = appCfg
	}
}

func WithProtocol(key string, opts ...global.ProtocolOption) InstanceOption {
	proCfg := new(global.ProtocolConfig)
	for _, opt := range opts {
		opt(proCfg)
	}

	return func(cfg *InstanceOptions) {
		if cfg.Protocols == nil {
			cfg.Protocols = make(map[string]*global.ProtocolConfig)
		}
		cfg.Protocols[key] = proCfg
	}
}

func WithRegistry(key string, opts ...global.RegistryOption) InstanceOption {
	regCfg := new(global.RegistryConfig)
	for _, opt := range opts {
		opt(regCfg)
	}

	return func(cfg *InstanceOptions) {
		if cfg.Registries == nil {
			cfg.Registries = make(map[string]*global.RegistryConfig)
		}
		cfg.Registries[key] = regCfg
	}
}

func WithConfigCenter(opts ...global.CenterOption) InstanceOption {
	ccCfg := new(global.CenterConfig)
	for _, opt := range opts {
		opt(ccCfg)
	}

	return func(cfg *InstanceOptions) {
		cfg.ConfigCenter = ccCfg
	}
}

func WithMetadataReport(opts ...global.MetadataReportOption) InstanceOption {
	mrCfg := new(global.MetadataReportConfig)
	for _, opt := range opts {
		opt(mrCfg)
	}

	return func(cfg *InstanceOptions) {
		cfg.MetadataReport = mrCfg
	}
}

func WithConsumer(opts ...global.ConsumerOption) InstanceOption {
	conCfg := new(global.ConsumerConfig)
	for _, opt := range opts {
		opt(conCfg)
	}

	return func(cfg *InstanceOptions) {
		cfg.Consumer = conCfg
	}
}

func WithMetric(opts ...global.MetricOption) InstanceOption {
	meCfg := new(global.MetricConfig)
	for _, opt := range opts {
		opt(meCfg)
	}

	return func(cfg *InstanceOptions) {
		cfg.Metric = meCfg
	}
}

func WithTracing(key string, opts ...global.TracingOption) InstanceOption {
	traCfg := new(global.TracingConfig)
	for _, opt := range opts {
		opt(traCfg)
	}

	return func(cfg *InstanceOptions) {
		if cfg.Tracing == nil {
			cfg.Tracing = make(map[string]*global.TracingConfig)
		}
		cfg.Tracing[key] = traCfg
	}
}

func WithLogger(opts ...global.LoggerOption) InstanceOption {
	logCfg := new(global.LoggerConfig)
	for _, opt := range opts {
		opt(logCfg)
	}

	return func(cfg *InstanceOptions) {
		cfg.Logger = logCfg
	}
}

func WithShutdown(opts ...global.ShutdownOption) InstanceOption {
	sdCfg := new(global.ShutdownConfig)
	for _, opt := range opts {
		opt(sdCfg)
	}

	return func(cfg *InstanceOptions) {
		cfg.Shutdown = sdCfg
	}
}

func WithEventDispatcherType(typ string) InstanceOption {
	return func(cfg *InstanceOptions) {
		cfg.EventDispatcherType = typ
	}
}

func WithCacheFile(file string) InstanceOption {
	return func(cfg *InstanceOptions) {
		cfg.CacheFile = file
	}
}

func WithCustom(opts ...global.CustomOption) InstanceOption {
	cusCfg := new(global.CustomConfig)
	for _, opt := range opts {
		opt(cusCfg)
	}

	return func(cfg *InstanceOptions) {
		cfg.Custom = cusCfg
	}
}

func WithProfiles(opts ...global.ProfilesOption) InstanceOption {
	proCfg := new(global.ProfilesConfig)
	for _, opt := range opts {
		opt(proCfg)
	}

	return func(cfg *InstanceOptions) {
		cfg.Profiles = proCfg
	}
}

func WithTLS(opts ...global.TLSOption) InstanceOption {
	tlsCfg := new(global.TLSConfig)
	for _, opt := range opts {
		opt(tlsCfg)
	}

	return func(cfg *InstanceOptions) {
		cfg.TLSConfig = tlsCfg
	}
}
