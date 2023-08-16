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

func NewResult() *Result {
	return &Result{
		Min:  math.MaxFloat64,
		Max:  math.SmallestNonzeroFloat64,
		Last: math.NaN(),
	}
}

func (r *Result) Update(v float64) {
	r.Min = math.Min(r.Min, v)
	r.Max = math.Max(r.Max, v)
	r.Last = v
	r.Total += v
	r.Count++
}

func (r *Result) Merge(o *Result) {
	r.Min = math.Min(r.Min, o.Min)
	r.Max = math.Max(r.Max, o.Max)
	r.Total += o.Total
	r.Count += o.Count
	r.Last = o.Last
}

func (r *Result) Get() *Result {
	if r.Count > 0 {
		r.Avg = r.Total / float64(r.Count)
	}
	return r
}

// Result returns the aggregate result of the sliding window by aggregating all panes.
func (t *TimeWindowAggregator) Result() *Result {
	t.mux.RLock()
	defer t.mux.RUnlock()
	res := NewResult()
	for _, v := range t.window.values(time.Now().UnixMilli()) {
		res.Merge(v.(*Result)) // Last not as expect, but agg result has no Last value
	}
	return res.Get()
}

// Add adds a value to the sliding window's current pane.
func (t *TimeWindowAggregator) Add(v float64) {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.window.currentPane(time.Now().UnixMilli(), t.newEmptyValue).value.(*Result).Update(v)
}

func (t *TimeWindowAggregator) newEmptyValue() interface{} {
	return NewResult()
}
