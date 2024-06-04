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
	"dubbo.apache.org/dubbo-go/v3/cluster/utils"
	"fmt"
	"gopkg.in/yaml.v2"
	"sort"
	"strconv"
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type conditionRoute []*StateRouter

func (p conditionRoute) route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if len(invokers) == 0 || len(p) == 0 {
		return invokers
	}
	for _, router := range p {
		invokers, _ = router.Route(invokers, url, invocation)
		if len(invokers) == 0 {
			break
		}
	}
	return invokers
}

type multiplyConditionRoute []*StateRouter

func (m multiplyConditionRoute) route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if len(invokers) == 0 || len(m) == 0 {
		return invokers
	}
	for _, router := range m {
		matchInvokers, isMatch := router.Route(invokers, url, invocation)
		if !isMatch || (len(matchInvokers) == 0 && !router.force) {
			continue
		}
		return matchInvokers
	}
	return []protocol.Invoker{}
}

type condRouter interface {
	route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker
}

type DynamicRouter struct {
	mu              sync.RWMutex
	force           bool
	enable          bool
	conditionRouter condRouter
}

func (d *DynamicRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
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
		if len(res) == 0 && !force {
			return invokers
		}
		return res
	} else {
		return invokers
	}
}

func (d *DynamicRouter) URL() *common.URL {
	return nil
}

func (d *DynamicRouter) Process(event *config_center.ConfigChangeEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if event.ConfigType == remoting.EventTypeDel {
		d.conditionRouter = nil
	} else {
		rc, force, enable, err := generateCondition(event.Value.(string))
		if err != nil {
			logger.Errorf("generate condition error: %v", err)
			d.conditionRouter = nil
		} else {
			d.force, d.enable, d.conditionRouter = force, enable, rc
		}
	}
}

/*
to check configVersion, here need decode twice.
From a performance perspective, decoding from a string and decoding from a map[string]interface{}
cost nearly same (a few milliseconds).

To keep the code simpler,
here use yaml-to-map and yaml-to-struct, not yaml-to-map-to-struct

generateCondition @return(router,force,enable,error)
*/
func generateCondition(rawConfig string) (condRouter, bool, bool, error) {
	m := map[string]interface{}{}

	err := yaml.Unmarshal([]byte(rawConfig), m)
	if err != nil {
		return nil, false, false, err
	}

	rawVersion, ok := m["configVersion"]
	if !ok {
		return nil, false, false, fmt.Errorf("miss `ConfigVersion` in %s", rawConfig)
	} else if version, ok := rawVersion.(string); !ok {
		return nil, false, false, fmt.Errorf("`ConfigVersion should be `string` got %v`", rawVersion)
	} else {
		var (
			cr     condRouter
			enable bool
			force  bool
		)
		if mode, compareErr := utils.CompareVersions(constant.RouteVersion, version); compareErr != nil {
			return nil, false, false, fmt.Errorf("invalid version %s, %s", version, compareErr.Error())
		} else {
			switch mode {
			case utils.VersionEqual:
				cr, force, enable, err = generateMultiConditionRoute(rawConfig)
			case utils.VersionGreater:
				cr, force, enable, err = generateConditionsRoute(rawConfig)
			case utils.VersionLess:
				err = fmt.Errorf("config version %s is greater than %s", rawVersion, constant.RouteVersion)
			default:
				panic("invalid version compare return")
			}
			if err != nil {
				err = fmt.Errorf("generate condition error: %s", err.Error())
			}
		}
		return cr, force, enable, err
	}
}

func generateMultiConditionRoute(rawConfig string) (multiplyConditionRoute, bool, bool, error) {
	routerConfig, err := parseMultiConditionRoute(rawConfig)
	if err != nil {
		logger.Warnf("[condition router]Build a new condition route config error, %s and we will use the original condition rule configuration.", err.Error())
		return nil, false, false, err
	}

	force, enable := routerConfig.Enabled, routerConfig.Force
	if !enable {
		return nil, false, false, nil
	}

	conditionRouters := make([]*StateRouter, 0, len(routerConfig.Conditions))
	for _, conditionRule := range routerConfig.Conditions {
		url, err := common.NewURL("condition://")
		if err != nil {
			return nil, false, false, err
		}
		url.AddParam(constant.RuleKey, conditionRule.Rule)
		url.AddParam(constant.ForceKey, strconv.FormatBool(conditionRule.Force))
		url.AddParam(constant.PriorityKey, strconv.FormatInt(int64(conditionRule.Priority), 10))
		conditionRoute, err := NewConditionStateRouter(url)
		if err != nil {
			return nil, false, false, err
		}
		conditionRouters = append(conditionRouters, conditionRoute)
	}

	sort.Slice(conditionRouters, func(i, j int) bool {
		return conditionRouters[i].priority > conditionRouters[j].priority
	})
	return conditionRouters, force, enable, nil
}

func generateConditionsRoute(rawConfig string) (conditionRoute, bool, bool, error) {
	routerConfig, err := parseConditionRoute(rawConfig)
	if err != nil {
		logger.Warnf("[condition router]Build a new condition route config error, %s and we will use the original condition rule configuration.", err.Error())
		return nil, false, false, err
	}

	force, enable := *routerConfig.Enabled, *routerConfig.Force
	if !enable {
		return nil, false, false, nil
	}

	conditionRouters := make([]*StateRouter, 0, len(routerConfig.Conditions))
	for _, conditionRule := range routerConfig.Conditions {
		url, err := common.NewURL("condition://")
		if err != nil {
			return nil, false, false, err
		}
		url.AddParam(constant.RuleKey, conditionRule)
		url.AddParam(constant.ForceKey, strconv.FormatBool(*routerConfig.Force))
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

func (s *ServiceRouter) Priority() int64 {
	return 140
}

func (s *ServiceRouter) Notify(invokers []protocol.Invoker) {
	if len(invokers) == 0 {
		return
	}
	url := invokers[0].GetURL()
	if url == nil {
		logger.Error("Failed to notify a dynamically condition rule, because url is empty")
		return
	}

	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		logger.Infof("Config center does not start, Condition router will not be enabled")
		return
	}
	key := strings.Join([]string{url.ColonSeparatedKey(), constant.ConditionRouterRuleSuffix}, "")
	dynamicConfiguration.AddListener(key, s)
	value, err := dynamicConfiguration.GetRule(key)
	if err != nil {
		logger.Errorf("Failed to query condition rule, key=%s, err=%v", key, err)
		return
	}
	s.Process(&config_center.ConfigChangeEvent{Key: key, Value: value, ConfigType: remoting.EventTypeAdd})
}

// ApplicationRouter is Application level router
type ApplicationRouter struct {
	DynamicRouter
	application        string
	currentApplication string
}

func NewApplicationRouter() *ApplicationRouter {
	applicationName := config.GetApplicationConfig().Name
	a := &ApplicationRouter{
		currentApplication: applicationName,
	}

	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration != nil {
		dynamicConfiguration.AddListener(strings.Join([]string{applicationName, constant.ConditionRouterRuleSuffix}, ""), a)
	}
	return a
}

func (a *ApplicationRouter) Priority() int64 {
	return 145
}

func (a *ApplicationRouter) Notify(invokers []protocol.Invoker) {
	if len(invokers) == 0 {
		return
	}
	url := invokers[0].GetURL()
	if url == nil {
		logger.Error("Failed to notify a dynamically condition rule, because url is empty")
		return
	}

	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		logger.Infof("Config center does not start, Condition router will not be enabled")
		return
	}

	providerApplicaton := url.GetParam("application", "")
	if providerApplicaton == "" || providerApplicaton == a.currentApplication {
		logger.Warn("condition router get providerApplication is empty, will not subscribe to provider app rules.")
		return
	}

	if providerApplicaton != a.application {
		if a.application != "" {
			dynamicConfiguration.RemoveListener(strings.Join([]string{a.application, constant.ConditionRouterRuleSuffix}, ""), a)
		}

		key := strings.Join([]string{providerApplicaton, constant.ConditionRouterRuleSuffix}, "")
		dynamicConfiguration.AddListener(key, a)
		a.application = providerApplicaton
		value, err := dynamicConfiguration.GetRule(key)
		if err != nil {
			logger.Errorf("Failed to query condition rule, key=%s, err=%v", key, err)
			return
		}
		a.Process(&config_center.ConfigChangeEvent{Key: key, Value: value, ConfigType: remoting.EventTypeUpdate})
	}
}
