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
	"strconv"
	"strings"
	"sync"
)

import (
	"github.com/RoaringBitmap/roaring"
	gxnet "github.com/dubbogo/gost/net"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/utils"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/config"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/remoting"
)

const (
	name          = "tag-router"
	staticPrefix  = "static-"
	dynamicPrefix = "dynamic-"
)

// addrMetadata keeps snapshot data for app name, and some infos extracted from dynamic tag rule in order to make
// Route() method lock-free.
type addrMetadata struct {
	// application name
	application string
	// is rule a runtime rule
	//ruleRuntime bool
	// is rule a force rule
	ruleForce bool
	// is rule a valid rule
	ruleValid bool
	// is rule an enabled rule
	ruleEnabled bool
}

// Source indicates where the metadata comes from.
func (m *addrMetadata) Source() string {
	return name
}

// tagRouter defines url, enable and the priority
type tagRouter struct {
	url           *common.URL
	tagRouterRule *RouterRule
	enabled       bool
	priority      int64
	application   string
	ruleChanged   bool
	mutex         sync.RWMutex
	notify        chan struct{}
}

// NewTagRouter returns a tagRouter instance if url is not nil
func NewTagRouter(url *common.URL, notify chan struct{}) (*tagRouter, error) {
	if url == nil {
		return nil, perrors.Errorf("Illegal route URL!")
	}
	return &tagRouter{
		url:      url,
		enabled:  url.GetParamBool(constant.RouterEnabled, true),
		priority: url.GetParamInt(constant.RouterPriority, 0),
		notify:   notify,
	}, nil
}

// nolint
func (c *tagRouter) isEnabled() bool {
	return c.enabled
}

// Route gets a list of invoker
func (c *tagRouter) Route(invokers *roaring.Bitmap, cache router.Cache, url *common.URL, invocation protocol.Invocation) *roaring.Bitmap {
	if !c.isEnabled() || invokers.IsEmpty() {
		return invokers
	}

	if shouldUseDynamicTag(cache.FindAddrMeta(c)) {
		return c.routeWithDynamicTag(invokers, cache, url, invocation)
	}
	return c.routeWithStaticTag(invokers, cache, url, invocation)
}

// routeWithStaticTag routes with static tag rule
func (c *tagRouter) routeWithStaticTag(invokers *roaring.Bitmap, cache router.Cache, url *common.URL, invocation protocol.Invocation) *roaring.Bitmap {
	tag := findTag(invocation, url)
	if tag == "" {
		return invokers
	}

	ret, _ := c.filterWithTag(invokers, cache, staticPrefix+tag)
	if ret.IsEmpty() && !isForceUseTag(url, invocation) {
		return invokers
	}

	return ret
}

// routeWithDynamicTag routes with dynamic tag rule
func (c *tagRouter) routeWithDynamicTag(invokers *roaring.Bitmap, cache router.Cache, url *common.URL, invocation protocol.Invocation) *roaring.Bitmap {
	tag := findTag(invocation, url)
	if tag == "" {
		return c.filterNotInDynamicTag(invokers, cache)
	}

	ret, ok := c.filterWithTag(invokers, cache, dynamicPrefix+tag)
	if ok && (!ret.IsEmpty() || isTagRuleForce(cache.FindAddrMeta(c))) {
		return ret
	}

	// dynamic tag group doesn't have any item about the requested app OR it's null after filtered by
	// dynamic tag group but force=false. check static tag
	if ret.IsEmpty() {
		ret, _ = c.filterWithTag(invokers, cache, staticPrefix+tag)
		// If there's no tagged providers that can match the current tagged request. force.tag is set by default
		// to false, which means it will invoke any providers without a tag unless it's explicitly disallowed.
		if !ret.IsEmpty() || isForceUseTag(url, invocation) {
			return ret
		}
		return c.filterNotInDynamicTag(invokers, cache)
	}
	return ret
}

// filterWithTag filters incoming invokers with the given tag
func (c *tagRouter) filterWithTag(invokers *roaring.Bitmap, cache router.Cache, tag string) (*roaring.Bitmap, bool) {
	if target, ok := cache.FindAddrPool(c)[tag]; ok {
		return utils.JoinIfNotEqual(target, invokers), true
	}
	return utils.EmptyAddr, false
}

// filterNotInDynamicTag filters incoming invokers not applied to dynamic tag rule
func (c *tagRouter) filterNotInDynamicTag(invokers *roaring.Bitmap, cache router.Cache) *roaring.Bitmap {
	// FAILOVER: return all Providers without any tags.
	invokers = invokers.Clone()
	for k, v := range cache.FindAddrPool(c) {
		if strings.HasPrefix(k, dynamicPrefix) {
			invokers.AndNot(v)
		}
	}
	return invokers
}

// Process parses dynamic tag rule
func (c *tagRouter) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("Notification of tag rule, change type is:[%s] , raw rule is:[%v]", event.ConfigType, event.Value)
	if remoting.EventTypeDel == event.ConfigType {
		c.tagRouterRule = nil
		return
	}
	content, ok := event.Value.(string)
	if !ok {
		logger.Errorf("Convert event content fail,raw content:[%s] ", event.Value)
		return
	}

	routerRule, err := getRule(content)
	if err != nil {
		logger.Errorf("Parse dynamic tag router rule fail,error:[%s] ", err)
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tagRouterRule = routerRule
	c.ruleChanged = true
	c.notify <- struct{}{}
}

// URL gets the url of tagRouter
func (c *tagRouter) URL() *common.URL {
	return c.url
}

// Priority gets the priority of tagRouter
func (c *tagRouter) Priority() int64 {
	return c.priority
}

// Pool divided invokers into different address pool by tag.
func (c *tagRouter) Pool(invokers []protocol.Invoker) (router.AddrPool, router.AddrMetadata) {
	c.fetchRuleIfNecessary(invokers)

	rb := make(router.AddrPool, 8)
	poolWithStaticTag(invokers, rb)

	c.mutex.Lock()
	defer c.mutex.Unlock()
	poolWithDynamicTag(invokers, c.tagRouterRule, rb)
	c.ruleChanged = false
	// create metadata in order to avoid lock in route()
	meta := addrMetadata{application: c.application}
	if c.tagRouterRule != nil {
		meta.ruleForce = c.tagRouterRule.Force
		meta.ruleEnabled = c.tagRouterRule.Enabled
		meta.ruleValid = c.tagRouterRule.Valid
	}

	return rb, &meta
}

// fetchRuleIfNecessary fetches, parses rule and register listener for the further change
func (c *tagRouter) fetchRuleIfNecessary(invokers []protocol.Invoker) {
	if len(invokers) == 0 {
		return
	}

	url := invokers[0].GetUrl()
	providerApplication := url.GetParam(constant.RemoteApplicationKey, "")
	if len(providerApplication) == 0 {
		logger.Error("TagRouter must getConfig from or subscribe to a specific application, but the application " +
			"in this TagRouter is not specified.")
		return
	}
	dynamicConfiguration := config.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		logger.Error("Get dynamicConfiguration fail, dynamicConfiguration is nil, init config center plugin please")
		return
	}

	if providerApplication != c.application {
		dynamicConfiguration.RemoveListener(c.application+constant.TagRouterRuleSuffix, c)
	} else {
		// if app name from URL is as same as the current app name, then it is safe to jump out
		return
	}

	c.application = providerApplication
	routerKey := providerApplication + constant.TagRouterRuleSuffix
	dynamicConfiguration.AddListener(routerKey, c)
	// get rule
	rule, err := dynamicConfiguration.GetRule(routerKey, config_center.WithGroup(config_center.DEFAULT_GROUP))
	if len(rule) == 0 || err != nil {
		logger.Errorf("Get rule fail, config rule{%s},  error{%v}", rule, err)
		return
	}
	if len(rule) > 0 {
		c.Process(&config_center.ConfigChangeEvent{
			Key:        routerKey,
			Value:      rule,
			ConfigType: remoting.EventTypeUpdate})
	}
}

// ShouldPool returns false, to make sure address cache for tag router happens once and only once.
func (c *tagRouter) ShouldPool() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.ruleChanged
}

// Name returns pool's name
func (c *tagRouter) Name() string {
	return name
}

// poolWithDynamicTag pools addresses with the tags defined in dynamic tag rule, all keys have prefix "dynamic-"
func poolWithDynamicTag(invokers []protocol.Invoker, rule *RouterRule, pool router.AddrPool) {
	if rule == nil {
		return
	}

	tagNameToAddresses := rule.getTagNameToAddresses()
	for tag, addrs := range tagNameToAddresses {
		pool[dynamicPrefix+tag] = addrsToBitmap(addrs, invokers)
	}
}

// poolWithStaticTag pools addresses with tags found from incoming URLs, all keys have prefix "static-"
func poolWithStaticTag(invokers []protocol.Invoker, pool router.AddrPool) {
	for i, invoker := range invokers {
		url := invoker.GetUrl()
		tag := url.GetParam(constant.Tagkey, "")
		if len(tag) > 0 {
			if _, ok := pool[staticPrefix+tag]; !ok {
				pool[staticPrefix+tag] = roaring.NewBitmap()
			}
			pool[staticPrefix+tag].AddInt(i)
		}
	}
}

// shouldUseDynamicTag uses the snapshot data from the parsed rule to decide if dynamic tag rule should be used or not
func shouldUseDynamicTag(meta router.AddrMetadata) bool {
	return meta.(*addrMetadata).ruleValid && meta.(*addrMetadata).ruleEnabled
}

// isTagRuleForce uses the snapshot data from the parsed rule to decide if dynamic tag rule is forced or not
func isTagRuleForce(meta router.AddrMetadata) bool {
	return meta.(*addrMetadata).ruleForce
}

// isForceUseTag returns whether force use tag
func isForceUseTag(url *common.URL, invocation protocol.Invocation) bool {
	if b, e := strconv.ParseBool(invocation.AttachmentsByKey(constant.ForceUseTag, url.GetParam(constant.ForceUseTag, "false"))); e == nil {
		return b
	}
	return false
}

// addrsToBitmap finds indexes for the given IP addresses in the target URL list, if any '0.0.0.0' IP address is met,
// then returns back all indexes of the URLs list.
func addrsToBitmap(addrs []string, invokers []protocol.Invoker) *roaring.Bitmap {
	ret := roaring.NewBitmap()
	for _, addr := range addrs {
		if isAnyHost(addr) {
			ret.AddRange(0, uint64(len(invokers)))
			return ret
		}

		index := findIndexWithIp(addr, invokers)
		if index != -1 {
			ret.AddInt(index)
		}
	}
	return ret
}

// findIndexWithIp finds index for one particular IP
func findIndexWithIp(addr string, invokers []protocol.Invoker) int {
	for i, invoker := range invokers {
		if gxnet.MatchIP(addr, invoker.GetUrl().Ip, invoker.GetUrl().Port) {
			return i
		}
	}
	return -1
}

// isAnyHost checks if an IP is '0.0.0.0'
func isAnyHost(addr string) bool {
	return strings.HasPrefix(addr, constant.ANYHOST_VALUE)
}

// findTag finds tag, first from invocation's attachment, then from URL
func findTag(invocation protocol.Invocation, consumerUrl *common.URL) string {
	tag, ok := invocation.Attachments()[constant.Tagkey]
	if !ok {
		return consumerUrl.GetParam(constant.Tagkey, "")
	} else if v, t := tag.(string); t {
		return v
	}
	return ""
}
