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

type DynamicRouter struct {
	conditionRouters []*StateRouter
	routerConfig     *config.RouterConfig
}

func (d *DynamicRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if len(invokers) == 0 || len(d.conditionRouters) == 0 {
		return invokers
	}

	for _, router := range d.conditionRouters {
		invokers = router.Route(invokers, url, invocation)
	}
	return invokers
}

func (d *DynamicRouter) URL() *common.URL {
	return nil
}

func (d *DynamicRouter) Process(event *config_center.ConfigChangeEvent) {
	if event.ConfigType == remoting.EventTypeDel {
		d.routerConfig = nil
		d.conditionRouters = make([]*StateRouter, 0)
	} else {
		routerConfig, err := parseRoute(event.Value.(string))
		if err != nil {
			logger.Warnf("[condition router]Build a new condition route config error, %+v and we will use the original condition rule configuration.", err)
			return
		}
		d.routerConfig = routerConfig
		conditions, err := generateConditions(d.routerConfig)
		if err != nil {
			logger.Warnf("[condition router]Build a new condition route config error, %+v and we will use the original condition rule configuration.", err)
			return
		}
		d.conditionRouters = conditions
	}
}

func generateConditions(routerConfig *config.RouterConfig) ([]*StateRouter, error) {
	if routerConfig == nil {
		return make([]*StateRouter, 0), nil
	}
	conditionRouters := make([]*StateRouter, 0, len(routerConfig.Conditions))
	for _, conditionRule := range routerConfig.Conditions {
		url, err := common.NewURL("condition://")
		if err != nil {
			return nil, err
		}
		url.AddParam(constant.RuleKey, conditionRule)
		url.AddParam(constant.ForceKey, strconv.FormatBool(*routerConfig.Force))
		url.AddParam(constant.EnabledKey, strconv.FormatBool(*routerConfig.Enabled))
		conditionRoute, err := NewConditionStateRouter(url)
		if err != nil {
			return nil, err
		}
		conditionRouters = append(conditionRouters, conditionRoute)
	}
	return conditionRouters, nil
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
		logger.Warnf("config center does not start, please check if the configuration center has been properly configured in dubbogo.yml")
		return
	}
	key := strings.Join([]string{strings.Join([]string{url.Service(), url.GetParam(constant.VersionKey, ""), url.GetParam(constant.GroupKey, "")}, ":"),
		constant.ConditionRouterRuleSuffix}, "")
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
	mu                 sync.Mutex
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
	providerApplicaton := url.GetParam("application", "")
	if providerApplicaton == "" || providerApplicaton == a.currentApplication {
		logger.Warn("condition router get providerApplication is empty, will not subscribe to provider app rules.")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if providerApplicaton != a.application {
		dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
		if dynamicConfiguration == nil {
			logger.Warnf("config center does not start, please check if the configuration center has been properly configured in dubbogo.yml")
			return
		}

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
