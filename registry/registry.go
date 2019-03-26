package registry

import (
	"fmt"
	"time"
)

import (
	"github.com/AlexStocks/goext/net"
)

//////////////////////////////////////////////
// Registry Interface
//////////////////////////////////////////////

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


// for service discovery/registry
type Registry interface {

	//used for service provider calling , register services to registry
	ProviderRegister(conf ServiceConfigIf) error
	//used for service consumer calling , register services cared about ,for dubbo's admin monitoring
	ConsumerRegister(conf ServiceConfig) error
	//unregister service for service provider
	//Unregister(conf interface{}) error
	//used for service consumer ,start listen goroutine
	Listen()

	//input service config & request id, should return url which registry used
	Filter(ServiceConfigIf, int64) (*ServiceURL, error)
	Close()
	//new Provider conf
	NewProviderServiceConfig(ServiceConfig)ServiceConfigIf
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

//////////////////////////////////////////////
// service config
//////////////////////////////////////////////


type ServiceConfigIf interface {
	String() string
	ServiceEqual(url *ServiceURL) bool
}
type ServiceConfig struct {
	Protocol string `required:"true",default:"dubbo"  yaml:"protocol"  json:"protocol,omitempty"`
	Service  string `required:"true"  yaml:"service"  json:"service,omitempty"`
	Group    string `yaml:"group" json:"group,omitempty"`
	Version  string `yaml:"version" json:"version,omitempty"`
	//add for provider
	Path    string	`yaml:"path" json:"path,omitempty"`
	Methods string	`yaml:"methods" json:"methods,omitempty"`
}

func (c ServiceConfig) Key() string {
	return fmt.Sprintf("%s@%s", c.Service, c.Protocol)
}

func (c ServiceConfig) String() string {
	return fmt.Sprintf("%s@%s-%s-%s", c.Service, c.Protocol, c.Group, c.Version)
}

func (c ServiceConfig) ServiceEqual(url *ServiceURL) bool {
	if c.Protocol != url.Protocol {
		return false
	}

	if c.Service != url.Query.Get("interface") {
		return false
	}

	if c.Group != url.Group {
		return false
	}

	if c.Version != url.Version {
		return false
	}

	return true
}

type ServerConfig struct {
	Protocol string `required:"true",default:"dubbo" yaml:"protocol" json:"protocol,omitempty"` // codec string, jsonrpc  etc
	IP       string `yaml:"ip" json:"ip,omitempty"`
	Port     int    `required:"true" yaml:"port" json:"port,omitempty"`
}

func (c *ServerConfig) Address() string {
	return gxnet.HostAddress(c.IP, c.Port)
}


type ApplicationConfig struct {
	Organization string `yaml:"organization"  json:"organization,omitempty"`
	Name         string `yaml:"name" json:"name,omitempty"`
	Module       string `yaml:"module" json:"module,omitempty"`
	Version      string `yaml:"version" json:"version,omitempty"`
	Owner        string `yaml:"owner" json:"owner,omitempty"`
	Environment  string `yaml:"environment" json:"environment,omitempty"`
}

func (c *ApplicationConfig) ToString() string {
	return fmt.Sprintf("ApplicationConfig is {name:%s, version:%s, owner:%s, module:%s, organization:%s}",
		c.Name, c.Version, c.Owner, c.Module, c.Organization)
}