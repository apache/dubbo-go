package registry

import (
	"fmt"
)

import (
	"github.com/AlexStocks/goext/net"
)

//////////////////////////////////////////////
// Registry Interface
//////////////////////////////////////////////

// for service discovery/registry
type Registry interface {
	Register(conf interface{}) error
	Close()
}

//////////////////////////////////////////////
// application config
//////////////////////////////////////////////

type ApplicationConfig struct {
	Organization string `yaml:"organization"  json:"organization,omitempty"`
	Name         string `yaml:"name" json:"name,omitempty"`
	Module       string `yaml:"module" json:"module,omitempty"`
	Version      string `yaml:"version" json:"version,omitempty"`
	Owner        string `yaml:"owner" json:"owner,omitempty"`
}

func (c *ApplicationConfig) ToString() string {
	return fmt.Sprintf("ApplicationConfig is {name:%s, version:%s, owner:%s, module:%s, organization:%s}",
		c.Name, c.Version, c.Owner, c.Module, c.Organization)
}

type RegistryConfig struct {
	Address  []string `required:"true" yaml:"address"  json:"address,omitempty"`
	UserName string   `yaml:"user_name" json:"user_name,omitempty"`
	Password string   `yaml:"password" json:"password,omitempty"`
	Timeout  int      `yaml:"timeout" default:"5" json:"timeout,omitempty"` // unit: second
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
