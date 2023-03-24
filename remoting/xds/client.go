/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package xds

import (
	"errors"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
	xdsCommon "dubbo.apache.org/dubbo-go/v3/remoting/xds/common"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds/ewatcher"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds/mapping"
	"dubbo.apache.org/dubbo-go/v3/xds/client"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
	"dubbo.apache.org/dubbo-go/v3/xds/utils/resolver"
)

const (
	// todo make istiodTokenPath configurable
	defaultIstiodTokenPath          = "/var/run/secrets/token/istio-token"
	defaultIstiodDebugPort          = "8080"
	gRPCUserAgentName               = "gRPC Go"
	clientFeatureNoOverprovisioning = "envoy.lb.does_not_support_overprovisioning"
)

// xdsWrappedClient should only init once
var xdsWrappedClient *WrappedClientImpl

type WrappedClientImpl struct {
	/*
		local info
	*/
	podName   string
	namespace string

	/*
		localIP is  to find local pod's cluster and hostAddr by cds and eds
	*/
	localIP string

	/*
		hostAddr is local pod's cluster and hostAddr, like dubbo-go-app.default.svc.cluster.local:20000
	*/
	hostAddr xdsCommon.HostAddr

	/*
		istiod info
		istiodAddr is istio $(istioSeviceFullName):$(xds-grpc-port) like istiod.istio-system.svc.cluster.local:15010
		istiodPodIP is to call istiod unexposed debug port 8080
	*/
	istiodAddr  xdsCommon.HostAddr
	istiodPodIP string

	/*
		grpc xdsClient sdk
	*/
	xdsClient client.XDSClient

	/*
		interfaceMapHandler manages dubbogo metadata containing service key -> hostAddr map
	*/
	interfaceMapHandler mapping.InterfaceMapHandler

	/*
		rdsMap cache router config
		mesh router would read config from it
	*/
	rdsMap     map[string]resource.RouteConfigUpdate
	rdsMapLock sync.RWMutex

	/*
		cdsMap cache full clusterID -> clusterUpdate map of this istiod
	*/
	cdsMap     map[string]resource.ClusterUpdate
	cdsMapLock sync.RWMutex

	/*
		cdsUpdateEventChan transfer cds update event from xdsClient
		if update event got, we will refresh cds watcher, stopping endPointWatcherCtx related to deleted cluster, and starting
		to watch new-coming cluster with endPointWatcherCtx

		cdsUpdateEventHandlers stores handlers to recv refresh event, refresh event is only a call without param,
		after the calling event, we can read cdsMap to get latest and full cluster info, and handle the difference.
	*/
	cdsUpdateEventChan         chan struct{}
	cdsUpdateEventHandlers     []func()
	cdsUpdateEventHandlersLock sync.RWMutex

	/*
		hostAddrListenerMap[hostAddr][serviceUniqueKey] -> registry.NotifyListener
		stores all directory listener, which receives events and refresh invokers
	*/
	hostAddrListenerMap     map[string]map[string]registry.NotifyListener
	hostAddrListenerMapLock sync.RWMutex

	/*
		hostAddrClusterCtxMap[hostAddr][clusterName] -> endPointWatcherCtx
	*/
	hostAddrClusterCtxMap     map[string]map[string]ewatcher.EWatcher
	hostAddrClusterCtxMapLock sync.RWMutex

	/*
		subscribeStopChMap stores subscription stop chan
	*/
	subscribeStopChMap sync.Map

	/*
		xdsSniffingTimeout stores xds sniffing timeout duration
	*/
	xdsSniffingTimeout time.Duration
}

func GetXDSWrappedClient() *WrappedClientImpl {
	return xdsWrappedClient
}

// NewXDSWrappedClient create or get singleton xdsWrappedClient
func NewXDSWrappedClient(config Config) (XDSWrapperClient, error) {
	// todo @(laurence) safety problem? what if to concurrent 'new' both create new client?
	if xdsWrappedClient != nil {
		return xdsWrappedClient, nil
	}
	if config.SniffingTimeout == 0 {
		config.SniffingTimeout, _ = time.ParseDuration(constant.DefaultRegTimeout)
	}
	if config.DebugPort == "" {
		config.DebugPort = "8080"
	}

	// write param
	newClient := &WrappedClientImpl{
		podName:    config.PodName,
		namespace:  config.Namespace,
		localIP:    config.LocalIP,
		istiodAddr: config.IstioAddr,

		rdsMap: make(map[string]resource.RouteConfigUpdate),
		cdsMap: make(map[string]resource.ClusterUpdate),

		hostAddrListenerMap:   make(map[string]map[string]registry.NotifyListener),
		hostAddrClusterCtxMap: make(map[string]map[string]ewatcher.EWatcher),

		cdsUpdateEventChan:     make(chan struct{}),
		cdsUpdateEventHandlers: make([]func(), 0),

		xdsSniffingTimeout: config.SniffingTimeout,
	}

	// 1. init xdsclient
	if err := newClient.initXDSClient(); err != nil {
		return nil, err
	}
	// 2. watching cds update event
	// todo @(laurence)  gr control
	go newClient.runWatchingCdsUpdateEvent()

	// 3. load basic info from istiod and start listening cds
	if err := newClient.startWatchingAllClusterAndLoadLocalHostAddrAndIstioPodIP(config.LocalDebugMode); err != nil {
		return nil, err
	}

	// 4. init interface map handler
	newClient.interfaceMapHandler = mapping.NewInterfaceMapHandlerImpl(
		newClient.xdsClient,
		defaultIstiodTokenPath,
		xdsCommon.NewHostNameOrIPAddr(newClient.istiodPodIP+":"+config.DebugPort),
		newClient.hostAddr, config.LocalDebugMode)

	xdsWrappedClient = newClient
	return newClient, nil
}

// GetHostAddrByServiceUniqueKey  todo 1. timeout 2. hostAddr change?
func (w *WrappedClientImpl) GetHostAddrByServiceUniqueKey(serviceUniqueKey string) (string, error) {
	return w.interfaceMapHandler.GetHostAddrMap(serviceUniqueKey)
}

// GetDubboGoMetadata get all registered metadata of dubbogo
func (w *WrappedClientImpl) GetDubboGoMetadata() (map[string]string, error) {
	return w.interfaceMapHandler.GetDubboGoMetadata()
}

// ChangeInterfaceMap change the map of serviceUniqueKey -> appname, if add is true, register, else unregister
func (w *WrappedClientImpl) ChangeInterfaceMap(serviceUniqueKey string, add bool) error {
	if add {
		return w.interfaceMapHandler.Register(serviceUniqueKey)
	}
	return w.interfaceMapHandler.UnRegister(serviceUniqueKey)
}

func (w *WrappedClientImpl) GetRouterConfig(hostAddr string) resource.RouteConfigUpdate {
	w.rdsMapLock.RLock()
	defer w.rdsMapLock.RUnlock()
	routeConfig, ok := w.rdsMap[hostAddr]
	if ok {
		return routeConfig
	}
	return resource.RouteConfigUpdate{}
}

func (w *WrappedClientImpl) GetClusterUpdateIgnoreVersion(hostAddr string) resource.ClusterUpdate {
	addr := xdsCommon.NewHostNameOrIPAddr(hostAddr)
	w.cdsMapLock.RLock()
	defer w.cdsMapLock.Unlock()
	for clusterName, v := range w.cdsMap {
		cluster := xdsCommon.NewCluster(clusterName)
		if cluster.Addr.Port == addr.Port && cluster.Addr.HostnameOrIP == addr.HostnameOrIP {
			return v
		}
	}
	return resource.ClusterUpdate{}
}

func (w *WrappedClientImpl) Subscribe(svcUniqueName, interfaceName, hostAddr string, lst registry.NotifyListener) error {
	_, ok := w.subscribeStopChMap.Load(svcUniqueName)
	if ok {
		return perrors.Errorf("XDS WrappedClientImpl subscribe interface %s failed, subscription already exist.", interfaceName)
	}
	stopCh := make(chan struct{})
	w.subscribeStopChMap.Store(svcUniqueName, stopCh)
	w.registerHostLevelSubscription(hostAddr, interfaceName, svcUniqueName, lst)
	<-stopCh
	w.unregisterHostLevelSubscription(hostAddr, svcUniqueName)
	return nil
}

func (w *WrappedClientImpl) UnSubscribe(svcUniqueName string) {
	if stopCh, ok := w.subscribeStopChMap.Load(svcUniqueName); ok {
		close(stopCh.(chan struct{}))
	}
	w.subscribeStopChMap.Delete(svcUniqueName)
}

func (w *WrappedClientImpl) GetHostAddress() xdsCommon.HostAddr {
	return w.hostAddr
}

func (w *WrappedClientImpl) GetIstioPodIP() string {
	return w.istiodPodIP
}

// registerHostLevelSubscription register: 1. all related cluster, 2. router config
func (w *WrappedClientImpl) registerHostLevelSubscription(hostAddr, interfaceName, svcUniqueName string, lst registry.NotifyListener) {
	// 1. listen all cluster related endpoint
	w.hostAddrListenerMapLock.Lock()
	if _, ok := w.hostAddrListenerMap[hostAddr]; ok {
		// if subscription exist, register listener directly
		w.hostAddrListenerMap[hostAddr][svcUniqueName] = lst
		w.hostAddrListenerMapLock.Unlock()
		return
	}
	// host HostAddr key must not exist in map, create one
	w.hostAddrListenerMap[hostAddr] = make(map[string]registry.NotifyListener)

	w.hostAddrClusterCtxMapLock.Lock()
	w.hostAddrClusterCtxMap[hostAddr] = make(map[string]ewatcher.EWatcher)
	w.hostAddrClusterCtxMapLock.Unlock()

	w.hostAddrListenerMap[hostAddr][svcUniqueName] = lst
	w.hostAddrListenerMapLock.Unlock()

	// watch cluster change, and start listening newcoming cluster
	w.cdsUpdateEventHandlersLock.Lock()
	w.cdsUpdateEventHandlers = append(w.cdsUpdateEventHandlers, func() {
		// todo @(laurnece) now this event would be called if any cluster is change, but not only this hostAddr's
		updatedAllVersionedClusterName := w.getAllVersionClusterName(hostAddr)
		// do patch
		w.hostAddrClusterCtxMapLock.RLock()
		listeningClustersCancelMap := w.hostAddrClusterCtxMap[hostAddr]
		w.hostAddrClusterCtxMapLock.RUnlock()

		oldlisteningClusterMap := make(map[string]bool)
		for cluster := range listeningClustersCancelMap {
			oldlisteningClusterMap[cluster] = false
		}
		for _, updatedClusterName := range updatedAllVersionedClusterName {
			if _, ok := listeningClustersCancelMap[updatedClusterName]; ok {
				// already listening
				oldlisteningClusterMap[updatedClusterName] = true
				continue
			}
			// new cluster
			watcher := ewatcher.NewEndpointWatcherCtxImpl(
				updatedClusterName, hostAddr, interfaceName, &w.hostAddrListenerMapLock, w.hostAddrListenerMap)
			cancel := w.xdsClient.WatchEndpoints(updatedClusterName, watcher.Handle)
			watcher.SetCancelFunction(cancel)
			w.hostAddrClusterCtxMapLock.Lock()
			w.hostAddrClusterCtxMap[hostAddr][updatedClusterName] = watcher
			w.hostAddrClusterCtxMapLock.Unlock()
		}

		// cancel not exist cluster
		for cluster, v := range oldlisteningClusterMap {
			if !v {
				// this cluster not exist in update cluster list
				w.hostAddrClusterCtxMapLock.Lock()
				if watchCtx, ok := w.hostAddrClusterCtxMap[hostAddr][cluster]; ok {
					delete(w.hostAddrClusterCtxMap[hostAddr], cluster)
					watchCtx.Destroy()
				}
				w.hostAddrClusterCtxMapLock.Unlock()
			}
		}
	})
	w.cdsUpdateEventHandlersLock.Unlock()

	// update cluster of now
	allVersionedClusterName := w.getAllVersionClusterName(hostAddr)
	for _, c := range allVersionedClusterName {
		watcher := ewatcher.NewEndpointWatcherCtxImpl(
			c, hostAddr, interfaceName, &w.hostAddrListenerMapLock, w.hostAddrListenerMap)
		watcher.SetCancelFunction(w.xdsClient.WatchEndpoints(c, watcher.Handle))

		w.hostAddrClusterCtxMapLock.Lock()
		w.hostAddrClusterCtxMap[hostAddr][c] = watcher
		w.hostAddrClusterCtxMapLock.Unlock()
	}

	// 2. cache route config
	// todo @(laurnece) cancel watching of this addr's rds
	_ = w.xdsClient.WatchRouteConfig(hostAddr, func(update resource.RouteConfigUpdate, err error) {
		if update.VirtualHosts == nil {
			return
		}
		w.rdsMapLock.Lock()
		defer w.rdsMapLock.Unlock()
		w.rdsMap[hostAddr] = update
	})
}

func (w *WrappedClientImpl) unregisterHostLevelSubscription(hostAddr, svcUniqueName string) {
	w.hostAddrListenerMapLock.Lock()
	defer w.hostAddrListenerMapLock.Unlock()
	if _, ok := w.hostAddrListenerMap[hostAddr]; ok {
		// if subscription exist, register listener directly
		if _, exist := w.hostAddrListenerMap[hostAddr][svcUniqueName]; exist {
			delete(w.hostAddrListenerMap[hostAddr], svcUniqueName)
		}
		if (len(w.hostAddrListenerMap[hostAddr])) == 0 {
			// if no subscription of this host cancel all cds subscription of this hostAddr
			keys := make([]string, 0)
			w.hostAddrClusterCtxMapLock.Lock()
			for k, c := range w.hostAddrClusterCtxMap[hostAddr] {
				c.Destroy()
				keys = append(keys, k)
			}
			for _, v := range keys {
				delete(w.hostAddrClusterCtxMap, v)
			}
			w.hostAddrClusterCtxMapLock.Unlock()
		}
	}
}

func (w *WrappedClientImpl) initXDSClient() error {
	xdsClient, err := xdsClientFactoryFunction(w.localIP, w.podName, w.namespace, w.istiodAddr)
	if err != nil {
		return err
	}
	w.xdsClient = xdsClient
	return nil
}

// startWatchingAllClusterAndLoadLocalHostAddrAndIstioPodIP is blocking function
// 1. start watching all cluster by cds
// 2. discovery local pod's hostAddr by cds and eds
// 3. discovery istiod pod ip by cds and eds
func (w *WrappedClientImpl) startWatchingAllClusterAndLoadLocalHostAddrAndIstioPodIP(localDebugMode bool) error {
	// call watch and refresh istiod debug interface
	foundLocalStopCh := make(chan struct{})
	foundIstiodStopCh := make(chan struct{})
	discoveryFinishedStopCh := make(chan struct{})
	// todo timeout configure
	timeoutCh := time.After(w.xdsSniffingTimeout)
	foundLocal := false
	foundIstiod := false
	var cancel1 func()
	var cancel2 func()
	logger.Infof("[XDS Wrapped Client] Start sniffing with istio hostname = %s, localIp = %s",
		w.istiodAddr.HostnameOrIP, w.localIP)

	// todo @(laurence) here, if istiod is unhealthy, here should be timeout and tell developer.
	_ = w.xdsClient.WatchCluster("*", func(update resource.ClusterUpdate, err error) {
		if update.ClusterName == "" {
			return
		}
		if update.ClusterName[:1] == constant.MeshDeleteClusterPrefix {
			// delete event
			w.cdsMapLock.Lock()
			defer w.cdsMapLock.Unlock()
			delete(w.cdsMap, update.ClusterName[1:])
			logger.Infof("[XDS Wrapped Client] Delete cluster %s", update.ClusterName)
			w.cdsUpdateEventChan <- struct{}{} // send update event
			return
		}
		w.cdsMapLock.Lock()
		w.cdsMap[update.ClusterName] = update
		w.cdsMapLock.Unlock()

		w.cdsUpdateEventChan <- struct{}{} // send update event
		if foundLocal && foundIstiod {
			return
		}
		logger.Infof("[XDS Wrapped Client] Sniffing with cluster name = %s", update.ClusterName)
		// only into here during start sniffing istiod/service prcedure
		cluster := xdsCommon.NewCluster(update.ClusterName)
		if cluster.Addr.HostnameOrIP == w.istiodAddr.HostnameOrIP {
			// 1. find istiod podIP
			// todo: When would eds level watch be canceled?
			logger.Info("[XDS Wrapped Client] Sniffing get istiod cluster")
			cancel1 = w.xdsClient.WatchEndpoints(update.ClusterName, func(endpoint resource.EndpointsUpdate, err error) {
				if foundIstiod {
					return
				}
				logger.Infof("[XDS Wrapped Client] Sniffing get istiod endpoint = %+v, localities = %+v", endpoint, endpoint.Localities)
				for _, v := range endpoint.Localities {
					for _, e := range v.Endpoints {
						w.istiodPodIP = xdsCommon.NewHostNameOrIPAddr(e.Address).HostnameOrIP
						logger.Infof("[XDS Wrapped Client] Sniffing found istiod podIP = %s", w.istiodPodIP)
						foundIstiod = true
						close(foundIstiodStopCh)
					}
				}
			})
			return
		}
		// 2. found local hostAddr
		// todo: When would eds level watch be canceled?
		cancel2 = w.xdsClient.WatchEndpoints(update.ClusterName, func(endpoint resource.EndpointsUpdate, err error) {
			if foundLocal {
				return
			}
			for _, v := range endpoint.Localities {
				for _, e := range v.Endpoints {
					logger.Infof("[XDS Wrapped Client] Sniffing Found eds endpoint = %+v", e)
					if xdsCommon.NewHostNameOrIPAddr(e.Address).HostnameOrIP == w.localIP {
						cluster := xdsCommon.NewCluster(update.ClusterName)
						w.hostAddr = cluster.Addr
						foundLocal = true
						close(foundLocalStopCh)
					}
				}
			}
		})
	})

	if localDebugMode {
		go func() {
			<-foundIstiodStopCh
			<-foundLocalStopCh
			cancel1()
			cancel2()
		}()
		return nil
	}

	go func() {
		<-foundIstiodStopCh
		<-foundLocalStopCh
		close(discoveryFinishedStopCh)
	}()

	select {
	case <-discoveryFinishedStopCh:
		// discovery success
		// waiting for cancel function to have value
		time.Sleep(time.Second)
		cancel1()
		cancel2()
		logger.Infof("[XDS Wrapper Client] Sniffing Finished with host addr = %s, istiod pod ip = %s", w.hostAddr, w.istiodPodIP)
		return nil
	case <-timeoutCh:
		logger.Warnf("[XDS Wrapper Client] Sniffing timeout with duration = %v", w.xdsSniffingTimeout)
		if cancel1 != nil {
			cancel1()
		}
		if cancel2 != nil {
			cancel2()
		}
		select {
		case <-foundIstiodStopCh:
			return DiscoverLocalError
		default:
			return DiscoverIstiodPodIpError
		}
	}
}

// runWatchingCdsUpdateEvent is blocking function, starts to read event from cdsUpdateEventChan and call cdsUpdateEventHandlers
func (w *WrappedClientImpl) runWatchingCdsUpdateEvent() {
	for range w.cdsUpdateEventChan {
		w.cdsUpdateEventHandlersLock.RLock()
		for _, h := range w.cdsUpdateEventHandlers {
			h()
		}
		w.cdsUpdateEventHandlersLock.RUnlock()
	}
}

// getAllVersionClusterName get all clusterID that is the subset of given hostAddr from cache: cdsMap
// like: if given hostAddr is 'outbound|20000||dubbo-go-app.default.svc.cluster.local', and result would be
// ['outbound|20000|v1|dubbo-go-app.default.svc.cluster.local',
// 'outbound|20000||dubbo-go-app.default.svc.cluster.local',
// 'outbound|20000|v2|dubbo-go-app.default.svc.cluster.local']
func (w *WrappedClientImpl) getAllVersionClusterName(hostAddr string) []string {
	addr := xdsCommon.NewHostNameOrIPAddr(hostAddr)
	allVersionClusterNames := make([]string, 0)
	w.cdsMapLock.RLock()
	defer w.cdsMapLock.RUnlock()
	for clusterName, _ := range w.cdsMap {
		cluster := xdsCommon.NewCluster(clusterName)
		if cluster.Addr.Port == addr.Port && cluster.Addr.HostnameOrIP == addr.HostnameOrIP {
			allVersionClusterNames = append(allVersionClusterNames, clusterName)
		}
	}
	return allVersionClusterNames
}

func (w *WrappedClientImpl) MatchRoute(routerConfig resource.RouteConfigUpdate, invocation protocol.Invocation) (*resource.Route, error) {
	ctx := invocation.GetAttachmentAsContext()
	rpcInfo := resolver.RPCInfo{
		Context: ctx,
		Method:  "/" + invocation.MethodName(),
	}
	// try to route to sub virtual host
	for _, vh := range routerConfig.VirtualHosts {
		for _, r := range vh.Routes {
			//route.
			matcher, err := resource.RouteToMatcher(r)
			if err != nil {
				return nil, err
			}
			if matcher.Match(rpcInfo) {
				return r, nil
			}
		}
	}
	return nil, errors.New("not found route")
}

type XDSWrapperClient interface {
	Subscribe(svcUniqueName, interfaceName, hostAddr string, lst registry.NotifyListener) error
	UnSubscribe(svcUniqueName string)
	GetRouterConfig(hostAddr string) resource.RouteConfigUpdate
	GetHostAddrByServiceUniqueKey(serviceUniqueKey string) (string, error)
	GetDubboGoMetadata() (map[string]string, error)
	ChangeInterfaceMap(serviceUniqueKey string, add bool) error
	GetClusterUpdateIgnoreVersion(hostAddr string) resource.ClusterUpdate
	GetHostAddress() xdsCommon.HostAddr
	GetIstioPodIP() string
	MatchRoute(routerConfig resource.RouteConfigUpdate, invocation protocol.Invocation) (*resource.Route, error)
}
