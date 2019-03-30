package registry

import (
	"github.com/dubbo/dubbo-go/service"
)

//////////////////////////////////////////////
// Registry Interface
//////////////////////////////////////////////

// for service discovery/registry
type Registry interface {

	//used for service provider calling , register services to registry
	ProviderRegister(conf service.ServiceConfigIf) error
	//used for service consumer calling , register services cared about ,for dubbo's admin monitoring
	ConsumerRegister(conf *service.ServiceConfig) error
	//used for service consumer ,start listen goroutine
	GetListenEvent() chan *ServiceURLEvent

	//input the serviceConfig , registry should return serviceUrlArray with multi location(provider nodes) available
	GetService(*service.ServiceConfig) ([]*service.ServiceURL, error)

	Close()
	//new Provider conf
	NewProviderServiceConfig(service.ServiceConfig) service.ServiceConfigIf
}
