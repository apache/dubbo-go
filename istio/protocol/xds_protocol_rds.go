package protocol

import (
	"dubbo.apache.org/dubbo-go/v3/istio/channel"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"github.com/dubbogo/gost/log/logger"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	v3discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"sync"
)

type RdsProtocol struct {
	xdsClientChannel *channel.XdsClientChannel
	resourcesMap     sync.Map
	stopChan         chan struct{}
	updateChan       chan resources.XdsUpdateEvent
}

func NewRdsProtocol(stopChan chan struct{}, updateChan chan resources.XdsUpdateEvent, xdsClientChannel *channel.XdsClientChannel) (*RdsProtocol, error) {
	edsProtocol := &RdsProtocol{
		xdsClientChannel: xdsClientChannel,
		stopChan:         stopChan,
		updateChan:       updateChan,
	}
	return edsProtocol, nil
}

func (rds *RdsProtocol) GetTypeUrl() string {
	return channel.EnvoyRoute
}

func (rds *RdsProtocol) SubscribeResource(resourceNames []string) error {
	return rds.xdsClientChannel.SendWithTypeUrlAndResourceNames(rds.GetTypeUrl(), resourceNames)
}

func (rds *RdsProtocol) ProcessProtocol(resp *v3discovery.DiscoveryResponse, xdsClientChannel *channel.XdsClientChannel) error {
	if resp.GetTypeUrl() != rds.GetTypeUrl() {
		return nil
	}

	xdsRouteConfigurations := make([]resources.EnvoyRouteConfig, 0)
	resourceNames := make([]string, 0)

	for _, resource := range resp.GetResources() {
		rdsResource := &route.RouteConfiguration{}
		if err := ptypes.UnmarshalAny(resource, rdsResource); err != nil {
			logger.Errorf("fail to extract route configuration: %v", err)
			continue
		}
		xdsRouteConfiguration := rds.parseRoute(rdsResource)
		resourceNames = append(resourceNames, xdsRouteConfiguration.Name)
		xdsRouteConfigurations = append(xdsRouteConfigurations, xdsRouteConfiguration)
	}

	// notify update
	updateEvent := resources.XdsUpdateEvent{
		Type:   resources.XdsEventUpdateRDS,
		Object: xdsRouteConfigurations,
	}
	rds.updateChan <- updateEvent

	info := &channel.ResponseInfo{
		VersionInfo:   resp.VersionInfo,
		ResponseNonce: resp.Nonce,
		ResourceNames: resourceNames,
	}
	rds.xdsClientChannel.ApiStore.Store(channel.EnvoyRoute, info)
	rds.xdsClientChannel.AckResponse(resp)

	return nil
}

func (rds *RdsProtocol) parseRoute(route *route.RouteConfiguration) resources.EnvoyRouteConfig {
	envoyRouteConfig := resources.EnvoyRouteConfig{
		Name:         route.Name,
		VirtualHosts: make(map[string]resources.EnvoyVirtualHost, 0),
	}
	envoyRouteConfig.Name = route.Name
	for _, vh := range route.GetVirtualHosts() {
		// virtual host
		envoyVirtualHost := resources.EnvoyVirtualHost{
			Name:    vh.Name,
			Domains: make([]string, 0),
			Routes:  make([]resources.EnvoyRoute, 0),
		}
		// domains
		for _, domain := range vh.GetDomains() {
			envoyVirtualHost.Domains = append(envoyVirtualHost.Domains, domain)
		}
		// routes
		for _, vhRoute := range vh.GetRoutes() {
			envoyRoute := resources.EnvoyRoute{}
			envoyRoute.Name = vhRoute.Name
			// route match
			envoyRouteMatch := resources.EnvoyRouteMatch{}
			if len(vhRoute.Match.GetPath()) > 0 {
				envoyRouteMatch.Path = vhRoute.Match.GetPath()
			}
			if len(vhRoute.Match.GetPrefix()) > 0 {
				envoyRouteMatch.Prefix = vhRoute.Match.GetPrefix()
			}
			if vhRoute.Match.GetCaseSensitive() != nil {
				envoyRouteMatch.CaseSensitive = vhRoute.Match.GetCaseSensitive().Value
			}
			if len(vhRoute.Match.GetSafeRegex().GetRegex()) > 0 {
				envoyRouteMatch.Regex = vhRoute.Match.GetSafeRegex().GetRegex()
			}
			envoyRoute.Match = envoyRouteMatch
			// route
			envoyRouteAction := resources.EnvoyRouteAction{
				Cluster:        vhRoute.GetRoute().GetCluster(),
				ClusterWeights: make([]resources.EnvoyClusterWeight, 0),
			}
			envoyRouteAction.Cluster = vhRoute.GetRoute().GetCluster()
			if vhRoute.GetRoute().GetWeightedClusters() != nil {
				for _, clusterWeight := range vhRoute.GetRoute().GetWeightedClusters().GetClusters() {
					envoyClusterWeight := resources.EnvoyClusterWeight{}
					envoyClusterWeight.Name = clusterWeight.Name
					envoyClusterWeight.Weight = clusterWeight.Weight.Value
					envoyRouteAction.ClusterWeights = append(envoyRouteAction.ClusterWeights, envoyClusterWeight)
				}
			}
			envoyRoute.Match = envoyRouteMatch
			envoyRoute.Action = envoyRouteAction
			envoyVirtualHost.Routes = append(envoyVirtualHost.Routes, envoyRoute)

		}
		envoyRouteConfig.VirtualHosts[envoyVirtualHost.Name] = envoyVirtualHost
		// parse more info here like ratelimit, retry policy
	}
	return envoyRouteConfig
}
