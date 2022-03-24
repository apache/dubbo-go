package xds

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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

var xdsWrappedClient *WrappedClient

type WrappedClient struct {
	interfaceAppNameMap map[string]string
	subscribeStopChMap  sync.Map
	podName             string
	namespace           string
	localIP             string // to find hostAddr by cds and eds
	istiodHostName      string // todo: here must be istiod-grpc.istio-system.svc.cluster.local
	hostAddr            string // dubbo-go-app.default.svc.cluster.local:20000
	istiodPodIP         string // to call istiod unexposed debug port 8080
	xdsClient           client.XDSClient
	lock                sync.Mutex

	endpointClusterMap sync.Map

	edsMap    map[string]resource.EndpointsUpdate // edsMap is used to send delete signal if one cluster is deleted
	edsRWLock sync.RWMutex
	rdsMap    map[string]resource.RouteConfigUpdate // rdsMap is used by mesh router
	rdsRWLock sync.RWMutex
	cdsMap    map[string]resource.ClusterUpdate // cdsMap is full clusterId -> clusterUpdate cache of this istio system
	cdsRWLock sync.RWMutex

	cdsUpdateEventChan     chan struct{}
	cdsUpdateEventHandlers []func()

	hostAddrListenerMap       map[string]map[string]registry.NotifyListener
	hostAddrListenerMapLock   sync.RWMutex
	hostAddrClusterCtxMap     map[string]map[string]endPointWatcherCtx
	hostAddrClusterCtxMapLock sync.RWMutex

	interfaceNameHostAddrMap     map[string]string
	interfaceNameHostAddrMapLock sync.RWMutex
}

func GetXDSWrappedClient() *WrappedClient {
	return xdsWrappedClient
}

func NewXDSWrappedClient(podName, namespace, localIP, istiodHostName string) (*WrappedClient, error) {
	// todo @(laurence) safety problem? what if to concurrent 'new' both create new client?
	if xdsWrappedClient != nil {
		return xdsWrappedClient, nil
	}
	// get hostname from http://localhost:8080/debug/endpointz
	newClient := &WrappedClient{
		podName:             podName,
		namespace:           namespace,
		localIP:             localIP,
		istiodHostName:      istiodHostName,
		interfaceAppNameMap: make(map[string]string),

		rdsMap: make(map[string]resource.RouteConfigUpdate),
		cdsMap: make(map[string]resource.ClusterUpdate),
		edsMap: make(map[string]resource.EndpointsUpdate),

		hostAddrListenerMap:   make(map[string]map[string]registry.NotifyListener),
		hostAddrClusterCtxMap: make(map[string]map[string]endPointWatcherCtx),

		interfaceNameHostAddrMap: make(map[string]string),
		cdsUpdateEventChan:       make(chan struct{}),
		cdsUpdateEventHandlers:   make([]func(), 0),
	}
	// todo gr control
	newClient.startWatchingResource()
	if err := newClient.initClientAndLoadLocalHostAddr(); err != nil {
		return nil, err
	}

	xdsWrappedClient = newClient
	return newClient, nil
}

func (w *WrappedClient) getServiceUniqueKeyHostAddrMapFromPilot() (map[string]string, error) {
	req, _ := http.NewRequest(http.MethodGet, "http://"+w.istiodPodIP+":8080/debug/adsz", nil)
	token, err := os.ReadFile("/var/run/secrets/token/istio-token")
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+string(token))
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Infof("[XDS Wrapped Client] Try getting interface host map from istio %s with error %s\n", w.istiodHostName, err)
		return nil, err
	}

	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil{
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
	if hostAddr, ok := w.interfaceNameHostAddrMap[serviceUniqueKey]; ok {
		return hostAddr, nil
	}
	for {
		if interfaceHostAddrMap, err := w.getServiceUniqueKeyHostAddrMapFromPilot(); err != nil {
			return "", err
		} else {
			w.interfaceNameHostAddrMap = interfaceHostAddrMap
			hostName, ok := interfaceHostAddrMap[serviceUniqueKey]
			if !ok {
				logger.Infof("[XDS Wrapped Client] Try getting interface %s 's host from istio %d\n", serviceUniqueKey, w.istiodHostName)
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

func (w *WrappedClient) getAllVersionClusterName(hostAddr string) []string {
	hostName, port := getHostNameAndPortFromAddr(hostAddr)
	allVersionClusterNames := make([]string, 0)
	for clusterName, _ := range w.cdsMap {
		clusterNameData := strings.Split(clusterName, "|")
		if clusterNameData[1] == port && clusterNameData[3] == hostName {
			allVersionClusterNames = append(allVersionClusterNames, clusterName)
		}
	}
	return allVersionClusterNames
}

func generateRegistryEvent(clusterName string, endpoint resource.Endpoint, interfaceName string) *registry.ServiceEvent {
	// todo 1. register protocol in server side metadata 2. get metadata from endpoint，like label, for router
	url, _ := common.NewURL(fmt.Sprintf("tri://%s/%s", endpoint.Address, interfaceName))
	logger.Infof("[XDS Registry] Get Update event from pilot: interfaceName = %s, addr = %s, healthy = %d\n",
		interfaceName, endpoint.Address, endpoint.HealthStatus)
	clusterNames := strings.Split(clusterName, "|")
	// todo const MeshSubsetKey
	url.AddParam("meshSubset", clusterNames[2])
	url.AddParam("meshClusterId", clusterName)
	url.AddParam("meshHostAddr", clusterNames[3]+":"+clusterNames[1])
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
	if _, ok := w.hostAddrListenerMap[hostAddr]; ok {
		// if subscription exist, register listener directly
		w.hostAddrListenerMap[hostAddr][svcUniqueName] = lst
		return
	}
	// host Addr key must not exist in map, create one
	w.hostAddrListenerMap[hostAddr] = make(map[string]registry.NotifyListener)
	w.hostAddrClusterCtxMap[hostAddr] = make(map[string]endPointWatcherCtx)
	w.hostAddrListenerMap[hostAddr][svcUniqueName] = lst

	// watch cluster change, and start listening newcoming cluster
	w.cdsUpdateEventHandlers = append(w.cdsUpdateEventHandlers, func() {
		// todo @(laurnece) now this event would be called if any cluster is change, but not only this hostAddr's
		updatedAllVersionedClusterName := w.getAllVersionClusterName(hostAddr)
		// do patch
		listeningClustersCancelMap := w.hostAddrClusterCtxMap[hostAddr]
		oldlisteningClusterMap := make(map[string]bool)
		for cluster, _ := range listeningClustersCancelMap {
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
				interfaceName:       interfaceName,
				clusterName:         updatedClusterName,
				hostAddr:            hostAddr,
				hostAddrListenerMap: w.hostAddrListenerMap,
				xdsClient:           w,
			}
			cancel := w.xdsClient.WatchEndpoints(updatedClusterName, watcher.handle)
			watcher.cancel = cancel
			w.hostAddrClusterCtxMap[hostAddr][updatedClusterName] = watcher
		}

		// cancel not exist cluster
		for cluster, v := range oldlisteningClusterMap {
			if !v {
				// this cluster not exist in update cluster list
				if watchCtx, ok := w.hostAddrClusterCtxMap[hostAddr][cluster]; ok {
					delete(w.hostAddrClusterCtxMap[hostAddr], cluster)
					watchCtx.destroy()
				}
			}
		}
	})

	// update cluster of now
	allVersionedClusterName := w.getAllVersionClusterName(hostAddr)
	for _, c := range allVersionedClusterName {
		watcher := endPointWatcherCtx{
			interfaceName:       interfaceName,
			clusterName:         c,
			hostAddr:            hostAddr,
			hostAddrListenerMap: w.hostAddrListenerMap,
		}
		watcher.cancel = w.xdsClient.WatchEndpoints(c, watcher.handle)

		w.hostAddrClusterCtxMap[hostAddr][c] = watcher
	}

	// 2. cache route config
	// todo @(laurnece) cancel watching of this addr's rds
	_ = w.xdsClient.WatchRouteConfig(hostAddr, func(update resource.RouteConfigUpdate, err error) {
		if update.VirtualHosts == nil {
			return
		}
		w.rdsMap[hostAddr] = update
	})
}

func (w *WrappedClient) GetRouterConfig(hostAddr string) resource.RouteConfigUpdate {
	rconf, ok := w.rdsMap[hostAddr]
	if ok {
		return rconf
	}
	return resource.RouteConfigUpdate{}
}

func (w *WrappedClient) unregisterHostLevelSubscription(hostAddr, svcUniqueName string) {
	if _, ok := w.hostAddrListenerMap[hostAddr]; ok {
		// if subscription exist, register listener directly
		if _, exist := w.hostAddrListenerMap[hostAddr][svcUniqueName]; exist {
			delete(w.hostAddrListenerMap[hostAddr], svcUniqueName)
		}
		if (len(w.hostAddrListenerMap[hostAddr])) == 0 {
			// if no subscription of this host cancel all cds subscription of this hostAddr
			keys := make([]string, 0)
			for k, c := range w.hostAddrClusterCtxMap[hostAddr] {
				c.destroy()
				keys = append(keys, k)
			}
			for _, v := range keys {
				delete(w.hostAddrClusterCtxMap, v)
			}
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
	data, _ := json.Marshal(w.interfaceAppNameMap)
	return string(data)
}

// ChangeInterfaceMap change the map of interfaceName -> appname, if add is true, register, else unregister
func (w *WrappedClient) ChangeInterfaceMap(interfaceName string, add bool) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	if add {
		w.interfaceAppNameMap[interfaceName] = w.hostAddr
	} else {
		delete(w.interfaceAppNameMap, interfaceName)
	}
	if w.xdsClient == nil {
		xdsClient, err := newxdsClient(w.localIP, w.podName, w.namespace, w.interfaceAppNameMap2String(), w.istiodHostName)
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
	xdsClient, err := newxdsClient(w.localIP, w.podName, w.namespace, w.interfaceAppNameMap2String(), w.istiodHostName)
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
		if update.ClusterName[:1] == "-" {
			// delete event
			delete(w.cdsMap, update.ClusterName[1:])
			w.cdsUpdateEventChan <- struct{}{} // send update event
			return
		}
		w.cdsMap[update.ClusterName] = update
		w.cdsUpdateEventChan <- struct{}{} // send update event
		if foundLocal && foundIstiod {
			return
		}
		// only into here during start sniffing istiod/service prcedure
		clusterNameList := strings.Split(update.ClusterName, "|")
		// todo: what's going on? istiod can't discover istiod.istio-system.svc.cluster.local!!
		if clusterNameList[3] == w.istiodHostName {
			// 1. find istiod podIP
			// todo: When would eds level watch be cancelled?
			cancel1 = xdsClient.WatchEndpoints(update.ClusterName, func(endpoint resource.EndpointsUpdate, err error) {
				if foundIstiod {
					return
				}
				for _, v := range endpoint.Localities {
					for _, e := range v.Endpoints {
						w.endpointClusterMap.Store(e.Address, update.ClusterName)
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
		// todo: When would eds level watch be cancelled?
		cancel2 = xdsClient.WatchEndpoints(update.ClusterName, func(endpoint resource.EndpointsUpdate, err error) {
			if foundLocal {
				return
			}
			for _, v := range endpoint.Localities {
				for _, e := range v.Endpoints {
					w.endpointClusterMap.Store(e.Address, update.ClusterName)
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

func newxdsClient(localIP, podName, namespace, dubboGoMetadata, istiodIP string) (client.XDSClient, error) {
	v3NodeProto := &v3corepb.Node{
		Id:                   "sidecar~" + localIP + "~" + podName + "." + namespace + "~" + namespace + ".svc.cluster.local",
		UserAgentName:        gRPCUserAgentName,
		UserAgentVersionType: &v3corepb.Node_UserAgentVersion{UserAgentVersion: "1.45.0"},
		ClientFeatures:       []string{clientFeatureNoOverprovisioning},
	}

	nonNilCredsConfigV2 := &bootstrap.Config{
		XDSServer: &bootstrap.ServerConfig{
			ServerURI:    istiodIP + ":15010",
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
			"CLUSTER_ID": {
				// Set cluster id to Kubernetes to ensure dubbo-go's xds client can get service
				// istiod.istio-system.svc.cluster.local's
				// pods ip from istiod by eds, to call no-endpoint port of istio like 8080
				Kind: &structpb.Value_StringValue{StringValue: "Kubernetes"},
			},
			"LABELS": {
				Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"DUBBO_GO": {
							Kind: &structpb.Value_StringValue{StringValue: dubboGoMetadata},
						},
					},
				}},
			},
		},
	}
}

func (w *WrappedClient) startWatchingResource() {
	go func() {
		for _ = range w.cdsUpdateEventChan {
			// todo cdHandler lock
			for _, h := range w.cdsUpdateEventHandlers {
				h()
			}
		}
	}()
}
