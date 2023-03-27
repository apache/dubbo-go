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

package argument

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher/pattern"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	ArgumentsPattern = regexp.MustCompile("arguments\\[([0-9]+)\\]")
)

/*
 * analysis the arguments in the rule.
 * Examples would be like this:
 * "arguments[0]=1", whenCondition is that the first argument is equal to '1'.
 * "arguments[1]=a", whenCondition is that the second argument is equal to 'a'.
 */

type ConditionMatcher struct {
	base.ConditionMatcher
}

func NewConditionMatcher(key string) *ConditionMatcher {
	valuePatterns := extension.GetValuePatterns()
	valueMatchers := make([]pattern.ValuePattern, 0, len(valuePatterns))
	for _, valuePattern := range valuePatterns {
		valueMatchers = append(valueMatchers, valuePattern())
	}

	base.SortValuePattern(valueMatchers)

	conditionMatcher := &ConditionMatcher{
		base.ConditionMatcher{
			Key:           key,
			Matches:       map[string]struct{}{},
			Mismatches:    map[string]struct{}{},
			ValueMatchers: valueMatchers,
		},
	}

	conditionMatcher.Matcher = conditionMatcher
	return conditionMatcher
}

func (c *ConditionMatcher) GetValue(sample map[string]string, url *common.URL, invocation protocol.Invocation) string {
	// split the rule
	expressArray := strings.Split(c.Key, "\\.")
	argumentExpress := expressArray[0]
	matcher := ArgumentsPattern.FindStringSubmatch(argumentExpress)
	if len(matcher) == 0 {
		return "dubbo internal not found argument condition value"
	}

	//extract the argument index
	index, err := strconv.Atoi(matcher[1])
	if err != nil {
		logger.Warn("Parse argument match condition failed,Invalid , will ignore., ")
		return "dubbo internal not found argument condition value"
	}
	if index < 0 || index > len(invocation.Arguments()) {
		return "dubbo internal not found argument condition value"
	}
	return fmt.Sprint(invocation.Arguments()[index])
}
