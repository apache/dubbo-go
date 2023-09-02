package global

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

func WithCenter_Protocol(protocol string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Protocol = protocol
	}
}

func WithCenter_Address(address string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Address = address
	}
}

func WithCenter_DataID(id string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.DataId = id
	}
}

func WithCenter_Cluster(cluster string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Cluster = cluster
	}
}

func WithCenter_Group(group string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Group = group
	}
}

func WithCenter_Username(name string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Username = name
	}
}

func WithCenter_Password(password string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Password = password
	}
}

func WithCenter_Namespace(namespace string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Namespace = namespace
	}
}

func WithCenter_AppID(id string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.AppID = id
	}
}

func WithCenter_Timeout(timeout string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.Timeout = timeout
	}
}

func WithCenter_Params(params map[string]string) CenterOption {
	return func(cfg *CenterConfig) {
		if cfg.Params == nil {
			cfg.Params = make(map[string]string)
		}
		for k, v := range params {
			cfg.Params[k] = v
		}
	}
}

func WithCenter_FileExtension(extension string) CenterOption {
	return func(cfg *CenterConfig) {
		cfg.FileExtension = extension
	}
}
