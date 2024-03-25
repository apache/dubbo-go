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

	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"github.com/dubbogo/gost/log/logger"
	envoy_admin_v3 "github.com/envoyproxy/go-control-plane/envoy/admin/v3"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_stats/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/http_inspector/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/original_dst/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/tls_inspector/v3"
	http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/rbac/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

var _ = &envoy_config_core_v3.Pipe{}
var _ = http_connection_manager_v3.HttpFilter{}

var (
	configLock     sync.Mutex
	envoyClusters  map[string]*envoy_config_cluster_v3.Cluster
	envoyListeners map[string]*envoy_config_listener_v3.Listener
	envoyRoutes    map[string]*envoy_config_route_v3.RouteConfiguration
)

func init() {
	envoyClusters = map[string]*envoy_config_cluster_v3.Cluster{}
	envoyListeners = map[string]*envoy_config_listener_v3.Listener{}
	envoyRoutes = map[string]*envoy_config_route_v3.RouteConfiguration{}
}

// EnvoyConfigDump dump all envoy config
func EnvoyConfigDump() []byte {
	dump := envoy_admin_v3.ConfigDump{}

	c := envoyConfigDumpClusters()
	dump.Configs = append(dump.Configs, c)

	l := envoyConfigDumpListeners()
	dump.Configs = append(dump.Configs, l)

	r := envoyConfigDumpRoutes()
	dump.Configs = append(dump.Configs, r)

	m := jsonpb.Marshaler{
		OrigName: true,
		Indent:   " ",
	}
	res, err := m.MarshalToString(&dump)
	if err != nil {
		logger.Errorf("[config] [envoy config dump] configmanager.EnvoyConfigDump MarshalToString error: %s", err)
	}
	return []byte(res)
}

func envoyConfigDumpClusters() *any.Any {
	clusterDump := envoy_admin_v3.ClustersConfigDump{
		VersionInfo: "",
	}
	for _, c := range envoyClusters {
		// Test marshal, in case of unrecognized Any type
		m := jsonpb.Marshaler{
			OrigName: true,
			Indent:   " ",
		}
		_, err := m.MarshalToString(c)
		if err != nil {
			logger.Errorf("[config] [envoy config dump] configmanager.envoyConfigDumpClusters MarshalToString error: %s, cluster: %+v", err, c)
			continue
		}

		value, err := ptypes.MarshalAny(c)
		if err != nil {
			logger.Errorf("[config] [envoy config dump] configmanager.envoyConfigDumpClusters MarshalAny error: %s", err)
		}
		clusterDump.DynamicActiveClusters = append(clusterDump.DynamicActiveClusters, &envoy_admin_v3.ClustersConfigDump_DynamicCluster{
			Cluster: value,
		})
	}
	result, err := ptypes.MarshalAny(&clusterDump)
	if err != nil {
		logger.Errorf("[config] [envoy config dump] configmanager.envoyConfigDumpClusters MarshalAny error: %s", err)
	}
	return result
}

func envoyConfigDumpListeners() *any.Any {
	listenerDump := envoy_admin_v3.ListenersConfigDump{
		VersionInfo: "",
	}
	for _, l := range envoyListeners {
		// Test marshal, in case of unrecognized Any type
		m := jsonpb.Marshaler{
			OrigName: true,
			Indent:   " ",
		}
		_, err := m.MarshalToString(l)
		if err != nil {
			logger.Errorf("[config] [envoy config dump] configmanager.envoyConfigDumpListeners MarshalToString error: %s, listener: %+v", err, l)
			continue
		}

		value, err := ptypes.MarshalAny(l)
		if err != nil {
			logger.Errorf("[config] [envoy config dump] configmanager.envoyConfigDumpListeners MarshalAny error: %s", err)
		}
		listenerDump.DynamicListeners = append(listenerDump.DynamicListeners, &envoy_admin_v3.ListenersConfigDump_DynamicListener{
			ActiveState: &envoy_admin_v3.ListenersConfigDump_DynamicListenerState{
				Listener: value,
			},
		})
	}
	result, err := ptypes.MarshalAny(&listenerDump)
	if err != nil {
		logger.Errorf("[config] [envoy config dump] configmanager.envoyConfigDumpListeners MarshalAny error: %s", err)
	}
	return result
}

func envoyConfigDumpRoutes() *any.Any {
	routeDump := envoy_admin_v3.RoutesConfigDump{}
	for _, r := range envoyRoutes {
		// Test marshal, in case of unrecognized Any type
		m := jsonpb.Marshaler{
			OrigName: true,
			Indent:   " ",
		}
		_, err := m.MarshalToString(r)
		if err != nil {
			logger.Errorf("[config] [envoy config dump] configmanager.envoyConfigDumpRoutes MarshalToString error: %s, route: %+v", err, r)
			continue
		}

		value, err := ptypes.MarshalAny(r)
		if err != nil {
			logger.Errorf("[config] [envoy config dump] configmanager.envoyConfigDumpRoutes MarshalAny error: %s", err)
		}
		routeDump.DynamicRouteConfigs = append(routeDump.DynamicRouteConfigs, &envoy_admin_v3.RoutesConfigDump_DynamicRouteConfig{
			RouteConfig: value,
		})
	}
	result, err := ptypes.MarshalAny(&routeDump)
	if err != nil {
		logger.Errorf("[config] [envoy config dump] configmanager.envoyConfigDumpRoutes MarshalAny error: %s", err)
	}
	return result
}

// EnvoyConfigUpdateClusters update envoy cluster config
func EnvoyConfigUpdateClusters(clusters []*envoy_config_cluster_v3.Cluster) {
	configLock.Lock()
	defer configLock.Unlock()
	logger.Infof("[config] [envoy config cluster] update")
	for _, c := range clusters {
		envoyClusters[c.GetName()] = c
	}
}

func EnvoyConfigDeleteClusterByName(name string) {
	configLock.Lock()
	defer configLock.Unlock()
	delete(envoyClusters, name)
}

// EnvoyConfigUpdateListeners update envoy listener config
func EnvoyConfigUpdateListeners(listeners []*envoy_config_listener_v3.Listener) {
	configLock.Lock()
	defer configLock.Unlock()
	logger.Infof("[config] [envoy config listener] update")
	for _, l := range listeners {
		// listener maybe contains no name, so we use address instead of it.
		name := l.GetName()
		if name == "" {
			name = utils.ConvertAddress(l.GetAddress()).String()
		}
		envoyListeners[name] = l
	}
}

func EnvoyConfigDeleteListeners(listeners []*envoy_config_listener_v3.Listener) {
	configLock.Lock()
	defer configLock.Unlock()
	for _, l := range listeners {
		name := l.GetName()
		if name == "" {
			name = utils.ConvertAddress(l.GetAddress()).String()
		}
		delete(envoyListeners, name)
	}
}

// EnvoyConfigUpdateRoutes update envoy route config
func EnvoyConfigUpdateRoutes(routes []*envoy_config_route_v3.RouteConfiguration) {
	configLock.Lock()
	defer configLock.Unlock()
	logger.Infof("[config] [envoy config route] update")
	for _, r := range routes {
		envoyRoutes[r.GetName()] = r
	}
}
