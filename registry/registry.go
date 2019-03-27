package registry

import (
	"fmt"
)

//////////////////////////////////////////////
// Registry Interface
//////////////////////////////////////////////



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

