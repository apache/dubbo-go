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
	"time"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/filter"
)

type LeakyBucket struct {
	capacity              int64     // leaky bucket capacity
	leaksIntervalInMillis int64     // leaky bucket flow out rate
	used                  int64     // leaky bucket used amount
	lastLeakTimestamp     time.Time // last leak timestamp
	mu                    *sync.Mutex
}

func NewLeakyBucket(leaksIntervalInMillis int, capacity int) *LeakyBucket {
	return &LeakyBucket{
		capacity:              int64(capacity),
		leaksIntervalInMillis: int64(leaksIntervalInMillis),
		used:                  0,
		lastLeakTimestamp:     time.Now(),
		mu:                    new(sync.Mutex),
	}
}

func (lb *LeakyBucket) tryConsume(flowIn int) bool {
	currentTime := time.Now()
	pastTime := time.Now().Sub(lb.lastLeakTimestamp)

	lb.mu.Lock()
	defer lb.mu.Unlock()
	if currentTime.Sub(lb.lastLeakTimestamp) > 0 {
		leaks := (pastTime.Nanoseconds() / constant.MsToNanoRate) / lb.leaksIntervalInMillis
		if leaks > 0 {
			if lb.used <= leaks {
				lb.used = 0
			} else {
				lb.used -= leaks
			}
		}
		lb.lastLeakTimestamp = time.Now()
	}

	if lb.used+int64(flowIn) > lb.capacity {
		return false
	}
	lb.used = lb.used + int64(flowIn)
	return true
}

func (lb *LeakyBucket) IsAllowable(count int) bool {
	return lb.tryConsume(count)
}

type leakyBucketStrategyCreator struct{}

func (creator *leakyBucketStrategyCreator) Create(rate int, interval int) filter.TpsLimitLeakyBucketStrategy {
	return NewLeakyBucket(rate, interval)
}
