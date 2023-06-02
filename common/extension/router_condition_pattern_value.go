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

package extension

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher/pattern_value"
)

// SetValuePattern sets create valuePattern function with @name
func SetValuePattern(name string, fun func() pattern_value.ValuePattern) {
	pattern_value.SetValuePattern(name, fun)
}

// GetValuePattern gets create valuePattern function by name
func GetValuePattern(name string) pattern_value.ValuePattern {
	return pattern_value.GetValuePattern(name)
}

// GetValuePatterns gets all create valuePattern function
func GetValuePatterns() map[string]func() pattern_value.ValuePattern {
	return pattern_value.GetValuePatterns()
}
