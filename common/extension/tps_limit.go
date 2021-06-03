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
	"dubbo.apache.org/dubbo-go/v3/filter"
)

var (
	tpsLimitStrategy = make(map[string]filter.TpsLimitStrategyCreator)
	tpsLimiter       = make(map[string]func() filter.TpsLimiter)
)

// SetTpsLimiter sets the TpsLimiter with @name
func SetTpsLimiter(name string, creator func() filter.TpsLimiter) {
	tpsLimiter[name] = creator
}

// GetTpsLimiter finds the TpsLimiter with @name
func GetTpsLimiter(name string) filter.TpsLimiter {
	creator, ok := tpsLimiter[name]
	if !ok {
		panic("TpsLimiter for " + name + " is not existing, make sure you have import the package " +
			"and you have register it by invoking extension.SetTpsLimiter.")
	}
	return creator()
}

// SetTpsLimitStrategy sets the TpsLimitStrategyCreator with @name
func SetTpsLimitStrategy(name string, creator filter.TpsLimitStrategyCreator) {
	tpsLimitStrategy[name] = creator
}

// GetTpsLimitStrategyCreator finds the TpsLimitStrategyCreator with @name
func GetTpsLimitStrategyCreator(name string) filter.TpsLimitStrategyCreator {
	creator, ok := tpsLimitStrategy[name]
	if !ok {
		panic("TpsLimitStrategy for " + name + " is not existing, make sure you have import the package " +
			"and you have register it by invoking extension.SetTpsLimitStrategy.")
	}
	return creator
}
