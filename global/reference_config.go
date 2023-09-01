package global

// ReferenceConfig is the configuration of service consumer
type ReferenceConfig struct {
	InterfaceName    string            `yaml:"interface"  json:"interface,omitempty" property:"interface"`
	Check            *bool             `yaml:"check"  json:"check,omitempty" property:"check"`
	URL              string            `yaml:"url"  json:"url,omitempty" property:"url"`
	Filter           string            `yaml:"filter" json:"filter,omitempty" property:"filter"`
	Protocol         string            `yaml:"protocol"  json:"protocol,omitempty" property:"protocol"`
	RegistryIDs      []string          `yaml:"registry-ids"  json:"registry-ids,omitempty"  property:"registry-ids"`
	Cluster          string            `yaml:"cluster"  json:"cluster,omitempty" property:"cluster"`
	Loadbalance      string            `yaml:"loadbalance"  json:"loadbalance,omitempty" property:"loadbalance"`
	Retries          string            `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	Group            string            `yaml:"group"  json:"group,omitempty" property:"group"`
	Version          string            `yaml:"version"  json:"version,omitempty" property:"version"`
	Serialization    string            `yaml:"serialization" json:"serialization" property:"serialization"`
	ProvidedBy       string            `yaml:"provided_by"  json:"provided_by,omitempty" property:"provided_by"`
	Methods          []*MethodConfig   `yaml:"methods"  json:"methods,omitempty" property:"methods"`
	Async            bool              `yaml:"async"  json:"async,omitempty" property:"async"`
	Params           map[string]string `yaml:"params"  json:"params,omitempty" property:"params"`
	Generic          string            `yaml:"generic"  json:"generic,omitempty" property:"generic"`
	Sticky           bool              `yaml:"sticky"   json:"sticky,omitempty" property:"sticky"`
	RequestTimeout   string            `yaml:"timeout"  json:"timeout,omitempty" property:"timeout"`
	ForceTag         bool              `yaml:"force.tag"  json:"force.tag,omitempty" property:"force.tag"`
	TracingKey       string            `yaml:"tracing-key" json:"tracing-key,omitempty" propertiy:"tracing-key"`
	MeshProviderPort int               `yaml:"mesh-provider-port" json:"mesh-provider-port,omitempty" propertiy:"mesh-provider-port"`
}

type ReferenceOption func(*ReferenceConfig)

func DefaultReferenceConfig() *ReferenceConfig {
	return &ReferenceConfig{
		Methods: make([]*MethodConfig, 0, 8),
		Params:  make(map[string]string, 8),
	}
}
