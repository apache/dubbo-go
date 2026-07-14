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
	"sync"
)

var (
	valuePatternsMu sync.RWMutex
	valuePatterns   = make(map[string]func() ValuePattern)
)

// SetValuePattern sets create valuePattern function with @name
func SetValuePattern(name string, fun func() ValuePattern) {
	valuePatternsMu.Lock()
	defer valuePatternsMu.Unlock()
	valuePatterns[name] = fun
}

// GetValuePattern gets create valuePattern function by name
func GetValuePattern(name string) ValuePattern {
	valuePatternsMu.RLock()
	pattern := valuePatterns[name]
	valuePatternsMu.RUnlock()
	if pattern == nil {
		panic("value_pattern for " + name + " is not existing, make sure you have imported the package.")
	}
	return pattern()
}

// GetValuePatterns gets all create valuePattern function
func GetValuePatterns() map[string]func() ValuePattern {
	valuePatternsMu.RLock()
	defer valuePatternsMu.RUnlock()
	patterns := make(map[string]func() ValuePattern, len(valuePatterns))
	for name, pattern := range valuePatterns {
		patterns[name] = pattern
	}
	return patterns
}
