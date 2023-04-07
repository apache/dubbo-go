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
	"regexp"
	"sort"
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher"
	"dubbo.apache.org/dubbo-go/v3/common"
	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

var (
	RoutePattern = regexp.MustCompile("([&!=,]*)\\s*([^&!=,\\s]+)")
)

type ConditionDynamicRouter struct {
	ruleKey          string
	conditionRouters []*ConditionStateRouter
	routerConfig     *config.RouterConfig
}

func NewConditionDynamicRouter() (*ConditionDynamicRouter, error) {
	return &ConditionDynamicRouter{}, nil
}

func (c *ConditionDynamicRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if len(invokers) == 0 || len(c.conditionRouters) == 0 {
		return invokers
	}
	for _, router := range c.conditionRouters {
		invokers = router.Route(invokers, url, invocation)
	}
	return invokers
}

func (c *ConditionDynamicRouter) URL() *common.URL {
	return nil
}

func (c *ConditionDynamicRouter) Priority() int64 {
	return 0
}

func (c *ConditionDynamicRouter) Notify(invokers []protocol.Invoker) {
	if len(invokers) == 0 {
		return
	}
	url := invokers[0].GetURL()
	if url == nil {
		logger.Error("url is empty")
		return
	}
	dynamicConfiguration := conf.GetEnvInstance().GetDynamicConfiguration()
	if dynamicConfiguration == nil {
		logger.Warnf("config center does not start, please check if the configuration center has been properly configured in dubbogo.yml")
		return
	}
	key := url.Service() + ":" + url.GetParam("version", "") + ":" + url.GetParam("group", "") + constant.ConditionRouterRuleSuffix
	dynamicConfiguration.AddListener(key, c)
	value, err := dynamicConfiguration.GetRule(key)
	if err != nil {
		logger.Errorf("query router rule fail,key=%s,err=%v", key, err)
		return
	}
	c.Process(&config_center.ConfigChangeEvent{Key: key, Value: value, ConfigType: remoting.EventTypeAdd})
}

func (c *ConditionDynamicRouter) Process(event *config_center.ConfigChangeEvent) {
	if event.ConfigType == remoting.EventTypeDel {
		c.routerConfig = nil
		c.conditionRouters = make([]*ConditionStateRouter, 0)
	} else {
		routerConfig, err := parseRoute(event.Value.(string))
		if err != nil {
			logger.Warnf("[condition router]Parse new condition route config error, %+v "+
				"and we will use the original condition rule configuration.", err)
			return
		}
		c.routerConfig = routerConfig
		if c.routerConfig != nil {
			conditionRouters := make([]*ConditionStateRouter, 0, len(c.routerConfig.Conditions))
			for _, conditionRule := range c.routerConfig.Conditions {
				url, err := common.NewURL("condition://")
				if err != nil {
					logger.Warnf("[condition router]Parse new condition route config error, %+v "+
						"and we will use the original condition rule configuration.", err)
					return
				}
				url.AddParam(constant.RuleKey, conditionRule)
				url.AddParam(constant.ForceKey, strconv.FormatBool(c.routerConfig.Force))
				url.AddParam(constant.EnabledKey, strconv.FormatBool(c.routerConfig.Enabled))
				conditionRoute, err := NewConditionStateRouter(url)
				if err != nil {
					logger.Warnf("[condition router]Parse new condition route config error, %+v "+
						"and we will use the original condition rule configuration.", err)
					return
				}
				conditionRouters = append(conditionRouters, conditionRoute)
			}
			c.conditionRouters = conditionRouters
		}
	}
}

type ConditionStateRouter struct {
	enable           bool
	force            bool
	url              *common.URL
	whenCondition    map[string]matcher.ConditionMatcher
	thenCondition    map[string]matcher.ConditionMatcher
	matcherFactories []matcher.ConditionMatcherFactory
}

func NewConditionStateRouter(url *common.URL) (*ConditionStateRouter, error) {

	matcherFactories := extension.GetMatcherFactories()
	if len(matcherFactories) == 0 {
		return nil, errors.Errorf("No ConditionMatcherFactory exits , create one please")
	}
	factories := make([]matcher.ConditionMatcherFactory, 0, len(matcherFactories))
	for _, matcherFactory := range matcherFactories {
		factories = append(factories, matcherFactory())
	}
	sortMatcherFactories(factories)

	force := url.GetParamBool(constant.ForceKey, false)
	enable := url.GetParamBool(constant.EnabledKey, true)
	c := &ConditionStateRouter{
		url:              url,
		force:            force,
		matcherFactories: factories,
		enable:           enable,
	}

	if enable {
		rule := url.GetParam(constant.RuleKey, "")
		if rule == "" || len(strings.Trim(rule, " ")) == 0 {
			return nil, errors.Errorf("Illegal route rule!")
		}
		rule = strings.Replace(rule, "consumer.", "", -1)
		rule = strings.Replace(rule, "privoder.", "", -1)
		i := strings.Index(rule, "=>")
		var whenRule string
		var thenRule string
		if i < 0 {
			whenRule = ""
			thenRule = strings.Trim(rule, " ")
		} else {
			whenRule = strings.Trim(rule[0:i], " ")
			thenRule = strings.Trim(rule[i+2:], " ")
		}
		var when map[string]matcher.ConditionMatcher
		var then map[string]matcher.ConditionMatcher
		var err error
		if whenRule == "" || whenRule == " " || whenRule == "true" {
			when = make(map[string]matcher.ConditionMatcher)
		} else {
			when, err = c.parseRule(whenRule)
			if err != nil {
				return nil, err
			}
		}

		if thenRule == "" || thenRule == " " || thenRule == "false" {
			then = nil
		} else {
			then, err = c.parseRule(thenRule)
			if err != nil {
				return nil, err
			}
		}
		// NOTE: It should be determined on the business level whether the `When condition` can be empty or not.
		c.whenCondition = when
		c.thenCondition = then
	}
	return c, nil
}

func (c *ConditionStateRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if !c.enable {
		return invokers
	}

	if len(invokers) == 0 {
		return invokers
	}

	if !c.matchWhen(url, invocation) {
		return invokers
	}

	if len(c.thenCondition) == 0 {
		logger.Warn("condition state router thenCondition is empty")
		return []protocol.Invoker{}
	}

	var result = make([]protocol.Invoker, len(invokers))
	copy(result, invokers)
	i := 0
	for _, invoker := range result {
		if c.matchThen(invoker.GetURL(), url) {
			result[i] = invoker
			i++
		}
	}
	result = result[:i]

	if len(result) != 0 {
		return result
	} else if c.force {
		logger.Warn("execute condition state router result list is empty. and force=true")
		return result
	}

	return invokers
}

func (c *ConditionStateRouter) URL() *common.URL {
	return c.url
}

func (c *ConditionStateRouter) Priority() int64 {
	return 0
}

func (c *ConditionStateRouter) Notify(invokers []protocol.Invoker) {
}

func (c *ConditionStateRouter) Process(event *config_center.ConfigChangeEvent) {
}

func (c *ConditionStateRouter) parseRule(rule string) (map[string]matcher.ConditionMatcher, error) {
	condition := make(map[string]matcher.ConditionMatcher)
	if rule == "" || rule == " " {
		return condition, nil
	}
	// Key-Value pair, stores both match and mismatch conditions
	var matcherPair matcher.ConditionMatcher
	// Multiple values
	values := make(map[string]struct{})
	allMatchers := RoutePattern.FindAllStringSubmatch(rule, -1)
	for _, matchers := range allMatchers {
		separator := matchers[1]
		content := matchers[2]
		// Start part of the condition expression.
		if separator == "" {
			matcherPair = c.getMatcher(content)
			condition[content] = matcherPair
		} else if "&" == separator {
			// The KV part of the condition expression
			if condition[content] == nil {
				matcherPair = c.getMatcher(content)
				condition[content] = matcherPair
			} else {
				matcherPair = condition[content]
			}
		} else if "=" == separator {
			// The Value in the KV part.
			if matcherPair == nil {
				return nil, errors.Errorf("Illegal route rule \"%s\", The error char '%s' before '%s'",
					rule, separator, content)
			}
			values = matcherPair.GetMatches()
			values[content] = struct{}{}
		} else if "!=" == separator {
			// The Value in the KV part.
			if matcherPair == nil {
				return nil, errors.Errorf("Illegal route rule \"%s\", The error char '%s' before '%s'",
					rule, separator, content)
			}
			values = matcherPair.GetMismatches()
			values[content] = struct{}{}
		} else if "," == separator { // Should be separated by ','
			// The Value in the KV part, if Value have more than one items.
			if values == nil || len(values) == 0 {
				return nil, errors.Errorf("Illegal route rule \"%s\", The error char '%s' before '%s'",
					rule, separator, content)
			}
			values[content] = struct{}{}
		} else {
			return nil, errors.Errorf("Illegal route rule \"%s\", The error char '%s' before '%s'",
				rule, separator, content)
		}
	}
	return condition, nil
}

func (c *ConditionStateRouter) getMatcher(key string) matcher.ConditionMatcher {
	for _, factory := range c.matcherFactories {
		if factory.ShouldMatch(key) {
			return factory.NewMatcher(key)
		}
	}
	return extension.GetMatcherFactory("param").NewMatcher(key)
}

func (c *ConditionStateRouter) matchWhen(url *common.URL, invocation protocol.Invocation) bool {
	if len(c.whenCondition) == 0 {
		return true
	}
	return doMatch(url, nil, invocation, c.whenCondition, true)
}

func (c *ConditionStateRouter) matchThen(url *common.URL, param *common.URL) bool {
	if len(c.thenCondition) == 0 {
		return false
	}
	return doMatch(url, param, nil, c.thenCondition, false)
}

func doMatch(url *common.URL, param *common.URL, invocation protocol.Invocation, conditions map[string]matcher.ConditionMatcher, isWhenCondition bool) bool {
	sample := url.ToMap()
	for _, matcherPair := range conditions {
		if !matcherPair.IsMatch(sample, param, invocation, isWhenCondition) {
			return false
		}
	}
	return true
}

func sortMatcherFactories(matcherFactories []matcher.ConditionMatcherFactory) {
	sort.Stable(byPriority(matcherFactories))
}

type byPriority []matcher.ConditionMatcherFactory

func (a byPriority) Len() int           { return len(a) }
func (a byPriority) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPriority) Less(i, j int) bool { return a[i].Priority() < a[j].Priority() }

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
