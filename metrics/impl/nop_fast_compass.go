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

type NopFastCompass struct {
}

func (n NopFastCompass) LastUpdateTime() int64 {
	return 0
}

func (n NopFastCompass) Record(duration time.Duration, subCategory string) {
}

func (n NopFastCompass) GetMethodCountPerCategory() metrics.FastCompassResult {
	return make(metrics.FastCompassResult, 0)
}

func (n NopFastCompass) GetMethodCountPerCategorySince(startTime int64) metrics.FastCompassResult {
	return make(metrics.FastCompassResult, 0)
}

func (n NopFastCompass) GetMethodRtPerCategory() metrics.FastCompassResult {
	return make(metrics.FastCompassResult, 0)
}

func (n NopFastCompass) GetMethodRtPerCategorySince(startTime int64) metrics.FastCompassResult {
	return make(metrics.FastCompassResult, 0)
}

func (n NopFastCompass) GetCountAndRtPerCategory() metrics.FastCompassResult {
	return make(metrics.FastCompassResult, 0)
}

func (n NopFastCompass) GetCountAndRtPerCategorySince(startTime int64) metrics.FastCompassResult {
	return make(metrics.FastCompassResult, 0)
}

func (n NopFastCompass) GetBucketInterval() time.Duration {
	return time.Second
}

var nopFastCompass = &NopFastCompass{}
