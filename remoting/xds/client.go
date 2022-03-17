package xds

import (
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/xds/client"
	"dubbo.apache.org/dubbo-go/v3/xds/client/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource/version"
	"encoding/json"
	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	perrors "github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sync"
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
	localIP             string
	istiodAddr          string
	xdsClient           client.XDSClient
	lock                sync.Mutex
}

func NewXDSWrappedClient(podName, namspace, localIP, istiodAddr string) *WrappedClient {
	return &WrappedClient{
		podName:    podName,
		namespace:  namspace,
		localIP:    localIP,
		istiodAddr: istiodAddr,
	}
}

func (w *WrappedClient) GetInterfaceAppNameMapFromPilot() map[string]string {
	// todo get map from
	// http://istiodAddr:8080/debug/adsz
	return nil
}

func (w *WrappedClient) Subscribe(hostName string, lst registry.NotifyListener) error {
	_, ok := w.subscribeStopChMap.Load(hostName)
	if ok {
		return perrors.Errorf("XDS WrappedClient subscribe hostName failed, subscription already exist.")
	}
	stopCh := make(chan struct{})
	w.subscribeStopChMap.Store(hostName, stopCh)
LOOP:
	for {
		// todo cds, eds
		select {
		case <-stopCh:
			break LOOP
		//case <- eventCh:
		default:
		}

		// todo to invoker

		// todo notify
		lst.Notify(&registry.ServiceEvent{})
	}
	return nil
}

func (w *WrappedClient) UnSubscribe(hostName string) {
	if stopCh, ok := w.subscribeStopChMap.Load(hostName); ok {
		close(stopCh.(chan struct{}))
	}
}

func (w *WrappedClient) interfaceAppNameMap2String() string {
	data, _ := json.Marshal(w.interfaceAppNameMap)
	return string(data)
}

// ChangeInterfaceMap change the map of interfaceName -> appname, if appName is empty, delete this interfaceName
func (w *WrappedClient) ChangeInterfaceMap(interfaceName, appName string) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.xdsClient != nil {
		w.xdsClient.Close()
	}
	if appName == "" {
		delete(w.interfaceAppNameMap, interfaceName)
	} else {
		w.interfaceAppNameMap[interfaceName] = appName
	}

	v3NodeProto := &v3corepb.Node{
		Id:                   "sidecar~" + w.localIP + "~" + w.podName + "." + w.namespace + "~" + w.namespace + ".svc.cluster.local",
		UserAgentName:        gRPCUserAgentName,
		UserAgentVersionType: &v3corepb.Node_UserAgentVersion{UserAgentVersion: "1.45.0"},
		ClientFeatures:       []string{clientFeatureNoOverprovisioning},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"LABELS": {
					Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"DUBBO_GO": {
								Kind: &structpb.Value_StringValue{StringValue: w.interfaceAppNameMap2String()},
							},
						},
					}},
				},
			},
		},
	}

	nonNilCredsConfigV2 := &bootstrap.Config{
		XDSServer: &bootstrap.ServerConfig{
			ServerURI:    w.istiodAddr + "15010",
			Creds:        grpc.WithTransportCredentials(insecure.NewCredentials()),
			TransportAPI: version.TransportV3,
			NodeProto:    v3NodeProto,
		},
		ClientDefaultListenerResourceNameTemplate: "%s",
	}

	xdsClient, err := client.NewWithConfig(nonNilCredsConfigV2)
	if err != nil {
		return err
	}
	w.xdsClient = xdsClient
	return nil
}
