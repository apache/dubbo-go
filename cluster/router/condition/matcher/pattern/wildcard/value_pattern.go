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

package wildcard

import (
	"math"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher/pattern"
	"dubbo.apache.org/dubbo-go/v3/cluster/utils"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

/**
 * Matches with patterns like 'key=hello', 'key=hello*', 'key=*hello', 'key=h*o' or 'key=*'
 * This pattern evaluator must be the last one being executed.
 */
type WildcardValuePattern struct {
}

func init() {
	extension.SetValuePattern("wildcard", NewValuePattern)
}

func NewValuePattern() pattern.ValuePattern {
	return &WildcardValuePattern{}
}

func (w *WildcardValuePattern) Priority() int64 {
	return math.MaxInt64
}

func (w *WildcardValuePattern) ShouldMatch(pattern string) bool {
	return true
}

func (w *WildcardValuePattern) Match(pattern string, value string, url *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool {
	return utils.IsMatchGlobPattern(pattern, value, url)
}
