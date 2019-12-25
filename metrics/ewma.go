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

package metrics

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ewmaInterval     = 5
	secondPerMinutes = 60.0
	oneMinute        = 1
	fiveMinutes      = 5
	fifteenMinutes   = 15
)

var (
	m1Alpha  = 1 - math.Exp(-ewmaInterval/secondPerMinutes/oneMinute)
	m5Alpha  = 1 - math.Exp(-ewmaInterval/secondPerMinutes/fiveMinutes)
	m15Alpha = 1 - math.Exp(-ewmaInterval/secondPerMinutes/fifteenMinutes)
)

/**
 * An exponentially-weighted moving average.
 *
 * http://www.teamquest.com/pdfs/whitepaper/ldavg1.pdf UNIX Load Average Part 1: How
 *      It Works
 * http://www.teamquest.com/pdfs/whitepaper/ldavg2.pdf UNIX Load Average Part 2: Not
 *      Your Average Average
 * http://en.wikipedia.org/wiki/Moving_average#Exponential_moving_average
 *
 * I had try to replace mutex with channel, but fail. The TickN function needs to synchronized strictly.
 */
type EWMA struct {
	initialized bool
	rate        float64
	uncounted   int64
	alpha       float64
	interval    int64

	mutex sync.Mutex
}

func (ewma *EWMA) Update(n int64) {
	atomic.AddInt64(&ewma.uncounted, n)
}

func (ewma *EWMA) TickN(count int64) {
	instantRate := float64(count) / float64(ewma.interval)
	ewma.mutex.Lock()
	defer ewma.mutex.Unlock()

	if ewma.initialized {
		delta := ewma.alpha * (instantRate - ewma.rate)
		ewma.rate += delta
		return
	}

	// initializing
	ewma.rate = instantRate
	ewma.initialized = true
}

func (ewma *EWMA) Tick() {
	old := ewma.uncounted
	// CAS
	for swapped := atomic.CompareAndSwapInt64(&ewma.uncounted, old, 0); !swapped; {
		old = ewma.uncounted
		swapped = atomic.CompareAndSwapInt64(&ewma.uncounted, old, 0)
	}
	ewma.TickN(old)
}

/**
 * return the rate in the given time units of time
 * for example, the timeUnit could be time.SECONDS
 */
func (ewma *EWMA) GetRate(timeUnit time.Duration) float64 {
	return ewma.rate * float64(timeUnit.Nanoseconds())
}

func NewOneMinuteEWMA() *EWMA {
	return newEWMA(m1Alpha, ewmaInterval*time.Second)
}

func NewFiveMinutesEWMA() *EWMA {
	return newEWMA(m5Alpha, ewmaInterval*time.Second)
}

func NewFifteenMinutesEWMA() *EWMA {
	return newEWMA(m15Alpha, ewmaInterval*time.Second)
}

func newEWMA(alpha float64, interval time.Duration) *EWMA {
	result := &EWMA{
		alpha:    alpha,
		interval: interval.Nanoseconds(),
	}
	return result
}
