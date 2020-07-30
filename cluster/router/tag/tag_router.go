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
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
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

func (c *tagRouter) SetApplication(app string) {
	c.application = app
}

func (c *tagRouter) tagRouterRuleCopy() RouterRule {
	fmt.Println(c.tagRouterRule, "fuck")
	routerRule := *c.tagRouterRule
	return routerRule
}

// Route gets a list of invoker
func (c *tagRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if !c.isEnabled() {
		return invokers
	}
	if len(invokers) == 0 {
		return invokers
	}
	if c.tagRouterRule == nil || !c.tagRouterRule.Valid || !c.tagRouterRule.Enabled {
		return filterUsingStaticTag(invokers, url, invocation)
	}
	// since the rule can be changed by config center, we should copy one to use.
	tagRouterRuleCopy := c.tagRouterRuleCopy()
	tag, ok := invocation.Attachments()[constant.Tagkey]
	if !ok {
		tag = url.GetParam(constant.Tagkey, "")
	}
	var (
		result    []protocol.Invoker
		addresses []string
	)
	if tag != "" {
		addresses, _ = tagRouterRuleCopy.getTagNameToAddresses()[tag]
		// filter by dynamic tag group first
		if len(addresses) > 0 {
			filterAddressMatches := func(invoker protocol.Invoker) bool {
				url := invoker.GetUrl()
				if len(addresses) > 0 && checkAddressMatch(addresses, url.Ip, url.Port) {
					return true
				}
				return false
			}
			result = filterInvoker(invokers, filterAddressMatches)
			if len(result) > 0 || tagRouterRuleCopy.Force {
				return result
			}
		} else {
			// dynamic tag group doesn't have any item about the requested app OR it's null after filtered by
			// dynamic tag group but force=false. check static tag
			filter := func(invoker protocol.Invoker) bool {
				if invoker.GetUrl().GetParam(constant.Tagkey, "") == tag {
					return true
				}
				return false
			}
			result = filterInvoker(invokers, filter)
		}
		// If there's no tagged providers that can match the current tagged request. force.tag is set by default
		// to false, which means it will invoke any providers without a tag unless it's explicitly disallowed.
		if len(result) > 0 || isForceUseTag(url, invocation) {
			return result
		} else {
			// FAILOVER: return all Providers without any tags.
			filterAddressNotMatches := func(invoker protocol.Invoker) bool {
				url := invoker.GetUrl()
				if len(addresses) == 0 || !checkAddressMatch(tagRouterRuleCopy.getAddresses(), url.Ip, url.Port) {
					return true
				}
				return false
			}
			filterTagIsEmpty := func(invoker protocol.Invoker) bool {
				if invoker.GetUrl().GetParam(constant.Tagkey, "") == "" {
					return true
				}
				return false
			}
			return filterInvoker(invokers, filterAddressNotMatches, filterTagIsEmpty)
		}
	} else {
		// return all addresses in dynamic tag group.
		addresses = tagRouterRuleCopy.getAddresses()
		if len(addresses) > 0 {
			filterAddressNotMatches := func(invoker protocol.Invoker) bool {
				url := invoker.GetUrl()
				if len(addresses) == 0 || !checkAddressMatch(addresses, url.Ip, url.Port) {
					return true
				}
				return false
			}
			result = filterInvoker(invokers, filterAddressNotMatches)
			// 1. all addresses are in dynamic tag group, return empty list.
			if len(result) == 0 {
				return result
			}
			// 2. if there are some addresses that are not in any dynamic tag group, continue to filter using the
			// static tag group.
		}
		filter := func(invoker protocol.Invoker) bool {
			localTag := invoker.GetUrl().GetParam(constant.Tagkey, "")
			return localTag == "" || !(tagRouterRuleCopy.hasTag(localTag))
		}
		return filterInvoker(result, filter)
	}
}

func (c *tagRouter) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("Notification of tag rule, change type is:[%s] , raw rule is:[%v]", event.ConfigType, event.Value)
	if remoting.EventTypeDel == event.ConfigType {
		c.tagRouterRule = nil
		return
	} else {
		content, ok := event.Value.(string)
		if !ok {
			msg := fmt.Sprintf("Convert event content fail,raw content:[%s] ", event.Value)
			logger.Error(msg)
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
}

func (c *tagRouter) Notify(invokers []protocol.Invoker) {
	if len(invokers) == 0 {
		return
	}
	invoker := invokers[0]
	url := invoker.GetUrl()
	providerApplication := url.GetParam(constant.RemoteApplicationKey, "")
	if providerApplication == "" {
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
	if rule != "" {
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

// filterUsingStaticTag gets a list of invoker using static tag, If there's no dynamic tag rule being set, use static tag in URL
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

// TODO 需要搬到 dubbogo/gost, 可以先 review
func checkAddressMatch(addresses []string, host, port string) bool {
	for _, address := range addresses {
		if matchIp(address, host, port) {
			return true
		}
		if address == constant.ANYHOST_VALUE+":"+port {
			return true
		}
	}
	return false
}

func matchIp(pattern, host, port string) bool {
	// if the pattern is subnet format, it will not be allowed to config port param in pattern.
	if strings.Contains(pattern, "/") {
		_, subnet, _ := net.ParseCIDR(pattern)
		if subnet != nil && subnet.Contains(net.ParseIP(host)) {
			return true
		}
		return false
	}
	return matchIpRange(pattern, host, port)
}

func matchIpRange(pattern, host, port string) bool {
	if pattern == "" || host == "" {
		logger.Error("Illegal Argument pattern or hostName. Pattern:" + pattern + ", Host:" + host)
		return false
	}

	pattern = strings.TrimSpace(pattern)
	if "*.*.*.*" == pattern || "*" == pattern {
		return true
	}

	isIpv4 := true
	ip4 := net.ParseIP(host).To4()

	if ip4 == nil {
		isIpv4 = false
	}

	hostAndPort := getPatternHostAndPort(pattern, isIpv4)
	if hostAndPort[1] != "" && hostAndPort[1] != port {
		return false
	}

	pattern = hostAndPort[0]
	// TODO 常量化
	splitCharacter := "\\."
	if !isIpv4 {
		splitCharacter = ":"
	}

	mask := strings.Split(pattern, splitCharacter)
	// check format of pattern
	if err := checkHostPattern(pattern, mask, isIpv4); err != nil {
		logger.Error(err)
		return false
	}

	if pattern == host {
		return true
	}

	// short name condition
	if !ipPatternContains(pattern) {
		return pattern == host
	}

	ipAddress := strings.Split(host, splitCharacter)
	for i := 0; i < len(mask); i++ {
		if "*" == mask[i] || mask[i] == ipAddress[i] {
			continue
		} else if strings.Contains(mask[i], "-") {
			rangeNumStrs := strings.Split(mask[i], "-")
			if len(rangeNumStrs) != 2 {
				logger.Error("There is wrong format of ip Address: " + mask[i])
				return false
			}
			min := getNumOfIpSegment(rangeNumStrs[0], isIpv4)
			max := getNumOfIpSegment(rangeNumStrs[1], isIpv4)
			ip := getNumOfIpSegment(ipAddress[i], isIpv4)
			if ip < min || ip > max {
				return false
			}
		} else if "0" == ipAddress[i] && "0" == mask[i] || "00" == mask[i] || "000" == mask[i] || "0000" == mask[i] {
			continue
		} else if mask[i] != ipAddress[i] {
			return false
		}
	}
	return true
}

func ipPatternContains(pattern string) bool {
	return strings.Contains(pattern, "*") || strings.Contains(pattern, "-")
}

func checkHostPattern(pattern string, mask []string, isIpv4 bool) error {
	if !isIpv4 {
		if len(mask) != 8 && ipPatternContains(pattern) {
			return errors.New("If you config ip expression that contains '*' or '-', please fill qualified ip pattern like 234e:0:4567:0:0:0:3d:*. ")
		}
		if len(mask) != 8 && !strings.Contains(pattern, "::") {
			return errors.New("The host is ipv6, but the pattern is not ipv6 pattern : " + pattern)
		}
	} else {
		if len(mask) != 4 {
			return errors.New("The host is ipv4, but the pattern is not ipv4 pattern : " + pattern)
		}
	}
	return nil
}

func getPatternHostAndPort(pattern string, isIpv4 bool) []string {
	result := make([]string, 2)
	if strings.HasPrefix(pattern, "[") && strings.Contains(pattern, "]:") {
		end := strings.Index(pattern, "]:")
		result[0] = pattern[1:end]
		result[1] = pattern[end+2:]
	} else if strings.HasPrefix(pattern, "[") && strings.HasSuffix(pattern, "]") {
		result[0] = pattern[1 : len(pattern)-1]
		result[1] = ""
	} else if isIpv4 && strings.Contains(pattern, ":") {
		end := strings.Index(pattern, ":")
		result[0] = pattern[:end]
		result[1] = pattern[end+1:]
	} else {
		result[0] = pattern
	}
	return result
}

func getNumOfIpSegment(ipSegment string, isIpv4 bool) int {
	if isIpv4 {
		ipSeg, _ := strconv.Atoi(ipSegment)
		return ipSeg
	}
	ipSeg, _ := strconv.ParseInt(ipSegment, 0, 16)
	return int(ipSeg)
}
