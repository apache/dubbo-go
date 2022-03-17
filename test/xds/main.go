package main

import (
	"dubbo.apache.org/dubbo-go/v3/xds/client"
	"dubbo.apache.org/dubbo-go/v3/xds/client/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource/version"
	"fmt"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc/credentials/insecure"

	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"google.golang.org/grpc"
)

import (
	_ "dubbo.apache.org/dubbo-go/v3/xds/client/controller/version/v2"
	_ "dubbo.apache.org/dubbo-go/v3/xds/client/controller/version/v3"
)

const (
	gRPCUserAgentName               = "gRPC Go"
	clientFeatureNoOverprovisioning = "envoy.lb.does_not_support_overprovisioning"
)

// ATTENTION! export GRPC_XDS_EXPERIMENTAL_SECURITY_SUPPORT=false
func main() {
	v3NodeProto := &v3corepb.Node{
		Id:                   "sidecar~172.1.1.1~sleep-55b5877479-rwcct.default~default.svc.cluster.local",
		UserAgentName:        gRPCUserAgentName,
		Cluster:              "testCluster",
		UserAgentVersionType: &v3corepb.Node_UserAgentVersion{UserAgentVersion: "1.45.0"},
		ClientFeatures:       []string{clientFeatureNoOverprovisioning},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"LABELS": {
					Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"label1": {
								Kind: &structpb.Value_StringValue{StringValue: "val1"},
							},
							"label2": {
								Kind: &structpb.Value_StringValue{StringValue: "val2"},
							},
						},
					}},
				},
			},
		},
	}

	nonNilCredsConfigV2 := &bootstrap.Config{
		XDSServer: &bootstrap.ServerConfig{
			ServerURI: "localhost:15010",
			Creds:     grpc.WithTransportCredentials(insecure.NewCredentials()),
			//CredsType: "google_default",
			TransportAPI: version.TransportV3,
			NodeProto:    v3NodeProto,
		},
		ClientDefaultListenerResourceNameTemplate: "%s",
	}

	xdsClient, err := client.NewWithConfig(nonNilCredsConfigV2)
	if err != nil {
		panic(err)
	}

	clusterName := "outbound|20000||dubbo-go-app.default.svc.cluster.local" //
	//clusterName :=  "outbound|27||laurence-svc.default.svc.cluster.local"
	xdsClient.WatchListener("", func(update resource.ListenerUpdate, err error) {
		fmt.Printf("%+v\n err = %s", update, err)
	})

	xdsClient.WatchCluster(clusterName, func(update resource.ClusterUpdate, err error) {
		fmt.Printf("%+v\n err = %s", update, err)
	})

	xdsClient.WatchEndpoints(clusterName, func(update resource.EndpointsUpdate, err error) {
		fmt.Printf("%+v\n err = %s", update, err)
	})
	select {}
}
