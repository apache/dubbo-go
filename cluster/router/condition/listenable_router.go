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
	"github.com/RoaringBitmap/roaring"
	"github.com/apache/dubbo-go/cluster/router"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
)

import (
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

type conditionRouterSnapshot struct {
	routers []*ConditionRouter
}

func (s *conditionRouterSnapshot) Source() string {
	return "listenable-router"
}

// listenableRouter Abstract router which listens to dynamic configuration
type listenableRouter struct {
	conditionRouters []*ConditionRouter
	routerRule       *RouterRule
	url              *common.URL
	routerKey        string
	changed          bool
	force            bool
	priority         int64
	mutex            sync.RWMutex
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
	l.routerKey = routerKey
	l.changed = true

	//add listener
	dynamicConfiguration := config.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		return nil, perrors.Errorf("Get dynamicConfiguration fail, dynamicConfiguration is nil, init config center plugin please")
	}

	dynamicConfiguration.AddListener(routerKey, l)
	//get rule
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

// Process Process config change event , generate routers and set them to the listenableRouter instance
func (l *listenableRouter) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("Notification of condition rule, change type is:[%s] , raw rule is:[%v]", event.ConfigType, event.Value)
	if remoting.EventTypeDel == event.ConfigType {
		l.routerRule = nil
		l.mutex.Lock()
		l.mutex.Unlock()
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

	l.mutex.Lock()
	defer l.mutex.Unlock()
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
	l.changed = true
}

// Route Determine the target invokers list.
func (l *listenableRouter) Route(invokers *roaring.Bitmap, cache *router.AddrCache, url *common.URL, invocation protocol.Invocation) *roaring.Bitmap {
	meta := cache.FindAddrMeta(l)
	conditionRouters := meta.(*conditionRouterSnapshot).routers

	if invokers.IsEmpty() || len(conditionRouters) == 0 {
		return invokers
	}

	//We will check enabled status inside each router.
	newCache := l.convertCache(cache, conditionRouters)
	for _, r := range conditionRouters {
		invokers = r.Route(invokers, newCache, url, invocation)
	}
	return invokers
}

func (l *listenableRouter) convertCache(cache *router.AddrCache, conditionRouters []*ConditionRouter) *router.AddrCache {
	pool := cache.FindAddrPool(l)
	pools := make(map[string]router.RouterAddrPool)
	for _, r := range conditionRouters {
		rb := pool[r.Name()]
		rp := make(router.RouterAddrPool)
		rp["matched"] = rb
		pools[r.Name()] = rp
	}

	newCache := &router.AddrCache{
		Invokers: cache.Invokers,
		Bitmap:   cache.Bitmap,
		AddrPool: pools,
	}
	return newCache
}

func (l *listenableRouter) Pool(invokers []protocol.Invoker) (router.RouterAddrPool, router.AddrMetadata) {
	l.mutex.RLock()
	routers := make([]*ConditionRouter, len(l.conditionRouters))
	copy(routers, l.conditionRouters)
	l.mutex.RUnlock()

	rb := make(router.RouterAddrPool)
	for _, r := range routers {
		pool, _ := r.Pool(invokers)
		rb[r.Name()] = pool["matched"]
	}
	return rb, &conditionRouterSnapshot{routers}
}

func (l *listenableRouter) ShouldRePool() bool {
	if l.changed {
		l.changed = false
		return true
	} else {
		return false
	}
}

func (l *listenableRouter) Name() string {
	return l.routerKey
}

// Priority Return Priority in listenable router
func (l *listenableRouter) Priority() int64 {
	return l.priority
}

// URL Return URL in listenable router
func (l *listenableRouter) URL() common.URL {
	return *l.url
}
