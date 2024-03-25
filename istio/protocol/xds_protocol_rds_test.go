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
	"testing"

	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
)

func TestParseRoute(t *testing.T) {
	tests := []struct {
		name        string
		routeConfig *route.RouteConfiguration
		parsedRoute resources.XdsRouteConfig
	}{
		{
			name: "Test ParseRoute with valid route configuration",
			routeConfig: &route.RouteConfiguration{
				Name: "test-route",
				VirtualHosts: []*route.VirtualHost{
					{
						Name:    "test-host",
						Domains: []string{"example.com"},
						Routes: []*route.Route{
							{
								Name: "test-route",
								Match: &route.RouteMatch{
									PathSpecifier: &route.RouteMatch_Prefix{Prefix: "/test"},
								},
								Action: &route.Route_Route{
									Route: &route.RouteAction{
										ClusterSpecifier: &route.RouteAction_Cluster{Cluster: "test-cluster"},
									},
								},
							},
						},
					},
				},
			},
			parsedRoute: resources.XdsRouteConfig{
				Name: "test-route",
				VirtualHosts: map[string]resources.XdsVirtualHost{
					"test-host": {
						Name:    "test-host",
						Domains: []string{"example.com"},
						Routes: []resources.XdsRoute{
							{
								Name: "test-route",
								Match: resources.XdsRouteMatch{
									Prefix: "/test",
								},
								Action: resources.XdsRouteAction{
									Cluster:        "test-cluster",
									ClusterWeights: []resources.XdsClusterWeight{},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Test ParseRoute with route configuration containing multiple cluster weights",
			routeConfig: &route.RouteConfiguration{
				Name: "test-route-multiple-clusters",
				VirtualHosts: []*route.VirtualHost{
					{
						Name:    "test-host",
						Domains: []string{"example.com"},
						Routes: []*route.Route{
							{
								Name: "test-route-1",
								Match: &route.RouteMatch{
									PathSpecifier: &route.RouteMatch_Prefix{Prefix: "/test-1"},
								},
								Action: &route.Route_Route{
									Route: &route.RouteAction{
										ClusterSpecifier: &route.RouteAction_WeightedClusters{
											WeightedClusters: &route.WeightedCluster{
												Clusters: []*route.WeightedCluster_ClusterWeight{
													{
														Name:   "cluster-1",
														Weight: &wrappers.UInt32Value{Value: 70},
													},
													{
														Name:   "cluster-2",
														Weight: &wrappers.UInt32Value{Value: 30},
													},
												},
											},
										},
									},
								},
							},
							{
								Name: "test-route-2",
								Match: &route.RouteMatch{
									PathSpecifier: &route.RouteMatch_Prefix{Prefix: "/test-2"},
								},
								Action: &route.Route_Route{
									Route: &route.RouteAction{
										ClusterSpecifier: &route.RouteAction_Cluster{Cluster: "cluster-3"},
									},
								},
							},
						},
					},
				},
			},
			parsedRoute: resources.XdsRouteConfig{
				Name: "test-route-multiple-clusters",
				VirtualHosts: map[string]resources.XdsVirtualHost{
					"test-host": {
						Name:    "test-host",
						Domains: []string{"example.com"},
						Routes: []resources.XdsRoute{
							{
								Name: "test-route-1",
								Match: resources.XdsRouteMatch{
									Prefix: "/test-1",
								},
								Action: resources.XdsRouteAction{
									ClusterWeights: []resources.XdsClusterWeight{
										{
											Name:   "cluster-1",
											Weight: 70,
										},
										{
											Name:   "cluster-2",
											Weight: 30,
										},
									},
								},
							},
							{
								Name: "test-route-2",
								Match: resources.XdsRouteMatch{
									Prefix: "/test-2",
								},
								Action: resources.XdsRouteAction{
									Cluster:        "cluster-3",
									ClusterWeights: []resources.XdsClusterWeight{},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new RdsProtocol instance
			rdsProtocol := &RdsProtocol{}

			// Call parseRoute with mock route configuration
			parsedRoute := rdsProtocol.parseRoute(tt.routeConfig)

			// Verify that the parsed route configuration matches the expected result
			assert.Equal(t, tt.parsedRoute, parsedRoute)
		})
	}
}
