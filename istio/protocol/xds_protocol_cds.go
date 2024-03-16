package protocol

import (
	"dubbo.apache.org/dubbo-go/v3/istio/channel"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"github.com/dubbogo/gost/log/logger"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	v3discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"sync"
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

	xdsClusters := make([]resources.EnvoyCluster, 0)
	clusterNames := make([]string, 0)
	for _, resource := range resp.GetResources() {
		cdsResource := &cluster.Cluster{}
		if err := ptypes.UnmarshalAny(resource, cdsResource); err != nil {
			logger.Errorf("fail to extract endpoint: %v", err)
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

func (cds *CdsProtocol) parseCluster(cluster *cluster.Cluster) (resources.EnvoyCluster, error) {
	clusterName := cluster.Name
	xdsCluster := resources.EnvoyCluster{
		Name: clusterName,
		Type: cluster.GetType().String(),
	}
	// Parse Tls
	clusterTlsMode := resources.EnvoyClusterTlsMode{}
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
					logger.Errorf("can not parse to upstream tls context")
					continue
				}
				clusterTlsMode.IsTls = true
				matchers := tlsContext.CommonTlsContext.GetCombinedValidationContext().DefaultValidationContext.GetMatchSubjectAltNames()
				for _, matcher := range matchers {
					if len(matcher.GetExact()) > 0 {
						clusterTlsMode.SubjectAltNamesMatch = "exact"
						clusterTlsMode.SubjectAltNamesValue = matcher.GetExact()
					}
					if len(matcher.GetPrefix()) > 0 {
						clusterTlsMode.SubjectAltNamesMatch = "prefix"
						clusterTlsMode.SubjectAltNamesValue = matcher.GetPrefix()
					}
					if len(matcher.GetContains()) > 0 {
						clusterTlsMode.SubjectAltNamesMatch = "contains"
						clusterTlsMode.SubjectAltNamesValue = matcher.GetContains()
					}
				}
			}
			if typeUrl == "type.googleapis.com/envoy.extensions.transport_sockets.raw_buffer.v3.RawBuffer" {
				clusterTlsMode.IsRawBuffer = true
			}
		}
	}
	xdsCluster.TlsMode = clusterTlsMode
	return xdsCluster, nil
}
