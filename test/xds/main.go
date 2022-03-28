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

package main

import (
	"fmt"
)

import (
	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

	structpb "github.com/golang/protobuf/ptypes/struct"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

import (
	"dubbo.apache.org/dubbo-go/v3/xds/client"
	"dubbo.apache.org/dubbo-go/v3/xds/client/bootstrap"
	_ "dubbo.apache.org/dubbo-go/v3/xds/client/controller/version/v2"
	_ "dubbo.apache.org/dubbo-go/v3/xds/client/controller/version/v3"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource/version"
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
				"CLUSTER_ID": {
					Kind: &structpb.Value_StringValue{StringValue: "Kubernetes"},
				},
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

	//clusterName := "outbound|20000||dubbo-go-app.default.svc.cluster.local" //
	//clusterName := "outbound|8848||nacos.default.svc.cluster.local"
	//endpointClusterMap := sync.Map{}
	//xdsClient.WatchCluster("*", func(update resource.ClusterUpdate, err error) {
	//	xdsClient.WatchEndpoints(update.ClusterName, func(endpoint resource.EndpointsUpdate, err error) {
	//		for _, v := range endpoint.Localities {
	//			for _, e := range v.Endpoints {
	//				endpointClusterMap.Store(e.Address, update.ClusterName)
	//			}
	//		}
	//	})
	//})

	//
	//xdsClient.WatchEndpoints("outbound|15010||istiod.istio-system.svc.cluster.local", func(update resource.EndpointsUpdate, err error) {
	//	fmt.Printf("%+v\n err = %s", update, err)
	//})

	xdsClient.WatchCluster("*", func(update resource.ClusterUpdate, err error) {
		fmt.Println(update)
	})

	//xdsClient.WatchCluster("*", func(update resource.ClusterUpdate, err error) {
	//	fmt.Printf("%+v\n err = %s", update, err)
	//
	//})
	//

	select {}
}
