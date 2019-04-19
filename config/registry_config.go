package config

import "time"

type RegistryConfig struct {
	Id         string        `required:"true" yaml:"id"  json:"id,omitempty"`
	Address    string      `required:"true" yaml:"address"  json:"address,omitempty"`
	UserName   string        `yaml:"user_name" json:"user_name,omitempty"`
	Password   string        `yaml:"password" json:"password,omitempty"`
	TimeoutStr string        `yaml:"timeout" default:"5s" json:"timeout,omitempty"` // unit: second
	Timeout    time.Duration `yaml:"-"  json:"-"`
	Url        ConfigURL
	Type       string
}
