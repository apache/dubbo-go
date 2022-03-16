package main

import (
	"dubbo.apache.org/dubbo-go/v3/xds/client"
	"dubbo.apache.org/dubbo-go/v3/xds/client/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource/version"
	"fmt"
	"google.golang.org/grpc/credentials/insecure"

	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"google.golang.org/grpc"
)

import (
	_ "dubbo.apache.org/dubbo-go/v3/xds/client/controller/version/v2"
	_ "dubbo.apache.org/dubbo-go/v3/xds/client/controller/version/v3"
)


const(
	gRPCUserAgentName               = "gRPC Go"
	clientFeatureNoOverprovisioning = "envoy.lb.does_not_support_overprovisioning"
)

func main(){
	v3NodeProto := &v3corepb.Node{
		Id:                   "sidecar~10.195.0.134~random~local",
		//Metadata:             metadata,
		//BuildVersion:         gRPCVersion,
		UserAgentName:        gRPCUserAgentName,
		UserAgentVersionType: &v3corepb.Node_UserAgentVersion{UserAgentVersion: "1.45.0"},
		ClientFeatures:       []string{clientFeatureNoOverprovisioning},
	}

	nonNilCredsConfigV2 := &bootstrap.Config{
		XDSServer: &bootstrap.ServerConfig{
			ServerURI: "localhost:15010",
			Creds:    grpc.WithTransportCredentials(insecure.NewCredentials()),
			//CredsType: "google_default",
			TransportAPI: version.TransportV3,
			NodeProto:    v3NodeProto,
		},
		ClientDefaultListenerResourceNameTemplate: "%s",
	}


	xdsClient, err := client.NewWithConfig(nonNilCredsConfigV2)
	if err != nil{
		panic(err)
	}

	serviceName := "myService"
	xdsClient.WatchListener(serviceName, func(update resource.ListenerUpdate, err error) {
		fmt.Printf("%+v\n err = %s", update, err)
	})
	select {

	}
}
