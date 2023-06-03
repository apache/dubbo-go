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

package matcher

import (
	"sort"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher/pattern_value"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	valueMatchers = make([]pattern_value.ValuePattern, 0, 8)

	once sync.Once
)

// BaseConditionMatcher records the match and mismatch patterns of this matcher while at the same time
// provides the common match logics.
type BaseConditionMatcher struct {
	key        string
	matches    map[string]struct{}
	misMatches map[string]struct{}
}

func NewBaseConditionMatcher(key string) *BaseConditionMatcher {
	return &BaseConditionMatcher{
		key:        key,
		matches:    map[string]struct{}{},
		misMatches: map[string]struct{}{},
	}
}

// GetValue returns a value from different places of the request context.
func (b *BaseConditionMatcher) GetValue(sample map[string]string, url *common.URL, invocation protocol.Invocation) string {
	return ""
}

// IsMatch indicates whether this matcher matches the patterns with request context.
func (b *BaseConditionMatcher) IsMatch(value string, param *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool {
	if value == "" {
		// if key does not present in whichever of url, invocation or attachment based on the matcher type, then return false.
		return false
	}

	if len(b.matches) != 0 && len(b.misMatches) == 0 {
		return b.patternMatches(value, param, invocation, isWhenCondition)
	}

	if len(b.misMatches) != 0 && len(b.matches) == 0 {
		return b.patternMisMatches(value, param, invocation, isWhenCondition)
	}

	if len(b.matches) != 0 && len(b.misMatches) != 0 {
		// when both mismatches and matches contain the same value, then using mismatches first
		return b.patternMisMatches(value, param, invocation, isWhenCondition) && b.patternMatches(value, param, invocation, isWhenCondition)
	}
	return false
}

// GetMatches returns matches.
func (b *BaseConditionMatcher) GetMatches() map[string]struct{} {
	return b.matches
}

// GetMismatches returns misMatches.
func (b *BaseConditionMatcher) GetMismatches() map[string]struct{} {
	return b.misMatches
}

func (b *BaseConditionMatcher) patternMatches(value string, param *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool {
	for match := range b.matches {
		if doPatternMatch(match, value, param, invocation, isWhenCondition) {
			return true
		}
	}
	return false
}

func (b *BaseConditionMatcher) patternMisMatches(value string, param *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool {
	for mismatch := range b.misMatches {
		if doPatternMatch(mismatch, value, param, invocation, isWhenCondition) {
			return false
		}
	}
	return true
}

func doPatternMatch(pattern string, value string, url *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool {
	once.Do(initValueMatchers)
	for _, valueMatcher := range valueMatchers {
		if valueMatcher.ShouldMatch(pattern) {
			return valueMatcher.Match(pattern, value, url, invocation, isWhenCondition)
		}
	}
	// If no value matcher is available, will force to use wildcard value matcher
	logger.Error("Executing condition rule value match expression error, will force to use wildcard value matcher")

	valuePattern := pattern_value.GetValuePattern(constant.Wildcard)
	return valuePattern.Match(pattern, value, url, invocation, isWhenCondition)
}

// GetSampleValueFromURL returns the value of the conditionKey in the URL
func GetSampleValueFromURL(conditionKey string, sample map[string]string, param *common.URL, invocation protocol.Invocation) string {
	var sampleValue string
	// get real invoked method name from invocation
	if invocation != nil && (constant.MethodKey == conditionKey || constant.MethodsKey == conditionKey) {
		sampleValue = invocation.MethodName()
	} else {
		sampleValue = sample[conditionKey]
	}
	return sampleValue
}

func Match(condition Matcher, sample map[string]string, param *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool {
	return condition.IsMatch(condition.GetValue(sample, param, invocation), param, invocation, isWhenCondition)
}

func initValueMatchers() {
	valuePatterns := pattern_value.GetValuePatterns()
	for _, valuePattern := range valuePatterns {
		valueMatchers = append(valueMatchers, valuePattern())
	}
	sortValuePattern(valueMatchers)
}

func sortValuePattern(valuePatterns []pattern_value.ValuePattern) {
	sort.Stable(byPriority(valuePatterns))
}

type byPriority []pattern_value.ValuePattern

func (a byPriority) Len() int           { return len(a) }
func (a byPriority) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPriority) Less(i, j int) bool { return a[i].Priority() < a[j].Priority() }
