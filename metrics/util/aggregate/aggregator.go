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
	"math"
	"sync"
	"time"
)

// TimeWindowAggregator wrappers sliding window to aggregate data.
//
// It is concurrent-safe.
// It uses custom struct aggregator to aggregate data.
// The window is divided into several panes, and each pane's value is an aggregator instance.
type TimeWindowAggregator struct {
	window *slidingWindow
	mux    sync.RWMutex
}

func NewTimeWindowAggregator(paneCount int, timeWindowSeconds int64) *TimeWindowAggregator {
	return &TimeWindowAggregator{
		window: newSlidingWindow(paneCount, timeWindowSeconds*1000),
	}
}

type Result struct {
	Total float64
	Min   float64
	Max   float64
	Avg   float64
	Count uint64
	Last  float64
}

// Result returns the aggregate result of the sliding window by aggregating all panes.
func (t *TimeWindowAggregator) Result() *Result {
	t.mux.RLock()
	defer t.mux.RUnlock()

	res := &Result{}

	total := 0.0
	count := uint64(0)
	max := math.SmallestNonzeroFloat64
	min := math.MaxFloat64
	last := math.NaN()

	for _, v := range t.window.values(time.Now().UnixMilli()) {
		total += v.(*aggregator).total
		count += v.(*aggregator).count
		max = math.Max(max, v.(*aggregator).max)
		min = math.Min(min, v.(*aggregator).min)
		last = v.(*aggregator).last
	}

	if count > 0 {
		res.Avg = total / float64(count)
		res.Count = count
		res.Total = total
		res.Max = max
		res.Min = min
		res.Last = last
	}

	return res
}

// Add adds a value to the sliding window's current pane.
func (t *TimeWindowAggregator) Add(v float64) {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.window.currentPane(time.Now().UnixMilli(), t.newEmptyValue).value.(*aggregator).add(v)
}

func (t *TimeWindowAggregator) newEmptyValue() interface{} {
	return newAggregator()
}

// aggregator is a custom struct to aggregate data.
//
// It is NOT concurrent-safe.
// It aggregates data by calculating the min, max, total and count.
type aggregator struct {
	min   float64
	max   float64
	total float64
	count uint64
	last  float64
}

func newAggregator() *aggregator {
	return &aggregator{
		min:   math.MaxFloat64,
		max:   math.SmallestNonzeroFloat64,
		total: float64(0),
		count: uint64(0),
	}
}

func (a *aggregator) add(v float64) {
	a.min = math.Min(a.min, v)
	a.max = math.Max(a.max, v)
	a.last = v
	a.total += v
	a.count++
}
