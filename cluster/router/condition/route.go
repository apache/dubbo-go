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
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"sync"
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
	routePattern     = regexp.MustCompile("([&!=,]*)\\s*([^&!=,\\s]+)")
	illegalMsg       = "Illegal route rule \"%s\", The error char '%s' before '%s'"
	matcherFactories = make([]matcher.ConditionMatcherFactory, 0, 8)
	once             sync.Once
)

type StateRouter struct {
	whenCondition map[string]matcher.Matcher
	thenCondition map[string]matcher.Matcher
}

func NewConditionStateRouter(url *common.URL) (*StateRouter, error) {
	once.Do(initMatcherFactories)

	if len(matcherFactories) == 0 {
		return nil, errors.Errorf("No ConditionMatcherFactory exists")
	}

	c := &StateRouter{}

	when, then, err := generateMatcher(url)
	if err != nil {
		return nil, err
	}
	c.whenCondition = when
	c.thenCondition = then
	return c, nil
}

// Route Determine the target invokers list.
// condition rule like `self_condition => peers_condition `
//
// @return active_peers_invokers, Is_self_condition_match_success
func (s *StateRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if len(invokers) == 0 {
		return invokers
	}

	if !s.matchWhen(url, invocation) {
		return invokers
	}

	if len(s.thenCondition) == 0 {
		logger.Warn("condition state router thenCondition is empty")
		return []protocol.Invoker{}
	}

	var result = make([]protocol.Invoker, 0, len(invokers))
	for _, invoker := range invokers {
		if s.matchThen(invoker.GetURL(), url) {
			result = append(result, invoker)
		}
	}

	return result
}

func (s *StateRouter) matchWhen(url *common.URL, invocation protocol.Invocation) bool {
	if len(s.whenCondition) == 0 {
		return true
	}
	return doMatch(url, nil, invocation, s.whenCondition, true)
}

func (s *StateRouter) matchThen(url *common.URL, param *common.URL) bool {
	if len(s.thenCondition) == 0 {
		return false
	}
	return doMatch(url, param, nil, s.thenCondition, false)
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
	if isRuleEmpty(rule) {
		return make(map[string]matcher.Matcher), nil
	}

	condition, err := processMatchers(rule)
	if err != nil {
		return nil, err
	}

	return condition, nil
}

func isRuleEmpty(rule string) bool {
	return rule == "" || (len(rule) == 1 && rule[0] == ' ')
}

func processMatchers(rule string) (map[string]matcher.Matcher, error) {
	condition := make(map[string]matcher.Matcher)
	var currentMatcher matcher.Matcher
	var err error
	values := make(map[string]struct{})

	for _, matchers := range routePattern.FindAllStringSubmatch(rule, -1) {
		separator := matchers[1]
		content := matchers[2]

		switch separator {
		case "":
			currentMatcher = getMatcher(content)
			condition[content] = currentMatcher
		case "&":
			currentMatcher, condition = processAndSeparator(content, condition)
		case "=", "!=":
			values, currentMatcher, err = processEqualNotEqualSeparator(separator, content, currentMatcher, rule)
			if err != nil {
				return nil, err
			}
		case ",":
			values, err = processCommaSeparator(content, values, rule)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.Errorf(illegalMsg, rule, separator, content)
		}
	}

	return condition, nil
}

func processAndSeparator(content string, condition map[string]matcher.Matcher) (matcher.Matcher, map[string]matcher.Matcher) {
	currentMatcher := condition[content]
	if currentMatcher == nil {
		currentMatcher = getMatcher(content)
		condition[content] = currentMatcher
	}
	return currentMatcher, condition
}

func processEqualNotEqualSeparator(separator, content string, currentMatcher matcher.Matcher, rule string) (map[string]struct{}, matcher.Matcher, error) {
	if currentMatcher == nil {
		return nil, nil, errors.Errorf(illegalMsg, rule, separator, content)
	}
	values := currentMatcher.GetMatches()
	if separator == "!=" {
		values = currentMatcher.GetMismatches()
	}
	values[content] = struct{}{}
	return values, currentMatcher, nil
}

func processCommaSeparator(content string, values map[string]struct{}, rule string) (map[string]struct{}, error) {
	if len(values) == 0 {
		return nil, errors.Errorf(illegalMsg, rule, ",", content)
	}
	values[content] = struct{}{}
	return values, nil
}

func getMatcher(key string) matcher.Matcher {
	for _, factory := range matcherFactories {
		if factory.ShouldMatch(key) {
			return factory.NewMatcher(key)
		}
	}
	return matcher.GetMatcherFactory(constant.Param).NewMatcher(key)
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

func initMatcherFactories() {
	factoriesMap := matcher.GetMatcherFactories()
	if len(factoriesMap) == 0 {
		return
	}
	for _, factory := range factoriesMap {
		matcherFactories = append(matcherFactories, factory())
	}
	sortMatcherFactories(matcherFactories)
}

func sortMatcherFactories(matcherFactories []matcher.ConditionMatcherFactory) {
	sort.Stable(byPriority(matcherFactories))
}

type byPriority []matcher.ConditionMatcherFactory

func (a byPriority) Len() int           { return len(a) }
func (a byPriority) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPriority) Less(i, j int) bool { return a[i].Priority() < a[j].Priority() }

func parseConditionRoute(routeContent string) (*config.RouterConfig, error) {
	routeDecoder := yaml.NewDecoder(strings.NewReader(routeContent))
	routerConfig := &config.RouterConfig{}
	err := routeDecoder.Decode(routerConfig)
	if err != nil {
		return nil, err
	}
	return routerConfig, nil
}

type FieldMatcher struct {
	rule  string
	match map[string]matcher.Matcher
}

func NewFieldMatcher(rule string) (FieldMatcher, error) {
	m, err := parseRule(rule)
	if err != nil {
		return FieldMatcher{}, err
	}
	return FieldMatcher{rule: rule, match: m}, nil
}

func (m *FieldMatcher) MatchRequest(url *common.URL, invocation protocol.Invocation) bool {
	return doMatch(url, nil, invocation, m.match, true)
}

func (m *FieldMatcher) MatchInvoker(url *common.URL, ivk protocol.Invoker, invocation protocol.Invocation) bool {
	return doMatch(ivk.GetURL(), url, nil, m.match, false)
}

// MultiDestRouter Multiply-Destination-Router
type MultiDestRouter struct {
	whenCondition FieldMatcher
	thenCondition []condSet
}

type condSet struct {
	FieldMatcher
	subSetWeight int
}

func newCondSet(rule string, subSetWeight int) (condSet, error) {
	if subSetWeight <= 0 {
		subSetWeight = constant.DefaultRouteConditionSubSetWeight
	}
	m, err := NewFieldMatcher(rule)
	if err != nil {
		return condSet{}, err
	}
	return condSet{FieldMatcher: m, subSetWeight: subSetWeight}, nil
}

type destination struct {
	matchRule string
	weight    int
	ivks      []protocol.Invoker
}

type destSets struct {
	destinations []*destination
	weightSum    int
}

func newDestSets() *destSets {
	return &destSets{
		destinations: []*destination{},
		weightSum:    0,
	}
}

func (s *destSets) addDest(weight int, rule string, ivks []protocol.Invoker) {
	s.destinations = append(s.destinations, &destination{weight: weight, matchRule: rule, ivks: ivks})
	s.weightSum += weight
}

func (s *destSets) randDest() *destination {
	if s.weightSum == 0 {
		return nil
	}
	if len(s.destinations) == 1 {
		return s.destinations[0]
	}
	sum := rand.Intn(s.weightSum)
	for _, d := range s.destinations {
		sum -= d.weight
		if sum <= 0 {
			return d
		}
	}
	return nil
}

func (m MultiDestRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) ([]protocol.Invoker, bool) {
	if len(invokers) == 0 {
		return invokers, false
	}

	if !m.whenCondition.MatchRequest(url, invocation) {
		return invokers, false
	}

	if len(m.thenCondition) == 0 {
		logger.Warn("condition state router thenCondition is empty")
		return []protocol.Invoker{}, true
	}

	destinations := newDestSets()
	for _, condition := range m.thenCondition {
		res := make([]protocol.Invoker, 0)
		for _, invoker := range invokers {
			if condition.MatchInvoker(url, invoker, invocation) {
				res = append(res, invoker)
			}
		}
		if len(res) != 0 {
			destinations.addDest(condition.subSetWeight, condition.rule, res)
		}
	}
	// use to print log, if route empty
	i, ok := invocation.Attributes()["condition-chain"].([]string)
	if !ok {
		i = []string{}
	}

	d := destinations.randDest()
	if d != nil {
		invocation.Attributes()["condition-chain"] = append(i, "request="+m.whenCondition.rule+",invokers="+d.matchRule)
		return d.ivks, true
	}

	thenRule := make([]string, 0, len(m.thenCondition))
	for _, set := range m.thenCondition {
		thenRule = append(thenRule, set.rule)
	}
	invocation.Attributes()["condition-chain"] = append(i, "request="+m.whenCondition.rule+",invokers!="+strings.Join(thenRule, ","))
	return []protocol.Invoker{}, true
}

func NewConditionMultiDestRouter(url *common.URL) (*MultiDestRouter, error) {
	once.Do(initMatcherFactories)

	if len(matcherFactories) == 0 {
		return nil, errors.Errorf("No ConditionMatcherFactory exists")
	}

	rawCondConf, ok := url.GetAttribute(constant.RuleKey)
	if !ok {
		return nil, errors.Errorf("Condition Router can't get the rule key")
	}
	condConf, ok := rawCondConf.(*config.ConditionRule)
	if !ok {
		return nil, errors.Errorf("Condition Router get the rule key invaild , got %T", rawCondConf)
	}
	// ensure config effective
	if (condConf.To == nil || len(condConf.To) == 0) && condConf.From.Match == "" {
		return nil, nil
	}

	c := &MultiDestRouter{
		thenCondition: make([]condSet, 0, len(condConf.To)),
	}

	var err error
	c.whenCondition, err = NewFieldMatcher(condConf.From.Match)
	if err != nil {
		return nil, err
	}

	for _, ruleTo := range condConf.To {
		cs, err := newCondSet(ruleTo.Match, ruleTo.Weight)
		if err != nil {
			return nil, err
		}
		c.thenCondition = append(c.thenCondition, cs)
	}

	return c, nil
}

func parseMultiConditionRoute(routeContent string) (*config.ConditionRouter, error) {
	routeDecoder := yaml.NewDecoder(strings.NewReader(routeContent))
	routerConfig := &config.ConditionRouter{}
	err := routeDecoder.Decode(routerConfig)
	if err != nil {
		return nil, err
	}
	return routerConfig, nil
}
