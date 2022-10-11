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

package rolling

import (
	"sync"
	"time"
)

// SlidingWindowCounter is a policy for ring window based on time duration.
// SlidingWindowCounter moves bucket offset with time duration.
// e.g. If the last point is appended one bucket duration ago,
// SlidingWindowCounter will increment current offset.
type SlidingWindowCounter struct {
	size           int
	mu             sync.Mutex
	buckets        []float64
	count          float64
	offset         int
	bucketDuration time.Duration
	lastAppendTime time.Time
}

// SlidingWindowCounterOpts contains the arguments for creating SlidingWindowCounter.
type SlidingWindowCounterOpts struct {
	Size           int
	BucketDuration time.Duration
}

// NewSlidingWindowCounter creates a new SlidingWindowCounter based on the given window and SlidingWindowCounterOpts.
func NewSlidingWindowCounter(opts SlidingWindowCounterOpts) *SlidingWindowCounter {
	buckets := make([]float64, opts.Size)

	return &SlidingWindowCounter{
		size:           opts.Size,
		offset:         0,
		buckets:        buckets,
		bucketDuration: opts.BucketDuration,
		lastAppendTime: time.Now(),
	}
}

func (c *SlidingWindowCounter) timespan() int {
	v := int(time.Since(c.lastAppendTime) / c.bucketDuration)
	if v > -1 { // maybe time backwards
		return v
	}
	return c.size
}

func (c *SlidingWindowCounter) Append(val float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// move offset
	timespan := c.timespan()
	if timespan > 0 {
		start := (c.offset + 1) % c.size
		end := (c.offset + timespan) % c.size
		// reset the expired buckets
		c.ResetBuckets(start, timespan)
		c.offset = end
		c.lastAppendTime = c.lastAppendTime.Add(time.Duration(timespan * int(c.bucketDuration)))
	}

	c.buckets[c.offset] += val
	c.count += val
}

func (c *SlidingWindowCounter) Value() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.count
}

// ResetBucket empties the bucket based on the given offset.
func (c *SlidingWindowCounter) ResetBucket(offset int) {
	c.count -= c.buckets[offset%c.size]
	c.buckets[offset%c.size] = 0
}

// ResetBuckets empties the buckets based on the given offsets.
func (c *SlidingWindowCounter) ResetBuckets(offset int, count int) {
	if count > c.size {
		count = c.size
	}
	for i := 0; i < count; i++ {
		c.ResetBucket(offset + i)
	}
}
