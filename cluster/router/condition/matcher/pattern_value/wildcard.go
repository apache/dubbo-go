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
	"math"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// WildcardValuePattern evaluator must be the last one being executed.
// matches with patterns like 'key=hello', 'key=hello*', 'key=*hello', 'key=h*o' or 'key=*'
type WildcardValuePattern struct {
}

func init() {
	SetValuePattern(constant.Wildcard, NewWildcardValuePattern)
}

func NewWildcardValuePattern() ValuePattern {
	return &WildcardValuePattern{}
}

func (w *WildcardValuePattern) Priority() int64 {
	return math.MaxInt64
}

func (w *WildcardValuePattern) ShouldMatch(pattern string) bool {
	return true
}

func (w *WildcardValuePattern) Match(pattern string, value string, url *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool {
	if url != nil && strings.HasPrefix(pattern, "$") {
		pattern = url.GetRawParam(pattern[1:])
	}

	if "*" == pattern {
		return true
	}
	if pattern == "" && value == "" {
		return true
	}
	if pattern == "" || value == "" {
		return false
	}

	i := strings.LastIndex(pattern, "*")
	// doesn't find "*"
	if i == -1 {
		return value == pattern
	} else if i == len(pattern)-1 {
		// "*" is at the end
		return strings.HasPrefix(value, pattern[0:i])
	} else if i == 0 {
		// "*" is at the beginning
		return strings.HasSuffix(value, pattern[i+1:])
	} else {
		// "*" is in the middle
		prefix := pattern[0:i]
		suffix := pattern[i+1:]
		return strings.HasPrefix(value, prefix) && strings.HasSuffix(value, suffix)
	}
}
