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
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
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

	// remaining procedure is like RootConfig.Init() without RootConfig.Start()
	// tasks of RootConfig.Start() would be decomposed to Client and Server
	rcCompat := compatRootConfig(rc)
	if err := rcCompat.Logger.Init(); err != nil { // init default logger
		return err
	}
	if err := rcCompat.ConfigCenter.Init(rcCompat); err != nil {
		logger.Infof("[Config Center] Config center doesn't start")
		logger.Debugf("config center doesn't start because %s", err)
	} else {
		if err = rcCompat.Logger.Init(); err != nil { // init logger using config from config center again
			return err
		}
	}

	if err := rcCompat.Application.Init(); err != nil {
		return err
	}

	// init user define
	if err := rcCompat.Custom.Init(); err != nil {
		return err
	}

	// init protocol
	protocols := rcCompat.Protocols
	if len(protocols) <= 0 {
		protocol := &config.ProtocolConfig{}
		protocols = make(map[string]*config.ProtocolConfig, 1)
		protocols[constant.Dubbo] = protocol
		rcCompat.Protocols = protocols
	}
	for _, protocol := range protocols {
		if err := protocol.Init(); err != nil {
			return err
		}
	}

	// init registry
	registries := rcCompat.Registries
	if registries != nil {
		for _, reg := range registries {
			if err := reg.Init(); err != nil {
				return err
			}
		}
	}

	if err := rcCompat.MetadataReport.Init(rcCompat); err != nil {
		return err
	}
	if err := rcCompat.Metric.Init(rcCompat); err != nil {
		return err
	}
	for _, t := range rcCompat.Tracing {
		if err := t.Init(); err != nil {
			return err
		}
	}

	routers := rcCompat.Router
	if len(routers) > 0 {
		for _, r := range routers {
			if err := r.Init(); err != nil {
				return err
			}
		}
		rcCompat.Router = routers
	}

	// provider、consumer must last init
	if err := rcCompat.Provider.Init(rcCompat); err != nil {
		return err
	}
	if err := rcCompat.Consumer.Init(rcCompat); err != nil {
		return err
	}
	if err := rcCompat.Shutdown.Init(); err != nil {
		return err
	}
	config.SetRootConfig(*rcCompat)

	return nil
}

func (rc *InstanceOptions) Prefix() string {
	return constant.Dubbo
}

type InstanceOption func(*InstanceOptions)

func WithOrganization(organization string) InstanceOption {
	return func(opts *InstanceOptions) {
		opts.Application.Organization = organization
	}
}

func WithName(name string) InstanceOption {
	return func(opts *InstanceOptions) {
		opts.Application.Name = name
	}
}

func WithModule(module string) InstanceOption {
	return func(opts *InstanceOptions) {
		opts.Application.Module = module
	}
}

func WithGroup(group string) InstanceOption {
	return func(opts *InstanceOptions) {
		opts.Application.Group = group
	}
}

func WithVersion(version string) InstanceOption {
	return func(opts *InstanceOptions) {
		opts.Application.Version = version
	}
}

func WithOwner(owner string) InstanceOption {
	return func(opts *InstanceOptions) {
		opts.Application.Owner = owner
	}
}

func WithEnvironment(environment string) InstanceOption {
	return func(opts *InstanceOptions) {
		opts.Application.Environment = environment
	}
}

func WithRemoteMetadata() InstanceOption {
	return func(opts *InstanceOptions) {
		opts.Application.MetadataType = constant.RemoteMetadataStorageType
	}
}

func WithTag(tag string) InstanceOption {
	return func(opts *InstanceOptions) {
		opts.Application.Tag = tag
	}
}

func WithProtocol(key string, opts ...protocol.Option) InstanceOption {
	proOpts := protocol.DefaultOptions()
	for _, opt := range opts {
		opt(proOpts)
	}

	return func(insOpts *InstanceOptions) {
		if insOpts.Protocols == nil {
			insOpts.Protocols = make(map[string]*global.ProtocolConfig)
		}
		insOpts.Protocols[key] = proOpts.Protocol
	}
}

func WithRegistry(key string, opts ...registry.Option) InstanceOption {
	regOpts := registry.DefaultOptions()
	for _, opt := range opts {
		opt(regOpts)
	}

	return func(insOpts *InstanceOptions) {
		if insOpts.Registries == nil {
			insOpts.Registries = make(map[string]*global.RegistryConfig)
		}
		insOpts.Registries[key] = regOpts.Registry
	}
}

//func WithConfigCenter(opts ...global.CenterOption) InstanceOption {
//	ccCfg := new(global.CenterConfig)
//	for _, opt := range opts {
//		opt(ccCfg)
//	}
//
//	return func(cfg *InstanceOptions) {
//		cfg.ConfigCenter = ccCfg
//	}
//}

//func WithMetadataReport(opts ...global.MetadataReportOption) InstanceOption {
//	mrCfg := new(global.MetadataReportConfig)
//	for _, opt := range opts {
//		opt(mrCfg)
//	}
//
//	return func(cfg *InstanceOptions) {
//		cfg.MetadataReport = mrCfg
//	}
//}

//func WithConsumer(opts ...global.ConsumerOption) InstanceOption {
//	conCfg := new(global.ConsumerConfig)
//	for _, opt := range opts {
//		opt(conCfg)
//	}
//
//	return func(cfg *InstanceOptions) {
//		cfg.Consumer = conCfg
//	}
//}

//func WithMetric(opts ...global.MetricOption) InstanceOption {
//	meCfg := new(global.MetricConfig)
//	for _, opt := range opts {
//		opt(meCfg)
//	}
//
//	return func(cfg *InstanceOptions) {
//		cfg.Metric = meCfg
//	}
//}

//func WithTracing(key string, opts ...global.TracingOption) InstanceOption {
//	traCfg := new(global.TracingConfig)
//	for _, opt := range opts {
//		opt(traCfg)
//	}
//
//	return func(cfg *InstanceOptions) {
//		if cfg.Tracing == nil {
//			cfg.Tracing = make(map[string]*global.TracingConfig)
//		}
//		cfg.Tracing[key] = traCfg
//	}
//}

//func WithLogger(opts ...global.LoggerOption) InstanceOption {
//	logCfg := new(global.LoggerConfig)
//	for _, opt := range opts {
//		opt(logCfg)
//	}
//
//	return func(cfg *InstanceOptions) {
//		cfg.Logger = logCfg
//	}
//}

func WithShutdown(opts ...graceful_shutdown.Option) InstanceOption {
	sdOpts := graceful_shutdown.DefaultOptions()
	for _, opt := range opts {
		opt(sdOpts)
	}

	return func(insOpts *InstanceOptions) {
		insOpts.Shutdown = sdOpts.Shutdown
	}
}

// todo(DMwangnima): enumerate specific EventDispatcherType
//func WithEventDispatcherType(typ string) InstanceOption {
//	return func(cfg *InstanceOptions) {
//		cfg.EventDispatcherType = typ
//	}
//}

//func WithCacheFile(file string) InstanceOption {
//	return func(cfg *InstanceOptions) {
//		cfg.CacheFile = file
//	}
//}

//func WithCustom(opts ...global.CustomOption) InstanceOption {
//	cusCfg := new(global.CustomConfig)
//	for _, opt := range opts {
//		opt(cusCfg)
//	}
//
//	return func(cfg *InstanceOptions) {
//		cfg.Custom = cusCfg
//	}
//}

//func WithProfiles(opts ...global.ProfilesOption) InstanceOption {
//	proCfg := new(global.ProfilesConfig)
//	for _, opt := range opts {
//		opt(proCfg)
//	}
//
//	return func(cfg *InstanceOptions) {
//		cfg.Profiles = proCfg
//	}
//}

//func WithTLS(opts ...global.TLSOption) InstanceOption {
//	tlsCfg := new(global.TLSConfig)
//	for _, opt := range opts {
//		opt(tlsCfg)
//	}
//
//	return func(cfg *InstanceOptions) {
//		cfg.TLSConfig = tlsCfg
//	}
//}
