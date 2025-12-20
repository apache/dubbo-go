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

package limiter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMinDuration(t *testing.T) {
	// Test when lhs is smaller than rhs
	dur1 := 2 * time.Second
	dur2 := 3 * time.Second
	result := minDuration(dur1, dur2)
	assert.Equal(t, dur1, result)

	// Test when rhs is smaller than lhs
	result = minDuration(dur2, dur1)
	assert.Equal(t, dur1, result)

	// Test when both durations are equal
	dur3 := 2 * time.Second
	result = minDuration(dur3, dur3)
	assert.Equal(t, dur3, result)
}
