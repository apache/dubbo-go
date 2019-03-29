package service

import "fmt"

//////////////////////////////////////////////
// service config
//////////////////////////////////////////////


type ServiceConfigIf interface {
	Key() string
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

