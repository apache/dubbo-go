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

package resources

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type XdsCluster struct {
	Type            string
	Name            string
	LbPolicy        string
	Invokers        []protocol.Invoker
	Service         XdsClusterService
	TransportSocket XdsUpstreamTransportSocket
	TlsMode         XdsTLSMode
}

type XdsClusterEndpoint struct {
	Name      string
	Endpoints []XdsEndpoint
}

type XdsEndpoint struct {
	ClusterName string
	Protocol    string
	Address     string
	Port        uint32
	Healthy     bool
	Weight      int
}

// filter from metadata
type XdsClusterService struct {
	Name      string
	Namespace string
	Host      string
}

type XdsUpstreamTransportSocket struct {
	SubjectAltNamesMatch string // exact, prefix,  contains
	SubjectAltNamesValue string
	//tlsContext           sockets_tls_v3.UpstreamTlsContext
}
