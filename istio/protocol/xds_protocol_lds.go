package protocol

import (
	"dubbo.apache.org/dubbo-go/v3/istio/channel"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"github.com/dubbogo/gost/log/logger"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	v3discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"sync"
)

type LdsProtocol struct {
	xdsClientChannel *channel.XdsClientChannel
	resourcesMap     sync.Map
	stopChan         chan struct{}
	updateChan       chan resources.XdsUpdateEvent
}

func NewLdsProtocol(stopChan chan struct{}, updateChan chan resources.XdsUpdateEvent, xdsClientChannel *channel.XdsClientChannel) (*LdsProtocol, error) {
	ldsProtocol := &LdsProtocol{
		xdsClientChannel: xdsClientChannel,
		stopChan:         stopChan,
		updateChan:       updateChan,
	}
	return ldsProtocol, nil
}

func (lds *LdsProtocol) GetTypeUrl() string {
	return channel.EnvoyListener
}

func (lds *LdsProtocol) SubscribeResource(resourceNames []string) error {
	return lds.xdsClientChannel.SendWithTypeUrlAndResourceNames(lds.GetTypeUrl(), resourceNames)
}

func (lds *LdsProtocol) ProcessProtocol(resp *v3discovery.DiscoveryResponse, xdsClientChannel *channel.XdsClientChannel) error {
	if resp.GetTypeUrl() != lds.GetTypeUrl() {
		return nil
	}
	//logger.Infof("DiscoveryResponse: %s", utils.ConvertJsonString(resp))
	xdsListeners := make([]resources.XdsListener, 0)
	rdsResourceNames := make([]string, 0)
	for _, resource := range resp.GetResources() {
		ldsResource := &listener.Listener{}
		if err := ptypes.UnmarshalAny(resource, ldsResource); err != nil {
			logger.Errorf("fail to extract listener: %v", err)
			continue
		}
		logger.Infof("Listener: %s", utils.ConvertJsonString(ldsResource))
		xdsListener, _ := lds.parseListener(ldsResource)
		if xdsListener.HasRds {
			rdsResourceNames = append(rdsResourceNames, xdsListener.RdsResourceNames...)
		}
		xdsListeners = append(xdsListeners, xdsListener)
	}

	// notify update
	updateEvent := resources.XdsUpdateEvent{
		Type:   resources.XdsEventUpdateLDS,
		Object: xdsListeners,
	}
	lds.updateChan <- updateEvent

	info := &channel.ResponseInfo{
		VersionInfo:   resp.VersionInfo,
		ResponseNonce: resp.Nonce,
		ResourceNames: []string{}, // always
	}
	lds.xdsClientChannel.ApiStore.Store(channel.EnvoyListener, info)
	lds.xdsClientChannel.AckResponse(resp)

	if len(rdsResourceNames) > 0 { // Load EDS
		lds.xdsClientChannel.ApiStore.SetResourceNames(channel.EnvoyRoute, rdsResourceNames)
		req := lds.xdsClientChannel.CreateRdsRequest()
		return lds.xdsClientChannel.Send(req)
	}
	return nil
}

func (lds *LdsProtocol) parseListener(listener *listener.Listener) (resources.XdsListener, error) {

	envoyListener := resources.XdsListener{
		//FilterChains:     make([]resources.XdsFilterChain, 0),
		RdsResourceNames: make([]string, 0),
	}
	envoyListener.Name = listener.Name
	// inbound15006 tls mode and downstream transport socket
	inboundTlsMode := resources.XdsTlsMode{}
	inboundDownstreamTransportSocket := resources.XdsDownstreamTransportSocket{}
	isVirtualInbound := false

	// get 15006 and traffic direction
	if listener.GetAddress() != nil {
		if listener.GetAddress().GetSocketAddress() != nil {
			port := listener.GetAddress().GetSocketAddress().GetPortValue()
			if port == 15006 && listener.Name == "virtualInbound" {
				isVirtualInbound = true
			}
		}
	}
	envoyListener.IsVirtualInbound = isVirtualInbound
	envoyListener.TrafficDirection = listener.GetTrafficDirection().String()

	for _, filterChain := range listener.GetFilterChains() {

		envoyFilterChain := resources.XdsFilterChain{}
		envoyFilterChain.Name = filterChain.Name
		// is inboud filterChain?
		foundInboundFilterChain := false
		foundInboundDownstreamTransportSocket := false

		if filterChain.Name == "virtualInbound" {
			foundInboundFilterChain = true
		}
		// get transport
		envoyFilterChainMatch := resources.XdsFilterChainMatch{}
		envoyDownstreamTransportSocket := resources.XdsDownstreamTransportSocket{}

		// get listener inboundTlsMode
		if filterChain.GetFilterChainMatch() != nil {
			filterChainMatch := filterChain.GetFilterChainMatch()
			if len(filterChainMatch.GetTransportProtocol()) > 0 {
				envoyFilterChainMatch.TransportProtocol = filterChainMatch.GetTransportProtocol()
				if filterChainMatch.GetTransportProtocol() == "tls" && foundInboundFilterChain && isVirtualInbound {
					inboundTlsMode.IsTls = true
				}
				if filterChainMatch.GetTransportProtocol() == "raw_buffer" && foundInboundFilterChain && isVirtualInbound {
					inboundTlsMode.IsRawBuffer = true
				}
			}
		}
		// parse downstream transport socket
		if filterChain.GetTransportSocket() != nil {
			var tlsContext sockets_tls_v3.DownstreamTlsContext
			transportSocket := filterChain.GetTransportSocket()
			typeUrl := transportSocket.GetTypedConfig().GetTypeUrl()
			if typeUrl == "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext" {
				err := transportSocket.GetTypedConfig().UnmarshalTo(&tlsContext)
				if err != nil {
					logger.Errorf("can not parse to downstream tls context")
					continue
				}
				foundInboundDownstreamTransportSocket = true
				envoyDownstreamTransportSocket.RequireClientCertificate = tlsContext.RequireClientCertificate.GetValue()
				matchers := tlsContext.CommonTlsContext.GetCombinedValidationContext().DefaultValidationContext.GetMatchSubjectAltNames()
				for _, matcher := range matchers {
					if len(matcher.GetExact()) > 0 {
						envoyDownstreamTransportSocket.SubjectAltNamesMatch = "exact"
						envoyDownstreamTransportSocket.SubjectAltNamesValue = matcher.GetExact()
					}
					if len(matcher.GetPrefix()) > 0 {
						envoyDownstreamTransportSocket.SubjectAltNamesMatch = "prefix"
						envoyDownstreamTransportSocket.SubjectAltNamesValue = matcher.GetPrefix()
					}
					if len(matcher.GetContains()) > 0 {
						envoyDownstreamTransportSocket.SubjectAltNamesMatch = "contains"
						envoyDownstreamTransportSocket.SubjectAltNamesValue = matcher.GetContains()
					}
				}
			}
		}

		// found inbound TransportSocket
		if isVirtualInbound && foundInboundFilterChain && foundInboundDownstreamTransportSocket {
			inboundDownstreamTransportSocket = envoyDownstreamTransportSocket
		}

		envoyFilterChain.FilterChainMatch = envoyFilterChainMatch
		envoyFilterChain.TransportSocket = envoyDownstreamTransportSocket
		envoyFilterChain.Filters = make([]resources.XdsFilter, 0)
		// Get Rds and filters
		for _, filter := range filterChain.GetFilters() {
			envoyFilter := resources.XdsFilter{}
			envoyFilter.Name = filter.Name
			if filter.GetTypedConfig().GetTypeUrl() == "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager" {
				// check is rds
				httpConnectionManager := &http_connection_manager_v3.HttpConnectionManager{}
				if err := filter.GetTypedConfig().UnmarshalTo(httpConnectionManager); err != nil {
					logger.Errorf("can not parse to Http Connection Manager")
					continue
				}
				envoyFilter.IncludeHttpConnectionManager = true
				if httpConnectionManager.GetRds() != nil {
					envoyFilter.HasRds = true
					envoyFilter.RdsResourceName = httpConnectionManager.GetRds().RouteConfigName
					envoyListener.HasRds = true
					envoyListener.RdsResourceNames = append(envoyListener.RdsResourceNames, httpConnectionManager.GetRds().RouteConfigName)
				}
			}
			envoyFilterChain.Filters = append(envoyFilterChain.Filters, envoyFilter)
		}

		//envoyListener.FilterChains = append(envoyListener.FilterChains, envoyFilterChain)

		// Set envoy listener
		envoyListener.InboundTlsMode = inboundTlsMode
		envoyListener.InboundDownstreamTransportSocket = inboundDownstreamTransportSocket
	}

	return envoyListener, nil
}
