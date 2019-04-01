package registry

//////////////////////////////////////////////
// Registry Interface
//////////////////////////////////////////////

// for service discovery/registry
type Registry interface {

	//used for service provider calling , register services to registry
	RegisterProvider(ServiceConfigIf) error
	//used for service consumer calling , register services cared about ,for dubbo's admin monitoring
	RegisterConsumer(ServiceConfigIf) error
	//used for service consumer ,start listen goroutine
	GetListenEvent() chan *ServiceEvent

	//input the serviceConfig , registry should return serviceUrlArray with multi location(provider nodes) available
	GetService(*ServiceConfig) ([]*ServiceURL, error)

	Close()
}
