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
	ins "dubbo.apache.org/dubbo-go/v3/cluster/router/script/instance"
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"github.com/dubbogo/gost/log/logger"
	"gopkg.in/yaml.v3"
	"strings"
	"sync"
)

// ScriptRouter only takes effect on consumers and only supports application granular management.
type ScriptRouter struct {
	mu         sync.RWMutex
	scriptType string
	key        string // key to application - name
	enabled    bool   // enabled
	rawScript  string
}

func NewScriptRouter() *ScriptRouter {
	applicationName := config.GetApplicationConfig().Name
	a := &ScriptRouter{
		key:     applicationName,
		enabled: false,
	}

	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration != nil {
		dynamicConfiguration.AddListener(strings.Join([]string{applicationName, constant.ScriptRouterRuleSuffix}, ""), a)
	}
	return a
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
	checkConfig := func(*config.RouterConfig) bool {
		if "" == cfg.ScriptType {
			logger.Errorf("`type` field must be set in config")
			return false
		}
		if "" == cfg.Script {
			logger.Errorf("`script` field must be set in config")
			return false
		}
		if "" == cfg.Key {
			logger.Errorf("`key` field must be set in config")
			return false
		}
		if cfg.Key != config.GetApplicationConfig().Name {
			logger.Errorf("`key` not equal applicationName , script route config load fail")
			return false
		}
		if !*cfg.Enabled {
			logger.Infof("`enabled` field equiles false, this rule will be ignored :%s", cfg.Script)
			return false
		}
		return true
	}
	switch event.ConfigType {
	case remoting.EventTypeAdd:
		if !checkConfig(cfg) {
			return
		}

		in, err := ins.GetInstances(cfg.ScriptType)
		if err != nil {
			logger.Errorf("GetInstances failed: %v", err)
		}

		err = in.Compile(cfg.Key, cfg.Script)
		if err != nil {
			logger.Errorf("Compile Script failed: %v", err)
		}
		s.enabled = true
	case remoting.EventTypeDel:
		s.enabled = false

		ins.RangeInstances(func(instance ins.ScriptInstances) bool {
			instance.Destroy()
			return true
		})
	case remoting.EventTypeUpdate:
		if !checkConfig(cfg) {
			return
		}
		in, err := ins.GetInstances(cfg.ScriptType)
		if err != nil {
			logger.Errorf("GetInstances failed: %v", err)
		}
		err = in.Compile(cfg.Key, cfg.Script)
		if err != nil {
			logger.Errorf("Compile Script failed: %v", err)
		}
		s.enabled = true
	}
}

func (s *ScriptRouter) runScript(scriptType, rawScript string, invokers []protocol.Invoker, invocation protocol.Invocation) ([]protocol.Invoker, error) {
	in, err := ins.GetInstances(scriptType)
	if err != nil {
		return nil, err
	}
	return in.RunScript(rawScript, invokers, invocation)
}

func (s *ScriptRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if invokers == nil || len(invokers) == 0 {
		return []protocol.Invoker{}
	}
	if s.enabled == false {
		return invokers
	}
	res, err := s.runScript(s.scriptType, s.rawScript, invokers, invocation)
	if err != nil {
		logger.Warnf("ScriptRouter.Route error: %v", err)
		return []protocol.Invoker{}
	}
	return res
}

func (s *ScriptRouter) URL() *common.URL {
	return nil
}

func (s *ScriptRouter) Priority() int64 {
	return 0
}

func (s *ScriptRouter) Notify(_ []protocol.Invoker) {
}
