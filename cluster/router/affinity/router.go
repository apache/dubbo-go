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

package affinity

import (
	"math"
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition"
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type ServiceAffinityRoute struct {
	affinityRoute
}

func newServiceAffinityRoute() *ServiceAffinityRoute {
	return &ServiceAffinityRoute{}
}

func (s *ServiceAffinityRoute) Notify(invokers []protocol.Invoker) {
	if len(invokers) == 0 {
		return
	}

	url := invokers[0].GetURL()
	if url == nil {
		logger.Error("Failed to notify a Service Affinity rule, because url is empty")
		return
	}

	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		logger.Infof("Config center does not start, Condition router will not be enabled")
		return
	}

	key := strings.Join([]string{url.ColonSeparatedKey(), constant.AffinityRuleSuffix}, "")
	dynamicConfiguration.AddListener(key, s)
	value, err := dynamicConfiguration.GetRule(key)
	if err != nil {
		logger.Errorf("Failed to query affinity rule, key=%s, err=%v", key, err)
		return
	}

	s.Process(&config_center.ConfigChangeEvent{Key: key, Value: value, ConfigType: remoting.EventTypeAdd})
}

type ApplicationAffinityRoute struct {
	affinityRoute
	application        string
	currentApplication string
}

func newApplicationAffinityRouter() *ApplicationAffinityRoute {
	applicationName := config.GetApplicationConfig().Name
	a := &ApplicationAffinityRoute{
		currentApplication: applicationName,
	}

	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration != nil {
		dynamicConfiguration.AddListener(strings.Join([]string{applicationName, constant.AffinityRuleSuffix}, ""), a)
	}
	return a
}

func (s *ApplicationAffinityRoute) Notify(invokers []protocol.Invoker) {
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

	providerApplication := url.GetParam("application", "")
	if providerApplication == "" || providerApplication == s.currentApplication {
		logger.Warn("condition router get providerApplication is empty, will not subscribe to provider app rules.")
		return
	}

	if providerApplication != s.application {
		if s.application != "" {
			dynamicConfiguration.RemoveListener(strings.Join([]string{s.application, constant.AffinityRuleSuffix}, ""), s)
		}
		s.application = providerApplication

		key := strings.Join([]string{providerApplication, constant.AffinityRuleSuffix}, "")
		dynamicConfiguration.AddListener(key, s)
		value, err := dynamicConfiguration.GetRule(key)
		if err != nil {
			logger.Errorf("Failed to query condition rule, key=%s, err=%v", key, err)
			return
		}

		s.Process(&config_center.ConfigChangeEvent{Key: key, Value: value, ConfigType: remoting.EventTypeUpdate})
	}
}

type affinityRoute struct {
	mu      sync.RWMutex
	matcher *condition.FieldMatcher
	enabled bool
	key     string
	ratio   int32
}

func (a *affinityRoute) Process(event *config_center.ConfigChangeEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.matcher, a.enabled, a.key, a.ratio = nil, false, "", 0

	switch event.ConfigType {
	case remoting.EventTypeDel:
	case remoting.EventTypeAdd, remoting.EventTypeUpdate:
		cfg, err := parseConfig(event.Value.(string))
		if err != nil {
			logger.Errorf("Failed to parse affinity config, key=%s, err=%v", a.key, err)
			return
		}

		if cfg.AffinityAware.Ratio < 0 || cfg.AffinityAware.Ratio > 100 {
			logger.Errorf("Failed to parse affinity config, affinity.ratio=%d, expect 0-100", a.ratio)
			return
		}

		key := strings.TrimSpace(cfg.AffinityAware.Key)
		if !cfg.Enabled || key == "" {
			return
		}
		rule := strings.Join([]string{key, key}, "=$")
		f, err := condition.NewFieldMatcher(rule)
		if err != nil {
			logger.Errorf("Failed to parse affinity config, key=%s, rule=%s ,err=%v", a.key, rule, err)
			return
		}

		a.matcher, a.enabled, a.key, a.ratio = &f, true, key, cfg.AffinityAware.Ratio
	}
}

func (a *affinityRoute) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if len(invokers) == 0 {
		return invokers
	}

	a.mu.RLock()
	enabled, matcher, ratio := a.enabled, a.matcher, a.ratio
	a.mu.RUnlock()

	if !enabled {
		return invokers
	}

	res := make([]protocol.Invoker, 0, len(invokers))
	for _, invoker := range invokers {
		if matcher.MatchInvoker(url, invoker, invocation) {
			res = append(res, invoker)
		}
	}
	if float32(len(res))/float32(len(invokers)) >= float32(ratio)/float32(100) {
		return res
	}

	return invokers
}

func (a *affinityRoute) URL() *common.URL {
	return nil
}

func (a *affinityRoute) Priority() int64 {
	// expect this router is the last one in the router chain
	return math.MinInt64
}

func (a *affinityRoute) Notify(_ []protocol.Invoker) {
	panic("this function should not be called")
}

func parseConfig(c string) (config.AffinityRouter, error) {
	res := config.AffinityRouter{}
	err := yaml.Unmarshal([]byte(c), &res)
	return res, err
}
