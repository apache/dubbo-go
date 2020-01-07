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

type CompassContext interface {
}

type Compass interface {
	Metered
	Sampling

	// Add a record duration
	Update(duration time.Duration)

	// Add a record duration.
	// isSuccess whether is success
	// errorCode and addon could be empty
	UpdateWithError(duration time.Duration, isSuccess bool, errorCode string, addon string)

	// Get the success count of the invocation
	GetSuccessCount() int64

	// Get the distribution of error code
	GetErrorCodeCounts() map[string]BucketCounter

	// Get the number of occurrence of added on metric
	GetAddonCounts() map[string]BucketCounter

	// Get the success count of the invocation
	GetBucketSuccessCount() BucketCounter
}
