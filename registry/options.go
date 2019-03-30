package registry

import (
	"fmt"
)

/////////////////////////////////
// dubbo role type
/////////////////////////////////

const (
	CONSUMER = iota
	CONFIGURATOR
	ROUTER
	PROVIDER
)

var (
	DubboNodes = [...]string{"consumers", "configurators", "routers", "providers"}
	DubboRole  = [...]string{"consumer", "", "", "provider"}
)

type DubboType int

func (t DubboType) String() string {
	return DubboNodes[t]
}

func (t DubboType) Role() string {
	return DubboRole[t]
}

/////////////////////////////////
// dubbo config & options
/////////////////////////////////

type RegistryOptions interface {
	ToString() string
}

type ApplicationConfig struct {
	Organization string `yaml:"organization"  json:"organization,omitempty"`
	Name         string `yaml:"name" json:"name,omitempty"`
	Module       string `yaml:"module" json:"module,omitempty"`
	Version      string `yaml:"version" json:"version,omitempty"`
	Owner        string `yaml:"owner" json:"owner,omitempty"`
	Environment  string `yaml:"environment" json:"environment,omitempty"`
}

type Options struct {
	ApplicationConfig
	DubboType DubboType
}

func (o *Options) ToString() string {
	return fmt.Sprintf("Options{name:%s, version:%s, owner:%s, module:%s, organization:%s, type:%s}",
		o.Name, o.Version, o.Owner, o.Module, o.Organization, o.DubboType)
}

type Option func(*Options)

func WithDubboType(typ DubboType) Option {
	return func(o *Options) {
		o.DubboType = typ
	}
}

func WithApplicationConf(conf ApplicationConfig) Option {
	return func(o *Options) {
		o.ApplicationConfig = conf
	}
}
