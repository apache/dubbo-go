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

package matcher_impl

import (
	"sort"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher"
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher/pattern"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// BaseConditionMatcher records the match and mismatch patterns of this matcher while at the same time
// provides the common match logics.
type BaseConditionMatcher struct {
	Matches       map[string]struct{}
	Mismatches    map[string]struct{}
	ValueMatchers []pattern.ValuePattern

	Matcher matcher.ConditionMatcher
	Key     string
}

func NewBaseConditionMatcher(key string) *BaseConditionMatcher {
	valuePatterns := extension.GetValuePatterns()
	valueMatchers := make([]pattern.ValuePattern, 0, len(valuePatterns))
	for _, valuePattern := range valuePatterns {
		valueMatchers = append(valueMatchers, valuePattern())
	}

	SortValuePattern(valueMatchers)

	b := &BaseConditionMatcher{
		Key:           key,
		Matches:       map[string]struct{}{},
		Mismatches:    map[string]struct{}{},
		ValueMatchers: valueMatchers,
	}

	return b
}

func (b *BaseConditionMatcher) GetValue(sample map[string]string, url *common.URL, invocation protocol.Invocation) (string, error) {
	return "", nil
}

func (b *BaseConditionMatcher) GetSampleValueFromURL(conditionKey string, sample map[string]string, param *common.URL, invocation protocol.Invocation) string {
	var sampleValue string
	// get real invoked method name from invocation
	if invocation != nil && (constant.MethodKey == conditionKey || constant.MethodsKey == conditionKey) {
		sampleValue = invocation.MethodName()
	} else {
		sampleValue = sample[conditionKey]
	}
	return sampleValue
}

func (b *BaseConditionMatcher) IsMatch(sample map[string]string, param *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool {
	value, err := b.Matcher.GetValue(sample, param, invocation)
	if err != nil {
		logger.Error(err)
		return false
	}

	if value == "" {
		// if key does not present in whichever of url, invocation or attachment based on the matcher type, then return false.
		return false
	}

	if len(b.Matches) != 0 && len(b.Mismatches) == 0 {
		for match := range b.Matches {
			if b.DoPatternMatch(match, value, param, invocation, isWhenCondition) {
				return true
			}
		}
		return false
	}

	if len(b.Mismatches) != 0 && len(b.Matches) == 0 {
		for mismatch := range b.Mismatches {
			if b.DoPatternMatch(mismatch, value, param, invocation, isWhenCondition) {
				return false
			}
		}
		return true
	}

	if len(b.Matches) != 0 && len(b.Mismatches) != 0 {
		// when both mismatches and matches contain the same value, then using mismatches first
		for mismatch := range b.Mismatches {
			if b.DoPatternMatch(mismatch, value, param, invocation, isWhenCondition) {
				return false
			}
		}

		for match := range b.Matches {
			if b.DoPatternMatch(match, value, param, invocation, isWhenCondition) {
				return true
			}
		}
		return false
	}
	return false
}

func (b *BaseConditionMatcher) GetMatches() map[string]struct{} {
	return b.Matches
}

func (b *BaseConditionMatcher) GetMismatches() map[string]struct{} {
	return b.Mismatches
}

func (b *BaseConditionMatcher) DoPatternMatch(pattern string, value string, url *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool {
	for _, valueMatcher := range b.ValueMatchers {
		if valueMatcher.ShouldMatch(pattern) {
			return valueMatcher.Match(pattern, value, url, invocation, isWhenCondition)
		}
	}
	// If no value matcher is available, will force to use wildcard value matcher
	logger.Error("Executing condition rule value match expression error, will force to use wildcard value matcher")

	valuePattern := extension.GetValuePattern("wildcard")
	return valuePattern.Match(pattern, value, url, invocation, isWhenCondition)
}

func SortValuePattern(valuePatterns []pattern.ValuePattern) {
	sort.Stable(byPriority(valuePatterns))
}

type byPriority []pattern.ValuePattern

func (a byPriority) Len() int           { return len(a) }
func (a byPriority) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPriority) Less(i, j int) bool { return a[i].Priority() < a[j].Priority() }
