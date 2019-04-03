package registry

//////////////////////////////////////////////
// Registry Interface
//////////////////////////////////////////////

// for service discovery/registry
type Registry interface {

	//used for service provider calling , register services to registry
	//And it is also used for service consumer calling , register services cared about ,for dubbo's admin monitoring.
	Register(ServiceConfigIf) error

	//used for service consumer ,start subscribe service event from registry
	Subscribe() (Listener, error)

	//input the serviceConfig , registry should return serviceUrlArray with multi location(provider nodes) available
	GetService(ServiceConfig) ([]*ServiceURL, error)
	//close the registry for Elegant closing
	Close()
	//return if the registry is closed for consumer subscribing
	IsClosed() bool
}

type Listener interface {
	Next() (*ServiceEvent, error)
	Close()
}
