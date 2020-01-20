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
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/remoting"
)

const (
	ROUTER_NAME      = "LISTENABLE_ROUTER"
	RULE_SUFFIX      = ".condition-router"
	DEFAULT_PRIORITY = ^int64(0)
)

//ListenableRouter Abstract router which listens to dynamic configuration
type listenableRouter struct {
	conditionRouters []*ConditionRouter
	routerRule       *RouterRule
	url              *common.URL
	force            bool
	priority         int64
}

func (l *listenableRouter) newListenableRouter(url *common.URL, ruleKey string) error {
	l.url = url
	l.priority = DEFAULT_PRIORITY
	if ruleKey == "" {
		return nil
	}

	routerKey := ruleKey + RULE_SUFFIX
	//TODO: add listener

	//TODO: add get rule
	rule := "rule"
	if rule != "" {
		l.Process(&config_center.ConfigChangeEvent{
			Key:        routerKey,
			Value:      rule,
			ConfigType: remoting.EventTypeUpdate})
	}
}

func (l *listenableRouter) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("Notification of condition rule, change type is:[%s] , raw rule is:[%v]", event.ConfigType, event.Value)
	if remoting.EventTypeDel == event.ConfigType {
		l.routerRule = nil
		l.conditionRouters = make([]*ConditionRouter, 0)
		return
	}
	content, ok := event.Value.(string)
	if !ok {
		logger.Errorf("Convert event content fail,raw content:[%s] ", event.Value)
		return
	}

	routerRule, err := Parse(content)
	if err != nil {
		logger.Errorf("Parse condition router rule fail,error:[%s] ", err)
		return
	}
	l.generateConditions(routerRule)
}

func (l *listenableRouter) generateConditions(rule *RouterRule) {
	if rule == nil || !rule.Valid {
		return
	}
	l.conditionRouters = make([]*ConditionRouter, 0)
	for _, c := range rule.Conditions {
		router, e := NewConditionRouterWithRule(c)
		if e != nil {
			logger.Errorf("Create condition router with rule fail,raw rule:[%s] ", c)
			continue
		}
		router.Force = rule.Force
		router.enabled = rule.Enabled
		l.conditionRouters = append(l.conditionRouters, router)
	}
}

func (l *listenableRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if len(invokers) == 0 || len(l.conditionRouters) == 0 {
		return invokers
	}
	//We will check enabled status inside each router.
	for _, r := range l.conditionRouters {
		invokers = r.Route(invokers, url, invocation)
	}
	return invokers
}

func (l *listenableRouter) Priority() int64 {
	return l.priority
}

func (l *listenableRouter) Url() common.URL {
	return *l.url
}
