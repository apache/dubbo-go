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
}

func NewXDSWrappedClient(podName, namespace, localIP, istiodHostName string) (*WrappedClient, error) {
	// get hostname from http://localhost:8080/debug/endpointz
	newClient := &WrappedClient{
		podName:             podName,
		namespace:           namespace,
		localIP:             localIP,
		istiodHostName:      istiodHostName,
		interfaceAppNameMap: make(map[string]string),
	}
	if err := newClient.initClientAndloadLocalHostAddr(); err != nil {
		return nil, err
	}
	return newClient, nil
}

func (w *WrappedClient) getInterfaceHostAddrMapFromPilot() (map[string]string, error) {
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
	adszRsp := &ADSZResponse{}
	if err := json.Unmarshal(data, adszRsp); err != nil {
		return nil, err
	}
	return adszRsp.GetMap(), nil
}

// GetHostAddrFromPilot  todo 1. timeout 2. hostAddr change?
func (w *WrappedClient) GetHostAddrFromPilot(serviceKey string) (string, error) {
	for {
		if interfaceHostMap, err := w.getInterfaceHostAddrMapFromPilot(); err != nil {
			return "", err
		} else {
			hostName, ok := interfaceHostMap[serviceKey]
			if !ok {
				logger.Infof("[XDS Wrapped Client] Try getting service %s 's host from istio %d\n", serviceKey, w.istiodHostName)
				time.Sleep(time.Millisecond * 100)
				continue
			}
			return hostName, nil
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
	ipPort := strings.Split(hostAddr, ":")
	hostName := ipPort[0]
	port := ipPort[1]
	// todo cds, eds
	cancel := w.xdsClient.WatchEndpoints(fmt.Sprintf("outbound|%s||%s", port, hostName), func(update resource.EndpointsUpdate, err error) {
		for _, v := range update.Localities {
			for _, e := range v.Endpoints {
				// todo 1. register protocol in server side metadata 2. get metadata from endpoint, for router
				url, _ := common.NewURL(fmt.Sprintf("tri://%s/%s", e.Address, interfaceName))
				logger.Infof("[XDS Registry] Get Update event from pilot: interfaceName = %s, addr = %s, healthy = %d\n",
					interfaceName, e.Address, e.HealthStatus)
				if e.HealthStatus == resource.EndpointHealthStatusHealthy {
					lst.Notify(&registry.ServiceEvent{
						Action:  remoting.EventTypeUpdate,
						Service: url,
					})
				} else {
					lst.Notify(&registry.ServiceEvent{
						Action:  remoting.EventTypeDel,
						Service: url,
					})
				}
			}
		}
	})
	<-stopCh
	cancel()
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

func (w *WrappedClient) initClientAndloadLocalHostAddr() error {
	// call watch and refresh istiod debug interface
	xdsClient, err := newxdsClient(w.localIP, w.podName, w.namespace, w.interfaceAppNameMap2String(), w.istiodHostName)
	if err != nil {
		return err
	}
	stopCh := make(chan struct{})
	foundLocal := false
	foundIstiod := false
	cancel := xdsClient.WatchCluster("*", func(update resource.ClusterUpdate, err error) {
		if update.ClusterName == "" {
			return
		}
		clusterNameList := strings.Split(update.ClusterName, "|")
		// todo: what's going on? istiod can't discover istiod.istio-system.svc.cluster.local!!
		if clusterNameList[3] == w.istiodHostName {
			// 1. find istiod podIP
			// todo: When would eds level watch be cancelled?
			_ = xdsClient.WatchEndpoints(update.ClusterName, func(endpoint resource.EndpointsUpdate, err error) {
				if foundIstiod {
					return
				}
				for _, v := range endpoint.Localities {
					for _, e := range v.Endpoints {
						w.endpointClusterMap.Store(e.Address, update.ClusterName)
						addrs := strings.Split(e.Address, ":")
						w.istiodPodIP = addrs[0]
						foundIstiod = true
						if foundLocal && foundIstiod {
							stopCh <- struct{}{}
						}
					}
				}
			})
			return
		}
		// 2. found local hostAddr
		// todo: When would eds level watch be cancelled?
		_ = xdsClient.WatchEndpoints(update.ClusterName, func(endpoint resource.EndpointsUpdate, err error) {
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
					}
					if foundLocal && foundIstiod {
						stopCh <- struct{}{}
					}
				}
			}
		})
	})
	<-stopCh
	cancel()
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
