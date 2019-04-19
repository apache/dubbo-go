package config

type ReferenceConfig struct {
	Service string `required:"true"  yaml:"service"  json:"service,omitempty"`
	URLs   []ConfigURL
}

func NewReferenceConfig() *ReferenceConfig {
	return &ReferenceConfig{}
}