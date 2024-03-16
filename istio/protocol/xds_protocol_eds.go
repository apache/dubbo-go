package protocol

import (
	"sync"

	"dubbo.apache.org/dubbo-go/v3/istio/channel"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"github.com/dubbogo/gost/log/logger"
	envoyendpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	v3discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
)

type EdsProtocol struct {
	xdsClientChannel *channel.XdsClientChannel
	resourcesMap     sync.Map
	stopChan         chan struct{}
	updateChan       chan resources.XdsUpdateEvent
}

func NewEdsProtocol(stopChan chan struct{}, updateChan chan resources.XdsUpdateEvent, xdsClientChannel *channel.XdsClientChannel) (*EdsProtocol, error) {
	edsProtocol := &EdsProtocol{
		xdsClientChannel: xdsClientChannel,
		stopChan:         stopChan,
		updateChan:       updateChan,
	}
	return edsProtocol, nil
}

func (eds *EdsProtocol) GetTypeUrl() string {
	return channel.EnvoyEndpoint
}

func (eds *EdsProtocol) SubscribeResource(resourceNames []string) error {
	return eds.xdsClientChannel.SendWithTypeUrlAndResourceNames(eds.GetTypeUrl(), resourceNames)
}

func (eds *EdsProtocol) ProcessProtocol(resp *v3discovery.DiscoveryResponse, xdsClientChannel *channel.XdsClientChannel) error {
	if resp.GetTypeUrl() != eds.GetTypeUrl() {
		return nil
	}

	xdsClusterEndpoints := make([]resources.XdsClusterEndpoint, 0)

	for _, resource := range resp.GetResources() {
		edsResource := &envoyendpoint.ClusterLoadAssignment{}
		if err := ptypes.UnmarshalAny(resource, edsResource); err != nil {
			logger.Errorf("fail to extract endpoint: %v", err)
			continue
		}
		xdsClusterEndpoint, _ := eds.parseEds(edsResource)
		xdsClusterEndpoints = append(xdsClusterEndpoints, xdsClusterEndpoint)
	}

	// notify update
	updateEvent := resources.XdsUpdateEvent{
		Type:   resources.XdsEventUpdateEDS,
		Object: xdsClusterEndpoints,
	}
	eds.updateChan <- updateEvent

	info := &channel.ResponseInfo{
		VersionInfo:   resp.VersionInfo,
		ResponseNonce: resp.Nonce,
		ResourceNames: eds.xdsClientChannel.ApiStore.Find(channel.EnvoyEndpoint).ResourceNames,
	}
	eds.xdsClientChannel.ApiStore.Store(channel.EnvoyEndpoint, info)
	eds.xdsClientChannel.AckResponse(resp)
	return nil
}

func (eds *EdsProtocol) parseEds(edsResource *envoyendpoint.ClusterLoadAssignment) (resources.XdsClusterEndpoint, error) {
	clusterName := edsResource.ClusterName
	xdsClusterEndpoint := resources.XdsClusterEndpoint{
		Name: clusterName,
	}
	endPoints := make([]resources.XdsEndpoint, 0)
	for _, lbeps := range edsResource.Endpoints {
		for _, ep := range lbeps.LbEndpoints {
			endpoint := resources.XdsEndpoint{}
			endpoint.Address = ep.GetEndpoint().Address.GetSocketAddress().Address
			endpoint.Port = ep.GetEndpoint().Address.GetSocketAddress().GetPortValue()
			endpoint.ClusterName = clusterName
			endPoints = append(endPoints, endpoint)
		}

	}
	xdsClusterEndpoint.Endpoints = endPoints
	return xdsClusterEndpoint, nil
}
