package registry

import (
	"time"
)

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
type ApplicationConfig struct {
	Organization string `yaml:"organization"  json:"organization,omitempty"`
	Name         string `yaml:"name" json:"name,omitempty"`
	Module       string `yaml:"module" json:"module,omitempty"`
	Version      string `yaml:"version" json:"version,omitempty"`
	Owner        string `yaml:"owner" json:"owner,omitempty"`
	Environment  string `yaml:"environment" json:"environment,omitempty"`
}




type OptionInf interface{
	OptionName()string
}
type Options struct{
	ApplicationConfig
	Mode           Mode
	ServiceTTL     time.Duration
	DubboType      DubboType
}



//func (c *ApplicationConfig) ToString() string {
//	return fmt.Sprintf("ApplicationConfig is {name:%s, version:%s, owner:%s, module:%s, organization:%s}",
//		c.Name, c.Version, c.Owner, c.Module, c.Organization)
//}

type Option func(*Options)

func(Option)OptionName() string {
	return "Abstact option func"
}

func WithDubboType(tp DubboType)Option{
	return func (o *Options){
		o.DubboType = tp
	}
}

func WithApplicationConf(conf  ApplicationConfig) Option {
	return func(o *Options) {
		o.ApplicationConfig = conf
	}
}

func WithServiceTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.ServiceTTL = ttl
	}
}
func WithBalanceMode(mode Mode) Option {
	return func(o *Options) {
		o.Mode = mode
	}
}
