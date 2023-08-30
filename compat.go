package dubbo

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"go.uber.org/atomic"
)

func compatApplicationConfig(c *ApplicationConfig) *config.ApplicationConfig {
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

func compatProtocolConfig(c *protocol.ProtocolConfig) *config.ProtocolConfig {
	return &config.ProtocolConfig{
		Name:                 c.Name,
		Ip:                   c.Ip,
		Port:                 c.Port,
		Params:               c.Params,
		MaxServerSendMsgSize: c.MaxServerSendMsgSize,
		MaxServerRecvMsgSize: c.MaxServerRecvMsgSize,
	}
}

func compatRegistryConfig(c *registry.RegistryConfig) *config.RegistryConfig {
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

func compatCenterConfig(c *config_center.CenterConfig) *config.CenterConfig {
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

func compatMetadataReportConfig(c *report.MetadataReportConfig) *config.MetadataReportConfig {
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

// todo:(DMwangnima): is there any need to compat ConsumerConfig?
func compatConsumerConfig(c *ConsumerConfig) *config.ConsumerConfig {
	return &config.ConsumerConfig{
		Filter:          c.Filter,
		RegistryIDs:     c.RegistryIDs,
		Protocol:        c.Protocol,
		RequestTimeout:  c.RequestTimeout,
		ProxyFactory:    c.ProxyFactory,
		Check:           c.Check,
		AdaptiveService: c.AdaptiveService,
		//References:                     c.References,
		TracingKey:                     c.TracingKey,
		FilterConf:                     c.FilterConf,
		MaxWaitTimeForServiceDiscovery: c.MaxWaitTimeForServiceDiscovery,
		MeshEnabled:                    c.MeshEnabled,
	}
}

func compatMetricConfig(c *metrics.MetricConfig) *config.MetricConfig {
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

func compatTracingConfig(c *TracingConfig) *config.TracingConfig {
	return &config.TracingConfig{
		Name:        c.Name,
		ServiceName: c.ServiceName,
		Address:     c.Address,
		UseAgent:    c.UseAgent,
	}
}

func compatLoggerConfig(c *LoggerConfig) *config.LoggerConfig {
	return &config.LoggerConfig{
		Driver:   c.Driver,
		Level:    c.Level,
		Format:   c.Format,
		Appender: c.Appender,
		File:     compatFile(c.File),
	}
}

func compatFile(c *File) *config.File {
	return &config.File{
		Name:       c.Name,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
		Compress:   c.Compress,
	}
}

func compatShutdownConfig(c *ShutdownConfig) *config.ShutdownConfig {
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

func compatCustomConfig(c *CustomConfig) *config.CustomConfig {
	return &config.CustomConfig{
		ConfigMap: c.ConfigMap,
	}
}

func compatProfilesConfig(c *ProfilesConfig) *config.ProfilesConfig {
	return &config.ProfilesConfig{
		Active: c.Active,
	}
}

func compatTLSConfig(c *TLSConfig) *config.TLSConfig {
	return &config.TLSConfig{
		CACertFile:    c.CACertFile,
		TLSCertFile:   c.TLSCertFile,
		TLSKeyFile:    c.TLSKeyFile,
		TLSServerName: c.TLSServerName,
	}
}
