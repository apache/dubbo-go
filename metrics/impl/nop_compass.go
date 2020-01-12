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
)
import (
	"github.com/apache/dubbo-go/metrics"
)

type NopCompass struct {
}

func (n NopCompass) LastUpdateTime() int64 {
	return 0
}

func (n NopCompass) GetCount() int64 {
	return 0
}

func (n NopCompass) GetInstantCount() map[int64]int64 {
	return make(map[int64]int64, 0)
}

func (n NopCompass) GetInstantCountSince(startTime int64) map[int64]int64 {
	return make(map[int64]int64, 0)
}

func (n NopCompass) GetInstantCountInterval() time.Duration {
	return 0
}

func (n NopCompass) GetFifteenMinuteRate() float64 {
	return 0
}

func (n NopCompass) GetFiveMinuteRate() float64 {
	return 0
}

func (n NopCompass) GetMeanRate() float64 {
	return 0
}

func (n NopCompass) GetOneMinuteRate() float64 {
	return 0
}

func (n NopCompass) GetSnapshot() metrics.Snapshot {
	return nopSnapshot
}

func (n NopCompass) Update(duration time.Duration) {
}

func (n NopCompass) UpdateWithError(duration time.Duration, isSuccess bool, errorCode string, addon string) {
}

func (n NopCompass) GetSuccessCount() int64 {
	return 0
}

func (n NopCompass) GetErrorCodeCounts() map[string]metrics.BucketCounter {
	return make(map[string]metrics.BucketCounter, 0)
}

func (n NopCompass) GetAddonCounts() map[string]metrics.BucketCounter {
	return make(map[string]metrics.BucketCounter, 0)
}

func (n NopCompass) GetBucketSuccessCount() metrics.BucketCounter {
	return nopBucketCounter
}

var nopCompass = &NopCompass{}
