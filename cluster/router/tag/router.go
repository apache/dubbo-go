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
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type PriorityRouter struct {
	routerConfigs sync.Map
}

func NewTagPriorityRouter() (*PriorityRouter, error) {
	return &PriorityRouter{}, nil
}

// Route Determine the target invokers list.
func (p *PriorityRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if len(invokers) == 0 {
		logger.Warnf("[tag router] invokers from previous router is empty")
		return invokers
	}
	key := url.Service() + constant.TagRouterRuleSuffix
	value, ok := p.routerConfigs.Load(key)
	if !ok {
		return staticTag(invokers, url, invocation)
	}
	routerCfg := value.(config.RouterConfig)
	if !routerCfg.Enabled || !routerCfg.Valid {
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

func (p *PriorityRouter) Notify(invokers []protocol.Invoker) {
	if len(invokers) == 0 {
		return
	}
	service := invokers[0].GetURL().Service()
	if service == "" {
		logger.Error("url service is empty")
		return
	}
	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		logger.Warnf("config center does not start, please check if the configuration center has been properly configured in dubbogo.yml")
		return
	}
	key := service + constant.TagRouterRuleSuffix
	dynamicConfiguration.AddListener(key, p)
	value, err := dynamicConfiguration.GetRule(key)
	if err != nil {
		logger.Errorf("query router rule fail,key=%s,err=%v", key, err)
		return
	}
	p.Process(&config_center.ConfigChangeEvent{Key: key, Value: value, ConfigType: remoting.EventTypeAdd})

}

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

func parseRoute(routeContent string) (*config.RouterConfig, error) {
	routeDecoder := yaml.NewDecoder(strings.NewReader(routeContent))
	routerConfig := &config.RouterConfig{}
	err := routeDecoder.Decode(routerConfig)
	if err != nil {
		return nil, err
	}
	routerConfig.Valid = true
	if len(routerConfig.Tags) == 0 {
		routerConfig.Valid = false
	}
	return routerConfig, nil
}
