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
	"sync/atomic"
	"time"
)

// TimeWindowAggregator wrappers sliding window to aggregate data.
//
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

type AggregateResult struct {
	Total float64
	Min   float64
	Max   float64
	Avg   float64
	Count uint64
}

// Result returns the aggregate result of the sliding window by aggregating all panes.
func (t *TimeWindowAggregator) Result() *AggregateResult {
	t.mux.RLock()
	defer t.mux.RUnlock()

	res := &AggregateResult{}

	total := 0.0
	count := uint64(0)
	max := math.SmallestNonzeroFloat64
	min := math.MaxFloat64

	for _, v := range t.window.values(time.Now().UnixMilli()) {
		total += v.(*aggregator).total.Load().(float64)
		count += v.(*aggregator).count.Load().(uint64)
		max = math.Max(max, v.(*aggregator).max.Load().(float64))
		min = math.Min(min, v.(*aggregator).min.Load().(float64))
	}

	if count > 0 {
		res.Avg = total / float64(count)
		res.Count = count
		res.Total = total
		res.Max = max
		res.Min = min
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
	return &aggregator{}
}

type aggregator struct {
	min   atomic.Value // float64
	max   atomic.Value // float64
	total atomic.Value // float64
	count atomic.Value // uint64
}

func (a *aggregator) add(v float64) {
	a.updateMin(v)
	a.updateMax(v)
	a.updateTotal(v)
	a.updateCount()
}

func (a *aggregator) updateMin(v float64) {
	for {
		store := a.min.Load()
		if store == nil {
			if ok := a.min.CompareAndSwap(nil, v); ok {
				return
			}
		} else {
			if ok := a.min.CompareAndSwap(store, math.Min(store.(float64), v)); ok {
				return
			}
		}
	}
}

func (a *aggregator) updateMax(v float64) {
	for {
		store := a.max.Load()
		if store == nil {
			if ok := a.max.CompareAndSwap(nil, v); ok {
				return
			}
		} else {
			if ok := a.max.CompareAndSwap(store, math.Max(store.(float64), v)); ok {
				return
			}
		}
	}
}

func (a *aggregator) updateTotal(v float64) {
	for {
		store := a.total.Load()
		if store == nil {
			if ok := a.total.CompareAndSwap(nil, v); ok {
				return
			}
		} else {
			if ok := a.total.CompareAndSwap(store, store.(float64)+v); ok {
				return
			}
		}
	}
}

func (a *aggregator) updateCount() {
	for {
		store := a.count.Load()
		if store == nil {
			if ok := a.count.CompareAndSwap(nil, uint64(1)); ok {
				return
			}
		} else {
			if ok := a.count.CompareAndSwap(store, store.(uint64)+1); ok {
				return
			}
		}
	}
}
