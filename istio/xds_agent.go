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

package istio

import (
	"crypto/tls"
	"crypto/x509"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
)

type PilotAgentType int32

const (
	PilotAgentTypeServerWorkload PilotAgentType = iota
	PilotAgentTypeClientWorkload
)

type OnRdsChangeListener func(serviceName string, xdsVirtualHost resources.XdsVirtualHost) error
type OnEdsChangeListener func(clusterName string, xdsCluster resources.XdsCluster, xdsClusterEndpoint resources.XdsClusterEndpoint) error

type XdsAgent interface {
	Run(pilotAgentType PilotAgentType) error
	GetWorkloadCertificateProvider() WorkloadCertificateProvider
	SubscribeRds(serviceName, listenerName string, listener OnRdsChangeListener)
	UnsubscribeRds(serviceName, listenerName string)
	SubscribeCds(clusterName, listenerName string, listener OnEdsChangeListener)
	UnsubscribeCds(clusterName, listenerName string)
	GetHostInboundListener() *resources.XdsHostInboundListener
	GetHostInboundMutualTLSMode() resources.MutualTLSMode
	GetHostInboundJwtAuthentication() *resources.JwtAuthentication
	GetHostInboundRBAC() *rbacv3.RBAC
	Stop()
}

type WorkloadCertificateProvider interface {
	GetWorkloadCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
	GetCACertPool() (*x509.CertPool, error)
}
