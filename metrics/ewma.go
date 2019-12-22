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
	"sync"
	"sync/atomic"
	"time"
)

const (
	ewmaInterval = 5
	secondPerMinutes = 60.0
	oneMinute = 1
	fiveMinutes = 5
	fifteenMinutes = 15
	m1Alpha = ewmaInterval / secondPerMinutes / oneMinute
	m5Alpha = ewmaInterval / secondPerMinutes / fiveMinutes
	m15Alpha = ewmaInterval / secondPerMinutes / fifteenMinutes
)

/**
 * An exponentially-weighted moving average.
 *
 * http://www.teamquest.com/pdfs/whitepaper/ldavg1.pdf UNIX Load Average Part 1: How
 *      It Works
 * http://www.teamquest.com/pdfs/whitepaper/ldavg2.pdf UNIX Load Average Part 2: Not
 *      Your Average Average
 * http://en.wikipedia.org/wiki/Moving_average#Exponential_moving_average
 */
type EWMA struct {
	initialized bool
	rate float64
	uncounted int64
	alpha float64
	interval int64

	// using channel to calculate the rate instead of using mutex
	rateChannel chan float64
	// only use in
	initMutex sync.Mutex
	initOnce sync.Once
}

func (ewma *EWMA) Update(n int64) {
	atomic.AddInt64(&ewma.uncounted, n)
}

func (ewma *EWMA) TickN(count int64)  {
	instantRate := float64(count)/ float64(ewma.interval)
	delta := ewma.alpha * (instantRate - ewma.rate)

	// In most case, ewma.initialized is true. So we avoid using the mutex. It's the quick path
	if ewma.initialized {
		ewma.rateChannel <- delta
		return
	}

	// lock and then init.
	ewma.initMutex.Lock()
	defer ewma.initMutex.Unlock()

	// initialized by another thread...
	if ewma.initialized {
		ewma.rateChannel <- delta
		return
	}

	// initializing
	ewma.rate = instantRate
	ewma.initialized = true
}

func (ewma *EWMA) Tick()  {
	old := ewma.uncounted
	// CAS
	for swapped := atomic.CompareAndSwapInt64(&ewma.uncounted, old, 0);!swapped; {
		old = ewma.uncounted
		swapped = atomic.CompareAndSwapInt64(&ewma.uncounted, old, 0)
	}
	ewma.TickN(old)
}

func (ewma *EWMA) start() {
	for {
		delta := <- ewma.rateChannel
		ewma.rate += delta
	}

}

/**
 * return the rate in the given time units of time
 * for example, the timeUnit could be time.SECONDS
 */
func (ewma *EWMA) GetRate(timeUnit time.Duration) float64 {
	return ewma.rate * float64(timeUnit.Nanoseconds())
}

func NewOneMinuteEWMA() *EWMA {
	return newEWMA(m1Alpha, ewmaInterval * time.Second)
}

func NewFiveMinutesEWMA() *EWMA {
	return newEWMA(m5Alpha, ewmaInterval * time.Second)
}

func NewFifteenMinutesEWMA() *EWMA {
	return newEWMA(m15Alpha, ewmaInterval * time.Second)
}

func newEWMA(alpha float64, interval time.Duration) *EWMA{
	result := &EWMA{
		alpha:       alpha,
		interval:    interval.Nanoseconds(),
		// if we found out that the blocking channel is the bottle neck of performance,
		// we should think about using non-blocking channel, which needs more effort to ensure the codes are right
		rateChannel: make(chan float64),
	}
	go result.start()
	return result
}