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

package impl

import (
	"container/list"
	"sync"
	"time"
)

import (
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter/impl/tps"
)

func init() {
	extension.SetTpsLimitStrategy("slidingWindow", &slidingWindowStrategyCreator{})
}

/**
 * it's thread-safe.
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   tps.limiter: "method-service" # the name of limiter
 *   tps.limit.strategy: "slidingWindow" # service-level
 *   methods:
 *    - name: "GetUser"
 *      tps.interval: 3000
 *      tps.limit.strategy: "slidingWindow" # method-level
 */
type SlidingWindowTpsLimitStrategyImpl struct {
	rate     int
	interval int64
	mutex    *sync.Mutex
	queue    *list.List
}

func (impl *SlidingWindowTpsLimitStrategyImpl) IsAllowable() bool {
	impl.mutex.Lock()
	defer impl.mutex.Unlock()
	// quick path
	size := impl.queue.Len()
	current := time.Now().UnixNano()
	if size < impl.rate {
		impl.queue.PushBack(current)
		return true
	}

	// slow path
	boundary := current - impl.interval

	timestamp := impl.queue.Front()
	// remove the element that out of the window
	for timestamp != nil && timestamp.Value.(int64) < boundary {
		impl.queue.Remove(timestamp)
		timestamp = impl.queue.Front()
	}
	if impl.queue.Len() < impl.rate {
		impl.queue.PushBack(current)
		return true
	}
	return false
}

type slidingWindowStrategyCreator struct{}

func (creator *slidingWindowStrategyCreator) Create(rate int, interval int) tps.TpsLimitStrategy {
	return &SlidingWindowTpsLimitStrategyImpl{
		rate:     rate,
		interval: int64(interval) * int64(time.Millisecond),
		mutex:    &sync.Mutex{},
		queue:    list.New(),
	}
}
