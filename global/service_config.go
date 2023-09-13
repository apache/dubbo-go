package global

// ServiceConfig is the configuration of the service provider
type ServiceConfig struct {
	Filter                      string            `yaml:"filter" json:"filter,omitempty" property:"filter"`
	ProtocolIDs                 []string          `yaml:"protocol-ids"  json:"protocol-ids,omitempty" property:"protocol-ids"` // multi protocolIDs support, split by ','
	Interface                   string            `yaml:"interface"  json:"interface,omitempty" property:"interface"`
	RegistryIDs                 []string          `yaml:"registry-ids"  json:"registry-ids,omitempty"  property:"registry-ids"`
	Cluster                     string            `default:"failover" yaml:"cluster"  json:"cluster,omitempty" property:"cluster"`
	Loadbalance                 string            `default:"random" yaml:"loadbalance"  json:"loadbalance,omitempty"  property:"loadbalance"`
	Group                       string            `yaml:"group"  json:"group,omitempty" property:"group"`
	Version                     string            `yaml:"version"  json:"version,omitempty" property:"version" `
	Methods                     []*MethodConfig   `yaml:"methods"  json:"methods,omitempty" property:"methods"`
	Warmup                      string            `yaml:"warmup"  json:"warmup,omitempty"  property:"warmup"`
	Retries                     string            `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	Serialization               string            `yaml:"serialization" json:"serialization" property:"serialization"`
	Params                      map[string]string `yaml:"params"  json:"params,omitempty" property:"params"`
	Token                       string            `yaml:"token" json:"token,omitempty" property:"token"`
	AccessLog                   string            `yaml:"accesslog" json:"accesslog,omitempty" property:"accesslog"`
	TpsLimiter                  string            `yaml:"tps.limiter" json:"tps.limiter,omitempty" property:"tps.limiter"`
	TpsLimitInterval            string            `yaml:"tps.limit.interval" json:"tps.limit.interval,omitempty" property:"tps.limit.interval"`
	TpsLimitRate                string            `yaml:"tps.limit.rate" json:"tps.limit.rate,omitempty" property:"tps.limit.rate"`
	TpsLimitStrategy            string            `yaml:"tps.limit.strategy" json:"tps.limit.strategy,omitempty" property:"tps.limit.strategy"`
	TpsLimitRejectedHandler     string            `yaml:"tps.limit.rejected.handler" json:"tps.limit.rejected.handler,omitempty" property:"tps.limit.rejected.handler"`
	ExecuteLimit                string            `yaml:"execute.limit" json:"execute.limit,omitempty" property:"execute.limit"`
	ExecuteLimitRejectedHandler string            `yaml:"execute.limit.rejected.handler" json:"execute.limit.rejected.handler,omitempty" property:"execute.limit.rejected.handler"`
	Auth                        string            `yaml:"auth" json:"auth,omitempty" property:"auth"`
	NotRegister                 bool              `yaml:"not_register" json:"not_register,omitempty" property:"not_register"`
	ParamSign                   string            `yaml:"param.sign" json:"param.sign,omitempty" property:"param.sign"`
	Tag                         string            `yaml:"tag" json:"tag,omitempty" property:"tag"`
	TracingKey                  string            `yaml:"tracing-key" json:"tracing-key,omitempty" propertiy:"tracing-key"`
}

type ServiceOption func(*ServiceConfig)

func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		Methods: make([]*MethodConfig, 0, 8),
		Params:  make(map[string]string, 8),
	}
}

func WithService_Filter(filter string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Filter = filter
	}
}

func WithService_ProtocolIDs(protocolIDs []string) ServiceOption {
	return func(cfg *ServiceConfig) {
		if len(protocolIDs) <= 0 {
			cfg.ProtocolIDs = protocolIDs
		}
	}
}

func WithService_Interface(name string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Interface = name
	}
}

func WithService_RegistryIDs(registryIDs []string) ServiceOption {
	return func(cfg *ServiceConfig) {
		if len(registryIDs) <= 0 {
			cfg.RegistryIDs = registryIDs
		}
	}
}

func WithService_Cluster(cluster string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Cluster = cluster
	}
}

func WithService_LoadBalance(loadBalance string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Loadbalance = loadBalance
	}
}

func WithService_Group(group string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Group = group
	}
}

func WithService_Version(version string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Version = version
	}
}

func WithService_Methods(methods []*MethodConfig) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Methods = methods
	}
}

func WithService_WarmUp(warmUp string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Warmup = warmUp
	}
}

func WithService_Retries(retries string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Retries = retries
	}
}

func WithService_Serialization(serialization string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Serialization = serialization
	}
}

func WithService_Params(params map[string]string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Params = params
	}
}

func WithService_Token(token string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Token = token
	}
}

func WithService_AccessLog(accessLog string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.AccessLog = accessLog
	}
}

func WithService_TpsLimiter(tpsLimiter string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.TpsLimiter = tpsLimiter
	}
}

func WithService_TpsLimitInterval(tpsLimitInterval string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.TpsLimitInterval = tpsLimitInterval
	}
}

func WithService_TpsLimitRate(tpsLimitRate string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.TpsLimitRate = tpsLimitRate
	}
}

func WithService_TpsLimitStrategy(tpsLimitStrategy string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.TpsLimitStrategy = tpsLimitStrategy
	}
}

func WithService_TpsLimitRejectedHandler(tpsLimitRejectedHandler string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.TpsLimitRejectedHandler = tpsLimitRejectedHandler
	}
}

func WithService_ExecuteLimit(executeLimit string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.ExecuteLimit = executeLimit
	}
}

func WithService_ExecuteLimitRejectedHandler(executeLimitRejectedHandler string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.ExecuteLimitRejectedHandler = executeLimitRejectedHandler
	}
}

func WithService_Auth(auth string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Auth = auth
	}
}

func WithService_NotRegister(notRegister bool) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.NotRegister = notRegister
	}
}

func WithService_ParamSign(paramSign string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.ParamSign = paramSign
	}
}

func WithService_Tag(tag string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.Tag = tag
	}
}

func WithService_TracingKey(tracingKey string) ServiceOption {
	return func(cfg *ServiceConfig) {
		cfg.TracingKey = tracingKey
	}
}
