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

package zookeeper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestWaitForRetry_ExitSignaled verifies that an already-closed exit channel
// makes waitForRetry return true promptly instead of blocking on the timer.
func TestWaitForRetry_ExitSignaled(t *testing.T) {
	l := &ZkEventListener{exit: make(chan struct{})}
	close(l.exit)

	start := time.Now()
	exited := l.waitForRetry(5 * time.Second)
	elapsed := time.Since(start)

	assert.True(t, exited, "waitForRetry should report exit when l.exit is closed")
	assert.Less(t, elapsed, 500*time.Millisecond, "waitForRetry should return immediately on a closed exit")
}

// TestWaitForRetry_TimerFires verifies that, without an exit signal,
// waitForRetry waits for the delay and reports false.
func TestWaitForRetry_TimerFires(t *testing.T) {
	l := &ZkEventListener{exit: make(chan struct{})}

	start := time.Now()
	exited := l.waitForRetry(20 * time.Millisecond)
	elapsed := time.Since(start)

	assert.False(t, exited, "waitForRetry should report false when the delay elapses")
	assert.GreaterOrEqual(t, elapsed, 20*time.Millisecond, "should wait for the delay")
	assert.Less(t, elapsed, 2*time.Second, "should not wait much longer than the delay")
}

// TestWaitForRetry_ExitDuringWait verifies the exit path taken mid-wait: when
// l.exit is closed while waiting, waitForRetry returns true quickly and the
// internal timer is stopped, avoiding the timer/goroutine leak that time.After
// would cause (see #3558).
func TestWaitForRetry_ExitDuringWait(t *testing.T) {
	l := &ZkEventListener{exit: make(chan struct{})}

	done := make(chan bool, 1)
	go func() {
		done <- l.waitForRetry(10 * time.Second)
	}()

	// Let it start waiting on the timer.
	time.Sleep(50 * time.Millisecond)
	close(l.exit)

	select {
	case exited := <-done:
		assert.True(t, exited, "waitForRetry should return true once l.exit is closed")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("waitForRetry did not return promptly after l.exit was closed")
	}
}
