package xds

import (
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

type endPointWatcherCtx struct {
	clusterName         string
	interfaceName       string
	hostAddr            string
	xdsClient           *WrappedClient
	hostAddrListenerMap map[string]map[string]registry.NotifyListener
	cancel              func()
}

func (watcher *endPointWatcherCtx) handle(update resource.EndpointsUpdate, err error) {
	for _, v := range update.Localities {
		for _, e := range v.Endpoints {
			// FIXME: is this c we want?
			event := generateRegistryEvent(watcher.clusterName, e, watcher.interfaceName)
			// todo lock WrappedClient's resource
			for _, l := range watcher.hostAddrListenerMap[watcher.hostAddr] {
				// notify all listeners listening this hostAddr
				l.Notify(event)
			}
		}
	}
}

func (watcher *endPointWatcherCtx) destroy() {
	watcher.cancel()
	event := generateRegistryEvent(watcher.clusterName, resource.Endpoint{
		HealthStatus: resource.EndpointHealthStatusUnhealthy,
		Address:      "*", // destroy all endpoint of this cluster
	}, watcher.interfaceName)
	for _, l := range watcher.hostAddrListenerMap[watcher.hostAddr] {
		// notify all listeners listening this hostAddr
		l.Notify(event)
	}
}