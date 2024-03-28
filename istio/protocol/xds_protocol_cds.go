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

package protocol

import (
	"sync"

	"dubbo.apache.org/dubbo-go/v3/istio/channel"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"github.com/dubbogo/gost/log/logger"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	v3discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
)

type CdsProtocol struct {
	xdsClientChannel *channel.XdsClientChannel
	resourcesMap     sync.Map
	stopChan         chan struct{}
	updateChan       chan resources.XdsUpdateEvent
}

func NewCdsProtocol(stopChan chan struct{}, updateChan chan resources.XdsUpdateEvent, xdsClientChannel *channel.XdsClientChannel) (*CdsProtocol, error) {
	edsProtocol := &CdsProtocol{
		xdsClientChannel: xdsClientChannel,
		stopChan:         stopChan,
		updateChan:       updateChan,
	}
	return edsProtocol, nil
}

func (cds *CdsProtocol) GetTypeUrl() string {
	return channel.EnvoyCluster
}

func (cds *CdsProtocol) SubscribeResource(resourceNames []string) error {
	return cds.xdsClientChannel.SendWithTypeUrlAndResourceNames(cds.GetTypeUrl(), resourceNames)
}

func (cds *CdsProtocol) ProcessProtocol(resp *v3discovery.DiscoveryResponse, xdsClientChannel *channel.XdsClientChannel) error {

	if resp.GetTypeUrl() != cds.GetTypeUrl() {
		return nil
	}

	xdsClusters := make([]resources.XdsCluster, 0)
	clusterNames := make([]string, 0)
	for _, resource := range resp.GetResources() {
		cdsResource := &cluster.Cluster{}
		if err := ptypes.UnmarshalAny(resource, cdsResource); err != nil {
			logger.Errorf("[Xds Protocol] fail to extract endpoint: %v", err)
			continue
		}
		clusterName := cdsResource.Name
		if cdsResource.GetType() == cluster.Cluster_EDS {
			clusterNames = append(clusterNames, clusterName)
			// TODO Add more info about tls, lb
			xdsCluster, _ := cds.parseCluster(cdsResource)
			// Only get EDS cluster type
			xdsClusters = append(xdsClusters, xdsCluster)
		}
	}

	// notify update
	updateEvent := resources.XdsUpdateEvent{
		Type:   resources.XdsEventUpdateCDS,
		Object: xdsClusters,
	}
	cds.updateChan <- updateEvent

	info := &channel.ResponseInfo{
		VersionInfo:   resp.VersionInfo,
		ResponseNonce: resp.Nonce,
		ResourceNames: []string{}, // CDS ResourcesNames keeps empty,
	}
	cds.xdsClientChannel.ApiStore.Store(channel.EnvoyCluster, info)
	cds.xdsClientChannel.AckResponse(resp)

	if len(clusterNames) > 0 { // Load EDS
		cds.xdsClientChannel.ApiStore.SetResourceNames(channel.EnvoyEndpoint, clusterNames)
		req := cds.xdsClientChannel.CreateEdsRequest()
		return cds.xdsClientChannel.Send(req)
	}
	return nil
}

func (cds *CdsProtocol) parseCluster(cluster *cluster.Cluster) (resources.XdsCluster, error) {
	clusterName := cluster.Name
	xdsCluster := resources.XdsCluster{
		Name: clusterName,
		Type: cluster.GetType().String(),
	}
	// Parse Tls
	clusterUpstreamTransportSocket := resources.XdsUpstreamTransportSocket{}
	clusterTlsMode := resources.XdsTLSMode{}
	if cluster.GetTransportSocketMatches() == nil {
		clusterTlsMode.IsTls = false
		clusterTlsMode.IsRawBuffer = true
	} else {
		for _, transportSocketMatch := range cluster.GetTransportSocketMatches() {
			var tlsContext sockets_tls_v3.UpstreamTlsContext
			typeUrl := transportSocketMatch.GetTransportSocket().GetTypedConfig().GetTypeUrl()
			if typeUrl == "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext" {
				err := transportSocketMatch.GetTransportSocket().GetTypedConfig().UnmarshalTo(&tlsContext)
				if err != nil {
					logger.Errorf("[Xds Protocol] can not parse to upstream tls context")
					continue
				}
				clusterTlsMode.IsTls = true
				matchers := tlsContext.CommonTlsContext.GetCombinedValidationContext().DefaultValidationContext.GetMatchSubjectAltNames()
				for _, matcher := range matchers {
					if len(matcher.GetExact()) > 0 {
						clusterUpstreamTransportSocket.SubjectAltNamesMatch = "exact"
						clusterUpstreamTransportSocket.SubjectAltNamesValue = matcher.GetExact()
					}
					if len(matcher.GetPrefix()) > 0 {
						clusterUpstreamTransportSocket.SubjectAltNamesMatch = "prefix"
						clusterUpstreamTransportSocket.SubjectAltNamesValue = matcher.GetPrefix()
					}
					if len(matcher.GetContains()) > 0 {
						clusterUpstreamTransportSocket.SubjectAltNamesMatch = "contains"
						clusterUpstreamTransportSocket.SubjectAltNamesValue = matcher.GetContains()
					}
				}
			}
			if typeUrl == "type.googleapis.com/envoy.extensions.transport_sockets.raw_buffer.v3.RawBuffer" {
				clusterTlsMode.IsRawBuffer = true
			}
		}
	}
	xdsCluster.TransportSocket = clusterUpstreamTransportSocket
	xdsCluster.TlsMode = clusterTlsMode
	return xdsCluster, nil
}
