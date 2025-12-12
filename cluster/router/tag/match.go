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
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

type predicate func(invoker base.Invoker, tag any) bool

// static tag matching. no used configuration center to create tag router configuration
func staticTag(invokers []base.Invoker, url *common.URL, invocation base.Invocation) []base.Invoker {
	var (
		tag    string
		ok     bool
		result []base.Invoker
	)
	if tag, ok = invocation.GetAttachment(constant.Tagkey); !ok {
		tag = url.GetParam(constant.Tagkey, "")
	}
	if tag != "" {
		// match dynamic tag
		result = filterInvokers(invokers, tag, func(invoker base.Invoker, tag any) bool {
			return invoker.GetURL().GetParam(constant.Tagkey, "") != tag
		})
	}

	// match empty tag
	if (len(result) == 0 && !requestIsForce(url, invocation)) || tag == "" {
		result = filterInvokers(invokers, tag, func(invoker base.Invoker, tag any) bool {
			return invoker.GetURL().GetParam(constant.Tagkey, "") != ""
		})
	}
	logger.Debugf("[tag router] filter static tag, invokers=%+v", result)
	return result
}

// dynamic tag matching. used configuration center to create tag router configuration
func dynamicTag(invokers []base.Invoker, url *common.URL, invocation base.Invocation, cfg global.RouterConfig) []base.Invoker {
	tag := invocation.GetAttachmentWithDefaultValue(constant.Tagkey, url.GetParam(constant.Tagkey, ""))
	if tag == "" {
		return requestEmptyTag(invokers, cfg)
	}
	return requestTag(invokers, url, invocation, cfg, tag)
}

// if request.tag is not set, only providers with empty tags will be matched.
// even if a service is available in the cluster, it cannot be invoked if the tag does not match,
// and requests without tags or other tags will never be able to access services with other tags.
func requestEmptyTag(invokers []base.Invoker, cfg global.RouterConfig) []base.Invoker {
	result := filterInvokers(invokers, "", func(invoker base.Invoker, tag any) bool {
		return invoker.GetURL().GetParam(constant.Tagkey, "") != ""
	})
	if len(result) == 0 {
		return result
	}
	for _, tagCfg := range cfg.Tags {
		if len(tagCfg.Addresses) == 0 {
			continue
		}
		result = filterInvokers(result, tagCfg.Addresses, getAddressPredicate(true))
		logger.Debugf("[tag router]filter empty tag address, invokers=%+v", result)
	}
	logger.Debugf("[tag router]filter empty tag, invokers=%+v", result)
	return result
}

// when request tag =tag1, the provider with tag=tag1 is preferred.
// if no service corresponding to the request tag exists in the cluster,
// the provider with the empty request tag is degraded by default.
// to change the default behavior that no provider matching TAG1 returns an exception, set request.tag.force=true.
func requestTag(invokers []base.Invoker, url *common.URL, invocation base.Invocation, cfg global.RouterConfig, tag string) []base.Invoker {
	var (
		addresses []string
		result    []base.Invoker
		match     []*common.ParamMatch
	)
	for _, tagCfg := range cfg.Tags {
		if tagCfg.Name == tag {
			addresses = tagCfg.Addresses
			match = tagCfg.Match
		}
	}

	// only one of 'match' and 'addresses' will take effect if both are specified.
	if len(match) != 0 {
		result = filterInvokers(invokers, match, func(invoker base.Invoker, match any) bool {
			matches := match.([]*common.ParamMatch)
			for _, m := range matches {
				if !m.IsMatch(invoker.GetURL()) {
					return true
				}
			}
			return false
		})
	} else {
		if len(addresses) == 0 {
			// filter tag does not match
			result = filterInvokers(invokers, tag, func(invoker base.Invoker, tag any) bool {
				return invoker.GetURL().GetParam(constant.Tagkey, "") != tag
			})
			logger.Debugf("[tag router] filter dynamic tag, tag=%s, invokers=%+v", tag, result)
		} else {
			// filter address does not match
			result = filterInvokers(invokers, addresses, getAddressPredicate(false))
			logger.Debugf("[tag router] filter dynamic tag address, invokers=%+v", result)
		}
	}
	// returns the result directly
	if *cfg.Force || requestIsForce(url, invocation) {
		return result
	}
	if len(result) != 0 {
		return result
	}
	// failover: return all Providers without any tags
	result = filterInvokers(invokers, tag, func(invoker base.Invoker, tag any) bool {
		return invoker.GetURL().GetParam(constant.Tagkey, "") != ""
	})
	if len(addresses) == 0 {
		return result
	}
	result = filterInvokers(invokers, addresses, getAddressPredicate(true))
	logger.Debugf("[tag router] failover match all providers without any tags, invokers=%+v", result)
	return result
}

// filterInvokers remove invokers that match with predicate from the original input.
func filterInvokers(invokers []base.Invoker, param any, predicate predicate) []base.Invoker {
	result := make([]base.Invoker, len(invokers))
	copy(result, invokers)
	for i := 0; i < len(result); i++ {
		if predicate(result[i], param) {
			result = append(result[:i], result[i+1:]...)
			i--
		}
	}
	return result
}

func requestIsForce(url *common.URL, invocation base.Invocation) bool {
	force := invocation.GetAttachmentWithDefaultValue(constant.ForceUseTag, url.GetParam(constant.ForceUseTag, "false"))
	ok, err := strconv.ParseBool(force)
	if err != nil {
		logger.Errorf("parse force param fail,force=%s,err=%v", force, err)
	}
	return ok
}

func getAddressPredicate(result bool) predicate {
	return func(invoker base.Invoker, param any) bool {
		address := param.([]string)
		for _, v := range address {
			invokerURL := invoker.GetURL()
			if v == invokerURL.Location || v == constant.AnyHostValue+constant.KeySeparator+invokerURL.Port {
				return result
			}
		}
		return !result
	}
}
