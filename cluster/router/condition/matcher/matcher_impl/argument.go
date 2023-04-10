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
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

import (
	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
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

type ArgumentConditionMatcher struct {
	BaseConditionMatcher
}

func NewArgumentConditionMatcher(key string) *ArgumentConditionMatcher {
	conditionMatcher := &ArgumentConditionMatcher{
		*NewBaseConditionMatcher(key),
	}
	conditionMatcher.Matcher = conditionMatcher
	return conditionMatcher
}

func (a *ArgumentConditionMatcher) GetValue(sample map[string]string, url *common.URL, invocation protocol.Invocation) (string, error) {
	// split the rule
	expressArray := strings.Split(a.Key, "\\.")
	argumentExpress := expressArray[0]
	matcher := ArgumentsPattern.FindStringSubmatch(argumentExpress)
	if len(matcher) == 0 {
		return "", errors.Errorf("dubbo internal not found argument condition value")
	}

	// extract the argument index
	index, err := strconv.Atoi(matcher[1])
	if err != nil {
		return "", errors.Errorf("dubbo internal not found argument condition value")
	}
	if index < 0 || index > len(invocation.Arguments()) {
		return "", errors.Errorf("dubbo internal not found argument condition value")
	}
	return fmt.Sprint(invocation.Arguments()[index]), nil
}
