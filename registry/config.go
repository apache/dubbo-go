package registry

import (
	"fmt"
)

//////////////////////////////////////////////
// application config
//////////////////////////////////////////////

type ApplicationConfig struct {
	Organization string
	Name         string
	Module       string
	Version      string
	Owner        string
}

func (c *ApplicationConfig) ToString() string {
	return fmt.Sprintf("ApplicationConfig is {name:%s, version:%s, owner:%s, module:%s, organization:%s}",
		c.Name, c.Version, c.Owner, c.Module, c.Organization)
}

type RegistryConfig struct {
	Address  []string `required:"true"`
	UserName string
	Password string
	Timeout  int `default:"5"` // unit: second
}

//////////////////////////////////////////////
// service config
//////////////////////////////////////////////

type ServiceConfigIf interface {
	String() string
	ServiceEqual(url *ServiceURL) bool
}

type ServiceConfig struct {
	Protocol string `required:"true",default:"dubbo"`
	Service  string `required:"true"`
	Group    string
	Version  string
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
