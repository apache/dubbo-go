package config

type ApplicationConfig struct {
	Organization string `yaml:"organization"  json:"organization,omitempty"`
	Name         string `yaml:"name" json:"name,omitempty"`
	Module       string `yaml:"module" json:"module,omitempty"`
	Version      string `yaml:"version" json:"version,omitempty"`
	Owner        string `yaml:"owner" json:"owner,omitempty"`
	Environment  string `yaml:"environment" json:"environment,omitempty"`
}
