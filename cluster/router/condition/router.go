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
	"strings"
)

import (
	"github.com/RoaringBitmap/roaring"
	"github.com/dubbogo/gost/container/set"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/utils"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

const (
	// pattern route pattern regex
	pattern = `([&!=,]*)\\s*([^&!=,\\s]+)`
)

var (
	routerPatternReg = regexp.MustCompile(`([&!=,]*)\s*([^&!=,\s]+)`)
)

// ConditionRouter Condition router struct
type ConditionRouter struct {
	Pattern       string
	url           *common.URL
	priority      int64
	Force         bool
	enabled       bool
	WhenCondition map[string]MatchPair
	ThenCondition map[string]MatchPair
}

// NewConditionRouterWithRule Init condition router by raw rule
func NewConditionRouterWithRule(rule string) (*ConditionRouter, error) {
	var (
		whenRule string
		thenRule string
		when     map[string]MatchPair
		then     map[string]MatchPair
	)
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
		Pattern:       pattern,
		WhenCondition: when,
		ThenCondition: then,
	}, nil
}

// NewConditionRouter Init condition router by URL
func NewConditionRouter(url *common.URL) (*ConditionRouter, error) {
	if url == nil {
		return nil, perrors.Errorf("Illegal route URL!")
	}
	rule, err := url.GetParamAndDecoded(constant.RULE_KEY)
	if err != nil || len(rule) == 0 {
		return nil, perrors.Errorf("Illegal route rule!")
	}

	router, err := NewConditionRouterWithRule(rule)
	if err != nil {
		return nil, err
	}

	router.url = url
	var defaultPriority int64 = 0
	if url.GetParam(constant.APPLICATION_KEY, "") != "" {
		defaultPriority = 150
	} else if url.GetParam(constant.INTERFACE_KEY, "") != "" {
		defaultPriority = 140
	}
	router.priority = url.GetParamInt(constant.RouterPriority, defaultPriority)
	router.Force = url.GetParamBool(constant.RouterForce, false)
	router.enabled = url.GetParamBool(constant.RouterEnabled, true)

	return router, nil
}

// Priority Return Priority in condition router
func (c *ConditionRouter) Priority() int64 {
	return c.priority
}

// URL Return URL in condition router
func (c *ConditionRouter) URL() *common.URL {
	return c.url
}

// Enabled Return is condition router is enabled
// true: enabled
// false: disabled
func (c *ConditionRouter) Enabled() bool {
	return c.enabled
}

// Route Determine the target invokers list.
func (c *ConditionRouter) Route(invokers *roaring.Bitmap, cache router.Cache, url *common.URL, invocation protocol.Invocation) *roaring.Bitmap {
	if !c.Enabled() {
		return invokers
	}

	if invokers.IsEmpty() {
		return invokers
	}

	isMatchWhen := c.MatchWhen(url, invocation)
	if !isMatchWhen {
		return invokers
	}

	if len(c.ThenCondition) == 0 {
		return utils.EmptyAddr
	}

	result := roaring.NewBitmap()
	for iter := invokers.Iterator(); iter.HasNext(); {
		index := iter.Next()
		invoker := cache.GetInvokers()[index]
		invokerUrl := invoker.GetUrl()
		isMatchThen := c.MatchThen(invokerUrl, url)
		if isMatchThen {
			result.Add(index)
		}
	}

	if !result.IsEmpty() {
		return result
	} else if c.Force {
		rule, _ := url.GetParamAndDecoded(constant.RULE_KEY)
		localIP := common.GetLocalIp()
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
	values := gxset.NewSet()
	matches := routerPatternReg.FindAllSubmatch([]byte(rule), -1)
	for _, groups := range matches {
		separator := string(groups[1])
		content := string(groups[2])
		switch separator {
		case "":
			pair = MatchPair{
				Matches:    gxset.NewSet(),
				Mismatches: gxset.NewSet(),
			}
			condition[content] = pair
		case "&":
			if r, ok := condition[content]; ok {
				pair = r
			} else {
				pair = MatchPair{
					Matches:    gxset.NewSet(),
					Mismatches: gxset.NewSet(),
				}
				condition[content] = pair
			}
		case "=":
			if &pair == nil {
				var startIndex = getStartIndex(rule)
				return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex, startIndex)
			}
			values = pair.Matches
			values.Add(content)
		case "!=":
			if &pair == nil {
				var startIndex = getStartIndex(rule)
				return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex, startIndex)
			}
			values = pair.Mismatches
			values.Add(content)
		case ",":
			if values.Empty() {
				var startIndex = getStartIndex(rule)
				return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex, startIndex)
			}
			values.Add(content)
		default:
			var startIndex = getStartIndex(rule)
			return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex, startIndex)

		}
	}
	return condition, nil
}

func getStartIndex(rule string) int {
	if indexTuple := routerPatternReg.FindIndex([]byte(rule)); len(indexTuple) > 0 {
		return indexTuple[0]
	}
	return -1
}

// MatchWhen MatchWhen
func (c *ConditionRouter) MatchWhen(url *common.URL, invocation protocol.Invocation) bool {
	condition := matchCondition(c.WhenCondition, url, nil, invocation)
	return len(c.WhenCondition) == 0 || condition
}

// MatchThen MatchThen
func (c *ConditionRouter) MatchThen(url *common.URL, param *common.URL) bool {
	condition := matchCondition(c.ThenCondition, url, param, nil)
	return len(c.ThenCondition) > 0 && condition
}

// MatchCondition MatchCondition
func matchCondition(pairs map[string]MatchPair, url *common.URL, param *common.URL, invocation protocol.Invocation) bool {
	sample := url.ToMap()
	if sample == nil {
		// because url.ToMap() may return nil, but it should continue to process make condition
		sample = make(map[string]string)
	}
	var result bool
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
				return false
			}

			result = true
		} else {
			if !(matchPair.Matches.Empty()) {
				return false
			}

			result = true
		}
	}
	return result
}

// MatchPair Match key pair, condition process
type MatchPair struct {
	Matches    *gxset.HashSet
	Mismatches *gxset.HashSet
}

func (pair MatchPair) isMatch(value string, param *common.URL) bool {
	if !pair.Matches.Empty() && pair.Mismatches.Empty() {

		for match := range pair.Matches.Items {
			if isMatchGlobalPattern(match.(string), value, param) {
				return true
			}
		}
		return false
	}
	if !pair.Mismatches.Empty() && pair.Matches.Empty() {

		for mismatch := range pair.Mismatches.Items {
			if isMatchGlobalPattern(mismatch.(string), value, param) {
				return false
			}
		}
		return true
	}
	if !pair.Mismatches.Empty() && !pair.Matches.Empty() {
		// when both mismatches and matches contain the same value, then using mismatches first
		for mismatch := range pair.Mismatches.Items {
			if isMatchGlobalPattern(mismatch.(string), value, param) {
				return false
			}
		}
		for match := range pair.Matches.Items {
			if isMatchGlobalPattern(match.(string), value, param) {
				return true
			}
		}
		return false
	}
	return false
}
