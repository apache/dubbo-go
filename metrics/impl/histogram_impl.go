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
	"github.com/apache/dubbo-go/metrics"
	"time"
)

type HistorgramImpl struct {
	reservoir metrics.Reservoir
    count     metrics.BucketCounter
}

func (hg *HistorgramImpl) Update(value int) {
	hg.UpdateN(int64(value))
}

func (hg *HistorgramImpl) UpdateN(value int64) {
	hg.count.Update()
	hg.reservoir.UpdateN(value)
}

func (hg *HistorgramImpl) LastUpdateTime() int64 {
	result := int64(0)
	updatedTime := hg.count.LastUpdateTime()
	if result < updatedTime {
		result = updatedTime
	}
	return result
}

func (hg *HistorgramImpl) GetSnapshot() metrics.Snapshot {
	return hg.reservoir.GetSnapshot()
}

func (hg *HistorgramImpl) GetCount() int64 {
	return hg.count.GetCount()
}


func newHistogram(reservoirType metrics.ReservoirType, duration time.Duration, numberOfBucket int, clock metrics.Clock) metrics.Histogram {
	bucketCount := newBucketCounterImpl(duration, numberOfBucket, clock, true)
	switch (reservoirType) {
	case metrics.UniformReservoirType:
		return &HistorgramImpl{
			reservoir: NewUniformReservoir(1024),
			count:     bucketCount,
		}

	case metrics.BucketReservoirType:
		return &HistorgramImpl{
			reservoir: NewBucketReservoir(duration, numberOfBucket, clock, bucketCount),
			count:     bucketCount,
		}
	default:
		return &HistorgramImpl{
			reservoir: NewUniformReservoir(1024),
			count:     bucketCount,
		}
	}

}