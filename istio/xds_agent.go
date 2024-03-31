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

// PilotAgentType represents the type of Pilot agent, either server workload or client workload.
type PilotAgentType int32

const (
	PilotAgentTypeServerWorkload PilotAgentType = iota
	PilotAgentTypeClientWorkload
)

// OnRdsChangeListener defines the signature for RDS change listeners.
type OnRdsChangeListener func(serviceName string, xdsVirtualHost resources.XdsVirtualHost) error

// OnEdsChangeListener defines the signature for EDS change listeners.
type OnEdsChangeListener func(clusterName string, xdsCluster resources.XdsCluster, xdsClusterEndpoint resources.XdsClusterEndpoint) error

// XdsAgent is the interface for managing xDS agents.
type XdsAgent interface {
	// Run starts the xDS agent with the specified Pilot agent type.
	Run(pilotAgentType PilotAgentType) error

	// GetWorkloadCertificateProvider returns the provider for workload certificates.
	GetWorkloadCertificateProvider() WorkloadCertificateProvider

	// SubscribeRds subscribes to changes in the Route Discovery Service (RDS) for the given service name.
	SubscribeRds(serviceName, listenerName string, listener OnRdsChangeListener)

	// UnsubscribeRds unsubscribes the listener from changes in the Route Discovery Service (RDS) for the given service name.
	UnsubscribeRds(serviceName, listenerName string)

	// SubscribeCds subscribes to changes in the Endpoint Discovery Service (EDS) for the given cluster name.
	SubscribeCds(clusterName, listenerName string, listener OnEdsChangeListener)

	// UnsubscribeCds unsubscribes the listener from changes in the Endpoint Discovery Service (EDS) for the given cluster name.
	UnsubscribeCds(clusterName, listenerName string)

	// GetHostInboundListener returns the configuration for the host inbound listener.
	GetHostInboundListener() *resources.XdsHostInboundListener

	// GetHostInboundMutualTLSMode returns the mutual TLS mode for host inbound connections.
	GetHostInboundMutualTLSMode() resources.MutualTLSMode

	// GetHostInboundJwtAuthentication returns the JWT authentication configuration for host inbound connections.
	GetHostInboundJwtAuthentication() *resources.JwtAuthentication

	// GetHostInboundRBAC returns the RBAC (Role-Based Access Control) configuration for host inbound connections.
	GetHostInboundRBAC() *rbacv3.RBAC

	// Stop stops the xDS agent.
	Stop()
}

// WorkloadCertificateProvider is the interface for providing workload certificates.
type WorkloadCertificateProvider interface {
	// GetWorkloadCertificate returns the TLS certificate for the given ClientHelloInfo.
	// This method is responsible for providing certificates based on the information
	// available in the ClientHelloInfo, such as SNI (Server Name Indication).
	GetWorkloadCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)

	// GetCACertPool returns the root CA certificate pool.
	// This method is responsible for providing the root CA certificate pool
	// which is used for verifying the authenticity of certificates presented
	// by peers during mutual TLS handshake.
	GetCACertPool() (*x509.CertPool, error)
}
