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

package script

import (
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"gopkg.in/yaml.v2"
)

import (
	ins "dubbo.apache.org/dubbo-go/v3/cluster/router/script/instance"
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// ScriptRouter only takes effect on consumers and only supports application granular management.
type ScriptRouter struct {
	applicationName string

	mu         sync.RWMutex
	enabled    bool
	scriptType string
	rawScript  string
}

func NewScriptRouter() *ScriptRouter {
	return &ScriptRouter{
		applicationName: "",
		enabled:         false,
	}
}

func parseRoute(routeContent string) (*config.RouterConfig, error) {
	routeDecoder := yaml.NewDecoder(strings.NewReader(routeContent))
	routerConfig := &config.RouterConfig{}
	err := routeDecoder.Decode(routerConfig)
	if err != nil {
		return nil, err
	}
	return routerConfig, nil
}

func (s *ScriptRouter) Process(event *config_center.ConfigChangeEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rawConf, ok := event.Value.(string)
	if !ok {
		panic(ok)
	}
	cfg, err := parseRoute(rawConf)
	if err != nil {
		logger.Errorf("Parse route cfg failed: %v", err)
		return
	}

	switch event.ConfigType {
	case remoting.EventTypeAdd, remoting.EventTypeUpdate:
		//destroy old instance
		if s.enabled && s.scriptType != "" {
			in, err := ins.GetInstances(s.scriptType)
			if err != nil {
				logger.Errorf("GetInstances failed to Destroy: %v", err)
			} else {
				in.Destroy(s.rawScript)
			}
		}
		// check new config
		if "" == cfg.ScriptType {
			logger.Errorf("`type` field must be set in config")
			return
		}
		if "" == cfg.Script {
			logger.Errorf("`script` field must be set in config")
			return
		}
		if "" == cfg.Key {
			logger.Errorf("`applicationName` field must be set in config")
			return
		}
		if !*cfg.Enabled {
			logger.Infof("`enabled` field equiles false, this rule will be ignored :%s", cfg.Script)
		}
		// rewrite to ScriptRouter
		s.enabled = *cfg.Enabled
		s.rawScript = cfg.Script
		s.scriptType = cfg.ScriptType

		// compile script
		in, err := ins.GetInstances(s.scriptType)
		if err != nil {
			logger.Errorf("GetInstances failed: %v", err)
			s.enabled = false
			return
		}
		if s.enabled {
			err = in.Compile(s.rawScript)
			// fail, disable rule
			if err != nil {
				s.enabled = false
				logger.Errorf("Compile Script failed: %v", err)
			}
		}

	case remoting.EventTypeDel:
		in, _ := ins.GetInstances(s.scriptType)

		if in != nil && s.enabled {
			in.Destroy(s.rawScript)
		}
		s.enabled = false
		s.rawScript = ""
		s.scriptType = ""
	}
}

func (s *ScriptRouter) runScript(scriptType, rawScript string, invokers []protocol.Invoker, invocation protocol.Invocation) ([]protocol.Invoker, error) {
	in, err := ins.GetInstances(scriptType)
	if err != nil {
		return nil, err
	}
	return in.Run(rawScript, invokers, invocation)
}

func (s *ScriptRouter) Route(invokers []protocol.Invoker, _ *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if invokers == nil || len(invokers) == 0 {
		return []protocol.Invoker{}
	}

	s.mu.RLock()
	enabled, scriptType, rawScript := s.enabled, s.scriptType, s.rawScript
	s.mu.RUnlock()

	if enabled == false || s.scriptType == "" || s.rawScript == "" {
		return invokers
	}

	res, err := s.runScript(scriptType, rawScript, invokers, invocation)
	if err != nil {
		logger.Warnf("ScriptRouter.Route error: %v", err)
	}

	return res
}

func (s *ScriptRouter) URL() *common.URL {
	return nil
}

func (s *ScriptRouter) Priority() int64 {
	return 0
}

func (s *ScriptRouter) Notify(invokers []protocol.Invoker) {
	if len(invokers) == 0 {
		return
	}
	url := invokers[0].GetURL()
	if url == nil {
		logger.Error("Failed to notify a dynamically Script rule, because url is empty")
		return
	}

	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		logger.Infof("Config center does not start, Script router will not be enabled")
		return
	}

	providerApplication := url.GetParam("application", "")
	if providerApplication == "" {
		logger.Warn("Script router get providerApplication is empty, will not subscribe to provider app rules.")
		return
	}

	var (
		listenTarget, value string
		err                 error
	)
	if providerApplication != s.applicationName {
		if s.applicationName != "" {
			dynamicConfiguration.RemoveListener(strings.Join([]string{s.applicationName, constant.ScriptRouterRuleSuffix}, ""), s)
		}

		listenTarget = strings.Join([]string{providerApplication, constant.ScriptRouterRuleSuffix}, "")
		dynamicConfiguration.AddListener(listenTarget, s)
		s.applicationName = providerApplication
		value, err = dynamicConfiguration.GetRule(listenTarget)
		if err != nil {
			logger.Errorf("Failed to query Script rule, applicationName=%s, listening=%s, err=%v", s.applicationName, listenTarget, err)
		}
		s.Process(&config_center.ConfigChangeEvent{Key: listenTarget, Value: value, ConfigType: remoting.EventTypeUpdate})
	}
}
