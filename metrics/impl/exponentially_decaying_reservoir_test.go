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

	"github.com/stretchr/testify/assert"

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


func int64SliceBetween(values []int64, min int64, max int64) bool {
	for _, value := range values {
		if value < min || value > max {
			return false
		}
	}

	return true
}
