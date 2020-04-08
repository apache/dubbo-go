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
	"fmt"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestLeakyBucket(t *testing.T) {
	lb := NewLeakyBucket(10, 10)
	start := time.Now()
	for j := 0; j < 5; j++ {
		for i := 0; i < 100; i++ {
			prev := time.Now()
			r := lb.IsAllowable()
			if r == true {
				fmt.Println(i, time.Now().Sub(prev), r)
			} else {
				fmt.Println(i, r)
			}
			time.Sleep(time.Millisecond)
		}
	}
	fmt.Println(time.Now().Sub(start).Seconds())
}

func TestLeakyBucket_IsAllowable(t *testing.T) {
	creator := &leakyBucketStrategyCreator{}
	strategy := creator.Create(2, 1000) // per request 50ms
	//
	assert.True(t, strategy.IsAllowable())
	//time.Sleep(490*time.Millisecond)
	//assert.True(t, strategy.IsAllowable())
	//time.Sleep(1*time.Millisecond)
	//assert.True(t, strategy.IsAllowable())
	start := time.Now()
	for i := 0; i < 50; i++ {
		if strategy.IsAllowable() {
			fmt.Println(i, "true", time.Now().Sub(start))
		} else {
			fmt.Println(i, "false")
		}
		time.Sleep(22 * time.Millisecond)
	}
	fmt.Println(time.Now().Sub(start).Nanoseconds() / 1e6)
}
