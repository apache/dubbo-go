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
	"github.com/apache/dubbo-go/filter"
)

const milliSecondUnit = int64(time.Nanosecond * 1e3)

type leakyBucket struct {
	lck           *sync.Mutex
	rate          int64     // water flow out rate
	balance       int64     // leaky bucket allowance
	limit         int64     // leaky bucket max limit
	lastCkeckTime time.Time // last check time
}

func NewLeakyBucket(limitPerMillisecond int, balance int) *leakyBucket {
	return &leakyBucket{
		lck:           new(sync.Mutex),
		rate:          int64(limitPerMillisecond),
		balance:       int64(balance),
		limit:         int64(balance),
		lastCkeckTime: time.Now(),
	}
}

func (lb *leakyBucket) Check() bool {
	ok := false
	lb.lck.Lock()
	now := time.Now()
	dur := now.Sub(lb.lastCkeckTime).Nanoseconds() / 1e6
	lb.lastCkeckTime = now
	water := dur * lb.rate // water flow out from leaky bucket during "dur" time
	lb.balance += water    // allowance + flow water is the total amount of leaky bucket
	if lb.balance < 0 {
		lb.balance = 0
	}

	if lb.balance >= lb.limit { // allowance enough to hold the request
		lb.balance -= lb.limit
		ok = true
	}
	lb.lck.Unlock()
	return ok
}

func (lb *leakyBucket) IsAllowable() bool {
	return lb.Check()
}

type leakyBucketStrategyCreator struct{}

func (creator *leakyBucketStrategyCreator) Create(rate int, interval int) filter.TpsLimitStrategy {
	return NewLeakyBucket(rate, interval)
}
