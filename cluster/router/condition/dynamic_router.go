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

package condition

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"go.yaml.in/yaml/v4"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/utils"
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// for version 3.0-
type stateRouters []*StateRouter

func (p stateRouters) route(invokers []base.Invoker, url *common.URL, invocation base.Invocation) []base.Invoker {
	if len(invokers) == 0 || len(p) == 0 {
		return invokers
	}
	for _, router := range p {
		invokers = router.Route(invokers, url, invocation)
		if len(invokers) == 0 {
			break
		}
	}
	return invokers
}

type multiplyConditionRoute struct {
	trafficDisabled []*FieldMatcher
	routes          []*MultiDestRouter
}

func (m *multiplyConditionRoute) route(invokers []base.Invoker, url *common.URL, invocation base.Invocation) []base.Invoker {
	if len(m.trafficDisabled) != 0 {
		for _, cond := range m.trafficDisabled {
			if cond.MatchRequest(url, invocation) {
				logger.Warnf("[Router][Condition] request traffic disabled: url=%s, rule=%s", url.String(), cond.rule)
				invocation.SetAttachment(constant.TrafficDisableKey, struct{}{})
				return []base.Invoker{}
			}
		}
	}

	if len(invokers) != 0 && len(m.routes) != 0 {
		for _, router := range m.routes {
			isMatchWhen := false
			invokers, isMatchWhen = router.Route(invokers, url, invocation)
			if !isMatchWhen {
				continue
			}
			if len(invokers) == 0 {
				routeChains, ok := invocation.Attributes()["condition-chain"].([]string)
				if ok {
					logger.Errorf("[Router][Condition] route returned empty set: chain=%s", strings.Join(routeChains, "-->"))
				}
				return []base.Invoker{}
			}
		}
		delete(invocation.Attributes(), "condition-chain")
	}

	return invokers
}

type condRouter interface {
	route(invokers []base.Invoker, url *common.URL, invocation base.Invocation) []base.Invoker
}

type DynamicRouter struct {
	mu              sync.RWMutex
	force           bool
	enable          bool
	conditionRouter condRouter
	// ruleEpoch counts config-center writes applied to this router.
	// It is captured before AddListener. A GetRule snapshot is applied only
	// when the epoch is unchanged, and SetStaticConfig does not replace a
	// router whose epoch has already moved.
	ruleEpoch uint64
}

func (d *DynamicRouter) Route(invokers []base.Invoker, url *common.URL, invocation base.Invocation) []base.Invoker {
	if len(invokers) == 0 {
		return invokers
	}

	d.mu.RLock()
	force, enable, cr := d.force, d.enable, d.conditionRouter
	d.mu.RUnlock()

	if !enable {
		return invokers
	}
	if cr != nil {
		res := cr.route(invokers, url, invocation)
		if len(res) == 0 {
			if invocation.GetAttachmentInterface(constant.TrafficDisableKey) == nil && !force {
				return invokers
			}
		}
		return res
	} else {
		return invokers
	}
}

func (d *DynamicRouter) URL() *common.URL {
	return nil
}

// SetStaticConfig applies a RouterConfig with string conditions directly, bypassing YAML parsing.
// This bootstraps static (code-configured) rules. Process applies config-center YAML.
// Static and dynamic rules are not merged. NewRouterChain builds the application
// router, which subscribes and loads the consumer dynamic rule, before
// injectStaticRouters. A dynamic write that has already landed stays in place;
// later Process calls still replace the current state.
func (d *DynamicRouter) SetStaticConfig(cfg *global.RouterConfig) {
	if cfg == nil || len(cfg.Conditions) == 0 {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ruleEpoch != 0 {
		logger.Infof("[Router][Condition] keep subscribed condition rule over static config: conditions=%d", len(cfg.Conditions))
		return
	}

	force := cfg.Force != nil && *cfg.Force
	enable := cfg.Enabled == nil || *cfg.Enabled

	if !enable {
		d.force, d.enable, d.conditionRouter = force, false, nil
		return
	}

	conditionRouters := make([]*StateRouter, 0, len(cfg.Conditions))
	for _, conditionRule := range cfg.Conditions {
		url, err := common.NewURL("condition://")
		if err != nil {
			logger.Warnf("[Router][Condition] failed to create condition URL: err=%v", err)
			continue
		}
		url.SetParam(constant.RuleKey, conditionRule)
		url.SetParam(constant.ForceKey, strconv.FormatBool(force))
		conditionRoute, err := NewConditionStateRouter(url)
		if err != nil {
			logger.Warnf("[Router][Condition] failed to parse condition rule: rule=%s, err=%v", conditionRule, err)
			continue
		}
		conditionRouters = append(conditionRouters, conditionRoute)
	}
	d.force, d.enable, d.conditionRouter = force, enable, stateRouters(conditionRouters)
}

// Process applies config-center updates as the authoritative rule source at runtime.
// It does not merge with static rules bootstrapped via SetStaticConfig; any later
// dynamic update replaces the current state, including a rule loaded from the
// initial GetRule snapshot.
func (d *DynamicRouter) Process(event *config_center.ConfigChangeEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ruleEpoch++
	d.applyEventLocked(event)
}

func (d *DynamicRouter) captureEpoch() uint64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.ruleEpoch
}

// subscribeInitialRule registers listener and applies the current config-center
// value. The epoch is captured before AddListener, so an update or delete
// delivered before the snapshot is applied discards that stale GetRule result.
func (d *DynamicRouter) subscribeInitialRule(
	dc config_center.DynamicConfiguration,
	listener config_center.ConfigurationListener,
	key string,
	eventType remoting.EventType,
) (string, error) {
	captured := d.captureEpoch()
	dc.AddListener(key, listener)
	value, err := dc.GetRule(key)
	if err != nil || value == "" {
		return value, err
	}
	d.applyInitialSnapshot(&config_center.ConfigChangeEvent{Key: key, Value: value, ConfigType: eventType}, captured)
	return value, nil
}

func (d *DynamicRouter) applyInitialSnapshot(event *config_center.ConfigChangeEvent, captured uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ruleEpoch != captured {
		logger.Infof("[Router][Condition] discard stale condition snapshot: key=%s", event.Key)
		return
	}
	d.ruleEpoch++
	d.applyEventLocked(event)
}

func (d *DynamicRouter) applyEventLocked(event *config_center.ConfigChangeEvent) {
	if event.ConfigType == remoting.EventTypeDel {
		d.conditionRouter = nil
		return
	}
	rc, force, enable, err := generateCondition(event.Value.(string))
	if err != nil {
		logger.Errorf("[Router][Condition] failed to generate condition: err=%v", err)
		d.conditionRouter = nil
		return
	}
	d.force, d.enable, d.conditionRouter = force, enable, rc
}

/*
to check configVersion, here need decode twice.
From a performance perspective, decoding from a string and decoding from a map[string]any
cost nearly same (a few milliseconds).

To keep the code simpler,
here use yaml-to-map and yaml-to-struct, not yaml-to-map-to-struct

generateCondition @return(router,force,enable,error)
*/
func generateCondition(rawConfig string) (condRouter, bool, bool, error) {
	m := map[string]any{}

	err := yaml.Unmarshal([]byte(rawConfig), m)
	if err != nil {
		return nil, false, false, err
	}

	rawVersion, ok := m["configVersion"]
	if !ok {
		return nil, false, false, fmt.Errorf("miss `ConfigVersion` in %s", rawConfig)
	}

	version, ok := rawVersion.(string)
	if !ok {
		return nil, false, false, fmt.Errorf("`ConfigVersion` should be of type `string`, got %T", rawVersion)
	}

	v, parseErr := utils.ParseVersion(version)
	if parseErr != nil {
		return nil, false, false, fmt.Errorf("invalid version %s: %s", version, parseErr.Error())
	}

	switch {
	case v.Equal(utils.V3_1) || v.Greater(utils.V3_1):
		return generateMultiConditionRoute(rawConfig)
	case v.Less(utils.V3_1):
		return generateConditionsRoute(rawConfig)
	default:
		panic("invalid version compare return")
	}
}

func generateMultiConditionRoute(rawConfig string) (*multiplyConditionRoute, bool, bool, error) {
	routerConfig, err := parseMultiConditionRoute(rawConfig)
	if err != nil {
		logger.Warnf("[Router][Condition] failed to build condition route config: err=%s", err.Error())
		return nil, false, false, err
	}

	enable, force := routerConfig.Enabled, routerConfig.Force
	if !enable {
		return nil, false, false, nil
	}

	// remove same condition
	removeDuplicates(routerConfig.Conditions)

	conditionRouters := make([]*MultiDestRouter, 0, len(routerConfig.Conditions))
	disableMultiConditions := make([]*FieldMatcher, 0)
	for _, conditionRule := range routerConfig.Conditions {
		// removeDuplicates will set nil
		if conditionRule == nil {
			continue
		}
		url, err1 := common.NewURL("condition://")
		if err1 != nil {
			return nil, false, false, err1
		}

		url.SetAttribute(constant.RuleKey, conditionRule)

		conditionRoute, err2 := NewConditionMultiDestRouter(url)
		if err2 != nil {
			return nil, false, false, err2
		}
		// got invalid condition config, continue
		if conditionRoute == nil {
			continue
		}
		if len(conditionRoute.thenCondition) != 0 {
			conditionRouters = append(conditionRouters, conditionRoute)
		} else {
			disableMultiConditions = append(disableMultiConditions, &conditionRoute.whenCondition)
		}
	}

	return &multiplyConditionRoute{
		trafficDisabled: disableMultiConditions,
		routes:          conditionRouters,
	}, force, enable, nil
}

func generateConditionsRoute(rawConfig string) (stateRouters, bool, bool, error) {
	routerConfig, err := parseConditionRoute(rawConfig)
	if err != nil {
		logger.Warnf("[Router][Condition] failed to build condition route config: err=%s", err.Error())
		return nil, false, false, err
	}

	force := routerConfig.Force != nil && *routerConfig.Force
	enable := routerConfig.Enabled == nil || *routerConfig.Enabled
	if !enable {
		return nil, false, false, nil
	}

	conditionRouters := make([]*StateRouter, 0, len(routerConfig.Conditions))
	for _, conditionRule := range routerConfig.Conditions {
		url, err := common.NewURL("condition://")
		if err != nil {
			return nil, false, false, err
		}
		url.SetParam(constant.RuleKey, conditionRule)
		url.SetParam(constant.ForceKey, strconv.FormatBool(force))
		conditionRoute, err := NewConditionStateRouter(url)
		if err != nil {
			return nil, false, false, err
		}
		conditionRouters = append(conditionRouters, conditionRoute)
	}
	return conditionRouters, force, enable, nil
}

// ServiceRouter is Service level router
type ServiceRouter struct {
	DynamicRouter
}

func NewServiceRouter() *ServiceRouter {
	return &ServiceRouter{}
}

// SetStaticConfig applies config only when scope is service and Conditions are present.
func (s *ServiceRouter) SetStaticConfig(cfg *global.RouterConfig) {
	if cfg == nil || cfg.Scope != constant.RouterScopeService || len(cfg.Conditions) == 0 {
		return
	}
	s.DynamicRouter.SetStaticConfig(cfg)
}

func (s *ServiceRouter) Priority() int64 {
	return 140
}

func (s *ServiceRouter) Notify(invokers []base.Invoker) {
	if len(invokers) == 0 {
		return
	}
	url := invokers[0].GetURL()
	if url == nil {
		logger.Error("[Router][Condition] failed to notify condition router: url is empty")
		return
	}

	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		logger.Info("[Router][Condition] Config center does not start, Condition router will not be enabled")
		return
	}
	key := strings.Join([]string{url.ColonSeparatedKey(), constant.ConditionRouterRuleSuffix}, "")
	value, err := s.subscribeInitialRule(dynamicConfiguration, s, key, remoting.EventTypeAdd)
	if err != nil {
		logger.Errorf("[Router][Condition] Failed to query condition rule, key=%s, err=%v", key, err)
		return
	}
	if value == "" {
		logger.Infof("[Router][Condition] Condition rule is empty, key=%s", key)
		return
	}
}

// ApplicationRouter is Application level router
type ApplicationRouter struct {
	DynamicRouter
	application        string
	currentApplication string
}

func NewApplicationRouter(url *common.URL) *ApplicationRouter {
	// get application name from url param
	applicationName := url.GetParam(constant.ApplicationKey, "")
	if applicationName == "" && url.SubURL != nil {
		applicationName = url.SubURL.GetParam(constant.ApplicationKey, "")
	}

	a := &ApplicationRouter{
		currentApplication: applicationName,
	}

	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration != nil && applicationName != "" {
		key := strings.Join([]string{applicationName, constant.ConditionRouterRuleSuffix}, "")
		if _, err := a.subscribeInitialRule(dynamicConfiguration, a, key, remoting.EventTypeAdd); err != nil {
			logger.Errorf("[Router][Condition] failed to query condition rule: key=%s, err=%v", key, err)
		}
	}
	return a
}

// SetStaticConfig applies config only when scope is application and Conditions are present.
func (a *ApplicationRouter) SetStaticConfig(cfg *global.RouterConfig) {
	if cfg == nil || cfg.Scope != constant.RouterScopeApplication || len(cfg.Conditions) == 0 {
		return
	}
	a.DynamicRouter.SetStaticConfig(cfg)
}

func (a *ApplicationRouter) Priority() int64 {
	return 145
}

func (a *ApplicationRouter) Notify(invokers []base.Invoker) {
	if len(invokers) == 0 {
		return
	}
	url := invokers[0].GetURL()
	if url == nil {
		logger.Error("[Router][Condition] failed to notify condition router: url is empty")
		return
	}

	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		logger.Info("[Router][Condition] Config center does not start, Condition router will not be enabled")
		return
	}

	providerApplication := url.GetParam("application", "")
	if providerApplication == "" || providerApplication == a.currentApplication {
		logger.Warn("[Router][Condition] provider application is empty, will not subscribe to provider app rules")
		return
	}

	if providerApplication != a.application {
		if a.application != "" {
			dynamicConfiguration.RemoveListener(strings.Join([]string{a.application, constant.ConditionRouterRuleSuffix}, ""), a)
		}

		key := strings.Join([]string{providerApplication, constant.ConditionRouterRuleSuffix}, "")
		a.application = providerApplication
		value, err := a.subscribeInitialRule(dynamicConfiguration, a, key, remoting.EventTypeUpdate)
		if err != nil {
			logger.Errorf("[Router][Condition] failed to query condition rule: key=%s, err=%v", key, err)
			return
		}
		if value == "" {
			logger.Infof("[Router][Condition] Condition rule is empty, key=%s", key)
			return
		}
	}
}

func removeDuplicates(rules []*global.ConditionRule) {
	for i := range rules {
		if rules[i] == nil {
			continue
		}
		for j := i + 1; j < len(rules); j++ {
			if rules[j] != nil && rules[i].Equal(rules[j]) {
				rules[j] = nil
			}
		}
	}
}
