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
	return &config.ProviderConfig{
		Filter:                 c.Filter,
		Register:               c.Register,
		RegistryIDs:            c.RegistryIDs,
		ProtocolIDs:            c.ProtocolIDs,
		TracingKey:             c.TracingKey,
		ProxyFactory:           c.ProxyFactory,
		FilterConf:             c.FilterConf,
		ConfigType:             c.ConfigType,
		AdaptiveService:        c.AdaptiveService,
		AdaptiveServiceVerbose: c.AdaptiveServiceVerbose,
	}
}

func compatConsumerConfig(c *global.ConsumerConfig) *config.ConsumerConfig {
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
	return &config.MetricConfig{
		Mode:               c.Mode,
		Namespace:          c.Namespace,
		Enable:             c.Enable,
		Port:               c.Port,
		Path:               c.Path,
		PushGatewayAddress: c.PushGatewayAddress,
		SummaryMaxAge:      c.SummaryMaxAge,
		Protocol:           c.Protocol,
	}
}

func compatTracingConfig(c *global.TracingConfig) *config.TracingConfig {
	return &config.TracingConfig{
		Name:        c.Name,
		ServiceName: c.ServiceName,
		Address:     c.Address,
		UseAgent:    c.UseAgent,
	}
}

func compatLoggerConfig(c *global.LoggerConfig) *config.LoggerConfig {
	return &config.LoggerConfig{
		Driver:   c.Driver,
		Level:    c.Level,
		Format:   c.Format,
		Appender: c.Appender,
		File:     compatFile(c.File),
	}
}

func compatFile(c *global.File) *config.File {
	return &config.File{
		Name:       c.Name,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
		Compress:   c.Compress,
	}
}

func compatShutdownConfig(c *global.ShutdownConfig) *config.ShutdownConfig {
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
	return &config.CustomConfig{
		ConfigMap: c.ConfigMap,
	}
}

func compatProfilesConfig(c *global.ProfilesConfig) *config.ProfilesConfig {
	return &config.ProfilesConfig{
		Active: c.Active,
	}
}

func compatTLSConfig(c *global.TLSConfig) *config.TLSConfig {
	return &config.TLSConfig{
		CACertFile:    c.CACertFile,
		TLSCertFile:   c.TLSCertFile,
		TLSKeyFile:    c.TLSKeyFile,
		TLSServerName: c.TLSServerName,
	}
}
