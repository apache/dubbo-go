package dubbo

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	"go.uber.org/atomic"
)

type ApplicationConfig struct {
	Organization string `default:"dubbo-go" yaml:"organization" json:"organization,omitempty" property:"organization"`
	Name         string `default:"dubbo.io" yaml:"name" json:"name,omitempty" property:"name"`
	Module       string `default:"sample" yaml:"module" json:"module,omitempty" property:"module"`
	Group        string `yaml:"group" json:"group,omitempty" property:"module"`
	Version      string `yaml:"version" json:"version,omitempty" property:"version"`
	Owner        string `default:"dubbo-go" yaml:"owner" json:"owner,omitempty" property:"owner"`
	Environment  string `yaml:"environment" json:"environment,omitempty" property:"environment"`
	// the metadata type. remote or local
	MetadataType string `default:"local" yaml:"metadata-type" json:"metadataType,omitempty" property:"metadataType"`
	Tag          string `yaml:"tag" json:"tag,omitempty" property:"tag"`
}

type ApplicationOption func(*ApplicationConfig)

// ---------- ApplicationOption ----------

func WithApplication_Organization(organization string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Organization = organization
	}
}

func WithApplication_Name(name string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Name = name
	}
}

func WithApplication_Module(module string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Module = module
	}
}

func WithApplication_Group(group string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Group = group
	}
}

func WithApplication_Version(version string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Version = version
	}
}

func WithApplication_Owner(owner string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Owner = owner
	}
}

func WithApplication_Environment(environment string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Environment = environment
	}
}

func WithApplication_MetadataType(metadataType string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.MetadataType = metadataType
	}
}

func WithApplication_Tag(tag string) ApplicationOption {
	return func(cfg *ApplicationConfig) {
		cfg.Tag = tag
	}
}

type ConsumerConfig struct {
	Filter                         string                             `yaml:"filter" json:"filter,omitempty" property:"filter"`
	RegistryIDs                    []string                           `yaml:"registry-ids" json:"registry-ids,omitempty" property:"registry-ids"`
	Protocol                       string                             `yaml:"protocol" json:"protocol,omitempty" property:"protocol"`
	RequestTimeout                 string                             `default:"3s" yaml:"request-timeout" json:"request-timeout,omitempty" property:"request-timeout"`
	ProxyFactory                   string                             `default:"default" yaml:"proxy" json:"proxy,omitempty" property:"proxy"`
	Check                          bool                               `yaml:"check" json:"check,omitempty" property:"check"`
	AdaptiveService                bool                               `default:"false" yaml:"adaptive-service" json:"adaptive-service" property:"adaptive-service"`
	References                     map[string]*client.ReferenceConfig `yaml:"references" json:"references,omitempty" property:"references"`
	TracingKey                     string                             `yaml:"tracing-key" json:"tracing-key" property:"tracing-key"`
	FilterConf                     interface{}                        `yaml:"filter-conf" json:"filter-conf,omitempty" property:"filter-conf"`
	MaxWaitTimeForServiceDiscovery string                             `default:"3s" yaml:"max-wait-time-for-service-discovery" json:"max-wait-time-for-service-discovery,omitempty" property:"max-wait-time-for-service-discovery"`
	MeshEnabled                    bool                               `yaml:"mesh-enabled" json:"mesh-enabled,omitempty" property:"mesh-enabled"`
	rootConfig                     *RootConfig
}

type ConsumerOption func(*ConsumerConfig)

// ---------- ConsumerOption ----------

func WithConsumer_Filter(filter string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.Filter = filter
	}
}

func WithConsumer_RegistryIDs(ids []string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.RegistryIDs = ids
	}
}

func WithConsumer_Protocol(protocol string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.Protocol = protocol
	}
}

func WithConsumer_RequestTimeout(timeout string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.RequestTimeout = timeout
	}
}

func WithConsumer_ProxyFactory(factory string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.ProxyFactory = factory
	}
}

func WithConsumer_Check(flag bool) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.Check = flag
	}
}

func WithConsumer_AdaptiveService(flag bool) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.AdaptiveService = flag
	}
}

// todo(DMwangnima): think about a more ideal way
//func WithConsumer_Reference()

func WithConsumer_TracingKey(key string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.TracingKey = key
	}
}

func WithConsumer_FilterConf(conf interface{}) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.FilterConf = conf
	}
}

func WithConsumer_MaxWaitTimeForServiceDiscovery(time string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.MaxWaitTimeForServiceDiscovery = time
	}
}

func WithConsumer_MeshEnabled(flag bool) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.MeshEnabled = flag
	}
}

// TracingConfig is the configuration of the tracing.
type TracingConfig struct {
	Name        string `default:"jaeger" yaml:"name" json:"name,omitempty" property:"name"` // jaeger or zipkin(todo)
	ServiceName string `yaml:"serviceName" json:"serviceName,omitempty" property:"serviceName"`
	Address     string `yaml:"address" json:"address,omitempty" property:"address"`
	UseAgent    *bool  `default:"false" yaml:"use-agent" json:"use-agent,omitempty" property:"use-agent"`
}

type TracingOption func(*TracingConfig)

// ---------- TracingOption ----------

func WithTracing_Name(name string) TracingOption {
	return func(cfg *TracingConfig) {
		cfg.Name = name
	}
}

func WithTracing_ServiceName(name string) TracingOption {
	return func(cfg *TracingConfig) {
		cfg.ServiceName = name
	}
}

func WithTracing_Address(address string) TracingOption {
	return func(cfg *TracingConfig) {
		cfg.Address = address
	}
}

func WithTracing_UseAgent(flag bool) TracingOption {
	return func(cfg *TracingConfig) {
		cfg.UseAgent = &flag
	}
}

type LoggerConfig struct {
	// logger driver default zap
	Driver string `default:"zap" yaml:"driver"`

	// logger level
	Level string `default:"info" yaml:"level"`

	// logger formatter default text
	Format string `default:"text" yaml:"format"`

	// supports simultaneous file and console eg: console,file default console
	Appender string `default:"console" yaml:"appender"`

	// logger file
	File *File `yaml:"file"`
}

type File struct {
	// log file name default dubbo.log
	Name string `default:"dubbo.log" yaml:"name"`

	// log max size default 100Mb
	MaxSize int `default:"100" yaml:"max-size"`

	// log max backups default 5
	MaxBackups int `default:"5" yaml:"max-backups"`

	// log file max age default 3 day
	MaxAge int `default:"3" yaml:"max-age"`

	Compress *bool `default:"true" yaml:"compress"`
}

type LoggerOption func(*LoggerConfig)

// ---------- LoggerOption ----------

func WithLogger_Driver(driver string) LoggerOption {
	return func(cfg *LoggerConfig) {
		cfg.Driver = driver
	}
}

func WithLogger_Level(level string) LoggerOption {
	return func(cfg *LoggerConfig) {
		cfg.Level = level
	}
}

func WithLogger_Format(format string) LoggerOption {
	return func(cfg *LoggerConfig) {
		cfg.Format = format
	}
}

func WithLogger_Appender(appender string) LoggerOption {
	return func(cfg *LoggerConfig) {
		cfg.Appender = appender
	}
}

func WithLogger_File_Name(name string) LoggerOption {
	return func(cfg *LoggerConfig) {
		if cfg.File == nil {
			cfg.File = new(File)
		}
		cfg.File.Name = name
	}
}

func WithLogger_File_MaxSize(size int) LoggerOption {
	return func(cfg *LoggerConfig) {
		if cfg.File == nil {
			cfg.File = new(File)
		}
		cfg.File.MaxSize = size
	}
}

func WithLogger_File_MaxBackups(backups int) LoggerOption {
	return func(cfg *LoggerConfig) {
		if cfg.File == nil {
			cfg.File = new(File)
		}
		cfg.File.MaxBackups = backups
	}
}

func WithLogger_File_MaxAge(age int) LoggerOption {
	return func(cfg *LoggerConfig) {
		if cfg.File == nil {
			cfg.File = new(File)
		}
		cfg.File.MaxAge = age
	}
}

func WithLogger_File_Compress(flag bool) LoggerOption {
	return func(cfg *LoggerConfig) {
		if cfg.File == nil {
			cfg.File = new(File)
		}
		cfg.File.Compress = &flag
	}
}

// ShutdownConfig is used as configuration for graceful shutdown
type ShutdownConfig struct {
	/*
	 * Total timeout. Even though we don't release all resources,
	 * the applicationConfig will shutdown if the costing time is over this configuration. The unit is ms.
	 * default value is 60 * 1000 ms = 1 minutes
	 * In general, it should be bigger than 3 * StepTimeout.
	 */
	Timeout string `default:"60s" yaml:"timeout" json:"timeout,omitempty" property:"timeout"`
	/*
	 * the timeout on each step. You should evaluate the response time of request
	 * and the time that client noticed that server shutdown.
	 * For example, if your client will received the notification within 10s when you start to close server,
	 * and the 99.9% requests will return response in 2s, so the StepTimeout will be bigger than(10+2) * 1000ms,
	 * maybe (10 + 2*3) * 1000ms is a good choice.
	 */
	StepTimeout string `default:"3s" yaml:"step-timeout" json:"step.timeout,omitempty" property:"step.timeout"`

	/*
	 * ConsumerUpdateWaitTime means when provider is shutting down, after the unregister, time to wait for client to
	 * update invokers. During this time, incoming invocation can be treated normally.
	 */
	ConsumerUpdateWaitTime string `default:"3s" yaml:"consumer-update-wait-time" json:"consumerUpdate.waitTIme,omitempty" property:"consumerUpdate.waitTIme"`
	// when we try to shutdown the applicationConfig, we will reject the new requests. In most cases, you don't need to configure this.
	RejectRequestHandler string `yaml:"reject-handler" json:"reject-handler,omitempty" property:"reject_handler"`
	// internal listen kill signal，the default is true.
	InternalSignal *bool `default:"true" yaml:"internal-signal" json:"internal.signal,omitempty" property:"internal.signal"`
	// offline request window length
	OfflineRequestWindowTimeout string `yaml:"offline-request-window-timeout" json:"offlineRequestWindowTimeout,omitempty" property:"offlineRequestWindowTimeout"`
	// true -> new request will be rejected.
	RejectRequest atomic.Bool
	// active invocation
	ConsumerActiveCount atomic.Int32
	ProviderActiveCount atomic.Int32

	// provider last received request timestamp
	ProviderLastReceivedRequestTime atomic.Time
}

type ShutdownOption func(*ShutdownConfig)

// ---------- ShutdownOption ----------

func WithShutdown_Timeout(timeout string) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.Timeout = timeout
	}
}

func WithShutdown_StepTimeout(timeout string) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.StepTimeout = timeout
	}
}

func WithShutdown_ConsumerUpdateWaitTime(duration string) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.ConsumerUpdateWaitTime = duration
	}
}

func WithShutdown_RejectRequestHandler(handler string) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.RejectRequestHandler = handler
	}
}

func WithShutdown_InternalSignal(signal bool) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.InternalSignal = &signal
	}
}

func WithShutdown_OfflineRequestWindowTimeout(timeout string) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.OfflineRequestWindowTimeout = timeout
	}
}

func WithShutdown_RejectRequest(flag bool) ShutdownOption {
	return func(cfg *ShutdownConfig) {
		cfg.RejectRequest.Store(flag)
	}
}

type CustomConfig struct {
	ConfigMap map[string]interface{} `yaml:"config-map" json:"config-map,omitempty" property:"config-map"`
}

type CustomOption func(*CustomConfig)

// ---------- CustomOption ----------

func WithCustom_ConfigMap(cfgMap map[string]interface{}) CustomOption {
	return func(cfg *CustomConfig) {
		cfg.ConfigMap = cfgMap
	}
}

type ProfilesConfig struct {
	// active profiles
	Active string
}

type ProfilesOption func(*ProfilesConfig)

// ---------- ProfilesOption ----------

func WithProfiles_Active(active string) ProfilesOption {
	return func(cfg *ProfilesConfig) {
		cfg.Active = active
	}
}

// TLSConfig tls config
type TLSConfig struct {
	CACertFile    string `yaml:"ca-cert-file" json:"ca-cert-file" property:"ca-cert-file"`
	TLSCertFile   string `yaml:"tls-cert-file" json:"tls-cert-file" property:"tls-cert-file"`
	TLSKeyFile    string `yaml:"tls-key-file" json:"tls-key-file" property:"tls-key-file"`
	TLSServerName string `yaml:"tls-server-name" json:"tls-server-name" property:"tls-server-name"`
}

type TLSOption func(*TLSConfig)

// ---------- TLSOption ----------

func WithTLS_CACertFile(file string) TLSOption {
	return func(cfg *TLSConfig) {
		cfg.CACertFile = file
	}
}

func WithTLS_TLSCertFile(file string) TLSOption {
	return func(cfg *TLSConfig) {
		cfg.TLSCertFile = file
	}
}

func WithTLS_TLSKeyFile(file string) TLSOption {
	return func(cfg *TLSConfig) {
		cfg.TLSKeyFile = file
	}
}

func WithTLS_TLSServerName(name string) TLSOption {
	return func(cfg *TLSConfig) {
		cfg.TLSServerName = name
	}
}
