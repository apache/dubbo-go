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

/**
 * It's the go version of com.alibaba.metrics.Metered
 */
type Metered interface {
	Metric
	Counting
	// Get the accurate number per collecting interval
	// keyed by timestamp in milliseconds
	GetInstantCount() map[int64]int64
	// Get the accurate number per collecting interval since (including) the start time
	// keyed by timestamp in milliseconds
	GetInstantCountSince(startTime int64) map[int64]int64

	// Get the collecting interval
	GetInstantCountInterval() time.Duration

	// Returns the fifteen-minute exponentially-weighted moving average rate at which events have
	// occurred since the meter was created.
	// This rate has the same exponential decay factor as the fifteen-minute load average in the `top` Unix command
	GetFifteenMinuteRate() float64

	// Returns the fifteen-minute exponentially-weighted moving average rate at which events have
	// occurred since the meter was created.
	// This rate has the same exponential decay factor as the five-minute load average in the `top` Unix command
	GetFiveMinuteRate() float64

	// Returns the mean rate at which events have occurred since the meter was created.
	GetMeanRate() float64

	// Returns the fifteen-minute exponentially-weighted moving average rate at which events have
	// occurred since the meter was created.
	// This rate has the same exponential decay factor as the one-minute load average in the `top` Unix command
	GetOneMinuteRate() float64
}
