package registry

import "github.com/dubbo/dubbo-go/config"

// Extension - Registry
type Registry interface {

	//used for service provider calling , register services to registry
	//And it is also used for service consumer calling , register services cared about ,for dubbo's admin monitoring.
	Register(config.ConfigURL) error

	//used for service consumer ,start subscribe service event from registry
	Subscribe(config.ConfigURL) (Listener, error)

	//input the serviceConfig , registry should return serviceUrlArray with multi location(provider nodes) available
	//GetService(ConfigURL) ([]ConfigURL, error)
	//close the registry for Elegant closing
	Close()
	//return if the registry is closed for consumer subscribing
	IsClosed() bool
}

type Listener interface {
	Next() (*ServiceEvent, error)
	Close()
}
