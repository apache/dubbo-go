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

	"github.com/stretchr/testify/assert"
)

// Test for HillClimbing limiter's Acquire method
func TestHillClimbing_Acquire(t *testing.T) {
	limiter := NewHillClimbing().(*HillClimbing)

	// Simulating that there is remaining capacity
	limiter.limitation.Store(100)
	limiter.inflight.Store(50)

	updater, err := limiter.Acquire()
	assert.NotNil(t, updater)
	assert.NoError(t, err)

	// Simulating no remaining capacity
	limiter.limitation.Store(50)
	limiter.inflight.Store(50)

	updater, err = limiter.Acquire()
	assert.Nil(t, updater)
	assert.Error(t, err, ErrReachLimitation)
}

// Test the HillClimbingUpdater's DoUpdate method
func TestHillClimbingUpdater_DoUpdate(t *testing.T) {
	limiter := NewHillClimbing().(*HillClimbing)
	updater := NewHillClimbingUpdater(limiter)

	// Simulate the limiter update with arbitrary values for RTT and inflight
	// Normally, this would adjust the limiter's limitation based on RTT and inflight metrics
	err := updater.DoUpdate()
	assert.NoError(t, err)
}

// Test adjustLimitation method with different options
func TestHillClimbingUpdater_AdjustLimitation(t *testing.T) {
	limiter := NewHillClimbing().(*HillClimbing)
	updater := NewHillClimbingUpdater(limiter)

	// Simulate a scenario where the limiter is set to extend its capacity
	err := updater.adjustLimitation(HillClimbingOptionExtend)
	assert.NoError(t, err)
	assert.True(t, limiter.limitation.Load() > initialLimitation)

	// Simulate a scenario where the limiter is set to shrink its capacity
	err = updater.adjustLimitation(HillClimbingOptionShrink)
	assert.NoError(t, err)
	assert.True(t, limiter.limitation.Load() < initialLimitation)
}

// Test HillClimbing's Remaining capacity behavior
func TestHillClimbing_Remaining(t *testing.T) {
	limiter := NewHillClimbing().(*HillClimbing)
	limiter.limitation.Store(100)
	limiter.inflight.Store(30)

	remaining := limiter.Remaining()
	assert.Equal(t, uint64(70), remaining)

	// Simulate that inflight requests exceed the limitation
	limiter.inflight.Store(120)
	remaining = limiter.Remaining()
	assert.Equal(t, uint64(0), remaining)
}
