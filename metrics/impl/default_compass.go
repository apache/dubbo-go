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
	"sync"
	"time"

	"github.com/apache/dubbo-go/metrics"
)

const tickInterval = 5 * time.Second

type DefaultCompass struct {
	// The count per collect interval
	totalCount *metrics.BucketCounter
	// The number of successful count per collect interval
	successCount *metrics.BucketCounter
	// The number of error count per code per collect interval
	errorCodes sync.Map
	// The number of addon count per addon per collect interval
	addons sync.Map

	// 1min moving average
	m1Rate metrics.EWMA
	// 5min moving average
	m5Rate metrics.EWMA
	// 15min moving average
	m15Rate metrics.EWMA

	// usually, it's the real time.
	// But it could be "logic" time, which depends on the Clock's implementation
	startTime int64

	// the last tick timestamp
	lastTick int64

	// The number of events that is not update to moving average yet
	uncounted int64

	// The clock implementation
	clock metrics.Clock

	// The max number of error code that is allowed
	maxErrorCodeCount int

	// The max number of addon that is allowed
	maxAddonCount int

	// The collect interval
	bucketInterval int

	// The number of bucket
	numberOfBucket int

	reservoir metrics.Reservoir
}

func (cp *DefaultCompass) GetSnapshot() metrics.Snapshot {
	panic("Implement me")
}
