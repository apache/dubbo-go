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
	"sync/atomic"
	"time"
	"unsafe"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

const (
	secondToMsRate     = int64(time.Second / time.Millisecond)
	msToNanoSecondRate = int64(time.Millisecond / time.Nanosecond)
)

/**
 * Record the accurate number of events within each given time interval, one for each bucket.
 * see com.alibaba.metrics.BucketCounterImpl
 */
type BucketCounterImpl struct {
	totalCount metrics.Counter
	// true -> update totalCount
	// false -> ignored
	updateTotalCount bool

	// latest updated bucket
	latestBucket *bucket

	buckets *bucketDequeue

	/**
	 * the clock is used to get time & time tick.
	 */
	clock metrics.Clock

	/**
	 * interval
	 */
	interval time.Duration

	/**
	 * The time stamp this object has been initialized.
	 * It's the "logic" time.
	 */
	initTimestamp int64
}

// return the last update time
// if the bucket had never been updated, return the init timestamp
func (bci *BucketCounterImpl) LastUpdateTime() int64 {
	result := bci.latestBucket.timestamp
	if result <= 0 {
		return bci.initTimestamp
	}
	return result
}

// you should notice that, if the updateTotalCount is false, 0 will be return
func (bci *BucketCounterImpl) GetCount() int64 {
	return bci.totalCount.GetCount()
}

func (bci *BucketCounterImpl) Inc() {
	bci.Update()
}

func (bci *BucketCounterImpl) IncN(n int64) {
	bci.UpdateN(n)
}

func (bci *BucketCounterImpl) Dec() {
	bci.UpdateN(-1)
}

func (bci *BucketCounterImpl) DecN(n int64) {
	bci.UpdateN(-n)
}

func (bci *BucketCounterImpl) Update() {
	bci.UpdateN(1)
}

func (bci *BucketCounterImpl) UpdateN(n int64) {
	if bci.updateTotalCount {
		bci.totalCount.IncN(n)
	}

	currentTs := bci.alignTimestamp(bci.clock.GetTime())
	oldBucket := bci.latestBucket

	if currentTs > oldBucket.timestamp {
		// we should create a new bucket
		newBucket := &bucket{timestamp: currentTs}
		up := (*unsafe.Pointer)(unsafe.Pointer(&bci.latestBucket))
		if atomic.CompareAndSwapPointer(up, unsafe.Pointer(oldBucket), unsafe.Pointer(newBucket)) {
			// only one goroutine can success
			bci.buckets.addLast(newBucket)
			oldBucket = newBucket
		} else {
			// it means that other goroutine had build a new bucket.
			oldBucket = bci.latestBucket
		}
	}
	atomic.AddInt64(&oldBucket.count, n)
}

func (bci *BucketCounterImpl) GetBucketCounts() map[int64]int64 {
	return bci.GetBucketCountsSince(0)
}

func (bci *BucketCounterImpl) GetBucketCountsSince(startTime int64) map[int64]int64 {

	currentTs := bci.alignTimestamp(bci.clock.GetTime())
	bucketList := bci.buckets.getBucketList()
	result := make(map[int64]int64, len(bucketList))
	for _, value := range bucketList {

		bkTimestampInMs := secondToMsRate * value.timestamp
		if bkTimestampInMs >= startTime && value.timestamp <= currentTs {
			result[bkTimestampInMs] = value.count
		}
	}
	return result
}

func (bci *BucketCounterImpl) GetBucketInterval() time.Duration {
	return bci.interval
}

// it will align the timestmp to N*interval and then convert the timestmp to seconds
// for example: if the timestmp is 11000ms, and the interval is 5s, the result will be 11000/1000 /5 *5 = 10
func (bci *BucketCounterImpl) alignTimestamp(timestmp int64) int64 {
	return timestmp / secondToMsRate / int64(bci.interval.Seconds()) * int64(bci.interval.Seconds())
}

// interval should be one of 1s, 5s, 10s, 30s, 60s
// see Interval
func newBucketCounterImpl(interval time.Duration, numbOfBuckets int,
	clock metrics.Clock, updateTotalCount bool) metrics.BucketCounter {
	buckets := newBucketDequeue(numbOfBuckets)
	return &BucketCounterImpl{
		totalCount:       newCounter(),
		updateTotalCount: updateTotalCount,
		latestBucket:     buckets.peek(),
		buckets:          buckets,
		clock:            clock,
		interval:         interval,
		initTimestamp:    clock.GetTime(),
	}
}
