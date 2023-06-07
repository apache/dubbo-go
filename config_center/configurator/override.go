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

package configurator

import (
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
	"strings"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

func init() {
	extension.SetDefaultConfigurator(newConfigurator)
}

func newConfigurator(url *common.URL) config_center.Configurator {
	return &overrideConfigurator{configuratorUrl: url}
}

type overrideConfigurator struct {
	configuratorUrl *common.URL
}

func (c *overrideConfigurator) GetUrl() *common.URL {
	return c.configuratorUrl
}

func (c *overrideConfigurator) Configure(url *common.URL) {
	// remove configuratorUrl some param that can not be configured
	if c.configuratorUrl.GetParam(constant.EnabledKey, "true") == "false" || len(c.configuratorUrl.Location) == 0 {
		return
	}

	// branch for version 2.7.x
	apiVersion := c.configuratorUrl.GetParam(constant.ConfigVersionKey, "")
	if len(apiVersion) != 0 {
		var host string
		currentSide := url.GetParam(constant.SideKey, "")
		configuratorSide := c.configuratorUrl.GetParam(constant.SideKey, "")
		if currentSide == configuratorSide && common.DubboRole[common.CONSUMER] == currentSide {
			host = common.GetLocalIp()
		} else if currentSide == configuratorSide && common.DubboRole[common.PROVIDER] == currentSide {
			host = url.Ip
		}

		if strings.HasPrefix(apiVersion, constant.ApiVersion) {
			c.configureIfMatchV3(host, url)
		} else {
			c.configureIfMatch(host, url)
		}
	} else {
		// branch for version 2.6.x and less
		c.configureDeprecated(url)
	}
}

// configureIfMatch
func (c *overrideConfigurator) configureIfMatchV3(host string, url *common.URL) {
	conditionKeys := getConditionKeys()
	matcher := c.configuratorUrl.GetAttribute(constant.MatchCondition)
	if matcher != nil {
		conditionMatcher := matcher.(parser.ConditionMatch)
		if conditionMatcher.IsMatch(host, url) {
			configUrl := c.configuratorUrl.CloneExceptParams(conditionKeys)
			url.SetParams(configUrl.GetParams())
		}
	}
}

func (c *overrideConfigurator) configureDeprecated(url *common.URL) {
	// If override url has port, means it is a provider address. We want to control a specific provider with this override url, it may take effect on the specific provider instance or on consumers holding this provider instance.
	if c.configuratorUrl.Port != "0" {
		if url.Port == c.configuratorUrl.Port {
			c.configureIfMatch(url.Ip, url)
		}
	} else {
		// override url don't have a port, means the ip override url specify is a consumer address or 0.0.0.0
		// 1.If it is a consumer ip address, the intention is to control a specific consumer instance, it must takes effect at the consumer side, any provider received this override url should ignore;
		// 2.If the ip is 0.0.0.0, this override url can be used on consumer, and also can be used on provider
		if url.GetParam(constant.SideKey, "") == common.DubboRole[common.CONSUMER] {
			localIP := common.GetLocalIp()
			c.configureIfMatch(localIP, url)
		} else {
			c.configureIfMatch(constant.AnyHostValue, url)
		}
	}
}

func (c *overrideConfigurator) configureIfMatchInternal(url *common.URL) {
	configApp := c.configuratorUrl.GetParam(constant.ApplicationKey, c.configuratorUrl.Username)
	currentApp := url.GetParam(constant.ApplicationKey, url.Username)
	if len(configApp) == 0 || constant.AnyValue == configApp || configApp == currentApp {
		conditionKeys := getConditionKeys()
		returnUrl := false
		c.configuratorUrl.RangeParams(func(k, _ string) bool {
			value := c.configuratorUrl.GetParam(k, "")
			if strings.HasPrefix(k, "~") || k == constant.ApplicationKey || k == constant.SideKey {
				conditionKeys.Add(k)
				if len(value) != 0 && value != constant.AnyValue && value != url.GetParam(strings.TrimPrefix(k, "~"), "") {
					returnUrl = true
					return false
				}
			}
			return true
		})
		if returnUrl {
			return
		}
		configUrl := c.configuratorUrl.CloneExceptParams(conditionKeys)
		url.SetParams(configUrl.GetParams())
	}
}

// configureIfMatch translate from java, compatible rules in java
func (c *overrideConfigurator) configureIfMatch(host string, url *common.URL) {
	if constant.AnyHostValue == c.configuratorUrl.Ip || host == c.configuratorUrl.Ip {
		providers := c.configuratorUrl.GetParam(constant.OverrideProvidersKey, "")
		if len(providers) == 0 || strings.Contains(providers, url.Location) || strings.Contains(providers, constant.AnyHostValue) {
			c.configureIfMatchInternal(url)
		}
	}
}

func getConditionKeys() *gxset.HashSet {
	conditionKeys := gxset.NewSet()
	conditionKeys.Add(constant.CategoryKey)
	conditionKeys.Add(constant.CheckKey)
	conditionKeys.Add(constant.EnabledKey)
	conditionKeys.Add(constant.GroupKey)
	conditionKeys.Add(constant.VersionKey)
	conditionKeys.Add(constant.ApplicationKey)
	conditionKeys.Add(constant.SideKey)
	conditionKeys.Add(constant.ConfigVersionKey)
	conditionKeys.Add(constant.CompatibleConfigKey)
	return conditionKeys
}
