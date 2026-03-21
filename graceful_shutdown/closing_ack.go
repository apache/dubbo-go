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

package graceful_shutdown

import (
	"strings"
	"sync"
	"sync/atomic"
)

// ClosingAckStats is a lightweight, consumer-side statistic for active closing notices.
// Received means the consumer observed an active notice; Removed means directory removal
// succeeded; Missed means the notice was observed but did not match a local directory entry.
type ClosingAckStats struct {
	Received uint64
	Removed  uint64
	Missed   uint64
}

type closingAckCounter struct {
	received atomic.Uint64
	removed  atomic.Uint64
	missed   atomic.Uint64
}

type closingAckTracker struct {
	counters sync.Map // map[string]*closingAckCounter
}

var defaultClosingAckTracker = &closingAckTracker{}

func isActiveClosingSource(source string) bool {
	return strings.HasSuffix(source, "-health-watch")
}

func (t *closingAckTracker) record(event ClosingEvent, removed bool) {
	if !isActiveClosingSource(event.Source) {
		return
	}

	counter := t.getOrCreateCounter(event.Source)
	counter.received.Add(1)
	if removed {
		counter.removed.Add(1)
		return
	}
	counter.missed.Add(1)
}

func (t *closingAckTracker) getOrCreateCounter(source string) *closingAckCounter {
	if value, ok := t.counters.Load(source); ok {
		return value.(*closingAckCounter)
	}
	counter := &closingAckCounter{}
	actual, _ := t.counters.LoadOrStore(source, counter)
	return actual.(*closingAckCounter)
}

func (t *closingAckTracker) snapshot() map[string]ClosingAckStats {
	stats := make(map[string]ClosingAckStats)
	t.counters.Range(func(key, value any) bool {
		counter := value.(*closingAckCounter)
		stats[key.(string)] = ClosingAckStats{
			Received: counter.received.Load(),
			Removed:  counter.removed.Load(),
			Missed:   counter.missed.Load(),
		}
		return true
	})
	return stats
}

func (t *closingAckTracker) reset() {
	t.counters.Range(func(key, _ any) bool {
		t.counters.Delete(key)
		return true
	})
}

// DefaultClosingAckStats returns the process-wide active-notice ack statistics.
func DefaultClosingAckStats() map[string]ClosingAckStats {
	return defaultClosingAckTracker.snapshot()
}
