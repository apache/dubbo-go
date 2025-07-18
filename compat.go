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
	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
)

func compatRootConfig(c *InstanceOptions) *config.RootConfig {
	if c == nil {
		return nil
	}
	proCompat := make(map[string]*config.ProtocolConfig)
	for k, v := range c.Protocols {
		proCompat[k] = compatProtocolConfig(v)
	}

	regCompat := make(map[string]*config.RegistryConfig)
	for k, v := range c.Registries {
		regCompat[k] = compatRegistryConfig(v)
	}

	return &config.RootConfig{
		Application:         compatApplicationConfig(c.Application),
		Protocols:           proCompat,
		Registries:          regCompat,
		ConfigCenter:        compatCenterConfig(c.ConfigCenter),
		MetadataReport:      compatMetadataReportConfig(c.MetadataReport),
		Provider:            compatProviderConfig(c.Provider),
		Consumer:            compatConsumerConfig(c.Consumer),
		Metrics:             compatMetricConfig(c.Metrics),
		Otel:                compatOtelConfig(c.Otel),
		Logger:              compatLoggerConfig(c.Logger),
		Shutdown:            compatShutdownConfig(c.Shutdown),
		EventDispatcherType: c.EventDispatcherType,
		CacheFile:           c.CacheFile,
		Custom:              compatCustomConfig(c.Custom),
		Profiles:            compatProfilesConfig(c.Profiles),
	}
}

func compatApplicationConfig(c *global.ApplicationConfig) *config.ApplicationConfig {
	if c == nil {
		return nil
	}
	return &config.ApplicationConfig{
		Organization:            c.Organization,
		Name:                    c.Name,
		Module:                  c.Module,
		Group:                   c.Group,
		Version:                 c.Version,
		Owner:                   c.Owner,
		Environment:             c.Environment,
		MetadataType:            c.MetadataType,
		Tag:                     c.Tag,
		MetadataServicePort:     c.MetadataServicePort,
		MetadataServiceProtocol: c.MetadataServiceProtocol,
	}
}

func compatProtocolConfig(c *global.ProtocolConfig) *config.ProtocolConfig {
	if c == nil {
		return nil
	}
	return &config.ProtocolConfig{
		Name:                 c.Name,
		Ip:                   c.Ip,
		Port:                 c.Port,
		Params:               c.Params,
		TripleConfig:         compatTripleConfig(c.TripleConfig),
		MaxServerSendMsgSize: c.MaxServerSendMsgSize,
		MaxServerRecvMsgSize: c.MaxServerRecvMsgSize,
	}
}

// just for compat
func compatTripleConfig(c *global.TripleConfig) *config.TripleConfig {
	if c == nil {
		return nil
	}
	return &config.TripleConfig{
		MaxServerSendMsgSize: c.MaxServerSendMsgSize,
		MaxServerRecvMsgSize: c.MaxServerRecvMsgSize,
		Http3:                compatHttp3Config(c.Http3),

		KeepAliveInterval: c.KeepAliveInterval,
		KeepAliveTimeout:  c.KeepAliveTimeout,
	}
}

// just for compat
func compatHttp3Config(c *global.Http3Config) *config.Http3Config {
	if c == nil {
		return nil
	}
	return &config.Http3Config{
		Enable: c.Enable,
	}
}

func compatRegistryConfig(c *global.RegistryConfig) *config.RegistryConfig {
	if c == nil {
		return nil
	}
	return &config.RegistryConfig{
		Protocol:          c.Protocol,
		Timeout:           c.Timeout,
		Group:             c.Group,
		Namespace:         c.Namespace,
		TTL:               c.TTL,
		Address:           c.Address,
		Username:          c.Username,
		Password:          c.Password,
		Simplified:        c.Simplified,
		Preferred:         c.Preferred,
		Zone:              c.Zone,
		Weight:            c.Weight,
		Params:            c.Params,
		RegistryType:      c.RegistryType,
		UseAsMetaReport:   c.UseAsMetaReport,
		UseAsConfigCenter: c.UseAsConfigCenter,
	}
}

func compatCenterConfig(c *global.CenterConfig) *config.CenterConfig {
	if c == nil {
		return nil
	}
	return &config.CenterConfig{
		Protocol:      c.Protocol,
		Address:       c.Address,
		DataId:        c.DataId,
		Cluster:       c.Cluster,
		Group:         c.Group,
		Username:      c.Username,
		Password:      c.Password,
		Namespace:     c.Namespace,
		AppID:         c.AppID,
		Timeout:       c.Timeout,
		Params:        c.Params,
		FileExtension: c.FileExtension,
	}
}

func compatMetadataReportConfig(c *global.MetadataReportConfig) *config.MetadataReportConfig {
	if c == nil {
		return nil
	}
	return &config.MetadataReportConfig{
		Protocol:  c.Protocol,
		Address:   c.Address,
		Username:  c.Username,
		Password:  c.Password,
		Timeout:   c.Timeout,
		Group:     c.Group,
		Namespace: c.Namespace,
		Params:    c.Params,
	}
}

func compatProviderConfig(c *global.ProviderConfig) *config.ProviderConfig {
	if c == nil {
		return nil
	}
	services := make(map[string]*config.ServiceConfig)
	for key, svc := range c.Services {
		services[key] = compatServiceConfig(svc)
	}
	return &config.ProviderConfig{
		Filter:                 c.Filter,
		Register:               c.Register,
		RegistryIDs:            c.RegistryIDs,
		ProtocolIDs:            c.ProtocolIDs,
		TracingKey:             c.TracingKey,
		Services:               services,
		ProxyFactory:           c.ProxyFactory,
		FilterConf:             c.FilterConf,
		ConfigType:             c.ConfigType,
		AdaptiveService:        c.AdaptiveService,
		AdaptiveServiceVerbose: c.AdaptiveServiceVerbose,
	}
}

func compatServiceConfig(c *global.ServiceConfig) *config.ServiceConfig {
	if c == nil {
		return nil
	}
	methods := make([]*config.MethodConfig, len(c.Methods))
	for i, method := range c.Methods {
		methods[i] = compatMethodConfig(method)
	}
	protocols := make(map[string]*config.ProtocolConfig)
	for key, pro := range c.RCProtocolsMap {
		protocols[key] = compatProtocolConfig(pro)
	}
	registries := make(map[string]*config.RegistryConfig)
	for key, reg := range c.RCRegistriesMap {
		registries[key] = compatRegistryConfig(reg)
	}
	return &config.ServiceConfig{
		Filter:                      c.Filter,
		ProtocolIDs:                 c.ProtocolIDs,
		Interface:                   c.Interface,
		RegistryIDs:                 c.RegistryIDs,
		Cluster:                     c.Cluster,
		Loadbalance:                 c.Loadbalance,
		Group:                       c.Group,
		Version:                     c.Version,
		Methods:                     methods,
		Warmup:                      c.Warmup,
		Retries:                     c.Retries,
		Serialization:               c.Serialization,
		Params:                      c.Params,
		Token:                       c.Token,
		AccessLog:                   c.AccessLog,
		TpsLimiter:                  c.TpsLimiter,
		TpsLimitInterval:            c.TpsLimitInterval,
		TpsLimitRate:                c.TpsLimitRate,
		TpsLimitStrategy:            c.TpsLimitStrategy,
		TpsLimitRejectedHandler:     c.TpsLimitRejectedHandler,
		ExecuteLimit:                c.ExecuteLimit,
		ExecuteLimitRejectedHandler: c.ExecuteLimitRejectedHandler,
		Auth:                        c.Auth,
		NotRegister:                 c.NotRegister,
		ParamSign:                   c.ParamSign,
		Tag:                         c.Tag,
		TracingKey:                  c.TracingKey,
		RCProtocolsMap:              protocols,
		RCRegistriesMap:             registries,
		ProxyFactoryKey:             c.ProxyFactoryKey,
	}
}

func compatMethodConfig(c *global.MethodConfig) *config.MethodConfig {
	if c == nil {
		return nil
	}
	return &config.MethodConfig{
		InterfaceId:                 c.InterfaceId,
		InterfaceName:               c.InterfaceName,
		Name:                        c.Name,
		Retries:                     c.Retries,
		LoadBalance:                 c.LoadBalance,
		Weight:                      c.Weight,
		TpsLimitInterval:            c.TpsLimitInterval,
		TpsLimitRate:                c.TpsLimitRate,
		TpsLimitStrategy:            c.TpsLimitStrategy,
		ExecuteLimit:                c.ExecuteLimit,
		ExecuteLimitRejectedHandler: c.ExecuteLimitRejectedHandler,
		Sticky:                      c.Sticky,
		RequestTimeout:              c.RequestTimeout,
	}
}

func compatConsumerConfig(c *global.ConsumerConfig) *config.ConsumerConfig {
	if c == nil {
		return nil
	}
	return &config.ConsumerConfig{
		Filter:                         c.Filter,
		RegistryIDs:                    c.RegistryIDs,
		Protocol:                       c.Protocol,
		RequestTimeout:                 c.RequestTimeout,
		ProxyFactory:                   c.ProxyFactory,
		Check:                          c.Check,
		AdaptiveService:                c.AdaptiveService,
		TracingKey:                     c.TracingKey,
		FilterConf:                     c.FilterConf,
		MaxWaitTimeForServiceDiscovery: c.MaxWaitTimeForServiceDiscovery,
		MeshEnabled:                    c.MeshEnabled,
		References:                     compatReferences(c.References),
	}
}

func compatReferences(c map[string]*global.ReferenceConfig) map[string]*config.ReferenceConfig {
	refs := make(map[string]*config.ReferenceConfig, len(c))
	for name, ref := range c {
		refs[name] = &config.ReferenceConfig{
			InterfaceName:        ref.InterfaceName,
			Check:                ref.Check,
			URL:                  ref.URL,
			Filter:               ref.Filter,
			Protocol:             ref.Protocol,
			RegistryIDs:          ref.RegistryIDs,
			Cluster:              ref.Cluster,
			Loadbalance:          ref.Loadbalance,
			Retries:              ref.Retries,
			Group:                ref.Group,
			Version:              ref.Version,
			Serialization:        ref.Serialization,
			ProvidedBy:           ref.ProvidedBy,
			MethodsConfig:        compatMethod(ref.MethodsConfig),
			ProtocolClientConfig: compatProtocolClientConfig(ref.ProtocolClientConfig),
			Async:                ref.Async,
			Params:               ref.Params,
			Generic:              ref.Generic,
			Sticky:               ref.Sticky,
			RequestTimeout:       ref.RequestTimeout,
			ForceTag:             ref.ForceTag,
			TracingKey:           ref.TracingKey,
			MeshProviderPort:     ref.MeshProviderPort,
		}
	}
	return refs
}

// TODO: merge compatGlobalMethod() and compatGlobalMethodConfig
func compatMethod(m []*global.MethodConfig) []*config.MethodConfig {
	methods := make([]*config.MethodConfig, 0, len(m))
	for _, method := range m {
		methods = append(methods, &config.MethodConfig{
			InterfaceId:                 method.InterfaceId,
			InterfaceName:               method.InterfaceName,
			Name:                        method.Name,
			Retries:                     method.Retries,
			LoadBalance:                 method.LoadBalance,
			Weight:                      method.Weight,
			TpsLimitInterval:            method.TpsLimitInterval,
			TpsLimitRate:                method.TpsLimitRate,
			TpsLimitStrategy:            method.TpsLimitStrategy,
			ExecuteLimit:                method.ExecuteLimit,
			ExecuteLimitRejectedHandler: method.ExecuteLimitRejectedHandler,
			Sticky:                      method.Sticky,
			RequestTimeout:              method.RequestTimeout,
		})
	}
	return methods
}

// just for compat
func compatProtocolClientConfig(c *global.ClientProtocolConfig) *config.ClientProtocolConfig {
	if c == nil {
		return nil
	}
	return &config.ClientProtocolConfig{
		Name:         c.Name,
		TripleConfig: compatTripleConfig(c.TripleConfig),
	}
}

func compatMetricConfig(c *global.MetricsConfig) *config.MetricsConfig {
	if c == nil {
		return nil
	}
	return &config.MetricsConfig{
		Enable:             c.Enable,
		Port:               c.Port,
		Path:               c.Path,
		Prometheus:         compatMetricPrometheusConfig(c.Prometheus),
		Aggregation:        compatMetricAggregationConfig(c.Aggregation),
		Protocol:           c.Protocol,
		EnableMetadata:     c.EnableMetadata,
		EnableRegistry:     c.EnableRegistry,
		EnableConfigCenter: c.EnableConfigCenter,
	}
}

func compatOtelConfig(c *global.OtelConfig) *config.OtelConfig {
	if c == nil {
		return nil
	}
	return &config.OtelConfig{
		TraceConfig: &config.OtelTraceConfig{
			Enable:      c.TracingConfig.Enable,
			Exporter:    c.TracingConfig.Exporter,
			Endpoint:    c.TracingConfig.Endpoint,
			Propagator:  c.TracingConfig.Propagator,
			SampleMode:  c.TracingConfig.SampleMode,
			SampleRatio: c.TracingConfig.SampleRatio,
		},
	}
}

func compatLoggerConfig(c *global.LoggerConfig) *config.LoggerConfig {
	if c == nil {
		return nil
	}
	return &config.LoggerConfig{
		Driver:   c.Driver,
		Level:    c.Level,
		Format:   c.Format,
		Appender: c.Appender,
		File:     compatFile(c.File),
	}
}

func compatFile(c *global.File) *config.File {
	if c == nil {
		return nil
	}
	return &config.File{
		Name:       c.Name,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
		Compress:   c.Compress,
	}
}

func compatShutdownConfig(c *global.ShutdownConfig) *config.ShutdownConfig {
	if c == nil {
		return nil
	}
	cfg := &config.ShutdownConfig{
		Timeout:                     c.Timeout,
		StepTimeout:                 c.StepTimeout,
		ConsumerUpdateWaitTime:      c.ConsumerUpdateWaitTime,
		RejectRequestHandler:        c.RejectRequestHandler,
		InternalSignal:              c.InternalSignal,
		OfflineRequestWindowTimeout: c.OfflineRequestWindowTimeout,
		RejectRequest:               atomic.Bool{},
	}
	cfg.RejectRequest.Store(c.RejectRequest.Load())

	return cfg
}

func compatCustomConfig(c *global.CustomConfig) *config.CustomConfig {
	if c == nil {
		return nil
	}
	return &config.CustomConfig{
		ConfigMap: c.ConfigMap,
	}
}

func compatProfilesConfig(c *global.ProfilesConfig) *config.ProfilesConfig {
	if c == nil {
		return nil
	}
	return &config.ProfilesConfig{
		Active: c.Active,
	}
}

func compatMetricAggregationConfig(a *global.AggregateConfig) *config.AggregateConfig {
	if a == nil {
		return nil
	}
	return &config.AggregateConfig{
		Enabled:           a.Enabled,
		BucketNum:         a.BucketNum,
		TimeWindowSeconds: a.TimeWindowSeconds,
	}
}

func compatMetricPrometheusConfig(c *global.PrometheusConfig) *config.PrometheusConfig {
	if c == nil {
		return nil
	}
	return &config.PrometheusConfig{
		Exporter:    compatMetricPrometheusExporter(c.Exporter),
		Pushgateway: compatMetricPrometheusGateway(c.Pushgateway),
	}
}

func compatMetricPrometheusExporter(e *global.Exporter) *config.Exporter {
	if e == nil {
		return nil
	}
	return &config.Exporter{
		Enabled: e.Enabled,
	}
}

func compatMetricPrometheusGateway(g *global.PushgatewayConfig) *config.PushgatewayConfig {
	if g == nil {
		return nil
	}
	return &config.PushgatewayConfig{
		Enabled:      g.Enabled,
		BaseUrl:      g.BaseUrl,
		Job:          g.Job,
		Username:     g.Username,
		Password:     g.Password,
		PushInterval: g.PushInterval,
	}
}

func compatInstanceOptions(cr *config.RootConfig, rc *InstanceOptions) {
	if cr == nil {
		return
	}

	proCompat := make(map[string]*global.ProtocolConfig)
	for k, v := range cr.Protocols {
		proCompat[k] = compatGlobalProtocolConfig(v)
	}

	regCompat := make(map[string]*global.RegistryConfig)
	for k, v := range cr.Registries {
		regCompat[k] = compatGlobalRegistryConfig(v)
	}

	rc.Application = compatGlobalApplicationConfig(cr.Application)
	rc.Protocols = proCompat
	rc.Registries = regCompat
	rc.ConfigCenter = compatGlobalCenterConfig(cr.ConfigCenter)
	rc.MetadataReport = compatGlobalMetadataReportConfig(cr.MetadataReport)
	rc.Provider = compatGlobalProviderConfig(cr.Provider)
	rc.Consumer = compatGlobalConsumerConfig(cr.Consumer)
	rc.Metrics = compatGlobalMetricConfig(cr.Metrics)
	rc.Otel = compatGlobalOtelConfig(cr.Otel)
	rc.Logger = compatGlobalLoggerConfig(cr.Logger)
	rc.Shutdown = compatGlobalShutdownConfig(cr.Shutdown)
	rc.EventDispatcherType = cr.EventDispatcherType
	rc.CacheFile = cr.CacheFile
	rc.Custom = compatGlobalCustomConfig(cr.Custom)
	rc.Profiles = compatGlobalProfilesConfig(cr.Profiles)
}

func compatGlobalProtocolConfig(c *config.ProtocolConfig) *global.ProtocolConfig {
	if c == nil {
		return nil
	}
	return &global.ProtocolConfig{
		Name:                 c.Name,
		Ip:                   c.Ip,
		Port:                 c.Port,
		Params:               c.Params,
		TripleConfig:         compatGlobalTripleConfig(c.TripleConfig),
		MaxServerSendMsgSize: c.MaxServerSendMsgSize,
		MaxServerRecvMsgSize: c.MaxServerRecvMsgSize,
	}
}

// just for compat
func compatGlobalTripleConfig(c *config.TripleConfig) *global.TripleConfig {
	if c == nil {
		return nil
	}
	return &global.TripleConfig{
		KeepAliveInterval: c.KeepAliveInterval,
		KeepAliveTimeout:  c.KeepAliveTimeout,
		Http3:             compatGlobalHttp3Config(c.Http3),

		MaxServerSendMsgSize: c.MaxServerSendMsgSize,
		MaxServerRecvMsgSize: c.MaxServerRecvMsgSize,
	}
}

// just for compat
func compatGlobalHttp3Config(c *config.Http3Config) *global.Http3Config {
	if c == nil {
		return nil
	}
	return &global.Http3Config{
		Enable: c.Enable,
	}
}

func compatGlobalRegistryConfig(c *config.RegistryConfig) *global.RegistryConfig {
	if c == nil {
		return nil
	}
	return &global.RegistryConfig{
		Protocol:          c.Protocol,
		Timeout:           c.Timeout,
		Group:             c.Group,
		Namespace:         c.Namespace,
		TTL:               c.TTL,
		Address:           c.Address,
		Username:          c.Username,
		Password:          c.Password,
		Simplified:        c.Simplified,
		Preferred:         c.Preferred,
		Zone:              c.Zone,
		Weight:            c.Weight,
		Params:            c.Params,
		RegistryType:      c.RegistryType,
		UseAsMetaReport:   c.UseAsMetaReport,
		UseAsConfigCenter: c.UseAsConfigCenter,
	}
}

func compatGlobalApplicationConfig(c *config.ApplicationConfig) *global.ApplicationConfig {
	if c == nil {
		return nil
	}
	return &global.ApplicationConfig{
		Organization:            c.Organization,
		Name:                    c.Name,
		Module:                  c.Module,
		Group:                   c.Group,
		Version:                 c.Version,
		Owner:                   c.Owner,
		Environment:             c.Environment,
		MetadataType:            c.MetadataType,
		Tag:                     c.Tag,
		MetadataServicePort:     c.MetadataServicePort,
		MetadataServiceProtocol: c.MetadataServiceProtocol,
	}
}

func compatGlobalCenterConfig(c *config.CenterConfig) *global.CenterConfig {
	if c == nil {
		return nil
	}
	return &global.CenterConfig{
		Protocol:      c.Protocol,
		Address:       c.Address,
		DataId:        c.DataId,
		Cluster:       c.Cluster,
		Group:         c.Group,
		Username:      c.Username,
		Password:      c.Password,
		Namespace:     c.Namespace,
		AppID:         c.AppID,
		Timeout:       c.Timeout,
		Params:        c.Params,
		FileExtension: c.FileExtension,
	}
}

func compatGlobalMetadataReportConfig(c *config.MetadataReportConfig) *global.MetadataReportConfig {
	if c == nil {
		return nil
	}
	return &global.MetadataReportConfig{
		Protocol:  c.Protocol,
		Address:   c.Address,
		Username:  c.Username,
		Password:  c.Password,
		Timeout:   c.Timeout,
		Group:     c.Group,
		Namespace: c.Namespace,
		Params:    c.Params,
	}
}

func compatGlobalProviderConfig(c *config.ProviderConfig) *global.ProviderConfig {
	if c == nil {
		return nil
	}
	services := make(map[string]*global.ServiceConfig)
	for key, svc := range c.Services {
		services[key] = compatGlobalServiceConfig(svc)
	}
	return &global.ProviderConfig{
		Filter:                 c.Filter,
		Register:               c.Register,
		RegistryIDs:            c.RegistryIDs,
		ProtocolIDs:            c.ProtocolIDs,
		TracingKey:             c.TracingKey,
		Services:               services,
		ProxyFactory:           c.ProxyFactory,
		FilterConf:             c.FilterConf,
		ConfigType:             c.ConfigType,
		AdaptiveService:        c.AdaptiveService,
		AdaptiveServiceVerbose: c.AdaptiveServiceVerbose,
	}
}

func compatGlobalServiceConfig(c *config.ServiceConfig) *global.ServiceConfig {
	if c == nil {
		return nil
	}
	methods := make([]*global.MethodConfig, len(c.Methods))
	for i, method := range c.Methods {
		methods[i] = compatGlobalMethodConfig(method)
	}
	protocols := make(map[string]*global.ProtocolConfig)
	for key, pro := range c.RCProtocolsMap {
		protocols[key] = compatGlobalProtocolConfig(pro)
	}
	registries := make(map[string]*global.RegistryConfig)
	for key, reg := range c.RCRegistriesMap {
		registries[key] = compatGlobalRegistryConfig(reg)
	}
	return &global.ServiceConfig{
		Filter:                      c.Filter,
		ProtocolIDs:                 c.ProtocolIDs,
		Interface:                   c.Interface,
		RegistryIDs:                 c.RegistryIDs,
		Cluster:                     c.Cluster,
		Loadbalance:                 c.Loadbalance,
		Group:                       c.Group,
		Version:                     c.Version,
		Methods:                     methods,
		Warmup:                      c.Warmup,
		Retries:                     c.Retries,
		Serialization:               c.Serialization,
		Params:                      c.Params,
		Token:                       c.Token,
		AccessLog:                   c.AccessLog,
		TpsLimiter:                  c.TpsLimiter,
		TpsLimitInterval:            c.TpsLimitInterval,
		TpsLimitRate:                c.TpsLimitRate,
		TpsLimitStrategy:            c.TpsLimitStrategy,
		TpsLimitRejectedHandler:     c.TpsLimitRejectedHandler,
		ExecuteLimit:                c.ExecuteLimit,
		ExecuteLimitRejectedHandler: c.ExecuteLimitRejectedHandler,
		Auth:                        c.Auth,
		NotRegister:                 c.NotRegister,
		ParamSign:                   c.ParamSign,
		Tag:                         c.Tag,
		TracingKey:                  c.TracingKey,
		RCProtocolsMap:              protocols,
		RCRegistriesMap:             registries,
		ProxyFactoryKey:             c.ProxyFactoryKey,
	}
}

func compatGlobalMethodConfig(c *config.MethodConfig) *global.MethodConfig {
	if c == nil {
		return nil
	}
	return &global.MethodConfig{
		InterfaceId:                 c.InterfaceId,
		InterfaceName:               c.InterfaceName,
		Name:                        c.Name,
		Retries:                     c.Retries,
		LoadBalance:                 c.LoadBalance,
		Weight:                      c.Weight,
		TpsLimitInterval:            c.TpsLimitInterval,
		TpsLimitRate:                c.TpsLimitRate,
		TpsLimitStrategy:            c.TpsLimitStrategy,
		ExecuteLimit:                c.ExecuteLimit,
		ExecuteLimitRejectedHandler: c.ExecuteLimitRejectedHandler,
		Sticky:                      c.Sticky,
		RequestTimeout:              c.RequestTimeout,
	}
}

func compatGlobalConsumerConfig(c *config.ConsumerConfig) *global.ConsumerConfig {
	if c == nil {
		return nil
	}
	return &global.ConsumerConfig{
		Filter:                         c.Filter,
		RegistryIDs:                    c.RegistryIDs,
		Protocol:                       c.Protocol,
		RequestTimeout:                 c.RequestTimeout,
		ProxyFactory:                   c.ProxyFactory,
		Check:                          c.Check,
		AdaptiveService:                c.AdaptiveService,
		References:                     compatGlobalReferences(c.References),
		TracingKey:                     c.TracingKey,
		FilterConf:                     c.FilterConf,
		MaxWaitTimeForServiceDiscovery: c.MaxWaitTimeForServiceDiscovery,
		MeshEnabled:                    c.MeshEnabled,
	}
}

func compatGlobalReferences(c map[string]*config.ReferenceConfig) map[string]*global.ReferenceConfig {
	refs := make(map[string]*global.ReferenceConfig, len(c))
	for name, ref := range c {
		refs[name] = &global.ReferenceConfig{
			InterfaceName:        ref.InterfaceName,
			Check:                ref.Check,
			URL:                  ref.URL,
			Filter:               ref.Filter,
			Protocol:             ref.Protocol,
			RegistryIDs:          ref.RegistryIDs,
			Cluster:              ref.Cluster,
			Loadbalance:          ref.Loadbalance,
			Retries:              ref.Retries,
			Group:                ref.Group,
			Version:              ref.Version,
			Serialization:        ref.Serialization,
			ProvidedBy:           ref.ProvidedBy,
			MethodsConfig:        compatGlobalMethod(ref.MethodsConfig),
			ProtocolClientConfig: compatGlobalProtocolClientConfig(ref.ProtocolClientConfig),
			Async:                ref.Async,
			Params:               ref.Params,
			Generic:              ref.Generic,
			Sticky:               ref.Sticky,
			RequestTimeout:       ref.RequestTimeout,
			ForceTag:             ref.ForceTag,
			TracingKey:           ref.TracingKey,
			MeshProviderPort:     ref.MeshProviderPort,
		}
	}
	return refs
}

// TODO: merge compatGlobalMethod() and compatGlobalMethodConfig
func compatGlobalMethod(m []*config.MethodConfig) []*global.MethodConfig {
	methods := make([]*global.MethodConfig, 0, len(m))
	for _, method := range m {
		methods = append(methods, &global.MethodConfig{
			InterfaceId:                 method.InterfaceId,
			InterfaceName:               method.InterfaceName,
			Name:                        method.Name,
			Retries:                     method.Retries,
			LoadBalance:                 method.LoadBalance,
			Weight:                      method.Weight,
			TpsLimitInterval:            method.TpsLimitInterval,
			TpsLimitRate:                method.TpsLimitRate,
			TpsLimitStrategy:            method.TpsLimitStrategy,
			ExecuteLimit:                method.ExecuteLimit,
			ExecuteLimitRejectedHandler: method.ExecuteLimitRejectedHandler,
			Sticky:                      method.Sticky,
			RequestTimeout:              method.RequestTimeout,
		})
	}
	return methods
}

// just for compat
func compatGlobalProtocolClientConfig(c *config.ClientProtocolConfig) *global.ClientProtocolConfig {
	if c == nil {
		return nil
	}
	return &global.ClientProtocolConfig{
		Name:         c.Name,
		TripleConfig: compatGlobalTripleConfig(c.TripleConfig),
	}
}

func compatGlobalMetricConfig(c *config.MetricsConfig) *global.MetricsConfig {
	if c == nil {
		return nil
	}
	return &global.MetricsConfig{
		Enable:             c.Enable,
		Port:               c.Port,
		Path:               c.Path,
		Prometheus:         compatGlobalMetricPrometheusConfig(c.Prometheus),
		Aggregation:        compatGlobalMetricAggregationConfig(c.Aggregation),
		Protocol:           c.Protocol,
		EnableMetadata:     c.EnableMetadata,
		EnableRegistry:     c.EnableRegistry,
		EnableConfigCenter: c.EnableConfigCenter,
	}
}

func compatGlobalMetricPrometheusConfig(c *config.PrometheusConfig) *global.PrometheusConfig {
	if c == nil {
		return nil
	}
	return &global.PrometheusConfig{
		Exporter:    compatGlobalMetricPrometheusExporter(c.Exporter),
		Pushgateway: compatGlobalMetricPrometheusGateway(c.Pushgateway),
	}
}

func compatGlobalMetricAggregationConfig(a *config.AggregateConfig) *global.AggregateConfig {
	if a == nil {
		return nil
	}
	return &global.AggregateConfig{
		Enabled:           a.Enabled,
		BucketNum:         a.BucketNum,
		TimeWindowSeconds: a.TimeWindowSeconds,
	}
}

func compatGlobalMetricPrometheusExporter(e *config.Exporter) *global.Exporter {
	if e == nil {
		return nil
	}
	return &global.Exporter{
		Enabled: e.Enabled,
	}
}

func compatGlobalMetricPrometheusGateway(g *config.PushgatewayConfig) *global.PushgatewayConfig {
	if g == nil {
		return nil
	}
	return &global.PushgatewayConfig{
		Enabled:      g.Enabled,
		BaseUrl:      g.BaseUrl,
		Job:          g.Job,
		Username:     g.Username,
		Password:     g.Password,
		PushInterval: g.PushInterval,
	}
}

func compatGlobalOtelConfig(c *config.OtelConfig) *global.OtelConfig {
	if c == nil {
		return nil
	}
	return &global.OtelConfig{
		TracingConfig: &global.OtelTraceConfig{
			Enable:      c.TraceConfig.Enable,
			Exporter:    c.TraceConfig.Exporter,
			Endpoint:    c.TraceConfig.Endpoint,
			Propagator:  c.TraceConfig.Propagator,
			SampleMode:  c.TraceConfig.SampleMode,
			SampleRatio: c.TraceConfig.SampleRatio,
		},
	}
}

func compatGlobalLoggerConfig(c *config.LoggerConfig) *global.LoggerConfig {
	if c == nil {
		return nil
	}
	return &global.LoggerConfig{
		Driver:   c.Driver,
		Level:    c.Level,
		Format:   c.Format,
		Appender: c.Appender,
		File:     compatGlobalFile(c.File),
	}
}

func compatGlobalFile(c *config.File) *global.File {
	if c == nil {
		return nil
	}
	return &global.File{
		Name:       c.Name,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
		Compress:   c.Compress,
	}
}

func compatGlobalShutdownConfig(c *config.ShutdownConfig) *global.ShutdownConfig {
	if c == nil {
		return nil
	}
	cfg := &global.ShutdownConfig{
		Timeout:                     c.Timeout,
		StepTimeout:                 c.StepTimeout,
		ConsumerUpdateWaitTime:      c.ConsumerUpdateWaitTime,
		RejectRequestHandler:        c.RejectRequestHandler,
		InternalSignal:              c.InternalSignal,
		OfflineRequestWindowTimeout: c.OfflineRequestWindowTimeout,
		RejectRequest:               atomic.Bool{},
	}
	cfg.RejectRequest.Store(c.RejectRequest.Load())

	return cfg
}

func compatGlobalCustomConfig(c *config.CustomConfig) *global.CustomConfig {
	if c == nil {
		return nil
	}
	return &global.CustomConfig{
		ConfigMap: c.ConfigMap,
	}
}

func compatGlobalProfilesConfig(c *config.ProfilesConfig) *global.ProfilesConfig {
	if c == nil {
		return nil
	}
	return &global.ProfilesConfig{
		Active: c.Active,
	}
}
