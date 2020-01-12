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
	"math"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

func TestWeightedSample_Quantile(t *testing.T) {
	snapshot := newWeightSnapshotWithValues([]int64{5, 1, 2, 3, 4}, []float64{1, 2, 3, 2, 2})

	values, _ := snapshot.GetValues()
	assert.Equal(t, 5, len(values))
	assert.Equal(t, int64(1), values[0])
	assert.Equal(t, int64(2), values[1])
	assert.Equal(t, int64(3), values[2])
	assert.Equal(t, int64(4), values[3])
	assert.Equal(t, int64(5), values[4])

	// small quantile
	value, _ := snapshot.GetValue(0.0)
	assert.True(t, equals(1.0, value, 0.1))

	// big quantile
	value, _ = snapshot.GetValue(1.0)
	assert.True(t, equals(5.0, value, 0.1))

	// invalid quantile
	value, err := snapshot.GetValue(math.NaN())
	assert.NotNil(t, err)

	value, err = snapshot.GetValue(1.000000001)
	assert.NotNil(t, err)

	value, err = snapshot.GetValue(-0.0000001)
	assert.NotNil(t, err)

	// median
	value, err = snapshot.GetMedian()
	assert.Nil(t, err)
	assert.True(t, equals(3.0, value, 0.1))

	// 75
	value, err = snapshot.Get75thPercentile()
	assert.Nil(t, err)
	assert.True(t, equals(4.0, value, 0.1))

	// 95
	value, err = snapshot.Get95thPercentile()
	assert.Nil(t, err)
	assert.True(t, equals(5.0, value, 0.1))

	// 98
	value, err = snapshot.Get98thPercentile()
	assert.Nil(t, err)
	assert.True(t, equals(5.0, value, 0.1))

	// 99
	value, err = snapshot.Get99thPercentile()
	assert.Nil(t, err)
	assert.True(t, equals(5.0, value, 0.1))

	// equals()

	// 999
	value, err = snapshot.Get999thPercentile()
	assert.Nil(t, err)
	assert.True(t, equals(5.0, value, 0.1))

	// mean
	value, err = snapshot.GetMean()
	assert.Nil(t, err)
	assert.True(t, equals(2.7, value, 0.1))

	// min
	value0, err := snapshot.GetMin()
	assert.Nil(t, err)
	assert.Equal(t, int64(1), value0)

	// max
	value0, err = snapshot.GetMax()
	assert.Nil(t, err)
	assert.Equal(t, int64(5), value0)
}

func TestWeightedSample_Emty(t *testing.T) {
	snapshot := NewWeightedSnapshot(nil)
	min, err := snapshot.GetMin()
	assert.Equal(t, int64(0), min)
	assert.NotNil(t, err)

	max, err := snapshot.GetMax()
	assert.Equal(t, int64(0), max)
	assert.NotNil(t, err)

	mean, err := snapshot.GetMean()
	assert.Equal(t, float64(0), mean)
	assert.NotNil(t, err)

	stdDev, err := snapshot.GetStdDev()
	assert.Equal(t, float64(0), stdDev)
	assert.Nil(t, err)

	snapshot = newWeightSnapshotWithValues([]int64{1}, []float64{1.0})
	stdDev, err = snapshot.GetStdDev()
	assert.Equal(t, float64(0), stdDev)
	assert.Nil(t, err)
}

func newWeightSnapshotWithValues(values []int64, weight []float64) metrics.Snapshot {
	samples := make([]*WeightedSample, 0, len(values))
	for index, _ := range values {
		sample := NewWeightSample(0, values[index], weight[index])
		samples = append(samples, sample)
	}
	return NewWeightedSnapshot(samples)
}
