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

type NopBucketCounter struct {
}

func (nbc *NopBucketCounter) LastUpdateTime() int64 {
	return 0
}

func (nbc *NopBucketCounter) GetCount() int64 {
	return 0
}

func (nbc *NopBucketCounter) Inc() {

}

func (nbc *NopBucketCounter) IncN(n int64) {
}

func (nbc *NopBucketCounter) Dec() {
}

func (nbc *NopBucketCounter) DecN(n int64) {
}

func (nbc *NopBucketCounter) Update() {
}

func (nbc *NopBucketCounter) UpdateN(n int64) {
}

func (nbc *NopBucketCounter) GetBucketCounts() map[int64]int64 {
	return make(map[int64]int64, 0)
}

func (nbc *NopBucketCounter) GetBucketCountsSince(startTime int64) map[int64]int64 {
	return make(map[int64]int64, 0)
}

func (nbc *NopBucketCounter) GetBucketInterval() time.Duration {
	return 0
}

var nopBucketCounter = &NopBucketCounter{}
