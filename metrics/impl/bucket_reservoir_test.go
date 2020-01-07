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

func TestBucketReservoir(t *testing.T) {
	clock := &ManualClock{}
	totalCount := newBucketCounterImpl(5*time.Second, 10, clock, true)
	reservoir := NewBucketReservoir(5*time.Second, 10, clock, totalCount)

	// 1 interval
	clock.Add(5 * time.Second)

	reservoir.UpdateN(10)
	reservoir.UpdateN(20)

	totalCount.UpdateN(2)

	// step into another interval
	clock.Add(2 * time.Second)

	snapshot := reservoir.GetSnapshot()

	mean, err := snapshot.GetMean()
	assert.Equal(t, float64(15), mean)
	assert.Nil(t, err)

	per75, err := snapshot.Get75thPercentile()
	assert.Equal(t, float64(metrics.NotAvailable), per75)
	assert.NotNil(t, err)

	per95, err := snapshot.Get95thPercentile()
	assert.Equal(t, float64(metrics.NotAvailable), per95)
	assert.NotNil(t, err)

	per98, err := snapshot.Get98thPercentile()
	assert.Equal(t, float64(metrics.NotAvailable), per98)
	assert.NotNil(t, err)

	per99, err := snapshot.Get99thPercentile()
	assert.Equal(t, float64(metrics.NotAvailable), per99)
	assert.NotNil(t, err)

	per999, err := snapshot.Get999thPercentile()
	assert.Equal(t, float64(metrics.NotAvailable), per999)
	assert.NotNil(t, err)

	max, err := snapshot.GetMax()
	assert.Equal(t, metrics.NotAvailable, max)
	assert.NotNil(t, err)

	min, err := snapshot.GetMin()
	assert.Equal(t, metrics.NotAvailable, min)
	assert.NotNil(t, err)
}
