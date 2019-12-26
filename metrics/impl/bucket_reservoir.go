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
	"time"

	"github.com/apache/dubbo-go/metrics"
)

type BucketReservoir struct {
	countPerBucket metrics.BucketCounter
	valuePerBucket metrics.BucketCounter
	clock          metrics.Clock
	interval       time.Duration
}

func (b *BucketReservoir) Size() int {
	return int(b.countPerBucket.GetCount())
}

func (b *BucketReservoir) UpdateN(value int64) {
	b.valuePerBucket.UpdateN(value)
}

func (b *BucketReservoir) GetSnapshot() metrics.Snapshot {
	panic("implement me")
}

func NewBucketReservoir(interval time.Duration, numOfBucket int, clock metrics.Clock, totalCount metrics.BucketCounter) metrics.Reservoir {
	return &BucketReservoir{
		countPerBucket: totalCount,
		valuePerBucket: newBucketCounterImpl(interval, numOfBucket, clock, true),
		clock:          clock,
		interval:       interval,
	}
}
