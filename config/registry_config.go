package config

type RegistryConfig struct {
	Id         string        `required:"true" yaml:"id"  json:"id,omitempty"`
	TimeoutStr string        `yaml:"timeout" default:"5s" json:"timeout,omitempty"` // unit: second
	RegistryURL
}
