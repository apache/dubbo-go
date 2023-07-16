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

import (
	"github.com/influxdata/tdigest"
)

// TimeWindowQuantile wrappers sliding window around T-Digest.
//
// It uses T-Digest algorithm to calculate quantile.
// The window is divided into several panes, and each pane's value is a TDigest instance.
type TimeWindowQuantile struct {
	compression float64
	window      *slidingWindow
	mux         sync.RWMutex
}

func NewTimeWindowQuantile(compression float64, paneCount int, timeWindowSeconds int64) *TimeWindowQuantile {
	return &TimeWindowQuantile{
		compression: compression,
		window:      newSlidingWindow(paneCount, timeWindowSeconds*1000),
	}
}

// Quantile returns a quantile of the sliding window by merging all panes.
func (t *TimeWindowQuantile) Quantile(q float64) float64 {
	return t.mergeTDigests().Quantile(q)
}

// Quantiles returns quantiles of the sliding window by merging all panes.
func (t *TimeWindowQuantile) Quantiles(qs []float64) []float64 {
	td := t.mergeTDigests()

	res := make([]float64, len(qs))
	for i, q := range qs {
		res[i] = td.Quantile(q)
	}

	return res
}

// mergeTDigests merges all panes' TDigests into one TDigest.
func (t *TimeWindowQuantile) mergeTDigests() *tdigest.TDigest {
	t.mux.RLock()
	defer t.mux.RUnlock()

	td := tdigest.NewWithCompression(t.compression)
	for _, v := range t.window.values(time.Now().UnixMilli()) {
		td.AddCentroidList(v.(*tdigest.TDigest).Centroids())
	}
	return td
}

// Add adds a value to the sliding window's current pane.
func (t *TimeWindowQuantile) Add(value float64) {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.window.currentPane(time.Now().UnixMilli(), t.newEmptyValue).value.(*tdigest.TDigest).Add(value, 1)
}

func (t *TimeWindowQuantile) newEmptyValue() interface{} {
	return tdigest.NewWithCompression(t.compression)
}
