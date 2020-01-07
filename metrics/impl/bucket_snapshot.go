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
	"errors"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

type BucketSnapshot struct {
	count int64
	value int64
}

func (b BucketSnapshot) GetValue(quantile float64) (float64, error) {
	return float64(metrics.NotAvailable), errors.New("BucketSnapshot do not support GetValue operation")
}

func (b BucketSnapshot) GetValues() ([]int64, error) {
	return []int64{0}, errors.New("BucketSnapshot do not support GetValues operation")
}

func (b BucketSnapshot) Size() (int, error) {
	return int(b.count), nil
}

func (b BucketSnapshot) GetMedian() (float64, error) {
	return float64(metrics.NotAvailable), errors.New("BucketSnapshot do not support GetMedian operation")
}

func (b BucketSnapshot) Get75thPercentile() (float64, error) {
	return float64(metrics.NotAvailable), errors.New("BucketSnapshot do not support Get75thPercentile operation")
}

func (b BucketSnapshot) Get95thPercentile() (float64, error) {
	return float64(metrics.NotAvailable), errors.New("BucketSnapshot do not support Get95thPercentile operation")
}

func (b BucketSnapshot) Get98thPercentile() (float64, error) {
	return float64(metrics.NotAvailable), errors.New("BucketSnapshot do not support Get98thPercentile operation")
}

func (b BucketSnapshot) Get99thPercentile() (float64, error) {
	return float64(metrics.NotAvailable), errors.New("BucketSnapshot do not support Get98thPercentile operation")
}

func (b BucketSnapshot) Get999thPercentile() (float64, error) {
	return float64(metrics.NotAvailable), errors.New("BucketSnapshot do not support Get999thPercentile operation")
}

func (b BucketSnapshot) GetMax() (int64, error) {
	return metrics.NotAvailable, errors.New("BucketSnapshot do not support GetMax operation")
}

func (b BucketSnapshot) GetMean() (float64, error) {
	if b.count == 0 {
		return 0, nil
	}
	return float64(b.value) / float64(b.count), nil
}

func (b BucketSnapshot) GetMin() (int64, error) {
	return metrics.NotAvailable, errors.New("BucketSnapshot do not support GetMin operation")
}

func (b BucketSnapshot) GetStdDev() (float64, error) {
	return float64(metrics.NotAvailable), errors.New("BucketSnapshot do not support GetStdDev operation")
}

func NewBucketSnapshot(count int64, value int64) metrics.Snapshot {
	return BucketSnapshot{
		count: count,
		value: value,
	}
}
