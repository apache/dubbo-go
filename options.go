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
	"time"
)

import (
	log "github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/logger"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/otel/trace"
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
	Metrics        *global.MetricsConfig             `yaml:"metrics" json:"metrics,omitempty" property:"metrics"`
	Otel           *global.OtelConfig                `yaml:"otel" json:"otel,omitempty" property:"otel"`
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
		Metrics:        global.DefaultMetricsConfig(),
		Otel:           global.DefaultOtelConfig(),
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
		log.Infof("[Config Center] Config center doesn't start")
		log.Debugf("config center doesn't start because %s", err)
	} else {
		compatInstanceOptions(rcCompat, rc)
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

	if err := rcCompat.Metrics.Init(rcCompat); err != nil {
		return err
	}
	if err := rcCompat.Otel.Init(rcCompat.Application); err != nil {
		return err
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

	if err := rc.initMetadataReport(); err != nil {
		return err
	}
	if err := rc.initRegistryMetadataReport(); err != nil {
		return err
	}

	return nil
}

func (rc *InstanceOptions) initMetadataReport() error {
	if rc.MetadataReport.Address != "" {
		opts, err := reportConfigToReportOptions(rc.MetadataReport)
		if err != nil {
			return err
		}
		if err := opts.Init(); err != nil {
			return err
		}
	}
	return nil
}

func reportConfigToReportOptions(mc *global.MetadataReportConfig) (*metadata.ReportOptions, error) {
	opts := metadata.NewReportOptions(
		metadata.WithRegistryId(constant.DefaultKey),
		metadata.WithProtocol(mc.Protocol),
		metadata.WithAddress(mc.Address),
		metadata.WithUsername(mc.Username),
		metadata.WithPassword(mc.Password),
		metadata.WithGroup(mc.Group),
		metadata.WithNamespace(mc.Namespace),
		metadata.WithParams(mc.Params),
	)
	if mc.Timeout != "" {
		timeout, err := time.ParseDuration(mc.Timeout)
		if err != nil {
			return nil, err
		}
		metadata.WithTimeout(timeout)(opts)
	}
	return opts, nil
}

func (rc *InstanceOptions) initRegistryMetadataReport() error {
	if len(rc.Registries) > 0 {
		for id, reg := range rc.Registries {
			if reg.UseAsMetaReport {
				opts, err := registryToReportOptions(id, reg)
				if err != nil {
					return err
				}
				if err := opts.Init(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func registryToReportOptions(id string, rc *global.RegistryConfig) (*metadata.ReportOptions, error) {
	opts := metadata.NewReportOptions(
		metadata.WithRegistryId(id),
		metadata.WithProtocol(rc.Protocol),
		metadata.WithAddress(rc.Address),
		metadata.WithUsername(rc.Username),
		metadata.WithPassword(rc.Password),
		metadata.WithGroup(rc.Group),
		metadata.WithNamespace(rc.Namespace),
		metadata.WithParams(rc.Params),
	)
	if rc.Timeout != "" {
		timeout, err := time.ParseDuration(rc.Timeout)
		if err != nil {
			return nil, err
		}
		metadata.WithTimeout(timeout)(opts)
	}
	return opts, nil
}

func (rc *InstanceOptions) Prefix() string {
	return constant.Dubbo
}

func (rc *InstanceOptions) CloneApplication() *global.ApplicationConfig {
	if rc.Application == nil {
		return nil
	}
	return rc.Application.Clone()
}

func (rc *InstanceOptions) CloneProtocols() map[string]*global.ProtocolConfig {
	if rc.Protocols == nil {
		return nil
	}
	protocols := make(map[string]*global.ProtocolConfig, len(rc.Protocols))
	for k, v := range rc.Protocols {
		protocols[k] = v.Clone()
	}
	return protocols
}

func (rc *InstanceOptions) CloneRegistries() map[string]*global.RegistryConfig {
	if rc.Registries == nil {
		return nil
	}
	registries := make(map[string]*global.RegistryConfig, len(rc.Registries))
	for k, v := range rc.Registries {
		registries[k] = v.Clone()
	}
	return registries
}

func (rc *InstanceOptions) CloneConfigCenter() *global.CenterConfig {
	if rc.ConfigCenter == nil {
		return nil
	}
	return rc.ConfigCenter.Clone()
}

func (rc *InstanceOptions) CloneMetadataReport() *global.MetadataReportConfig {
	if rc.MetadataReport == nil {
		return nil
	}
	return rc.MetadataReport.Clone()
}

func (rc *InstanceOptions) CloneProvider() *global.ProviderConfig {
	if rc.Provider == nil {
		return nil
	}
	return rc.Provider.Clone()
}

func (rc *InstanceOptions) CloneConsumer() *global.ConsumerConfig {
	if rc.Consumer == nil {
		return nil
	}
	return rc.Consumer.Clone()
}

func (rc *InstanceOptions) CloneMetrics() *global.MetricsConfig {
	if rc.Metrics == nil {
		return nil
	}
	return rc.Metrics.Clone()
}

func (rc *InstanceOptions) CloneOtel() *global.OtelConfig {
	if rc.Otel == nil {
		return nil
	}
	return rc.Otel.Clone()
}

func (rc *InstanceOptions) CloneLogger() *global.LoggerConfig {
	if rc.Logger == nil {
		return nil
	}
	return rc.Logger.Clone()
}

func (rc *InstanceOptions) CloneShutdown() *global.ShutdownConfig {
	if rc.Shutdown == nil {
		return nil
	}
	return rc.Shutdown.Clone()
}

func (rc *InstanceOptions) CloneCustom() *global.CustomConfig {
	if rc.Custom == nil {
		return nil
	}
	return rc.Custom.Clone()
}

func (rc *InstanceOptions) CloneProfiles() *global.ProfilesConfig {
	if rc.Profiles == nil {
		return nil
	}
	return rc.Profiles.Clone()
}

func (rc *InstanceOptions) CloneTLSConfig() *global.TLSConfig {
	if rc.TLSConfig == nil {
		return nil
	}
	return rc.TLSConfig.Clone()
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

func WithProtocol(opts ...protocol.Option) InstanceOption {
	proOpts := protocol.NewOptions(opts...)

	return func(insOpts *InstanceOptions) {
		if insOpts.Protocols == nil {
			insOpts.Protocols = make(map[string]*global.ProtocolConfig)
		}
		insOpts.Protocols[proOpts.ID] = proOpts.Protocol
	}
}

func WithRegistry(opts ...registry.Option) InstanceOption {
	regOpts := registry.NewOptions(opts...)

	return func(insOpts *InstanceOptions) {
		if insOpts.Registries == nil {
			insOpts.Registries = make(map[string]*global.RegistryConfig)
		}
		insOpts.Registries[regOpts.ID] = regOpts.Registry
	}
}

// WithTracing otel configuration, currently only supports tracing
func WithTracing(opts ...trace.Option) InstanceOption {
	traceOpts := trace.NewOptions(opts...)

	return func(insOpts *InstanceOptions) {
		insOpts.Otel.TracingConfig = traceOpts.Otel.TracingConfig
	}
}

func WithConfigCenter(opts ...config_center.Option) InstanceOption {
	configOpts := config_center.NewOptions(opts...)

	return func(cfg *InstanceOptions) {
		cfg.ConfigCenter = configOpts.Center
	}
}

func WithMetadataReport(opts ...metadata.ReportOption) InstanceOption {
	metadataOpts := metadata.NewReportOptions(opts...)

	return func(cfg *InstanceOptions) {
		cfg.MetadataReport = metadataOpts.MetadataReportConfig
	}
}

func WithMetrics(opts ...metrics.Option) InstanceOption {
	metricOpts := metrics.NewOptions(opts...)

	return func(cfg *InstanceOptions) {
		cfg.Metrics = metricOpts.Metrics
	}
}

func WithLogger(opts ...logger.Option) InstanceOption {
	loggerOpts := logger.NewOptions(opts...)

	return func(cfg *InstanceOptions) {
		cfg.Logger = loggerOpts.Logger
	}
}

func WithShutdown(opts ...graceful_shutdown.Option) InstanceOption {
	sdOpts := graceful_shutdown.NewOptions(opts...)

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
