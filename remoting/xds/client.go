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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

import (
	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

	structpb "github.com/golang/protobuf/ptypes/struct"

	perrors "github.com/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/xds/client"
	"dubbo.apache.org/dubbo-go/v3/xds/client/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource/version"
)

const (
	gRPCUserAgentName               = "gRPC Go"
	clientFeatureNoOverprovisioning = "envoy.lb.does_not_support_overprovisioning"
)

const (
	istiodTokenPath     = "/var/run/secrets/token/istio-token"
	authorizationHeader = "Authorization"
	istiodTokenPrefix   = "Bearer "
)

var xdsWrappedClient *WrappedClient

type WrappedClient struct {
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
	hostAddr string

	/*
		istiod info
		istiodAddr is istio $(istioSeviceFullName):$(xds-grpc-port) like istiod.istio-system.svc.cluster.local:15010
		istiodPodIP is to call istiod unexposed debug port 8080
	*/
	istiodAddr  Addr
	istiodPodIP string

	/*
		grpc xdsClient sdk
	*/
	xdsClient client.XDSClient

	/*
		interfaceAppNameMap store map of serviceUniqueKey -> hostAddr
	*/
	interfaceAppNameMap     map[string]string
	interfaceAppNameMapLock sync.RWMutex

	/*
		rdsMap cache router config
		mesh router would read config from it
	*/
	rdsMap     map[string]resource.RouteConfigUpdate
	rdsMapLock sync.RWMutex

	/*
		cdsMap cache full clusterId -> clusterUpdate map of this istiod
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
	hostAddrClusterCtxMap     map[string]map[string]endPointWatcherCtx
	hostAddrClusterCtxMapLock sync.RWMutex

	/*
		interfaceNameHostAddrMap cache the dubbo interface unique key -> hostName
		the data is read from istiod:8080/debug/adsz, connection metadata["LABELS"]["DUBBO_GO"]
	*/
	interfaceNameHostAddrMap     map[string]string
	interfaceNameHostAddrMapLock sync.RWMutex

	/*
		subscribeStopChMap stores subscription stop chan
	*/
	subscribeStopChMap sync.Map
}

func GetXDSWrappedClient() *WrappedClient {
	return xdsWrappedClient
}

func NewXDSWrappedClient(podName, namespace, localIP string, istioAddr Addr) (*WrappedClient, error) {
	// todo @(laurence) safety problem? what if to concurrent 'new' both create new client?
	if xdsWrappedClient != nil {
		return xdsWrappedClient, nil
	}
	// get hostname from http://localhost:8080/debug/endpointz
	newClient := &WrappedClient{
		podName:             podName,
		namespace:           namespace,
		localIP:             localIP,
		istiodAddr:          istioAddr,
		interfaceAppNameMap: make(map[string]string),

		rdsMap: make(map[string]resource.RouteConfigUpdate),
		cdsMap: make(map[string]resource.ClusterUpdate),

		hostAddrListenerMap:   make(map[string]map[string]registry.NotifyListener),
		hostAddrClusterCtxMap: make(map[string]map[string]endPointWatcherCtx),

		interfaceNameHostAddrMap: make(map[string]string),
		cdsUpdateEventChan:       make(chan struct{}),
		cdsUpdateEventHandlers:   make([]func(), 0),
	}
	// todo @(laurence)  gr control
	go newClient.runWatchingResource()
	if err := newClient.initClientAndLoadLocalHostAddr(); err != nil {
		return nil, err
	}

	xdsWrappedClient = newClient
	return newClient, nil
}

func (w *WrappedClient) getServiceUniqueKeyHostAddrMapFromPilot() (map[string]string, error) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:8080/debug/adsz", w.istiodPodIP), nil)
	token, err := ioutil.ReadFile(istiodTokenPath)
	if err != nil {
		return nil, err
	}
	req.Header.Add(authorizationHeader, istiodTokenPrefix+string(token))
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Infof("[XDS Wrapped Client] Try getting interface host map from istio %s, IP %s with error %s\n",
			w.istiodAddr.HostnameOrIP, w.istiodPodIP, err)
		return nil, err
	}

	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	adszRsp := &ADSZResponse{}
	if err := json.Unmarshal(data, adszRsp); err != nil {
		return nil, err
	}
	return adszRsp.GetMap(), nil
}

// GetHostAddrByServiceUniqueKey  todo 1. timeout 2. hostAddr change?
func (w *WrappedClient) GetHostAddrByServiceUniqueKey(serviceUniqueKey string) (string, error) {
	w.interfaceNameHostAddrMapLock.RLock()
	if hostAddr, ok := w.interfaceNameHostAddrMap[serviceUniqueKey]; ok {
		return hostAddr, nil
	}
	w.interfaceNameHostAddrMapLock.Unlock()

	for {
		if interfaceHostAddrMap, err := w.getServiceUniqueKeyHostAddrMapFromPilot(); err != nil {
			return "", err
		} else {
			w.interfaceNameHostAddrMapLock.Lock()
			w.interfaceNameHostAddrMap = interfaceHostAddrMap
			w.interfaceNameHostAddrMapLock.Unlock()
			hostName, ok := interfaceHostAddrMap[serviceUniqueKey]
			if !ok {
				logger.Infof("[XDS Wrapped Client] Try getting interface %s 's host from istio %d:8080\n", serviceUniqueKey, w.istiodPodIP)
				time.Sleep(time.Millisecond * 100)
				continue
			}
			return hostName, nil
		}
	}
}

func getHostNameAndPortFromAddr(hostAddr string) (string, string) {
	ipPort := strings.Split(hostAddr, ":")
	hostName := ipPort[0]
	port := ipPort[1]
	return hostName, port
}

func (w *WrappedClient) GetClusterUpdateIgnoreVersion(hostAddr string) resource.ClusterUpdate {
	hostName, port := getHostNameAndPortFromAddr(hostAddr)
	w.cdsMapLock.RLock()
	defer w.cdsMapLock.Unlock()
	for clusterName, v := range w.cdsMap {
		clusterNameData := strings.Split(clusterName, "|")
		if clusterNameData[1] == port && clusterNameData[3] == hostName {
			return v
		}
	}
	return resource.ClusterUpdate{}
}

func (w *WrappedClient) getAllVersionClusterName(hostAddr string) []string {
	hostName, port := getHostNameAndPortFromAddr(hostAddr)
	allVersionClusterNames := make([]string, 0)
	w.cdsMapLock.RLock()
	defer w.cdsMapLock.Unlock()
	for clusterName := range w.cdsMap {
		clusterNameData := strings.Split(clusterName, "|")
		if clusterNameData[1] == port && clusterNameData[3] == hostName {
			allVersionClusterNames = append(allVersionClusterNames, clusterName)
		}
	}
	return allVersionClusterNames
}

func generateRegistryEvent(clusterName string, endpoint resource.Endpoint, interfaceName string) *registry.ServiceEvent {
	// todo now only support triple protocol
	url, _ := common.NewURL(fmt.Sprintf("tri://%s/%s", endpoint.Address, interfaceName))
	logger.Infof("[XDS Registry] Get Update event from pilot: interfaceName = %s, addr = %s, healthy = %d\n",
		interfaceName, endpoint.Address, endpoint.HealthStatus)
	clusterNames := strings.Split(clusterName, "|")
	// todo const MeshSubsetKey
	url.AddParam(constant.MeshSubsetKey, clusterNames[2])
	url.AddParam(constant.MeshClusterIDKey, clusterName)
	url.AddParam(constant.MeshHostAddrKey, clusterNames[3]+":"+clusterNames[1])
	if endpoint.HealthStatus == resource.EndpointHealthStatusUnhealthy {
		return &registry.ServiceEvent{
			Action:  remoting.EventTypeDel,
			Service: url,
		}
	}
	return &registry.ServiceEvent{
		Action:  remoting.EventTypeUpdate,
		Service: url,
	}
}

// registerHostLevelSubscription register: 1. all related cluster, 2. router config
func (w *WrappedClient) registerHostLevelSubscription(hostAddr, interfaceName, svcUniqueName string, lst registry.NotifyListener) {
	// 1. listen all cluster related endpoint
	w.hostAddrListenerMapLock.Lock()
	if _, ok := w.hostAddrListenerMap[hostAddr]; ok {
		// if subscription exist, register listener directly
		w.hostAddrListenerMap[hostAddr][svcUniqueName] = lst
		w.hostAddrListenerMapLock.Unlock()
		return
	}
	// host Addr key must not exist in map, create one
	w.hostAddrListenerMap[hostAddr] = make(map[string]registry.NotifyListener)

	w.hostAddrClusterCtxMapLock.Lock()
	w.hostAddrClusterCtxMap[hostAddr] = make(map[string]endPointWatcherCtx)
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
		w.hostAddrClusterCtxMapLock.Unlock()

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
			watcher := endPointWatcherCtx{
				interfaceName: interfaceName,
				clusterName:   updatedClusterName,
				hostAddr:      hostAddr,
				xdsClient:     w,
			}
			cancel := w.xdsClient.WatchEndpoints(updatedClusterName, watcher.handle)
			watcher.cancel = cancel
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
					watchCtx.destroy()
				}
				w.hostAddrClusterCtxMapLock.Unlock()
			}
		}
	})
	w.cdsUpdateEventHandlersLock.Unlock()

	// update cluster of now
	allVersionedClusterName := w.getAllVersionClusterName(hostAddr)
	for _, c := range allVersionedClusterName {
		watcher := endPointWatcherCtx{
			interfaceName: interfaceName,
			clusterName:   c,
			hostAddr:      hostAddr,
			xdsClient:     w,
		}
		watcher.cancel = w.xdsClient.WatchEndpoints(c, watcher.handle)

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

func (w *WrappedClient) GetRouterConfig(hostAddr string) resource.RouteConfigUpdate {
	w.rdsMapLock.RLock()
	defer w.rdsMapLock.Unlock()
	routeConfig, ok := w.rdsMap[hostAddr]
	if ok {
		return routeConfig
	}
	return resource.RouteConfigUpdate{}
}

func (w *WrappedClient) unregisterHostLevelSubscription(hostAddr, svcUniqueName string) {
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
				c.destroy()
				keys = append(keys, k)
			}
			for _, v := range keys {
				delete(w.hostAddrClusterCtxMap, v)
			}
			w.hostAddrClusterCtxMapLock.Unlock()
		}
	}
}

func (w *WrappedClient) Subscribe(svcUniqueName, interfaceName, hostAddr string, lst registry.NotifyListener) error {
	_, ok := w.subscribeStopChMap.Load(svcUniqueName)
	if ok {
		return perrors.Errorf("XDS WrappedClient subscribe interface %s failed, subscription already exist.", interfaceName)
	}
	stopCh := make(chan struct{})
	w.subscribeStopChMap.Store(svcUniqueName, stopCh)
	w.registerHostLevelSubscription(hostAddr, interfaceName, svcUniqueName, lst)
	<-stopCh
	w.unregisterHostLevelSubscription(hostAddr, svcUniqueName)
	return nil
}

func (w *WrappedClient) UnSubscribe(svcUniqueName string) {
	if stopCh, ok := w.subscribeStopChMap.Load(svcUniqueName); ok {
		close(stopCh.(chan struct{}))
	}
	w.subscribeStopChMap.Delete(svcUniqueName)
}

func (w *WrappedClient) interfaceAppNameMap2String() string {
	w.interfaceAppNameMapLock.RLock()
	defer w.interfaceAppNameMapLock.Unlock()
	data, _ := json.Marshal(w.interfaceAppNameMap)
	return string(data)
}

// ChangeInterfaceMap change the map of serviceUniqueKey -> appname, if add is true, register, else unregister
func (w *WrappedClient) ChangeInterfaceMap(serviceUniqueKey string, add bool) error {
	w.interfaceAppNameMapLock.Lock()
	defer w.interfaceAppNameMapLock.Unlock()
	if add {
		w.interfaceAppNameMap[serviceUniqueKey] = w.hostAddr
	} else {
		delete(w.interfaceAppNameMap, serviceUniqueKey)
	}
	if w.xdsClient == nil {
		xdsClient, err := newxdsClient(w.localIP, w.podName, w.namespace, w.interfaceAppNameMap2String(), w.istiodAddr)
		if err != nil {
			return err
		}
		w.xdsClient = xdsClient
		return nil
	}

	if err := w.xdsClient.SetMetadata(getDubboGoMetadata(w.interfaceAppNameMap2String())); err != nil {
		return err
	}
	return nil
}

func (w *WrappedClient) initClientAndLoadLocalHostAddr() error {
	// call watch and refresh istiod debug interface
	xdsClient, err := newxdsClient(w.localIP, w.podName, w.namespace, w.interfaceAppNameMap2String(), w.istiodAddr)
	if err != nil {
		return err
	}
	foundLocalStopCh := make(chan struct{})
	foundIstiodStopCh := make(chan struct{})
	foundLocal := false
	foundIstiod := false
	var cancel1 func()
	var cancel2 func()
	// todo @(laurence) here, if istiod is unhealthy, here should be timeout and tell developer.
	_ = xdsClient.WatchCluster("*", func(update resource.ClusterUpdate, err error) {
		if update.ClusterName == "" {
			return
		}
		if update.ClusterName[:1] == constant.MeshDeleteClusterPrefix {
			// delete event
			w.cdsMapLock.Lock()
			defer w.cdsMapLock.Unlock()
			delete(w.cdsMap, update.ClusterName[1:])
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
		// only into here during start sniffing istiod/service prcedure
		clusterNameList := strings.Split(update.ClusterName, "|")
		// todo: what's going on? istiod can't discover istiod.istio-system.svc.cluster.local!!
		if clusterNameList[3] == w.istiodAddr.HostnameOrIP {
			// 1. find istiod podIP
			// todo: When would eds level watch be canceled?
			cancel1 = xdsClient.WatchEndpoints(update.ClusterName, func(endpoint resource.EndpointsUpdate, err error) {
				if foundIstiod {
					return
				}
				for _, v := range endpoint.Localities {
					for _, e := range v.Endpoints {
						addrs := strings.Split(e.Address, ":")
						w.istiodPodIP = addrs[0]
						foundIstiod = true
						close(foundLocalStopCh)
					}
				}
			})
			return
		}
		// 2. found local hostAddr
		// todo: When would eds level watch be canceled?
		cancel2 = xdsClient.WatchEndpoints(update.ClusterName, func(endpoint resource.EndpointsUpdate, err error) {
			if foundLocal {
				return
			}
			for _, v := range endpoint.Localities {
				for _, e := range v.Endpoints {
					addrs := strings.Split(e.Address, ":")
					if addrs[0] == w.localIP {
						clusterNames := strings.Split(update.ClusterName, "|")
						w.hostAddr = clusterNames[3] + ":" + clusterNames[1]
						foundLocal = true
						close(foundIstiodStopCh)
					}
				}
			}
		})
	})
	<-foundIstiodStopCh
	<-foundLocalStopCh
	cancel1()
	cancel2()
	w.xdsClient = xdsClient
	return nil
}

func newxdsClient(localIP, podName, namespace, dubboGoMetadata string, istioAddr Addr) (client.XDSClient, error) {
	// todo fix these ugly magic num
	v3NodeProto := &v3corepb.Node{
		Id:                   "sidecar~" + localIP + "~" + podName + "." + namespace + "~" + namespace + ".svc.cluster.local",
		UserAgentName:        gRPCUserAgentName,
		UserAgentVersionType: &v3corepb.Node_UserAgentVersion{UserAgentVersion: "1.45.0"},
		ClientFeatures:       []string{clientFeatureNoOverprovisioning},
	}

	nonNilCredsConfigV2 := &bootstrap.Config{
		XDSServer: &bootstrap.ServerConfig{
			ServerURI:    istioAddr.String(),
			Creds:        grpc.WithTransportCredentials(insecure.NewCredentials()),
			TransportAPI: version.TransportV3,
			NodeProto:    v3NodeProto,
		},
		ClientDefaultListenerResourceNameTemplate: "%s",
	}

	newClient, err := client.NewWithConfig(nonNilCredsConfigV2)
	if err != nil {
		return nil, err
	}
	if err := newClient.SetMetadata(getDubboGoMetadata(dubboGoMetadata)); err != nil {
		return nil, err
	}
	return newClient, nil
}

func getDubboGoMetadata(dubboGoMetadata string) *structpb.Struct {
	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			constant.XDSMetadataClusterIDKey: {
				// Set cluster id to Kubernetes to ensure dubbo-go's xds client can get service
				// istiod.istio-system.svc.cluster.local's
				// pods ip from istiod by eds, to call no-endpoint port of istio like 8080
				Kind: &structpb.Value_StringValue{StringValue: constant.XDSMetadataDefaultDomainName},
			},
			constant.XDSMetadataLabelsKey: {
				Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						constant.XDSMetadataDubboGoMapperKey: {
							Kind: &structpb.Value_StringValue{StringValue: dubboGoMetadata},
						},
					},
				}},
			},
		},
	}
}

func (w *WrappedClient) runWatchingResource() {
	for range w.cdsUpdateEventChan {
		w.cdsUpdateEventHandlersLock.RLock()
		for _, h := range w.cdsUpdateEventHandlers {
			h()
		}
		w.cdsUpdateEventHandlersLock.Unlock()
	}
}
