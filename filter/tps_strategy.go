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

package filter

// TpsLimitStrategy is the interface which defines how to do the TPS limiting in method level.
//
// IsAllowable will return true if this invocation is not over limitation.
type TpsLimitStrategy interface {
	IsAllowable() bool
}

// TpsLimitStrategyCreator is the interface which creates TpsLimitStrategy.
type TpsLimitStrategyCreator interface {
	// Create will create an instance of TpsLimitStrategy
	// It will be a little hard to understand this method.
	// The unit of interval is ms
	// for example, if the limit = 100, interval = 1000
	// which means that the limiter limitation is 100 times per 1000ms (100/1000ms)
	Create(limit int, interval int) TpsLimitStrategy
}
