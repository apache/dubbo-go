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

// SlidingWindow adopts sliding window algorithm for statistics.
//
// It is not thread-safe.
// A window contains paneCount panes.
// intervalInMs = paneCount * paneIntervalInMs.
type slidingWindow struct {
	paneCount        int
	intervalInMs     int64
	paneIntervalInMs int64
	paneSlice        []*pane
}

func newSlidingWindow(paneCount int, intervalInMs int64) *slidingWindow {
	return &slidingWindow{
		paneCount:        paneCount,
		intervalInMs:     intervalInMs,
		paneIntervalInMs: intervalInMs / int64(paneCount),
		paneSlice:        make([]*pane, paneCount),
	}
}

// values get all values from the slidingWindow's paneSlice.
func (s *slidingWindow) values(timeMillis int64) []interface{} {
	if timeMillis < 0 {
		return make([]interface{}, 0)
	}

	res := make([]interface{}, 0, s.paneCount)

	for _, p := range s.paneSlice {
		if p == nil || s.isPaneDeprecated(p, timeMillis) {
			continue
		}
		res = append(res, p.value)
	}

	return res
}

// isPaneDeprecated checks if the specified pane is deprecated at the specified timeMillis
func (s *slidingWindow) isPaneDeprecated(pane *pane, timeMillis int64) bool {
	return timeMillis-pane.startInMs > s.intervalInMs
}

// currentPane get the pane at the specified timestamp in milliseconds.
func (s *slidingWindow) currentPane(timeMillis int64, newEmptyValue func() interface{}) *pane {
	if timeMillis < 0 {
		return nil
	}
	paneIdx := s.calcPaneIdx(timeMillis)
	paneStart := s.calcPaneStart(timeMillis)

	if s.paneSlice[paneIdx] == nil {
		p := newPane(s.paneIntervalInMs, paneStart, newEmptyValue())
		s.paneSlice[paneIdx] = p
		return p
	} else {
		p := s.paneSlice[paneIdx]
		if paneStart == p.startInMs {
			return p
		} else if paneStart > p.startInMs {
			// The pane has deprecated. To avoid the overhead of creating a new instance, reset the original pane directly.
			p.resetTo(paneStart, newEmptyValue())
			return p
		} else {
			// The specified timestamp has passed.
			return newPane(s.paneIntervalInMs, paneStart, newEmptyValue())
		}
	}
}

func (s *slidingWindow) calcPaneIdx(timeMillis int64) int {
	return int(timeMillis/s.paneIntervalInMs) % s.paneCount
}

func (s *slidingWindow) calcPaneStart(timeMillis int64) int64 {
	return timeMillis - timeMillis%s.paneIntervalInMs
}
