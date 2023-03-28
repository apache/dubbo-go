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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type ConditionMatcherFactory interface {
	//ShouldMatch return whether the key is of the form of the current matcher type which this factory instance represents..
	ShouldMatch(key string) bool
	//NewMatcher return a matcher instance for the key.
	NewMatcher(key string) ConditionMatcher
	// Priority Return Priority in router
	// 0 to ^int(0) is better
	Priority() int64
}

/*
ConditionMatcher represents a specific match condition of a condition rule.
The following condition rule '=bar&arguments[0]=hello* => region=hangzhou' consists of three ConditionMatchers:
1. param.ConditionMatcher represented by 'foo=bar'
2. argument.ConditionMatcher represented by 'arguments[0]=hello*'
3. param.ConditionMatcher represented by 'region=hangzhou'
*/
type ConditionMatcher interface {
	// IsMatch return weather the patterns of this matcher matches with request context.
	IsMatch(sample map[string]string, param *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool
	// GetMatches return Matches
	// match patterns extracted from when condition
	GetMatches() map[string]struct{}
	// GetMismatches return Mismatches
	// mismatch patterns extracted from then condition
	GetMismatches() map[string]struct{}
	// GetValue return value from different places of the request context, for example, url, attachment and invocation.
	// This makes condition rule possible to check values in any place of a request.
	GetValue(sample map[string]string, url *common.URL, invocation protocol.Invocation) string
}
