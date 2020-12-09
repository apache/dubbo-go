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
)

import (
	"github.com/RoaringBitmap/roaring"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/config"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/remoting"
)

const (
	// Default priority for listenable router, use the maximum int64 value
	listenableRouterDefaultPriority = ^int64(0)
)

// listenableRouter Abstract router which listens to dynamic configuration
type listenableRouter struct {
	conditionRouters []*ConditionRouter
	routerRule       *RouterRule
	url              *common.URL
	force            bool
	priority         int64
}

// RouterRule Get RouterRule instance from listenableRouter
func (l *listenableRouter) RouterRule() *RouterRule {
	return l.routerRule
}

func newListenableRouter(url *common.URL, ruleKey string) (*AppRouter, error) {
	if ruleKey == "" {
		return nil, perrors.Errorf("NewListenableRouter ruleKey is nil, can't create Listenable router")
	}
	l := &AppRouter{}

	l.url = url
	l.priority = listenableRouterDefaultPriority

	routerKey := ruleKey + constant.ConditionRouterRuleSuffix
	// add listener
	dynamicConfiguration := config.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		return nil, perrors.Errorf("Get dynamicConfiguration fail, dynamicConfiguration is nil, init config center plugin please")
	}

	dynamicConfiguration.AddListener(routerKey, l)
	// get rule
	rule, err := dynamicConfiguration.GetRule(routerKey, config_center.WithGroup(config_center.DEFAULT_GROUP))
	if len(rule) == 0 || err != nil {
		return nil, perrors.Errorf("Get rule fail, config rule{%s},  error{%v}", rule, err)
	}
	l.Process(&config_center.ConfigChangeEvent{
		Key:        routerKey,
		Value:      rule,
		ConfigType: remoting.EventTypeUpdate})

	logger.Info("Init app router success")
	return l, nil
}

// Process Process config change event, generate routers and set them to the listenableRouter instance
func (l *listenableRouter) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("Notification of condition rule, change type is:[%s] , raw rule is:[%v]", event.ConfigType, event.Value)
	if remoting.EventTypeDel == event.ConfigType {
		l.routerRule = nil
		if l.conditionRouters != nil {
			l.conditionRouters = l.conditionRouters[:0]
		}
		return
	}
	content, ok := event.Value.(string)
	if !ok {
		msg := fmt.Sprintf("Convert event content fail,raw content:[%s] ", event.Value)
		logger.Error(msg)
		return
	}

	routerRule, err := getRule(content)
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
	l.conditionRouters = make([]*ConditionRouter, 0, len(rule.Conditions))
	l.routerRule = rule
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

// Route Determine the target invokers list.
func (l *listenableRouter) Route(invokers *roaring.Bitmap, cache router.Cache, url *common.URL, invocation protocol.Invocation) *roaring.Bitmap {
	if invokers.IsEmpty() || len(l.conditionRouters) == 0 {
		return invokers
	}
	// We will check enabled status inside each router.
	for _, r := range l.conditionRouters {
		invokers = r.Route(invokers, cache, url, invocation)
	}
	return invokers
}

// Priority Return Priority in listenable router
func (l *listenableRouter) Priority() int64 {
	return l.priority
}

// URL Return URL in listenable router
func (l *listenableRouter) URL() *common.URL {
	return l.url
}
