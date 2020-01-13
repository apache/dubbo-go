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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

func TestExponentiallyDecayingReservoir_GetSnapshotFor1000(t *testing.T) {
	reservoir := NewExponentiallyDecayingReservoir(100, 0.99, metrics.DefaultClock)
	for i := 0; i < 1000; i++ {
		reservoir.UpdateN(int64(i))
	}

	assert.Equal(t, 100, reservoir.Size())

	snapshot := reservoir.GetSnapshot()
	size, err := snapshot.Size()
	assert.Equal(t, 100, size)
	assert.Nil(t, err)

	values, err := snapshot.GetValues()
	assert.True(t, int64SliceBetween(values, 0, 1000))

}

func TestExponentiallyDecayingReservoir_GetSnapshotFor10(t *testing.T) {
	reservoir := NewExponentiallyDecayingReservoir(100, 0.99, metrics.DefaultClock)
	for i := 0; i < 10; i++ {
		reservoir.UpdateN(int64(i))
	}

	assert.Equal(t, 10, reservoir.Size())

	snapshot := reservoir.GetSnapshot()
	size, err := snapshot.Size()
	assert.Equal(t, 10, size)
	assert.Nil(t, err)

	values, err := snapshot.GetValues()
	assert.True(t, int64SliceBetween(values, 0, 10))

}

func TestExponentiallyDecayingReservoir_HeavilyBiased_GetSnapshotFor1000(t *testing.T) {
	reservoir := NewExponentiallyDecayingReservoir(1000, 0.01, metrics.DefaultClock)
	for i := 0; i < 100; i++ {
		reservoir.UpdateN(int64(i))
	}

	assert.Equal(t, 100, reservoir.Size())

	snapshot := reservoir.GetSnapshot()
	size, err := snapshot.Size()
	assert.Equal(t, 100, size)
	assert.Nil(t, err)

	values, err := snapshot.GetValues()
	assert.True(t, int64SliceBetween(values, 0, 100))

}

func TestExponentiallyDecayingReservoir_longPeriod(t *testing.T) {
	clock := &ManualClock{}
	reservoir := NewExponentiallyDecayingReservoir(10, 0.015, clock)
	for i := 0; i < 1000; i++ {
		reservoir.UpdateN(int64(1000 + i))
		clock.Add(100 * time.Millisecond)
	}

	assert.Equal(t, 10, reservoir.Size())

	snapshot := reservoir.GetSnapshot()
	size, err := snapshot.Size()
	assert.Equal(t, 10, size)
	assert.Nil(t, err)

	values, err := snapshot.GetValues()
	assert.True(t, int64SliceBetween(values, 1000, 2000))

	// wait for 15 hours and add another value.
	// this should trigger a rescale. Note that the number of samples will be reduced to 2
	// because of the very small scaling factor that will make all existing priorities equal to
	// zero after rescale.
	clock.Add(15 * time.Hour)
	reservoir.UpdateN(2000)

	snapshot = reservoir.GetSnapshot()
	size, err = snapshot.Size()
	assert.Equal(t, 1, size)
	assert.Nil(t, err)
	values, err = snapshot.GetValues()
	assert.True(t, int64SliceBetween(values, 1000, 3000))

	for i := 0; i < 1000; i++ {
		reservoir.UpdateN(int64(3000 + i))
		clock.Add(100 * time.Millisecond)
	}

	snapshot = reservoir.GetSnapshot()
	size, err = snapshot.Size()
	assert.Equal(t, 10, size)
	assert.Nil(t, err)
	values, err = snapshot.GetValues()
	assert.True(t, int64SliceBetween(values, 3000, 4000))

}

func TestExponentiallyDecayingReservoir_longPeriod_fetchShouldResample(t *testing.T) {
	clock := &ManualClock{}
	reservoir := NewExponentiallyDecayingReservoir(10, 0.015, clock)

	for i := 0; i < 1000; i++ {
		reservoir.UpdateN(int64(1000 + i))
		clock.Add(100 * time.Millisecond)
	}

	// wait for 20 hours and add another value.
	// this should trigger a rescale. Note that the number of samples will be reduced to 2
	// because of the very small scaling factor that will make all existing priorities equal to
	// zero after rescale.
	clock.Add(20 * time.Hour)

	snapshot := reservoir.GetSnapshot()

	max, err := snapshot.GetMax()
	assert.Equal(t, int64(0), max)
	assert.NotNil(t, err)
	mean, err := snapshot.GetMean()
	assert.True(t, equals(float64(0), mean, 0.0000001))

	median, err := snapshot.GetMedian()
	assert.True(t, equals(float64(0), median, 0.0000001))

	size, err := snapshot.Size()
	assert.Equal(t, 0, size)
}

func TestExponentiallyDecayingReservoir_spotLift(t *testing.T) {
	clock := &ManualClock{}
	reservoir := NewExponentiallyDecayingReservoir(1000, 0.015, clock)
	valuesRatePerMinute := 10
	// 1m = 60 * 1000ms
	valuesIntervalMillis := 60 * 1000 / valuesRatePerMinute

	// mode 1: steady regime for 120 minutes
	for i := 0; i < 120*valuesRatePerMinute; i++ {
		reservoir.UpdateN(177)
		clock.Add(time.Duration(valuesIntervalMillis) * time.Millisecond)
	}

	// switching to mode 2: 10 minutes more with the same rate, but larger value
	for i := 0; i < 10*valuesRatePerMinute; i++ {
		reservoir.UpdateN(9999)
		clock.Add(time.Duration(valuesIntervalMillis) * time.Millisecond)
	}

	// expect that quantiles should be more about mode 2 after 10 minutes
	snapshot := reservoir.GetSnapshot()
	median, err := snapshot.GetMedian()
	assert.Equal(t, float64(9999), median)
	assert.Nil(t, err)
}

func TestExponentiallyDecayingReservoir_spotFall(t *testing.T) {
	clock := &ManualClock{}
	reservoir := NewExponentiallyDecayingReservoir(1000, 0.015, clock)
	valuesRatePerMinute := 10
	// 1m = 60 * 1000ms
	valuesIntervalMillis := 60 * 1000 / valuesRatePerMinute

	// mode 1: steady regime for 120 minutes
	for i := 0; i < 120*valuesRatePerMinute; i++ {
		reservoir.UpdateN(9998)
		clock.Add(time.Duration(valuesIntervalMillis) * time.Millisecond)
	}

	// switching to mode 2: 10 minutes more with the same rate, but larger value
	for i := 0; i < 10*valuesRatePerMinute; i++ {
		reservoir.UpdateN(178)
		clock.Add(time.Duration(valuesIntervalMillis) * time.Millisecond)
	}

	// expect that quantiles should be more about mode 2 after 10 minutes
	snapshot := reservoir.GetSnapshot()
	median, err := snapshot.GetMedian()
	assert.Equal(t, float64(178), median)
	assert.Nil(t, err)
}

func TestExponentiallyDecayingReservoir_quantiliesShouldBeBasedOnWeights(t *testing.T) {
	clock := &ManualClock{}
	reservoir := NewExponentiallyDecayingReservoir(1000, 0.015, clock)
	for i := 0; i < 40; i++ {
		reservoir.UpdateN(177)
	}

	clock.Add(120 * time.Second)

	for i := 0; i < 10; i++ {
		reservoir.UpdateN(9999)
	}

	// the first added 40 items (177) have weights 1
	// the next added 10 items (9999) have weights ~6
	// so, it's 40 vs 60 distribution, not 40 vs 10

	snapshot := reservoir.GetSnapshot()
	median, _ := snapshot.GetMedian()
	assert.True(t, equals(float64(9999), median, 0.000000001))

	quantile75, _ := snapshot.Get75thPercentile()
	assert.True(t, equals(float64(9999), quantile75, 0.000000001))

}

func int64SliceBetween(values []int64, min int64, max int64) bool {
	for _, value := range values {
		if value < min || value > max {
			return false
		}
	}

	return true
}
