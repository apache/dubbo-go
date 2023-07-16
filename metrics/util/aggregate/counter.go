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

package aggregate

import (
	"sync"
	"time"
)

// TimeWindowCounter wrappers sliding window around a counter.
//
// It is concurrent-safe.
// When it works for calculating QPS, the counter's value means the number of requests in a pane.
type TimeWindowCounter struct {
	window *slidingWindow
	mux    sync.RWMutex
}

func NewTimeWindowCounter(paneCount int, timeWindowSeconds int64) *TimeWindowCounter {
	return &TimeWindowCounter{
		window: newSlidingWindow(paneCount, timeWindowSeconds*1000),
	}
}

// Count returns the sum of all panes' value.
func (t *TimeWindowCounter) Count() float64 {
	t.mux.RLock()
	defer t.mux.RUnlock()

	total := float64(0)
	for _, v := range t.window.values(time.Now().UnixMilli()) {
		total += v.(*counter).value
	}
	return total
}

// LivedSeconds returns the lived seconds of the sliding window.
func (t *TimeWindowCounter) LivedSeconds() int64 {
	t.mux.RLock()
	defer t.mux.RUnlock()

	windowLength := len(t.window.values(time.Now().UnixMilli()))
	return int64(windowLength) * t.window.paneIntervalInMs / 1000
}

// Add adds a step to the counter.
func (t *TimeWindowCounter) Add(step float64) {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.window.currentPane(time.Now().UnixMilli(), t.newEmptyValue).value.(*counter).add(step)
}

// Inc increments the counter by 1.
func (t *TimeWindowCounter) Inc() {
	t.Add(1)
}

func (t *TimeWindowCounter) newEmptyValue() interface{} {
	return &counter{0}
}

type counter struct {
	value float64
}

func (c *counter) add(v float64) {
	c.value += v
}
