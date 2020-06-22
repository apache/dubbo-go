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

package tps

import (
	"sync"
)

import (
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter"
)

func init() {
	extension.SetTpsLimitStrategy("threadSafeFixedWindow", &threadSafeFixedWindowStrategyCreator{
		fixedWindowStrategyCreator: &fixedWindowStrategyCreator{},
	})
}

// ThreadSafeFixedWindowTpsLimitStrategyImpl is the thread-safe implementation.
// It's also a thread-safe decorator of FixedWindowTpsLimitStrategyImpl
/**
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   tps.limiter: "method-service" # the name of limiter
 *   tps.limit.strategy: "threadSafeFixedWindow" # service-level
 *   methods:
 *    - name: "GetUser"
 *      tps.interval: 3000
 *      tps.limit.strategy: "threadSafeFixedWindow" # method-level
 */
type ThreadSafeFixedWindowTpsLimitStrategyImpl struct {
	mutex       *sync.Mutex
	fixedWindow *FixedWindowTpsLimitStrategyImpl
}

// IsAllowable implements thread-safe then run the FixedWindowTpsLimitStrategy
func (impl *ThreadSafeFixedWindowTpsLimitStrategyImpl) IsAllowable() bool {
	impl.mutex.Lock()
	defer impl.mutex.Unlock()
	return impl.fixedWindow.IsAllowable()
}

type threadSafeFixedWindowStrategyCreator struct {
	fixedWindowStrategyCreator *fixedWindowStrategyCreator
}

// Create returns ThreadSafeFixedWindowTpsLimitStrategyImpl instance
func (creator *threadSafeFixedWindowStrategyCreator) Create(rate int, interval int) filter.TpsLimitStrategy {
	fixedWindowStrategy := creator.fixedWindowStrategyCreator.Create(rate, interval).(*FixedWindowTpsLimitStrategyImpl)
	return &ThreadSafeFixedWindowTpsLimitStrategyImpl{
		fixedWindow: fixedWindowStrategy,
		mutex:       &sync.Mutex{},
	}
}
