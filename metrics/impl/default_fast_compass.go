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
	countOffset int64 = 38
	/*
	 * The base number of count that is added to total rt,
	 * to derive a number which will be added to
	 */
	countBase   int64 = 1 << 38

	/*
	 * The base number is used to do BITWISE AND operation with the value of int64
	 * to derive the total number of execution time
	 */
	rtBitwiseAndBase = (1 << 38) -1

	defaultBucketCount = 10

)

/**
 * The default implementation of FastCompass
 * All the data will be saved in memory.
 * It's the go-version of com.alibaba.metrics.FastCompassImpl In java
 *
 * Since the total number of method invocation and total time of method execution
 * within a collecting interval never exceed the range a int64 can represent,
 * we can use one int64 to record both the total count and total number
 */
type FastCompassImpl struct {
	maxSubCategoryCount int32
	bucketInterval int32
	numberOfBuckets int32
	clock metrics.Clock
	maxCategoryCount int32
	subCategories sync.Map
}

func (f FastCompassImpl) LastUpdateTime() int64 {
	panic("implement me")
}

func (f FastCompassImpl) Record(duration time.Duration, subCategory string) {
	if duration < 0 || len(subCategory) <= 0{
		return
	}
}

func (f FastCompassImpl) GetMethodCountPerCategory() metrics.FastCompassResult {
	panic("implement me")
}

func (f FastCompassImpl) GetMethodCountPerCategorySince(startTime int64) metrics.FastCompassResult {
	panic("implement me")
}

func (f FastCompassImpl) GetMethodRtPerCategory() metrics.FastCompassResult {
	panic("implement me")
}

func (f FastCompassImpl) GetMethodRtPerCategorySince(startTime int64) metrics.FastCompassResult {
	panic("implement me")
}

func (f FastCompassImpl) GetCountAndRtPerCategory() metrics.FastCompassResult {
	panic("implement me")
}

func (f FastCompassImpl) GetCountAndRtPerCategorySince(startTime int64) metrics.FastCompassResult {
	panic("implement me")
}

func (f FastCompassImpl) GetBucketInterval() int32 {
	panic("implement me")
}

var (
	fastCompassInstance *FastCompassImpl
	fastCompassInitOnce sync.Once
)

func GetFastCompassImpl() metrics.FastCompass {
	fastCompassInitOnce.Do(func() {
		fastCompassInstance = &FastCompassImpl{
			maxSubCategoryCount: config.GetMetricConfig().GetMaxSubCategoryCount(),
		}
	})
	return fastCompassInstance
}
