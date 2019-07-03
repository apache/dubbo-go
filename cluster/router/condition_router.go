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

package router

import (
	"reflect"
	"regexp"
	"strings"
)

import (
	"github.com/dubbogo/gost/container"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/common/utils"
	"github.com/apache/dubbo-go/protocol"
)

const (
	ROUTE_PATTERN = `([&!=,]*)\\s*([^&!=,\\s]+)`
	FORCE         = "force"
	PRIORITY      = "priority"
)

//ConditionRouter condition router struct
type ConditionRouter struct {
	Pattern       string
	Url           *common.URL
	Priority      int64
	Force         bool
	WhenCondition map[string]MatchPair
	ThenCondition map[string]MatchPair
}

func newConditionRouter(url *common.URL) (*ConditionRouter, error) {
	var (
		whenRule string
		thenRule string
		when     map[string]MatchPair
		then     map[string]MatchPair
	)
	rule, err := url.GetParamAndDecoded(constant.RULE_KEY)
	if err != nil || len(rule) == 0 {
		return nil, perrors.Errorf("Illegal route rule!")
	}
	rule = strings.Replace(rule, "consumer.", "", -1)
	rule = strings.Replace(rule, "provider.", "", -1)
	i := strings.Index(rule, "=>")
	if i > 0 {
		whenRule = rule[0:i]
	}
	if i < 0 {
		thenRule = rule
	} else {
		thenRule = rule[i+2:]
	}
	whenRule = strings.Trim(whenRule, " ")
	thenRule = strings.Trim(thenRule, " ")
	w, err := parseRule(whenRule)
	if err != nil {
		return nil, perrors.Errorf("%s", "")
	}
	t, err := parseRule(thenRule)
	if err != nil {
		return nil, perrors.Errorf("%s", "")
	}
	if len(whenRule) == 0 || "true" == whenRule {
		when = make(map[string]MatchPair, 16)
	} else {
		when = w
	}
	if len(thenRule) == 0 || "false" == thenRule {
		when = make(map[string]MatchPair, 16)
	} else {
		then = t
	}
	return &ConditionRouter{
		ROUTE_PATTERN,
		url,
		url.GetParamInt(PRIORITY, 0),
		url.GetParamBool(FORCE, false),
		when,
		then,
	}, nil
}

//Router determine the target server list.
func (c *ConditionRouter) Route(invokers []protocol.Invoker, url common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if len(invokers) == 0 {
		return invokers
	}
	isMatchWhen, err := c.MatchWhen(url, invocation)
	if err != nil {

		var urls []string
		for _, invo := range invokers {
			urls = append(urls, reflect.TypeOf(invo).String())
		}
		logger.Warnf("Failed to execute condition router rule: %s , invokers: [%s], cause: %v", c.Url.String(), strings.Join(urls, ","), err)
		return invokers
	}
	if !isMatchWhen {
		return invokers
	}
	var result []protocol.Invoker
	if len(c.ThenCondition) == 0 {
		return result
	}
	localIP, _ := utils.GetLocalIP()
	for _, invoker := range invokers {
		isMatchThen, err := c.MatchThen(invoker.GetUrl(), url)
		if err != nil {
			var urls []string
			for _, invo := range invokers {
				urls = append(urls, reflect.TypeOf(invo).String())
			}
			logger.Warnf("Failed to execute condition router rule: %s , invokers: [%s], cause: %v", c.Url.String(), strings.Join(urls, ","), err)
			return invokers
		}
		if isMatchThen {
			result = append(result, invoker)
		}
	}
	if len(result) > 0 {
		return result
	} else if c.Force {
		rule, _ := url.GetParamAndDecoded(constant.RULE_KEY)
		logger.Warnf("The route result is empty and force execute. consumer: %s, service: %s, router: %s", localIP, url.Service(), rule)
		return result
	}
	return invokers
}

func parseRule(rule string) (map[string]MatchPair, error) {
	condition := make(map[string]MatchPair, 16)
	if len(rule) == 0 {
		return condition, nil
	}
	var pair MatchPair
	values := container.NewSet()
	reg := regexp.MustCompile(`([&!=,]*)\s*([^&!=,\s]+)`)
	var startIndex = 0
	if indexTuple := reg.FindIndex([]byte(rule)); len(indexTuple) > 0 {
		startIndex = indexTuple[0]
	}
	matches := reg.FindAllSubmatch([]byte(rule), -1)
	for _, groups := range matches {
		separator := string(groups[1])
		content := string(groups[2])
		switch separator {
		case "":
			pair = MatchPair{
				Matches:    container.NewSet(),
				Mismatches: container.NewSet(),
			}
			condition[content] = pair
		case "&":
			if r, ok := condition[content]; ok {
				pair = r
			} else {
				pair = MatchPair{
					Matches:    container.NewSet(),
					Mismatches: container.NewSet(),
				}
				condition[content] = pair
			}
		case "=":
			if &pair == nil {
				return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex, startIndex)
			}
			values = pair.Matches
			values.Add(content)
		case "!=":
			if &pair == nil {
				return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex, startIndex)
			}
			values = pair.Mismatches
			values.Add(content)
		case ",":
			if values.Empty() {
				return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex, startIndex)
			}
			values.Add(content)
		default:
			return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex, startIndex)

		}
	}
	return condition, nil
}

//
func (c *ConditionRouter) MatchWhen(url common.URL, invocation protocol.Invocation) (bool, error) {
	condition, err := MatchCondition(c.WhenCondition, &url, nil, invocation)
	return len(c.WhenCondition) == 0 || condition, err
}

//MatchThen MatchThen
func (c *ConditionRouter) MatchThen(url common.URL, param common.URL) (bool, error) {
	condition, err := MatchCondition(c.ThenCondition, &url, &param, nil)
	return len(c.ThenCondition) > 0 && condition, err
}

//MatchCondition MatchCondition
func MatchCondition(pairs map[string]MatchPair, url *common.URL, param *common.URL, invocation protocol.Invocation) (bool, error) {
	sample := url.ToMap()
	if sample == nil {
		return true, perrors.Errorf("url is not allowed be nil")
	}
	result := false
	for key, matchPair := range pairs {
		var sampleValue string

		if invocation != nil && ((constant.METHOD_KEY == key) || (constant.METHOD_KEYS == key)) {
			sampleValue = invocation.MethodName()
		} else {
			sampleValue = sample[key]
			if len(sampleValue) == 0 {
				sampleValue = sample[constant.PREFIX_DEFAULT_KEY+key]
			}
		}
		if len(sampleValue) > 0 {
			if !matchPair.isMatch(sampleValue, param) {
				return false, nil
			} else {
				result = true
			}
		} else {
			if !(matchPair.Matches.Empty()) {
				return false, nil
			} else {
				result = true
			}
		}
	}
	return result, nil
}

type MatchPair struct {
	Matches    *container.HashSet
	Mismatches *container.HashSet
}

func (pair MatchPair) isMatch(value string, param *common.URL) bool {
	if !pair.Matches.Empty() && pair.Mismatches.Empty() {

		for match := range pair.Matches.Items {
			if isMatchGlobPattern(match.(string), value, param) {
				return true
			}
		}
		return false
	}
	if !pair.Mismatches.Empty() && pair.Matches.Empty() {

		for mismatch := range pair.Mismatches.Items {
			if isMatchGlobPattern(mismatch.(string), value, param) {
				return false
			}
		}
		return true
	}
	if !pair.Mismatches.Empty() && !pair.Matches.Empty() {
		for mismatch := range pair.Mismatches.Items {
			if isMatchGlobPattern(mismatch.(string), value, param) {
				return false
			}
		}
		for match := range pair.Matches.Items {
			if isMatchGlobPattern(match.(string), value, param) {
				return true
			}
		}
		return false
	}
	return false
}

func isMatchGlobPattern(pattern string, value string, param *common.URL) bool {
	if param != nil && strings.HasPrefix(pattern, "$") {
		pattern = param.GetRawParam(pattern[1:])
	}
	if "*" == pattern {
		return true
	}
	if len(pattern) == 0 && len(value) == 0 {
		return true
	}
	if len(pattern) == 0 || len(value) == 0 {
		return false
	}
	i := strings.LastIndex(pattern, "*")
	switch i {
	case -1:
		return value == pattern
	case len(pattern) - 1:
		return strings.HasPrefix(value, pattern[0:i])
	case 0:
		return strings.HasSuffix(value, pattern[:i+1])
	default:
		prefix := pattern[0:1]
		suffix := pattern[i+1:]
		return strings.HasPrefix(value, prefix) && strings.HasSuffix(value, suffix)
	}
}
