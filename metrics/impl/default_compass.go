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
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

const tickInterval = 5 * time.Second

type DefaultCompass struct {
	// The count per collect interval
	totalCount metrics.BucketCounter
	// The number of successful count per collect interval
	successCount metrics.BucketCounter
	// The number of error count per code per collect interval
	errorCodes      map[string]metrics.BucketCounter
	errorCodesMutex sync.Mutex

	// The number of addon count per addon per collect interval
	addons      map[string]metrics.BucketCounter
	addonsMutex sync.Mutex

	// 1min moving average
	m1Rate metrics.EWMA
	// 5min moving average
	m5Rate metrics.EWMA
	// 15min moving average
	m15Rate metrics.EWMA

	// usually, it's the real time.
	// But it could be "logic" time, which depends on the Clock's implementation
	startTime int64

	// the last tick timestamp
	lastTick int64

	// The number of events that is not update to moving average yet
	uncounted int64

	// The clock implementation
	clock metrics.Clock

	// The max number of error code that is allowed
	maxErrorCodeCount int

	// The max number of addon that is allowed
	maxAddonCount int

	// The collect interval
	bucketInterval time.Duration

	// The number of bucket
	numberOfBucket int

	reservoir metrics.Reservoir
}

func (cp *DefaultCompass) GetInstantCountInterval() time.Duration {
	return cp.bucketInterval
}

func (cp *DefaultCompass) GetFifteenMinuteRate() float64 {
	cp.tickIfNecessary()
	return cp.m15Rate.GetRate(time.Second)
}

func (cp *DefaultCompass) GetFiveMinuteRate() float64 {
	cp.tickIfNecessary()
	return cp.m5Rate.GetRate(time.Second)
}

func (cp *DefaultCompass) GetMeanRate() float64 {
	count := cp.GetCount()
	elapsed := cp.clock.GetTick() - cp.startTime
	if count == 0 || elapsed <= 0 {
		return 0
	}
	return float64(count) / float64(elapsed) * float64(time.Second.Nanoseconds())
}

func (cp *DefaultCompass) GetOneMinuteRate() float64 {
	cp.tickIfNecessary()
	return cp.m1Rate.GetRate(time.Second)
}

func (cp *DefaultCompass) GetBucketSuccessCount() metrics.BucketCounter {
	return cp.successCount
}

func (cp *DefaultCompass) GetAddonCounts() map[string]metrics.BucketCounter {
	result := make(map[string]metrics.BucketCounter, len(cp.addons))
	for key, value := range cp.addons {
		result[key] = value
	}
	return result
}

func (cp *DefaultCompass) GetErrorCodeCounts() map[string]metrics.BucketCounter {
	result := make(map[string]metrics.BucketCounter, len(cp.errorCodes))
	for key, value := range cp.errorCodes {
		result[key] = value
	}
	return result
}

func (cp *DefaultCompass) GetSuccessCount() int64 {
	return cp.successCount.GetCount()
}

func (cp *DefaultCompass) GetInstantCount() map[int64]int64 {
	return cp.totalCount.GetBucketCounts()
}

func (cp *DefaultCompass) GetInstantCountSince(startTime int64) map[int64]int64 {
	return cp.totalCount.GetBucketCountsSince(startTime)
}

func (cp *DefaultCompass) LastUpdateTime() int64 {
	return cp.totalCount.LastUpdateTime()
}

func (cp *DefaultCompass) GetCount() int64 {
	return cp.totalCount.GetCount()
}

func (cp *DefaultCompass) UpdateWithError(duration time.Duration, isSuccess bool, errorCode string, addon string) {
	cp.Update(duration)
	if isSuccess {
		cp.successCount.Update()
	}

	if len(errorCode) > 0 {

		cp.errorCodesMutex.Lock()
		errorCounter := cp.findOrNewCounter(cp.errorCodes, errorCode, cp.maxErrorCodeCount)
		cp.errorCodesMutex.Unlock()
		errorCounter.Update()
	}

	if len(addon) > 0 {
		cp.addonsMutex.Lock()
		addonCounter := cp.findOrNewCounter(cp.addons, addon, cp.maxAddonCount)
		cp.addonsMutex.Unlock()
		addonCounter.Update()
	}
}

func (cp *DefaultCompass) findOrNewCounter(counters map[string]metrics.BucketCounter, key string, maxCount int) metrics.BucketCounter {
	counter, loaded := counters[key]
	if !loaded && len(counters) < maxCount {
		counter = newBucketCounterImpl(cp.bucketInterval, cp.numberOfBucket, cp.clock, true)
		counters[key] = counter
	}
	return counter
}

func (cp *DefaultCompass) Update(duration time.Duration) {
	cp.tickIfNecessary()
	atomic.AddInt64(&cp.uncounted, 1)
	cp.totalCount.Update()
	cp.reservoir.UpdateN(duration.Nanoseconds())
}

func (cp *DefaultCompass) GetSnapshot() metrics.Snapshot {
	return cp.reservoir.GetSnapshot()
}

func (cp *DefaultCompass) tickIfNecessary() {
	oldTick := cp.lastTick
	newTick := cp.clock.GetTick()
	age := newTick - oldTick
	if age > tickInterval.Nanoseconds() {
		newIntervalStartTick := newTick - age%cp.bucketInterval.Nanoseconds()
		if atomic.CompareAndSwapInt64(&cp.lastTick, oldTick, newIntervalStartTick) {
			requiredTicks := age / tickInterval.Nanoseconds()
			for i := 0; int64(i) < requiredTicks; i++ {

				// fetch the uncounted's value and then reset it to 0
				count := atomic.LoadInt64(&cp.uncounted)
				for !atomic.CompareAndSwapInt64(&cp.uncounted, count, 0) {
					count = atomic.LoadInt64(&cp.uncounted)
				}
				cp.m1Rate.TickN(count)
				cp.m5Rate.TickN(count)
				cp.m15Rate.TickN(count)
			}
		}
	}
}

func NewCompassWithType(reservoirType metrics.ReservoirType, clock metrics.Clock,
	numOfBucket int, bucketInterval time.Duration,
	maxErrorCodeCount int, maxAddonCount int) metrics.Compass {
	totalCount := newBucketCounterImpl(bucketInterval, numOfBucket, clock, true)
	now := clock.GetTime()
	var reservoir metrics.Reservoir
	switch reservoirType {
	case metrics.ExponentiallyDecayingReservoirType:
		reservoir = NewExponentiallyDecayingReservoir(EdrDefaultSize, EdrDefaultAlpha, clock)
	case metrics.BucketReservoirType:
		reservoir = NewBucketReservoir(bucketInterval, numOfBucket, clock, totalCount)
	case metrics.UniformReservoirType:
		reservoir = NewUniformReservoir(defaultUniformReservoirSize)
	default:
		reservoir = NewExponentiallyDecayingReservoir(EdrDefaultSize, EdrDefaultAlpha, clock)
	}

	return &DefaultCompass{
		reservoir:         reservoir,
		bucketInterval:    bucketInterval,
		clock:             clock,
		numberOfBucket:    numOfBucket,
		maxAddonCount:     maxAddonCount,
		maxErrorCodeCount: maxErrorCodeCount,

		totalCount:   totalCount,
		startTime:    now,
		lastTick:     now,
		successCount: newBucketCounterImpl(bucketInterval, numOfBucket, clock, true),
		errorCodes:   make(map[string]metrics.BucketCounter, maxErrorCodeCount),
		addons:       make(map[string]metrics.BucketCounter, maxAddonCount),
	}
}

func NewCompass(reservoir metrics.Reservoir, clock metrics.Clock,
	numOfBucket int, bucketInterval time.Duration,
	maxErrorCodeCount int, maxAddonCount int) metrics.Compass {
	now := clock.GetTime()
	return &DefaultCompass{
		reservoir:         reservoir,
		bucketInterval:    bucketInterval,
		clock:             clock,
		numberOfBucket:    numOfBucket,
		maxAddonCount:     maxAddonCount,
		maxErrorCodeCount: maxErrorCodeCount,

		totalCount:   newBucketCounterImpl(bucketInterval, numOfBucket, clock, true),
		startTime:    now,
		lastTick:     now,
		successCount: newBucketCounterImpl(bucketInterval, numOfBucket, clock, true),
		errorCodes:   make(map[string]metrics.BucketCounter, maxErrorCodeCount),
		addons:       make(map[string]metrics.BucketCounter, maxAddonCount),
	}
}
