package config_center

// CenterConfig is configuration for config center
//
// ConfigCenter also introduced concepts of namespace and group to better manage Key-Value pairs by group,
// those configs are already built-in in many professional third-party configuration centers.
// In most cases, namespace is used to isolate different tenants, while group is used to divide the key set from one tenant into groups.
//
// CenterConfig has currently supported Zookeeper, Nacos, Etcd, Consul, Apollo
type CenterConfig struct {
	Protocol  string            `validate:"required" yaml:"protocol"  json:"protocol,omitempty"`
	Address   string            `validate:"required" yaml:"address" json:"address,omitempty"`
	DataId    string            `yaml:"data-id" json:"data-id,omitempty"`
	Cluster   string            `yaml:"cluster" json:"cluster,omitempty"`
	Group     string            `yaml:"group" json:"group,omitempty"`
	Username  string            `yaml:"username" json:"username,omitempty"`
	Password  string            `yaml:"password" json:"password,omitempty"`
	Namespace string            `yaml:"namespace"  json:"namespace,omitempty"`
	AppID     string            `default:"dubbo" yaml:"app-id"  json:"app-id,omitempty"`
	Timeout   string            `default:"10s" yaml:"timeout"  json:"timeout,omitempty"`
	Params    map[string]string `yaml:"params"  json:"parameters,omitempty"`

	//FileExtension the suffix of config dataId, also the file extension of config content
	FileExtension string `default:"yaml" yaml:"file-extension" json:"file-extension" `
}

func DefaultCenterConfig() *CenterConfig {
	return &CenterConfig{
		Params: make(map[string]string),
	}
}

type CenterOption func(*CenterConfig)

func WithProtocol(protocol string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Protocol = protocol
	}
}

func WithAddress(address string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Address = address
	}
}

func WithDataID(id string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.DataId = id
	}
}

func WithCluster(cluster string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Cluster = cluster
	}
}

// todo: think about changing the name of another WithGroup in this package
func WithGroup_(group string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Group = group
	}
}

func WithUsername(name string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Username = name
	}
}

func WithPassword(password string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Password = password
	}
}

func WithNamespace(namespace string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Namespace = namespace
	}
}

func WithAppID(id string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.AppID = id
	}
}

// todo: think about changing the name of another WithTimeout in this package
func WithTimeout_(timeout string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Timeout = timeout
	}
}

func WithParams(params map[string]string) CenterOption {
	return func(cfg *CenterConfig) {
		if cfg.Params == nil {
			cfg.Params = make(map[string]string)
		}
		for k, v := range params {
			cfg.Params[k] = v
		}
	}
}

func WithFileExtension(extension string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.FileExtension = extension
	}
}
