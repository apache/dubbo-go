package xds

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

// endPointWatcherCtx is endpoint watching context
type endPointWatcherCtx struct {
	clusterName   string
	interfaceName string
	hostAddr      string
	xdsClient     *WrappedClient
	cancel        func()
}

// handle handles endpoint update event and send to directory to refresh invoker
func (watcher *endPointWatcherCtx) handle(update resource.EndpointsUpdate, err error) {
	for _, v := range update.Localities {
		for _, e := range v.Endpoints {
			event := generateRegistryEvent(watcher.clusterName, e, watcher.interfaceName)
			watcher.xdsClient.hostAddrListenerMapLock.RLock()
			for _, l := range watcher.xdsClient.hostAddrListenerMap[watcher.hostAddr] {
				// notify all listeners listening this hostAddr
				l.Notify(event)
			}
			watcher.xdsClient.hostAddrListenerMapLock.Unlock()
		}
	}
}

// destroy call cancel and send event to listener to remove related invokers of current deleated cluster
func (watcher *endPointWatcherCtx) destroy() {
	watcher.cancel()
	/*
		directory would identify this by EndpointHealthStatusUnhealthy and Location == "*" and none empty clusterId
		and delete related invokers
	*/
	event := generateRegistryEvent(watcher.clusterName, resource.Endpoint{
		HealthStatus: resource.EndpointHealthStatusUnhealthy,
		Address:      constant.MeshAnyAddrMatcher, // destroy all endpoint of this cluster
	}, watcher.interfaceName)
	watcher.xdsClient.hostAddrListenerMapLock.RLock()
	for _, l := range watcher.xdsClient.hostAddrListenerMap[watcher.hostAddr] {
		// notify all listeners listening this hostAddr
		l.Notify(event)
	}
	watcher.xdsClient.hostAddrListenerMapLock.Unlock()
}
