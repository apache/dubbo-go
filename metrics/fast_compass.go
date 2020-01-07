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

package metrics

import (
	"time"
)

type FastCompassResult map[string]map[int64]int64

func NewFastCompassResult(expected int) FastCompassResult {
	return make(map[string]map[int64]int64, expected)
}

/**
 * This interface will be used to record the qps, rt and error code.
 * Specially, it's designed for high concurrency environment
 */
type FastCompass interface {
	Metric

	// record a method invocation with execution time and sub-categories
	Record(duration time.Duration, subCategory string)

	// return method count per bucket per category
	GetMethodCountPerCategory() FastCompassResult
	/* return method count per bucket per category since starTime
	 * sometimes the startTime is not the real time but the logic time
	 * it comes from Clock interface's
	 */
	GetMethodCountPerCategorySince(startTime int64) FastCompassResult

	// return method execution time per bucket per category
	GetMethodRtPerCategory() FastCompassResult
	/*
	 * return method execution time per bucket per category since starTime
	 * sometimes the startTime is not the real time but the logic time
	 * it comes from Clock interface's
	 */
	GetMethodRtPerCategorySince(startTime int64) FastCompassResult

	// return method execution time and count per bucket per category
	GetCountAndRtPerCategory() FastCompassResult
	/*
	 * return method execution time and count per bucket per category since startTime
	 * sometimes the startTime is not the real time but the logic time
	 * it comes from Clock interface's
	 */
	GetCountAndRtPerCategorySince(startTime int64) FastCompassResult

	GetBucketInterval() time.Duration
}
