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

package tps

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestLeakyBucket(t *testing.T) {
	lb := NewLeakyBucket(1, 10)

	r := lb.IsAllowable(8)
	assert.True(t, true, r)
	r = lb.IsAllowable(5)
	assert.False(t, false, r)
	time.Sleep(time.Millisecond * 5)
	r = lb.IsAllowable(5)
	assert.True(t, true, r)

}

func TestLeakyBucket_IsAllowable(t *testing.T) {

	creator := &leakyBucketStrategyCreator{}
	bucket := creator.Create(1, 10)

	allowed := bucket.IsAllowable(5)
	assert.True(t, true, allowed)
	allowed = bucket.IsAllowable(5)
	assert.True(t, true, allowed)
	allowed = bucket.IsAllowable(5)
	assert.False(t, false, allowed)
	time.Sleep(time.Millisecond * 5)
	allowed = bucket.IsAllowable(5)
	assert.True(t, true, allowed)

}
