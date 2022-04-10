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
	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

import (
	xdsCommon "dubbo.apache.org/dubbo-go/v3/remoting/xds/common"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds/mapping"
	"dubbo.apache.org/dubbo-go/v3/xds/client"
	"dubbo.apache.org/dubbo-go/v3/xds/client/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource/version"
)

// xdsClientFactoryFunction generates new xds client
// when running ut, it's for for ut to replace
var xdsClientFactoryFunction = func(localIP, podName, namespace string, istioAddr xdsCommon.HostAddr) (client.XDSClient, error) {
	// todo fix these ugly magic num
	v3NodeProto := &v3corepb.Node{
		Id:                   "sidecar~" + localIP + "~" + podName + "." + namespace + "~" + namespace + ".svc.cluster.local",
		UserAgentName:        gRPCUserAgentName,
		UserAgentVersionType: &v3corepb.Node_UserAgentVersion{UserAgentVersion: "1.45.0"},
		ClientFeatures:       []string{clientFeatureNoOverprovisioning},
		Metadata:             mapping.GetDubboGoMetadata(""),
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
	return newClient, nil
}
