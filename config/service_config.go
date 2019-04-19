package config


type ServiceConfig struct {
	Service string `required:"true"  yaml:"service"  json:"service,omitempty"`
	URLs   []ConfigURL
}

func NewDefaultProviderServiceConfig() *ServiceConfig {
	return &ServiceConfig{}
}
