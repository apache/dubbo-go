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

package pattern_value

import (
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// ScopeValuePattern matches with patterns like 'key=1~100', 'key=~100' or 'key=1~'
type ScopeValuePattern struct {
}

func init() {
	SetValuePattern(constant.Scope, NewScopeValuePattern)
}

func NewScopeValuePattern() ValuePattern {
	return &ScopeValuePattern{}
}

func (s *ScopeValuePattern) Priority() int64 {
	return 100
}

func (s *ScopeValuePattern) ShouldMatch(pattern string) bool {
	return strings.Contains(pattern, "~")
}

func (s *ScopeValuePattern) Match(pattern string, value string, url *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool {
	defaultValue := !isWhenCondition
	intValue, err := strconv.Atoi(value)
	if err != nil {
		logError(pattern, value)
		return defaultValue
	}

	arr := strings.Split(pattern, "~")
	if len(arr) < 2 {
		logError(pattern, value)
		return defaultValue
	}

	rawStart := arr[0]
	rawEnd := arr[1]

	if rawStart == "" && rawEnd == "" {
		return defaultValue
	}

	return s.matchRange(intValue, rawStart, rawEnd, defaultValue, pattern, value)
}

func (s *ScopeValuePattern) matchRange(intValue int, rawStart, rawEnd string, defaultValue bool, pattern, value string) bool {
	if rawStart == "" {
		end, err := strconv.Atoi(rawEnd)
		if err != nil {
			logError(pattern, value)
			return defaultValue
		}
		if intValue > end {
			return false
		}
	} else if rawEnd == "" {
		start, err := strconv.Atoi(rawStart)
		if err != nil {
			logError(pattern, value)
			return defaultValue
		}
		if intValue < start {
			return false
		}
	} else {
		start, end, err := convertToIntRange(rawStart, rawEnd)
		if err != nil {
			logError(pattern, value)
			return defaultValue
		}
		if intValue < start || intValue > end {
			return false
		}
	}

	return true
}

func convertToIntRange(rawStart, rawEnd string) (int, int, error) {
	start, err := strconv.Atoi(rawStart)
	if err != nil {
		return 0, 0, err
	}
	end, err := strconv.Atoi(rawEnd)
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

func logError(pattern, value string) {
	logger.Errorf("Parse integer error, Invalid condition rule '%s' or value '%s', will ignore.", pattern, value)
}
