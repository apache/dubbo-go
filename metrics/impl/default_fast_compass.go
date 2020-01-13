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
)

import (
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/metrics"
)

const (
	/*
	 * This magic number divide a int64 into two parts,
	 * where the higher part is used to record the total number of method invocations,
	 * and the lower part is used to record the total method execution time.
	 * The max number of count per collecting interval will be 2 ^ (64 -38 -1) = 33554432
	 */
	countOffset uint64 = 38
	/*
	 * The base number of count that is added to total rt,
	 * to derive a number which will be added to
	 */
	countBase int64 = 1 << 38

	/*
	 * The base number is used to do BITWISE AND operation with the value of int64
	 * to derive the total number of execution time
	 */
	rtBitwiseAndBase = (1 << 38) - 1

	defaultBucketCount = 10
)

/**
 * The default implementation of GetFastCompass
 * All the data will be saved in memory.
 * It's the go-version of com.alibaba.metrics.FastCompassImpl In java
 *
 * Since the total number of method invocation and total time of method execution
 * within a collecting interval never exceed the range a int64 can represent,
 * we can use one int64 to record both the total count and total number
 */
type FastCompassImpl struct {
	maxSubCategoryCount int
	bucketInterval      time.Duration
	numberOfBuckets     int
	clock               metrics.Clock

	subCategoriesMutex sync.Mutex
	subCategories      map[string]metrics.BucketCounter
}

func (fc *FastCompassImpl) LastUpdateTime() int64 {
	result := int64(0)
	for _, value := range fc.subCategories {
		updatedTime := value.LastUpdateTime()
		if result < updatedTime {
			result = updatedTime
		}
	}
	return result
}

func (fc *FastCompassImpl) Record(duration time.Duration, subCategory string) {
	if duration < 0 || len(subCategory) <= 0 {
		return
	}

	fc.subCategoriesMutex.Lock()
	defer fc.subCategoriesMutex.Unlock()
	bucketCounter, found := fc.subCategories[subCategory]
	if !found {
		if len(fc.subCategories) >= fc.maxSubCategoryCount {
			// do nothing
			return
		}
		bucketCounter = newBucketCounterImpl(
			fc.bucketInterval,
			fc.numberOfBuckets,
			fc.clock,
			false)
		fc.subCategories[subCategory] = bucketCounter
	}

	bucketCounter.UpdateN(countBase + duration.Nanoseconds()/1e6)
}

func (fc *FastCompassImpl) GetMethodCountPerCategory() metrics.FastCompassResult {
	return fc.GetMethodCountPerCategorySince(0)
}

func (fc *FastCompassImpl) GetMethodCountPerCategorySince(startTime int64) metrics.FastCompassResult {
	result := metrics.NewFastCompassResult(len(fc.subCategories))
	for key, value := range fc.subCategories {
		bkCnt := value.GetBucketCountsSince(startTime)
		methodBucketCount := make(map[int64]int64, len(bkCnt))
		for timestamp, count := range bkCnt {
			// It's impossible for count to be negative
			methodBucketCount[timestamp] = int64(uint64(count) >> countOffset)
		}
		result[key] = methodBucketCount
	}
	return result
}

func (fc *FastCompassImpl) GetMethodRtPerCategory() metrics.FastCompassResult {
	return fc.GetMethodRtPerCategorySince(0)
}

func (fc *FastCompassImpl) GetMethodRtPerCategorySince(startTime int64) metrics.FastCompassResult {
	result := metrics.NewFastCompassResult(len(fc.subCategories))
	for key, value := range fc.subCategories {
		bkCnt := value.GetBucketCountsSince(startTime)
		methodBucketCount := make(map[int64]int64, len(bkCnt))
		for timestamp, count := range bkCnt {
			// It's impossible for count to be negative
			methodBucketCount[timestamp] = int64(uint64(count) & rtBitwiseAndBase)
		}
		result[key] = methodBucketCount
	}
	return result
}

func (fc *FastCompassImpl) GetCountAndRtPerCategory() metrics.FastCompassResult {
	return fc.GetCountAndRtPerCategorySince(0)
}

func (fc *FastCompassImpl) GetCountAndRtPerCategorySince(startTime int64) metrics.FastCompassResult {
	result := metrics.NewFastCompassResult(len(fc.subCategories))
	for key, value := range fc.subCategories {
		bkCnt := value.GetBucketCountsSince(startTime)
		methodBucketCount := make(map[int64]int64, len(bkCnt))
		for timestamp, count := range bkCnt {
			// It's impossible for count to be negative
			methodBucketCount[timestamp] = count
		}
		result[key] = methodBucketCount
	}
	return result
}

func (fc *FastCompassImpl) GetBucketInterval() time.Duration {
	return fc.bucketInterval
}

func newFastCompass(duration time.Duration, bucketCount int) metrics.FastCompass {
	return &FastCompassImpl{
		maxSubCategoryCount: config.GetMetricConfig().GetMaxSubCategoryCount(),
		bucketInterval:      duration,
		numberOfBuckets:     bucketCount,
		clock:               metrics.DefaultClock,
		subCategoriesMutex:  sync.Mutex{},
		subCategories:       make(map[string]metrics.BucketCounter),
	}
}
