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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	routePattern = regexp.MustCompile("([&!=,]*)\\s*([^&!=,\\s]+)")

	illegalMsg = "Illegal route rule \"%s\", The error char '%s' before '%s'"

	matcherFactories = make([]matcher.ConditionMatcherFactory, 0, 8)
)

func init() {
	factoriesMap := matcher.GetMatcherFactories()
	if len(factoriesMap) == 0 {
		return
	}
	for _, factory := range factoriesMap {
		matcherFactories = append(matcherFactories, factory())
	}
	sortMatcherFactories(matcherFactories)
}

type StateRouter struct {
	enable        bool
	force         bool
	url           *common.URL
	whenCondition map[string]matcher.Matcher
	thenCondition map[string]matcher.Matcher
}

func NewConditionStateRouter(url *common.URL) (*StateRouter, error) {
	if len(matcherFactories) == 0 {
		return nil, errors.Errorf("No ConditionMatcherFactory exists")
	}

	force := url.GetParamBool(constant.ForceKey, false)
	enable := url.GetParamBool(constant.EnabledKey, true)
	c := &StateRouter{
		url:    url,
		force:  force,
		enable: enable,
	}

	if enable {
		when, then, err := generateMatcher(url)
		if err != nil {
			return nil, err
		}
		c.whenCondition = when
		c.thenCondition = then
	}
	return c, nil
}

// Route Determine the target invokers list.
func (c *StateRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
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

	var result = make([]protocol.Invoker, 0, len(invokers))
	for _, invoker := range invokers {
		if c.matchThen(invoker.GetURL(), url) {
			result = append(result, invoker)
		}
	}

	if len(result) != 0 {
		return result
	} else if c.force {
		logger.Warn("execute condition state router result list is empty. and force=true")
		return result
	}

	return invokers
}

func (c *StateRouter) URL() *common.URL {
	return c.url
}

func (c *StateRouter) matchWhen(url *common.URL, invocation protocol.Invocation) bool {
	if len(c.whenCondition) == 0 {
		return true
	}
	return doMatch(url, nil, invocation, c.whenCondition, true)
}

func (c *StateRouter) matchThen(url *common.URL, param *common.URL) bool {
	if len(c.thenCondition) == 0 {
		return false
	}
	return doMatch(url, param, nil, c.thenCondition, false)
}

func generateMatcher(url *common.URL) (when, then map[string]matcher.Matcher, err error) {
	rule := url.GetParam(constant.RuleKey, "")
	if rule == "" || len(strings.Trim(rule, " ")) == 0 {
		return nil, nil, errors.Errorf("Illegal route rule!")
	}
	rule = strings.Replace(rule, "consumer.", "", -1)
	rule = strings.Replace(rule, "provider.", "", -1)
	i := strings.Index(rule, "=>")
	// for the case of `{when rule} => {then rule}`
	var whenRule string
	var thenRule string
	if i < 0 {
		whenRule = ""
		thenRule = strings.Trim(rule, " ")
	} else {
		whenRule = strings.Trim(rule[0:i], " ")
		thenRule = strings.Trim(rule[i+2:], " ")
	}

	when, err = parseWhen(whenRule)
	if err != nil {
		return nil, nil, err
	}

	then, err = parseThen(thenRule)
	if err != nil {
		return nil, nil, err
	}
	// NOTE: It should be determined on the business level whether the `When condition` can be empty or not.
	return when, then, nil
}

func parseWhen(whenRule string) (map[string]matcher.Matcher, error) {
	if whenRule == "" || whenRule == " " || whenRule == "true" {
		return make(map[string]matcher.Matcher), nil
	} else {
		when, err := parseRule(whenRule)
		if err != nil {
			return nil, err
		}
		return when, nil
	}
}

func parseThen(thenRule string) (map[string]matcher.Matcher, error) {
	if thenRule == "" || thenRule == " " || thenRule == "false" {
		return nil, nil
	} else {
		when, err := parseRule(thenRule)
		if err != nil {
			return nil, err
		}
		return when, nil
	}
}

func parseRule(rule string) (map[string]matcher.Matcher, error) {
	condition := make(map[string]matcher.Matcher)
	if rule == "" || rule == " " {
		return condition, nil
	}
	// key-Value pair, stores both match and mismatch conditions
	var matcherPair matcher.Matcher
	// Multiple values
	values := make(map[string]struct{})
	allMatchers := routePattern.FindAllStringSubmatch(rule, -1)
	for _, matchers := range allMatchers {
		separator := matchers[1]
		content := matchers[2]
		// Start part of the condition expression.
		if separator == "" {
			matcherPair = getMatcher(content)
			condition[content] = matcherPair

		} else if "&" == separator {
			// The KV part of the condition expression
			if condition[content] == nil {
				matcherPair = getMatcher(content)
				condition[content] = matcherPair
			} else {
				matcherPair = condition[content]
			}

		} else if "=" == separator {
			// The Value in the KV part.
			if matcherPair == nil {
				return nil, errors.Errorf(illegalMsg, rule, separator, content)
			}
			values = matcherPair.GetMatches()
			values[content] = struct{}{}

		} else if "!=" == separator {
			// The Value in the KV part.
			if matcherPair == nil {
				return nil, errors.Errorf(illegalMsg, rule, separator, content)
			}
			values = matcherPair.GetMismatches()
			values[content] = struct{}{}

		} else if "," == separator { // Should be separated by ','
			// The Value in the KV part, if Value have more than one items.
			if values == nil || len(values) == 0 {
				return nil, errors.Errorf(illegalMsg, rule, separator, content)
			}
			values[content] = struct{}{}

		} else {
			return nil, errors.Errorf(illegalMsg, rule, separator, content)
		}
	}
	return condition, nil
}

func getMatcher(key string) matcher.Matcher {
	for _, factory := range matcherFactories {
		if factory.ShouldMatch(key) {
			return factory.NewMatcher(key)
		}
	}
	return matcher.GetMatcherFactory("param").NewMatcher(key)
}

func doMatch(url *common.URL, param *common.URL, invocation protocol.Invocation, conditions map[string]matcher.Matcher, isWhenCondition bool) bool {
	sample := url.ToMap()
	for _, matcherPair := range conditions {
		if !matcher.Match(matcherPair, sample, param, invocation, isWhenCondition) {
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
	return routerConfig, nil
}
