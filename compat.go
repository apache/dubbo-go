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

	traCompat := make(map[string]*config.TracingConfig)
	for k, v := range c.Tracing {
		traCompat[k] = compatTracingConfig(v)
	}

	return &config.RootConfig{
		Application:         compatApplicationConfig(c.Application),
		Protocols:           proCompat,
		Registries:          regCompat,
		ConfigCenter:        compatCenterConfig(c.ConfigCenter),
		MetadataReport:      compatMetadataReportConfig(c.MetadataReport),
		Provider:            compatProviderConfig(c.Provider),
		Consumer:            compatConsumerConfig(c.Consumer),
		Metric:              compatMetricConfig(c.Metric),
		Tracing:             traCompat,
		Logger:              compatLoggerConfig(c.Logger),
		Shutdown:            compatShutdownConfig(c.Shutdown),
		EventDispatcherType: c.EventDispatcherType,
		CacheFile:           c.CacheFile,
		Custom:              compatCustomConfig(c.Custom),
		Profiles:            compatProfilesConfig(c.Profiles),
		TLSConfig:           compatTLSConfig(c.TLSConfig),
	}
}

func compatApplicationConfig(c *global.ApplicationConfig) *config.ApplicationConfig {
	if c == nil {
		return nil
	}
	return &config.ApplicationConfig{
		Organization: c.Organization,
		Name:         c.Name,
		Module:       c.Module,
		Group:        c.Group,
		Version:      c.Version,
		Owner:        c.Owner,
		Environment:  c.Environment,
		MetadataType: c.MetadataType,
		Tag:          c.Tag,
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
		MaxServerSendMsgSize: c.MaxServerSendMsgSize,
		MaxServerRecvMsgSize: c.MaxServerRecvMsgSize,
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
	}
}

func compatMetricConfig(c *global.MetricConfig) *config.MetricConfig {
	if c == nil {
		return nil
	}
	return &config.MetricConfig{
		Enable:      c.Enable,
		Port:        c.Port,
		Path:        c.Path,
		Prometheus:  compatMetricPrometheusConfig(c.Prometheus),
		Aggregation: compatMetricAggregationConfig(c.Aggregation),
		Protocol:    c.Protocol,
	}
}

func compatTracingConfig(c *global.TracingConfig) *config.TracingConfig {
	if c == nil {
		return nil
	}
	return &config.TracingConfig{
		Name:        c.Name,
		ServiceName: c.ServiceName,
		Address:     c.Address,
		UseAgent:    c.UseAgent,
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

func compatTLSConfig(c *global.TLSConfig) *config.TLSConfig {
	if c == nil {
		return nil
	}
	return &config.TLSConfig{
		CACertFile:    c.CACertFile,
		TLSCertFile:   c.TLSCertFile,
		TLSKeyFile:    c.TLSKeyFile,
		TLSServerName: c.TLSServerName,
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
