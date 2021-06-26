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
	"encoding/json"
	"io"
	"strings"
)

import (
	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/cluster/router/v3router/k8s_api"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// RouterChain contains all uniform router logic
// it has UniformRouter list,
type RouterChain struct {
	routers                    []*UniformRouter
	virtualServiceConfigBytes  []byte
	destinationRuleConfigBytes []byte
	notify                     chan struct{}
}

// NewUniformRouterChain return
func NewUniformRouterChain(virtualServiceConfig, destinationRuleConfig []byte) (router.PriorityRouter, error) {
	fromFileConfig := true
	uniformRouters, err := parseFromConfigToRouters(virtualServiceConfig, destinationRuleConfig)
	if err != nil {
		fromFileConfig = false
		logger.Warnf("parse router config form local file failed, error = %+v", err)
	}
	r := &RouterChain{
		virtualServiceConfigBytes:  virtualServiceConfig,
		destinationRuleConfigBytes: destinationRuleConfig,
		routers:                    uniformRouters,
	}
	if err := k8s_api.SetK8sEventListener(r); err != nil {
		logger.Warnf("try listen K8s router config failed, error = %+v", err)
		if !fromFileConfig {
			panic("No config file from both local file and k8s")
		}
	}
	return r, nil
}

// Route route invokers using RouterChain's routers one by one
func (r *RouterChain) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	for _, v := range r.routers {
		invokers = v.Route(invokers, url, invocation)
	}
	return invokers
}

func (r *RouterChain) Process(event *config_center.ConfigChangeEvent) {
	logger.Debugf("on processed event = %+v\n", *event)
	if event.ConfigType == remoting.EventTypeAdd || event.ConfigType == remoting.EventTypeUpdate {
		switch event.Key {
		case k8s_api.VirtualServiceEventKey:
			logger.Debug("virtul service event")
			newVSValue, ok := event.Value.(*config.VirtualServiceConfig)
			if !ok {
				logger.Error("event.Value assertion error")
				return
			}

			newVSJsonValue, ok := newVSValue.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
			if !ok {
				logger.Error("newVSValue.ObjectMeta.Annotations has no key named kubectl.kubernetes.io/last-applied-configuration")
				return
			}
			logger.Debugf("new virtual service json value = \n%v\n", newVSJsonValue)
			newVirtualServiceConfig := &config.VirtualServiceConfig{}
			if err := json.Unmarshal([]byte(newVSJsonValue), newVirtualServiceConfig); err != nil {
				logger.Error("on process json data unmarshal error = ", err)
				return
			}
			newVirtualServiceConfig.YamlAPIVersion = newVirtualServiceConfig.APIVersion
			newVirtualServiceConfig.YamlKind = newVirtualServiceConfig.Kind
			newVirtualServiceConfig.MetaData.Name = newVirtualServiceConfig.ObjectMeta.Name
			logger.Debugf("get event after asseration = %+v\n", newVirtualServiceConfig)
			data, err := yaml.Marshal(newVirtualServiceConfig)
			if err != nil {
				logger.Error("Process change of virtual service: event.Value marshal error:", err)
				return
			}
			r.routers, err = parseFromConfigToRouters(data, r.destinationRuleConfigBytes)
			if err != nil {
				logger.Error("Process change of virtual service: parseFromConfigToRouters:", err)
				return
			}
		case k8s_api.DestinationRuleEventKey:
			logger.Debug("handling dest rule event")
			newDRValue, ok := event.Value.(*config.DestinationRuleConfig)
			if !ok {
				logger.Error("event.Value assertion error")
				return
			}

			newDRJsonValue, ok := newDRValue.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
			if !ok {
				logger.Error("newVSValue.ObjectMeta.Annotations has no key named kubectl.kubernetes.io/last-applied-configuration")
				return
			}
			newDestRuleConfig := &config.DestinationRuleConfig{}
			if err := json.Unmarshal([]byte(newDRJsonValue), newDestRuleConfig); err != nil {
				logger.Error("on process json data unmarshal error = ", err)
				return
			}
			newDestRuleConfig.YamlAPIVersion = newDestRuleConfig.APIVersion
			newDestRuleConfig.YamlKind = newDestRuleConfig.Kind
			newDestRuleConfig.MetaData.Name = newDestRuleConfig.ObjectMeta.Name
			logger.Debugf("get event after asseration = %+v\n", newDestRuleConfig)
			data, err := yaml.Marshal(newDestRuleConfig)
			if err != nil {
				logger.Error("Process change of dest rule: event.Value marshal error:", err)
				return
			}
			r.routers, err = parseFromConfigToRouters(r.virtualServiceConfigBytes, data)
			if err != nil {
				logger.Error("Process change of dest rule: parseFromConfigToRouters:", err)
				return
			}
		default:
			logger.Error("unknown unsupported event key:", event.Key)
		}
	}

	// todo delete router
	//if event.ConfigType == remoting.EventTypeDel {
	//
	//}
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

// parseFromConfigToRouters parse virtualService and destinationRule yaml file bytes to target router list
func parseFromConfigToRouters(virtualServiceConfig, destinationRuleConfig []byte) ([]*UniformRouter, error) {
	var virtualServiceConfigList []*config.VirtualServiceConfig
	destRuleConfigsMap := make(map[string]map[string]map[string]string)

	vsDecoder := yaml.NewDecoder(strings.NewReader(string(virtualServiceConfig)))
	drDecoder := yaml.NewDecoder(strings.NewReader(string(destinationRuleConfig)))
	// parse virtual service
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

	// parse destination rule
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

	for _, v := range virtualServiceConfigList {
		tempSerivceNeedsDescMap := make(map[string]map[string]string)
		for _, host := range v.Spec.Hosts {
			targetDestMap := destRuleConfigsMap[host]

			// copy to new Map
			mapCombine(tempSerivceNeedsDescMap, targetDestMap)
		}
		// change single config to one rule
		newRule, err := newDubboRouterRule(v.Spec.Dubbo, tempSerivceNeedsDescMap)
		if err != nil {
			logger.Error("Parse config to uniform rule err = ", err)
			return nil, err
		}
		rtr, err := NewUniformRouter(newRule)
		if err != nil {
			logger.Error("new uniform router err = ", err)
			return nil, err
		}
		routers = append(routers, rtr)
	}
	logger.Debug("parsed successed! with router size = ", len(routers))
	return routers, nil
}

func mapCombine(dist map[string]map[string]string, from map[string]map[string]string) {
	for k, v := range from {
		dist[k] = v
	}
}
