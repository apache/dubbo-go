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

package tag

import (
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type PriorityRouter struct {
	routerConfigs sync.Map
}

func NewTagPriorityRouter() (*PriorityRouter, error) {
	return &PriorityRouter{}, nil
}

// Route Determine the target invokers list.
func (p *PriorityRouter) Route(invokers []base.Invoker, url *common.URL, invocation base.Invocation) []base.Invoker {
	if len(invokers) == 0 {
		logger.Warnf("[tag router] invokers from previous router is empty")
		return invokers
	}
	// get application name from invoker to look up tag routing config
	application := invokers[0].GetURL().GetParam(constant.ApplicationKey, "")
	key := strings.Join([]string{application, constant.TagRouterRuleSuffix}, "")
	value, ok := p.routerConfigs.Load(key)
	if !ok {
		return staticTag(invokers, url, invocation)
	}
	routerCfg := value.(global.RouterConfig)
	enabled := routerCfg.Enabled == nil || *routerCfg.Enabled
	valid := (routerCfg.Valid != nil && *routerCfg.Valid) || (routerCfg.Valid == nil && len(routerCfg.Tags) > 0)
	if !enabled || !valid {
		return staticTag(invokers, url, invocation)
	}
	return dynamicTag(invokers, url, invocation, routerCfg)
}

func (p *PriorityRouter) URL() *common.URL {
	return nil
}

func (p *PriorityRouter) Priority() int64 {
	return 0
}

func (p *PriorityRouter) Notify(invokers []base.Invoker) {
	if len(invokers) == 0 {
		return
	}
	application := invokers[0].GetURL().GetParam(constant.ApplicationKey, "")
	if application == "" {
		logger.Warn("url application is empty, tag router will not be enabled")
		return
	}
	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		logger.Infof("Config center does not start, Tag router will not be enabled")
		return
	}
	key := strings.Join([]string{application, constant.TagRouterRuleSuffix}, "")
	dynamicConfiguration.AddListener(key, p)
	value, err := dynamicConfiguration.GetRule(key)
	if err != nil {
		logger.Errorf("query router rule fail,key=%s,err=%v", key, err)
		return
	}
	if value == "" {
		logger.Infof("router rule is empty,key=%s", key)
		return
	}
	p.Process(&config_center.ConfigChangeEvent{Key: key, Value: value, ConfigType: remoting.EventTypeAdd})
}

// SetStaticConfig applies a RouterConfig directly, bypassing YAML parsing.
// This is the correct entry point for static (code-configured) rules;
// Process is designed for dynamic config-center updates that arrive as YAML text.
// Static and dynamic rules are not merged: later Process updates replace the current state built here.
func (p *PriorityRouter) SetStaticConfig(cfg *global.RouterConfig) {
	if cfg == nil || cfg.Scope != constant.RouterScopeApplication || len(cfg.Tags) == 0 {
		return
	}
	cfgCopy := cfg.Clone()
	cfgCopy.Valid = new(bool)
	*cfgCopy.Valid = len(cfgCopy.Tags) > 0
	if cfgCopy.Enabled == nil {
		cfgCopy.Enabled = new(bool)
		*cfgCopy.Enabled = true
	}
	// Derive storage key the same way Notify() does: application + suffix
	key := strings.Join([]string{cfg.Key, constant.TagRouterRuleSuffix}, "")
	p.routerConfigs.Store(key, *cfgCopy)
	logger.Infof("[tag router] Applied static tag router config: key=%s", key)
}

// Process applies config-center updates as the authoritative rule source at runtime.
// It does not merge with static rules bootstrapped via SetStaticConfig; any later
// dynamic update replaces the current static-derived state.
func (p *PriorityRouter) Process(event *config_center.ConfigChangeEvent) {
	if event.ConfigType == remoting.EventTypeDel {
		p.routerConfigs.Delete(event.Key)
		return
	}
	routerConfig, err := parseRoute(event.Value.(string))
	if err != nil {
		logger.Warnf("[tag router]Parse new tag route config error, %+v "+
			"and we will use the original tag rule configuration.", err)
		return
	}
	p.routerConfigs.Store(event.Key, *routerConfig)
	logger.Infof("[tag router]Parse tag router config success,routerConfig=%+v", routerConfig)
}

func parseRoute(routeContent string) (*global.RouterConfig, error) {
	routeDecoder := yaml.NewDecoder(strings.NewReader(routeContent))
	routerConfig := &global.RouterConfig{}
	err := routeDecoder.Decode(routerConfig)
	if err != nil {
		return nil, err
	}
	routerConfig.Valid = new(bool)
	*routerConfig.Valid = true
	if len(routerConfig.Tags) == 0 {
		*routerConfig.Valid = false
	}
	return routerConfig, nil
}
