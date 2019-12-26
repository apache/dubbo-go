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

	assert.Equal(t, int64(15), snapshot.GetMean())
	assert.Equal(t, float64(metrics.NotAvailable), snapshot.Get75thPercentile())
	assert.Equal(t, float64(metrics.NotAvailable), snapshot.Get95thPercentile())
	assert.Equal(t, float64(metrics.NotAvailable), snapshot.Get98thPercentile())
	assert.Equal(t, float64(metrics.NotAvailable), snapshot.Get99thPercentile())
	assert.Equal(t, float64(metrics.NotAvailable), snapshot.Get999thPercentile())
	assert.Equal(t, metrics.NotAvailable, snapshot.GetMax())
	assert.Equal(t, metrics.NotAvailable, snapshot.GetMin())
}
