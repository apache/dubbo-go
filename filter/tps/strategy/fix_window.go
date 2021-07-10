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

package strategy

import (
	"sync/atomic"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
)

const (
	// FixedWindowKey defines limiter limit algorithm
	FixedWindowKey = "fixedWindow"
)

func init() {
	creator := &fixedWindowStrategyCreator{}
	extension.SetTpsLimitStrategy(FixedWindowKey, creator)
	extension.SetTpsLimitStrategy(constant.DEFAULT_KEY, creator)
}

// FixedWindowTpsLimitStrategy implements the TPS limit strategy base on requests count during the interval
/**
 * It's the same as default implementation in Java
 * It's not a thread-safe implementation.
 * It you want to use the thread-safe implementation, please use ThreadSafeFixedWindowTpsLimitStrategy
 * This is the default implementation.
 *
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   tps.limiter: "method-service" # the name of limiter
 *   tps.limit.strategy: "default" or "fixedWindow" # service-level
 *   methods:
 *    - name: "GetUser"
 *      tps.interval: 3000
 *      tps.limit.strategy: "default" or "fixedWindow" # method-level
 */
type FixedWindowTpsLimitStrategy struct {
	rate      int32
	interval  int64
	count     int32
	timestamp int64
}

// IsAllowable determines if the requests over the TPS limit within the interval.
// It is not thread-safe.
func (impl *FixedWindowTpsLimitStrategy) IsAllowable() bool {
	current := time.Now().UnixNano()
	if impl.timestamp+impl.interval < current {
		// it's a new window
		// if a lot of threads come here, the count will be set to 0 several times.
		// so the return statement will be wrong.
		impl.timestamp = current
		impl.count = 0
	}
	// this operation is thread-safe, but count + 1 may be overflow
	return atomic.AddInt32(&impl.count, 1) <= impl.rate
}

type fixedWindowStrategyCreator struct{}

// Create returns a FixedWindowTpsLimitStrategy instance with pre-configured limit rate and interval
func (creator *fixedWindowStrategyCreator) Create(rate int, interval int) filter.TpsLimitStrategy {
	return &FixedWindowTpsLimitStrategy{
		rate:      int32(rate),
		interval:  int64(interval) * int64(time.Millisecond), // convert to ns
		count:     0,
		timestamp: time.Now().UnixNano(),
	}
}
