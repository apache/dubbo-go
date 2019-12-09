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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

func TestFastCompassImpl_Record(t *testing.T) {
	mc := &ManualClock{}
	fastCompass := newFastCompassForTest(60*time.Second, 10, mc, 10)
	fastCompass.Record(10*time.Millisecond, "success")
	fastCompass.Record(20*time.Millisecond, "error")
	fastCompass.Record(15*time.Millisecond, "success")

	mc.Add(60 * time.Second)

	methodCount := fastCompass.GetMethodCountPerCategory()
	_, load := methodCount["success"]
	assert.True(t, load)
	_, load = methodCount["error"]
	assert.True(t, load)

	methodCount = fastCompass.GetMethodCountPerCategorySince(0)
	assert.Equal(t, int64(2), methodCount["success"][0])
	assert.Equal(t, int64(1), methodCount["error"][0])

	methodRt := fastCompass.GetMethodRtPerCategory()

	_, load = methodRt["success"]
	assert.True(t, load)
	_, load = methodRt["error"]
	assert.True(t, load)

	methodRt = fastCompass.GetMethodRtPerCategorySince(0)
	assert.Equal(t, int64(25), methodRt["success"][0])
	assert.Equal(t, int64(20), methodRt["error"][0])

	countAndRt := fastCompass.GetCountAndRtPerCategory()
	_, load = countAndRt["success"]
	assert.True(t, load)
	_, load = countAndRt["error"]
	assert.True(t, load)

	countAndRt = fastCompass.GetCountAndRtPerCategorySince(0)
	assert.Equal(t, int64((2<<38)+25), countAndRt["success"][0])
	assert.Equal(t, int64((1<<38)+20), countAndRt["error"][0])
}

func TestFastCompassImpl_MaxCategoryCount(t *testing.T) {
	mc := &ManualClock{}
	fastCompass := newFastCompassForTest(60*time.Second, 10, mc, 2)
	fastCompass.Record(10*time.Millisecond, "success")
	fastCompass.Record(20*time.Millisecond, "error1")
	fastCompass.Record(15*time.Millisecond, "error2")
	assert.Equal(t, 2, len(fastCompass.GetMethodCountPerCategory()))
}

func newFastCompassForTest(bucketInterval time.Duration, numOfBuckets int,
	clock metrics.Clock, maxSubCategoryCount int) metrics.FastCompass {
	return &FastCompassImpl{
		maxSubCategoryCount: maxSubCategoryCount,
		bucketInterval:      bucketInterval,
		numberOfBuckets:     numOfBuckets,
		clock:               clock,
		subCategoriesMutex:  sync.Mutex{},
		subCategories:       make(map[string]metrics.BucketCounter),
	}
}
