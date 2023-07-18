package consumer

type Options struct {
	// todo: confused
	RequestTimeout string `default:"3s" yaml:"request-timeout" json:"request-timeout,omitempty" property:"request-timeout"`
	ProxyFactory   string `default:"default" yaml:"proxy" json:"proxy,omitempty" property:"proxy"`
	// would influence reference
	AdaptiveService bool                        `default:"false" yaml:"adaptive-service" json:"adaptive-service" property:"adaptive-service"`
	References      map[string]*ReferenceConfig `yaml:"references" json:"references,omitempty" property:"references"`
	// would influence hystrix
	FilterConf                     interface{} `yaml:"filter-conf" json:"filter-conf,omitempty" property:"filter-conf"`
	MaxWaitTimeForServiceDiscovery string      `default:"3s" yaml:"max-wait-time-for-service-discovery" json:"max-wait-time-for-service-discovery,omitempty" property:"max-wait-time-for-service-discovery"`
	// would influence reference
	MeshEnabled bool `yaml:"mesh-enabled" json:"mesh-enabled,omitempty" property:"mesh-enabled"`
	// todo: use idl to provide Interface
	// unique
	InterfaceName string `yaml:"interface"  json:"interface,omitempty" property:"interface"`
	// maybe use other way to replace this field?
	// would set default from consumer
	Check *bool `yaml:"check"  json:"check,omitempty" property:"check"`
	// unique
	URL string `yaml:"url"  json:"url,omitempty" property:"url"`
	// would set default from consumer
	Filter string `yaml:"filter" json:"filter,omitempty" property:"filter"`
	// would set default from consumer
	Protocol string `yaml:"protocol"  json:"protocol,omitempty" property:"protocol"`
	// todo: figure out how to use this field
	// would set default from consumer
	RegistryIDs []string `yaml:"registry-ids"  json:"registry-ids,omitempty"  property:"registry-ids"`
	// would set default directly
	Cluster string `yaml:"cluster"  json:"cluster,omitempty" property:"cluster"`
	// unique, influenced by consumer
	Loadbalance string `yaml:"loadbalance"  json:"loadbalance,omitempty" property:"loadbalance"`
	// todo: move to runtime config
	// unique
	Retries string `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	// would set default from root(Application)
	Group string `yaml:"group"  json:"group,omitempty" property:"group"`
	// would set default from root(Application)
	Version string `yaml:"version"  json:"version,omitempty" property:"version"`
	// unique
	Serialization string `yaml:"serialization" json:"serialization" property:"serialization"`
	// unique
	ProvidedBy string `yaml:"provided_by"  json:"provided_by,omitempty" property:"provided_by"`
	// todo: figure out how to use MethodConfig
	Methods []*MethodConfig `yaml:"methods"  json:"methods,omitempty" property:"methods"`
	// unique
	Async bool `yaml:"async"  json:"async,omitempty" property:"async"`
	// unique
	Params map[string]string `yaml:"params"  json:"params,omitempty" property:"params"`
	// unique
	Generic string `yaml:"generic"  json:"generic,omitempty" property:"generic"`
	// unique
	Sticky bool `yaml:"sticky"   json:"sticky,omitempty" property:"sticky"`
	//// unique
	//RequestTimeout string `yaml:"timeout"  json:"timeout,omitempty" property:"timeout"`
	// unique
	ForceTag bool `yaml:"force.tag"  json:"force.tag,omitempty" property:"force.tag"`
	// would set default from Consumer
	TracingKey       string `yaml:"tracing-key" json:"tracing-key,omitempty" propertiy:"tracing-key"`
	metaDataType     string
	MeshProviderPort int `yaml:"mesh-provider-port" json:"mesh-provider-port,omitempty" propertiy:"mesh-provider-port"`
}

type ReferenceOptions struct {
}
type ConsumeOptions struct {
	RequestTimeout string
	Retries        string
}

type ConsumeOption func(*ConsumeOptions)

func WithRequestTimeout(timeout string) ConsumeOption {
	return func(opts *ConsumeOptions) {
		opts.RequestTimeout = timeout
	}
}

func WithRetries(retries string) ConsumeOption {
	return func(opts *ConsumeOptions) {
		opts.Retries = retries
	}
}

type TracingOptions struct {
}
