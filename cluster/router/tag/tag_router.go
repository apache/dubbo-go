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
	"net"
	"strconv"
)

import (
	gxnet "github.com/dubbogo/gost/net"
	"github.com/jinzhu/copier"
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

// tagRouter defines url, enable and the priority
type tagRouter struct {
	url           *common.URL
	tagRouterRule *RouterRule
	enabled       bool
	priority      int64
	application   string
}

// NewTagRouter returns a tagRouter instance if url is not nil
func NewTagRouter(url *common.URL) (*tagRouter, error) {
	if url == nil {
		return nil, perrors.Errorf("Illegal route URL!")
	}
	return &tagRouter{
		url:      url,
		enabled:  url.GetParamBool(constant.RouterEnabled, true),
		priority: url.GetParamInt(constant.RouterPriority, 0),
	}, nil
}

// nolint
func (c *tagRouter) isEnabled() bool {
	return c.enabled
}

// Route gets a list of invoker
func (c *tagRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	var (
		result    []protocol.Invoker
		addresses []string
		tag       string
	)

	if !c.isEnabled() || len(invokers) == 0 {
		return invokers
	}

	// Use static tags if dynamic tags are not set or invalid
	if c.tagRouterRule == nil || !c.tagRouterRule.Valid || !c.tagRouterRule.Enabled {
		return filterUsingStaticTag(invokers, url, invocation)
	}

	// since the rule can be changed by config center, we should copy one to use.
	tagRouterRuleCopy := new(RouterRule)
	_ = copier.Copy(tagRouterRuleCopy, c.tagRouterRule)
	tagValue, ok := invocation.Attachments()[constant.Tagkey]
	if !ok {
		tag = url.GetParam(constant.Tagkey, "")
	} else {
		tag, ok = tagValue.(string)
		if !ok {
			tag = url.GetParam(constant.Tagkey, "")
		}
	}

	// if we are requesting for a Provider with a specific tag
	if len(tag) > 0 {
		return filterInvokersWithTag(invokers, url, invocation, *tagRouterRuleCopy, tag)
	}

	// return all addresses in dynamic tag group.
	addresses = tagRouterRuleCopy.getAddresses()
	if len(addresses) > 0 {
		filterAddressNotMatches := func(invoker protocol.Invoker) bool {
			url := invoker.GetUrl()
			return len(addresses) == 0 || !checkAddressMatch(addresses, url.Ip, url.Port)
		}
		result = filterInvoker(invokers, filterAddressNotMatches)
		// 1. all addresses are in dynamic tag group, return empty list.
		if len(result) == 0 {
			return result
		}
	}
	// 2. if there are some addresses that are not in any dynamic tag group, continue to filter using the
	// static tag group.
	filter := func(invoker protocol.Invoker) bool {
		localTag := invoker.GetUrl().GetParam(constant.Tagkey, "")
		return len(localTag) == 0 || !(tagRouterRuleCopy.hasTag(localTag))
	}
	return filterInvoker(result, filter)
}

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
	c.tagRouterRule = routerRule
	return
}

func (c *tagRouter) Notify(invokers []protocol.Invoker) {
	if len(invokers) == 0 {
		return
	}
	invoker := invokers[0]
	url := invoker.GetUrl()
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
	}

	routerKey := providerApplication + constant.TagRouterRuleSuffix
	dynamicConfiguration.AddListener(routerKey, c)
	//get rule
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

// URL gets the url of tagRouter
func (c *tagRouter) URL() common.URL {
	return *c.url
}

// Priority gets the priority of tagRouter
func (c *tagRouter) Priority() int64 {
	return c.priority
}

// filterUsingStaticTag gets a list of invoker using static tag
func filterUsingStaticTag(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if tag, ok := invocation.Attachments()[constant.Tagkey]; ok {
		result := make([]protocol.Invoker, 0, 8)
		for _, v := range invokers {
			if v.GetUrl().GetParam(constant.Tagkey, "") == tag {
				result = append(result, v)
			}
		}
		if len(result) == 0 && !isForceUseTag(url, invocation) {
			return invokers
		}
		return result
	}
	return invokers
}

// filterInvokersWithTag gets a list of invoker using dynamic route with tag
func filterInvokersWithTag(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation, tagRouterRule RouterRule, tag string) []protocol.Invoker {
	var (
		result    []protocol.Invoker
		addresses []string
	)
	addresses, _ = tagRouterRule.getTagNameToAddresses()[tag]
	// filter by dynamic tag group first
	if len(addresses) > 0 {
		filterAddressMatches := func(invoker protocol.Invoker) bool {
			url := invoker.GetUrl()
			return len(addresses) > 0 && checkAddressMatch(addresses, url.Ip, url.Port)
		}
		result = filterInvoker(invokers, filterAddressMatches)
		if len(result) > 0 || tagRouterRule.Force {
			return result
		}
	}
	// dynamic tag group doesn't have any item about the requested app OR it's null after filtered by
	// dynamic tag group but force=false. check static tag
	filter := func(invoker protocol.Invoker) bool {
		return invoker.GetUrl().GetParam(constant.Tagkey, "") == tag
	}
	result = filterInvoker(invokers, filter)
	// If there's no tagged providers that can match the current tagged request. force.tag is set by default
	// to false, which means it will invoke any providers without a tag unless it's explicitly disallowed.
	if len(result) > 0 || isForceUseTag(url, invocation) {
		return result
	}
	// FAILOVER: return all Providers without any tags.
	filterAddressNotMatches := func(invoker protocol.Invoker) bool {
		url := invoker.GetUrl()
		return len(addresses) == 0 || !checkAddressMatch(tagRouterRule.getAddresses(), url.Ip, url.Port)
	}
	filterTagIsEmpty := func(invoker protocol.Invoker) bool {
		return invoker.GetUrl().GetParam(constant.Tagkey, "") == ""
	}
	return filterInvoker(invokers, filterAddressNotMatches, filterTagIsEmpty)
}

// isForceUseTag returns whether force use tag
func isForceUseTag(url *common.URL, invocation protocol.Invocation) bool {
	if b, e := strconv.ParseBool(invocation.AttachmentsByKey(constant.ForceUseTag, url.GetParam(constant.ForceUseTag, "false"))); e == nil {
		return b
	}
	return false
}

type filter func(protocol.Invoker) bool

func filterInvoker(invokers []protocol.Invoker, filters ...filter) []protocol.Invoker {
	var res []protocol.Invoker
OUTER:
	for _, invoker := range invokers {
		for _, filter := range filters {
			if !filter(invoker) {
				continue OUTER
			}
		}
		res = append(res, invoker)
	}
	return res
}

func checkAddressMatch(addresses []string, host, port string) bool {
	addr := net.JoinHostPort(constant.ANYHOST_VALUE, port)
	for _, address := range addresses {
		if gxnet.MatchIP(address, host, port) {
			return true
		}
		if address == addr {
			return true
		}
	}
	return false
}
