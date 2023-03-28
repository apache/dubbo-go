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

package utils

import (
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func IsMatchGlobPattern(pattern string, value string, param *common.URL) bool {
	if param != nil && strings.HasPrefix(pattern, "$") {
		pattern = param.GetRawParam(pattern[1:])
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
