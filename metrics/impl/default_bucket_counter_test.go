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
	"sort"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

func TestBucketCounterImpl_SlowUpdated(t *testing.T) {
	bucketCounter := newBucketCounterImpl(1*time.Second, 5, metrics.DefaultClock, true)
	for i := 0; i < 10; i++ {
		bucketCounter.Update()
	}

	time.Sleep(time.Second)

	for i := 0; i < 20; i++ {
		bucketCounter.Update()
	}

	time.Sleep(time.Second)

	// sleep for another 1 second
	time.Sleep(time.Second)

	result := bucketCounter.GetBucketCounts()
	assert.Equal(t, 2, len(result))
	values := make([]int, 0, 2)
	for _, value := range result {
		values = append(values, int(value))
	}
	sort.Ints(values)
	assert.Equal(t, 10, values[0])
	assert.Equal(t, 20, values[1])
}

func TestBucketCounterImpl_SingleRoutineUpdate(t *testing.T) {
	bci := newBucketCounterImpl(1*time.Second, 5, metrics.DefaultClock, true)
	for k := 0; k <= 7; k++ {
		for i := 0; i < k*10; i++ {
			bci.Update()
		}
		time.Sleep(time.Second)
	}
	result := bci.GetBucketCounts()
	assert.Equal(t, 5, len(result))
	values := make([]int, 0, 2)
	for _, value := range result {
		values = append(values, int(value))
	}
	sort.Ints(values)
	assert.Equal(t, 30, values[0])
	assert.Equal(t, 40, values[1])
	assert.Equal(t, 50, values[2])
	assert.Equal(t, 60, values[3])
	assert.Equal(t, 70, values[4])
}

func TestBucketCounterImpl_LatestIndexAtFirst(t *testing.T) {
	bci := newBucketCounterImpl(1*time.Second, 5, metrics.DefaultClock, true)
	for k := 0; k <= 6; k++ {
		for i := 0; i < k*10; i++ {
			bci.Update()
		}
		time.Sleep(time.Second)
	}
	result := bci.GetBucketCounts()
	assert.Equal(t, 5, len(result))
	values := make([]int, 0, 2)
	for _, value := range result {
		values = append(values, int(value))
	}
	sort.Ints(values)
	assert.Equal(t, 20, values[0])
	assert.Equal(t, 30, values[1])
	assert.Equal(t, 40, values[2])
	assert.Equal(t, 50, values[3])
	assert.Equal(t, 60, values[4])
}

func TestBucketCounterImpl_MultiGoroutineUpdate(t *testing.T) {
	wg := sync.WaitGroup{}
	bci := newBucketCounterImpl(1*time.Second, 15, metrics.DefaultClock, true)
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go func() {
			for k := 0; k <= 10; k++ {
				for i := 0; i < k*10; i++ {
					bci.Update()
				}
				time.Sleep(time.Second)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	result := bci.GetBucketCounts()
	assert.Equal(t, 10, len(result))
	values := make([]int, 0, 20)
	for _, value := range result {
		values = append(values, int(value))
	}
	sort.Ints(values)
	// expected = {8000, 7200, 6400, 5600, 4800, 4000, 3200, 2400, 1600, 800};
	assert.Equal(t, 800, values[0])
	assert.Equal(t, 1600, values[1])
	assert.Equal(t, 2400, values[2])
	assert.Equal(t, 3200, values[3])
	assert.Equal(t, 4000, values[4])
	assert.Equal(t, 4800, values[5])
	assert.Equal(t, 5600, values[6])
	assert.Equal(t, 6400, values[7])
	assert.Equal(t, 7200, values[8])
	assert.Equal(t, 8000, values[9])
	assert.Equal(t, int64(44000), bci.GetCount())
}

func TestBucketCounterImpl_10sInterval(t *testing.T) {
	mc := &ManualClock{}
	bci := newBucketCounterImpl(10*time.Second, 5, mc, true)
	for k := 0; k <= 7; k++ {
		for i := 0; i < k*10; i++ {
			bci.Update()
		}
		mc.Add(10 * time.Second)
	}

	result := bci.GetBucketCounts()
	assert.Equal(t, 5, len(result))
	values := make([]int, 0, 10)
	for _, value := range result {
		values = append(values, int(value))
	}
	sort.Ints(values)

	assert.Equal(t, 30, values[0])
	assert.Equal(t, 40, values[1])
	assert.Equal(t, 50, values[2])
	assert.Equal(t, 60, values[3])
	assert.Equal(t, 70, values[4])
	assert.Equal(t, int64(280), bci.GetCount())
}

func TestBucketCounterImpl_QueryWithStartTime(t *testing.T) {
	mc := &ManualClock{}
	bci := newBucketCounterImpl(10*time.Second, 5, mc, true)
	for k := 1; k <= 7; k++ {
		for i := 0; i < k*10; i++ {
			bci.Update()
		}
		mc.Add(10 * time.Second)
	}

	result := bci.GetBucketCountsSince(70000)
	assert.Equal(t, 0, len(result))

	result = bci.GetBucketCountsSince(60000)
	assert.Equal(t, 1, len(result))
	values := make([]int, 0, 10)
	for _, value := range result {
		values = append(values, int(value))
	}
	sort.Ints(values)

	assert.Equal(t, 70, values[0])
	assert.Equal(t, int64(280), bci.GetCount())

}

func TestBucketCounterImpl_QueryWith0(t *testing.T) {
	mc := &ManualClock{}
	bci := newBucketCounterImpl(10*time.Second, 10, mc, true)
	for k := 1; k <= 5; k++ {
		for i := 0; i < k*100; i++ {
			bci.Update()
		}
		mc.Add(10 * time.Second)
	}

	result := bci.GetBucketCountsSince(0)
	values := make([]int, 0, 10)
	keys := make([]int, 0, 10)
	for key, value := range result {
		values = append(values, int(value))
		keys = append(keys, int(key))
	}
	sort.Ints(values)
	sort.Ints(keys)
	assert.Equal(t, 5, len(values))
	assert.Equal(t, 5, len(keys))

	assert.Equal(t, 100, values[0])
	assert.Equal(t, 200, values[1])
	assert.Equal(t, 300, values[2])
	assert.Equal(t, 400, values[3])
	assert.Equal(t, 500, values[4])

	assert.Equal(t, 0, keys[0])
	assert.Equal(t, 10000, keys[1])
	assert.Equal(t, 20000, keys[2])
	assert.Equal(t, 30000, keys[3])
	assert.Equal(t, 40000, keys[4])

}

func TestBucketCounterImpl_AccurateQuery(t *testing.T) {
	mc := &ManualClock{}
	bci := newBucketCounterImpl(10*time.Second, 10, mc, true)
	mc.Add(10 * time.Second)
	bci.IncN(100)
	assert.Equal(t, int64(100), bci.GetBucketCounts()[10000])
}

func TestBucketCounterImpl_UpdateTotal(t *testing.T) {
	mc := &ManualClock{}
	bci := newBucketCounterImpl(10*time.Second, 10, mc, true)
	bci.Update()
	bci.Update()
	assert.Equal(t, int64(2), bci.GetCount())
}

func TestBucketCounterImpl_NotUpdateTotal(t *testing.T) {
	mc := &ManualClock{}
	bci := newBucketCounterImpl(10*time.Second, 10, mc, false)
	bci.Update()
	bci.Update()
	assert.Equal(t, int64(0), bci.GetCount())
}

func TestBucketCounterImpl_AlignTime(t *testing.T) {
	bci := BucketCounterImpl{
		interval: 1 * time.Second,
	}
	var origin int64 = 1575796315755
	assert.Equal(t, int64(1575796315), bci.alignTimestamp(origin))
}
