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
	"fmt"
	"strconv"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/remoting"
)

type tagRouter struct {
	url           *common.URL
	tagRouterRule *RouterRule
	enabled       bool
	priority      int64
}

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

func (c *tagRouter) isEnabled() bool {
	return c.enabled
}

func (c *tagRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if !c.isEnabled() {
		return invokers
	}
	if len(invokers) == 0 {
		return invokers
	}
	// since the rule can be changed by config center, we should copy one to use.
	tagRouterRuleCopy := c.tagRouterRule
	if tagRouterRuleCopy == nil || !tagRouterRuleCopy.Valid || !tagRouterRuleCopy.Enabled {
		return filterUsingStaticTag(invokers, url, invocation)
	}
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
			result = filterAddressMatches(invokers, addresses)
			if len(result) > 0 || tagRouterRuleCopy.Force {
				return result
			}
		} else {
			// dynamic tag group doesn't have any item about the requested app OR it's null after filtered by
			// dynamic tag group but force=false. check static tag
			cond := func(invoker protocol.Invoker) bool {
				if invoker.GetUrl().GetParam(constant.Tagkey, "") == tag {
					return true
				}
				return false
			}
			result = filterCondition(invokers, cond)
		}
		// If there's no tagged providers that can match the current tagged request. force.tag is set by default
		// to false, which means it will invoke any providers without a tag unless it's explicitly disallowed.
		if len(result) > 0 || isForceUseTag(url, invocation) {
			return result
		} else {
			// FAILOVER: return all Providers without any tags.
			result = filterAddressNotMatches(invokers, tagRouterRuleCopy.getAddresses())
			cond := func(invoker protocol.Invoker) bool {
				if invoker.GetUrl().GetParam(constant.Tagkey, "") == "" {
					return true
				}
				return false
			}
			return filterCondition(result, cond)
		}
	} else {
		// return all addresses in dynamic tag group.
		addresses = tagRouterRuleCopy.getAddresses()
		if len(addresses) > 0 {
			result = filterAddressNotMatches(invokers, addresses)
			// 1. all addresses are in dynamic tag group, return empty list.
			if len(result) == 0 {
				return result
			}
			// 2. if there are some addresses that are not in any dynamic tag group, continue to filter using the
			// static tag group.
		}
		cond := func(invoker protocol.Invoker) bool {
			localTag := invoker.GetUrl().GetParam(constant.Tagkey, "")
			return localTag == "" || !(tagRouterRuleCopy.hasTag(localTag))
		}
		return filterCondition(result, cond)
	}
}

func (c *tagRouter) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("Notification of dynamic tag rule, change type is:[%s] , raw rule is:[%v]", event.ConfigType, event.Value)
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

func (c *tagRouter) URL() common.URL {
	return *c.url
}

func (c *tagRouter) Priority() int64 {
	return c.priority
}

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

func isForceUseTag(url *common.URL, invocation protocol.Invocation) bool {
	if b, e := strconv.ParseBool(invocation.AttachmentsByKey(constant.ForceUseTag, url.GetParam(constant.ForceUseTag, "false"))); e == nil {
		return b
	}
	return false
}

func filterAddressMatches(invokers []protocol.Invoker, addresses []string) []protocol.Invoker {
	var idx int
	for _, invoker := range invokers {
		url := invoker.GetUrl()
		if !(len(addresses) > 0 && checkAddressMatch(addresses, url.Ip, url.Port)) {
			continue
		}
		invokers[idx] = invoker
		idx++
	}
	return invokers[:idx]
}

func filterAddressNotMatches(invokers []protocol.Invoker, addresses []string) []protocol.Invoker {
	var idx int
	for _, invoker := range invokers {
		url := invoker.GetUrl()
		if !(len(addresses) == 0 || !checkAddressMatch(addresses, url.Ip, url.Port)) {
			continue
		}
		invokers[idx] = invoker
		idx++
	}
	return invokers[:idx]
}

func filterCondition(invokers []protocol.Invoker, condition func(protocol.Invoker) bool) []protocol.Invoker {
	var idx int
	for _, invoker := range invokers {
		if !condition(invoker) {
			continue
		}
		invokers[idx] = invoker
		idx++
	}
	return invokers[:idx]
}

func checkAddressMatch(addresses []string, host, port string) bool {
	for _, address := range addresses {
		// TODO ip match
		if address == host+":"+port {
			return true
		}
		if address == constant.ANYHOST_VALUE+":"+port {
			return true
		}
	}
	return false
}
