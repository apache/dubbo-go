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

package scope

import (
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher/pattern"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

/**
 * Matches with patterns like 'key=1~100', 'key=~100' or 'key=1~'
 */

type ValuePattern struct {
}

func init() {
	extension.SetValuePattern("scope", NewValuePattern)
}

func NewValuePattern() pattern.ValuePattern {
	return &ValuePattern{}
}

func (v *ValuePattern) Priority() int64 {
	return 100
}

func (v *ValuePattern) ShouldMatch(pattern string) bool {
	return strings.Contains(pattern, "~")
}

func (v *ValuePattern) Match(pattern string, value string, url *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool {
	defaultValue := !isWhenCondition

	intValue, err := strconv.Atoi(value)
	if err != nil {
		logger.Errorf("Parse integer error,Invalid condition rule '%s' or value '%s', will ignore.", pattern, value)
		return defaultValue
	}

	arr := strings.Split(pattern, "~")
	if len(arr) < 2 {
		logger.Errorf("Invalid condition rule %s or value %s, will ignore.", pattern, value)
		return defaultValue
	}

	rawStart := arr[0]
	rawEnd := arr[1]

	if rawStart == "" && rawEnd == "" {
		return defaultValue
	}

	if rawStart == "" {
		end, err := strconv.Atoi(rawEnd)
		if err != nil {
			logger.Errorf("Parse integer error,Invalid condition rule '%s' or value '%s', will ignore.", pattern, value)
			return defaultValue
		}
		if intValue > end {
			return false
		}
	} else if rawEnd == "" {
		start, err := strconv.Atoi(rawStart)
		if err != nil {
			logger.Errorf("Parse integer error,Invalid condition rule '%s' or value '%s', will ignore.", pattern, value)
			return defaultValue
		}
		if intValue < start {
			return false
		}
	} else {
		start, err := strconv.Atoi(rawStart)
		if err != nil {
			logger.Errorf("Parse integer error,Invalid condition rule '%s' or value '%s', will ignore.", pattern, value)
			return defaultValue
		}
		end, err := strconv.Atoi(rawEnd)
		if err != nil {
			logger.Errorf("Parse integer error,Invalid condition rule '%s' or value '%s', will ignore.", pattern, value)
			return defaultValue
		}
		if intValue < start || intValue > end {
			return false
		}
	}

	return true
}
