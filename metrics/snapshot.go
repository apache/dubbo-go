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

/**
 * It's the go version of com.alibaba.metrics.Snapshot
 */
type Snapshot interface {
	// return the value at the given quantile.
	GetValue(quantile float64) (float64, error)

	// return the entire set of values in the snapshot
	GetValues() ([]int64, error)

	// return the number of values in the snapshot
	Size() (int, error)

	// Get the median value in the distribution
	GetMedian() (float64, error)

	// Get the value at 75th percentile in the distribution
	Get75thPercentile() (float64, error)

	// Get the value at 95th percentile in the distribution
	Get95thPercentile() (float64, error)

	// Get the value at 98th percentile in the distribution
	Get98thPercentile() (float64, error)

	// Get the value at 99th percentile in the distribution
	Get99thPercentile() (float64, error)

	// Get the value at 999th percentile in the distribution
	Get999thPercentile() (float64, error)

	// Returns the highest value in the snapshot.
	GetMax() (int64, error)

	// Returns the arithmetic mean of the values in the snapshot.
	GetMean() (float64, error)

	// Returns the lowest value in the snapshot.
	GetMin() (int64, error)

	// Returns the standard deviation of the values in the snapshot.
	GetStdDev() (float64, error)
}
