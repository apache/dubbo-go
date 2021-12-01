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

package v3router

import (
	"io"
	"strings"
)

import (
	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// RouterChain contains all uniform router logic
// it has UniformRouter list,
type RouterChain struct {
	routers []*UniformRouter
	notify  chan struct{}
}

// nolint
func NewUniformRouterChain() (router.PriorityRouter, error) {
	// 1. Add mesh route listener
	r := &RouterChain{}
	rootConfig := config.GetRootConfig()
	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		logger.Infof("[NewUniformRouterChain] Config center does not start, please check if the configuration center has been properly configured in dubbogo.yml")
		return nil, nil
	}

	// 2. Try to get mesh rules configuration, default key is "dubbo.io.MESHAPPRULE" with group "dubbo"
	key := rootConfig.Application.Name + constant.MeshRouteSuffix
	group := rootConfig.ConfigCenter.Group
	if group == "" {
		group = constant.Dubbo
	}
	dynamicConfiguration.AddListener(group+constant.PathSeparator+key, r)
	meshRouteValue, err := dynamicConfiguration.GetProperties(key, config_center.WithGroup(rootConfig.ConfigCenter.Group))
	if err != nil {
		// The mesh rules may not be initialized now
		logger.Warnf("[NewUniformRouterChain]Can not get mesh rules for group=%s, key=%s, error=%+v", rootConfig.ConfigCenter.Group, key, err)
		return r, nil
	}
	logger.Debugf("[NewUniformRouterChain]Successfully get mesh rules:%s", meshRouteValue)
	routes, err := parseRoute(meshRouteValue)
	if err != nil {
		logger.Warnf("[NewUniformRouterChain]Parse mesh rules failed, error=%+v", err)
		return nil, err
	}
	r.routers = routes
	logger.Infof("[NewUniformRouterChain]Successfully init mesh rules with:\n%s", meshRouteValue)
	return r, nil
}

// Route route invokers using RouterChain's routers one by one
func (r *RouterChain) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	for _, v := range r.routers {
		invokers = v.Route(invokers, url, invocation)
	}
	return invokers
}

// Process process route config change event
func (r *RouterChain) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("[RouteChain]Process config change event:%+v", event)
	if event.ConfigType == remoting.EventTypeDel {
		r.routers = nil
		return
	}
	routers, err := parseRoute(event.Value.(string))
	if err != nil {
		logger.Warnf("[RouteChain]Parse new mesh route config error, %+v "+
			"and we will use the original mesh rule configuration.", err)
		return
	}
	r.routers = routers
	logger.Infof("[RouteChain]Parse Mesh Rule Success.")
}

// Name get name of ConnCheckerRouter
func (r *RouterChain) Name() string {
	return name
}

// Priority get Router priority level
func (r *RouterChain) Priority() int64 {
	return 0
}

// URL Return URL in router
func (r *RouterChain) URL() *common.URL {
	return nil
}

// Deprecated parseFromConfigToRouters parse virtualService and destinationRule yaml file bytes to target router list
func parseFromConfigToRouters(virtualServiceConfig, destinationRuleConfig []byte) ([]*UniformRouter, error) {
	var virtualServiceConfigList []*config.VirtualServiceConfig
	destRuleConfigsMap := make(map[string]map[string]map[string]string)

	vsDecoder := yaml.NewDecoder(strings.NewReader(string(virtualServiceConfig)))
	drDecoder := yaml.NewDecoder(strings.NewReader(string(destinationRuleConfig)))
	// 1. parse virtual service config
	for {
		virtualServiceCfg := &config.VirtualServiceConfig{}

		err := vsDecoder.Decode(virtualServiceCfg)
		if err == io.EOF {
			break
		}

		if err != nil {
			logger.Error("parseFromConfigTo virtual service err = ", err)
			return nil, err
		}
		virtualServiceConfigList = append(virtualServiceConfigList, virtualServiceCfg)
	}

	// 2. parse destination rule config
	for {
		destRuleCfg := &config.DestinationRuleConfig{}
		err := drDecoder.Decode(destRuleCfg)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("parseFromConfigTo destination rule err = ", err)
			return nil, err
		}

		// name -> labels
		destRuleCfgMap := make(map[string]map[string]string)
		for _, v := range destRuleCfg.Spec.SubSets {
			destRuleCfgMap[v.Name] = v.Labels
		}

		// host -> name -> labels
		destRuleConfigsMap[destRuleCfg.Spec.Host] = destRuleCfgMap
	}

	routers := make([]*UniformRouter, 0)

	// 3. construct virtual service host to destination mapping
	for _, v := range virtualServiceConfigList {
		tempServiceNeedsDescMap := make(map[string]map[string]string)
		for _, host := range v.Spec.Hosts {
			// name -> labels
			targetDestMap := destRuleConfigsMap[host]

			// copy to new Map, FIXME name collision
			mapCopy(tempServiceNeedsDescMap, targetDestMap)
		}
		// transform single config to one rule
		routers = append(routers, NewUniformRouter(v.Spec.Dubbo, tempServiceNeedsDescMap))
	}
	logger.Debug("parsed successfully with router size = ", len(routers))
	return routers, nil
}

func parseRoute(routeContent string) ([]*UniformRouter, error) {
	var virtualServiceConfigList []*config.VirtualServiceConfig
	destRuleConfigsMap := make(map[string]map[string]map[string]string)

	meshRouteDecoder := yaml.NewDecoder(strings.NewReader(routeContent))
	for {
		meshRouteMetadata := &config.MeshRouteMetadata{}
		err := meshRouteDecoder.Decode(meshRouteMetadata)
		if err == io.EOF {
			break
		} else if err != nil {
			logger.Error("parseRoute route metadata err = ", err)
			return nil, err
		}

		bytes, err := yaml.Marshal(meshRouteMetadata.Spec)
		if err != nil {
			return nil, err
		}
		specDecoder := yaml.NewDecoder(strings.NewReader(string(bytes)))
		switch meshRouteMetadata.YamlKind {
		case "VirtualService":
			meshRouteConfigSpec := &config.UniformRouterConfigSpec{}
			err := specDecoder.Decode(meshRouteConfigSpec)
			if err != nil {
				return nil, err
			}
			virtualServiceConfigList = append(virtualServiceConfigList, &config.VirtualServiceConfig{
				YamlAPIVersion: meshRouteMetadata.YamlAPIVersion,
				YamlKind:       meshRouteMetadata.YamlKind,
				TypeMeta:       meshRouteMetadata.TypeMeta,
				ObjectMeta:     meshRouteMetadata.ObjectMeta,
				MetaData:       meshRouteMetadata.MetaData,
				Spec:           *meshRouteConfigSpec,
			})
		case "DestinationRule":
			meshRouteDestinationRuleSpec := &config.DestinationRuleSpec{}
			err := specDecoder.Decode(meshRouteDestinationRuleSpec)
			if err != nil {
				return nil, err
			}
			destRuleCfgMap := make(map[string]map[string]string)
			for _, v := range meshRouteDestinationRuleSpec.SubSets {
				destRuleCfgMap[v.Name] = v.Labels
			}

			destRuleConfigsMap[meshRouteDestinationRuleSpec.Host] = destRuleCfgMap
		}
	}

	routers := make([]*UniformRouter, 0)

	for _, v := range virtualServiceConfigList {
		tempServiceNeedsDescMap := make(map[string]map[string]string)
		for _, host := range v.Spec.Hosts {
			targetDestMap := destRuleConfigsMap[host]
			mapCopy(tempServiceNeedsDescMap, targetDestMap)
		}
		routers = append(routers, NewUniformRouter(v.Spec.Dubbo, tempServiceNeedsDescMap))
	}
	logger.Debug("parsed successfully with router size = ", len(routers))
	return routers, nil
}

func mapCopy(dist map[string]map[string]string, source map[string]map[string]string) {
	for k, v := range source {
		dist[k] = v
	}
}
