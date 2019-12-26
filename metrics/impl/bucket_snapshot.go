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
	"io"
)

import (
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/metrics"
)

type BucketSnapshot struct {
	count int64
	value int64
}

func (b BucketSnapshot) GetValue(quantile float64) float64 {
	return float64(metrics.NotAvailable)
}

func (b BucketSnapshot) GetValues() []int64 {
	return []int64{0}
}

func (b BucketSnapshot) Size() int {
	return int(b.count)
}

func (b BucketSnapshot) GetMedian() float64 {
	return float64(metrics.NotAvailable)
}

func (b BucketSnapshot) Get75thPercentile() float64 {
	return float64(metrics.NotAvailable)
}

func (b BucketSnapshot) Get95thPercentile() float64 {
	return float64(metrics.NotAvailable)
}

func (b BucketSnapshot) Get98thPercentile() float64 {
	return float64(metrics.NotAvailable)
}

func (b BucketSnapshot) Get99thPercentile() float64 {
	return float64(metrics.NotAvailable)
}

func (b BucketSnapshot) Get999thPercentile() float64 {
	return float64(metrics.NotAvailable)
}

func (b BucketSnapshot) GetMax() int64 {
	return metrics.NotAvailable
}

func (b BucketSnapshot) GetMean() int64 {
	if b.count == 0 {
		return 0
	}
	return b.value / b.count
}

func (b BucketSnapshot) GetMin() int64 {
	return metrics.NotAvailable
}

func (b BucketSnapshot) GetStdDev() float64 {
	return float64(metrics.NotAvailable)
}

func (b BucketSnapshot) Dump(writer io.Writer) {
	logger.Warn("This implementation will do nothing. ")
}

func NewBucketSnapshot(count int64, value int64) metrics.Snapshot {
	return BucketSnapshot{
		count: count,
		value: value,
	}
}
