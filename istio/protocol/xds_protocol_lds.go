package protocol

import (
	"dubbo.apache.org/dubbo-go/v3/istio/channel"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"github.com/dubbogo/gost/log/logger"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
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
	xdsListeners := make([]resources.EnvoyListener, 0)
	rdsResourceNames := make([]string, 0)
	for _, resource := range resp.GetResources() {
		ldsResource := &listener.Listener{}
		if err := ptypes.UnmarshalAny(resource, ldsResource); err != nil {
			logger.Errorf("fail to extract listener: %v", err)
			continue
		}
		xdsListener, _ := lds.parseListener(ldsResource)
		if xdsListener.IsRds {
			rdsResourceNames = append(rdsResourceNames, xdsListener.RdsResourceName)
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

func (lds *LdsProtocol) parseListener(listener *listener.Listener) (resources.EnvoyListener, error) {
	envoyListener := resources.EnvoyListener{}
	envoyListener.Name = listener.Name
	for _, filterChain := range listener.GetFilterChains() {
		for _, filter := range filterChain.GetFilters() {
			if filter.GetTypedConfig().GetTypeUrl() == "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager" {
				// check is rds
				httpConnectionManager := &http_connection_manager_v3.HttpConnectionManager{}
				if err := filter.GetTypedConfig().UnmarshalTo(httpConnectionManager); err != nil {
					logger.Errorf("can not parse to Http Connection Manager")
					continue
				}
				if httpConnectionManager.GetRds() != nil {
					envoyListener.IsRds = true
					envoyListener.RdsResourceName = httpConnectionManager.GetRds().RouteConfigName
				}
			}
		}
	}
	return envoyListener, nil
}
