package registry

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/config"
)

// Extension - Registry
type Registry interface {
	common.Node
	//used for service provider calling , register services to registry
	//And it is also used for service consumer calling , register services cared about ,for dubbo's admin monitoring.
	Register(url config.URL) error

	//used for service consumer ,start subscribe service event from registry
	Subscribe(config.URL) (Listener, error)
}

type Listener interface {
	Next() (*ServiceEvent, error)
	Close()
}
