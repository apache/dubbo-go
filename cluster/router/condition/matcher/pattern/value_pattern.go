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

package pattern

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type ValuePattern interface {
	//ShouldMatch return weather the input pattern of a specific form, for example, range pattern '1~100', wildcard pattern 'hello*', etc.
	ShouldMatch(pattern string) bool
	//Match is the pattern matches with the request context
	Match(pattern string, value string, url *common.URL, invocation protocol.Invocation, isWhenCondition bool) bool
	// Priority Return Priority in valuePattern
	// 0 to ^int(0) is better
	Priority() int64
}
