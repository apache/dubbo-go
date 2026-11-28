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

package iwrr

import (
	"math/rand"
	"slices"
	"sync"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

type iwrrEntry struct {
	weight int64
	index  int

	next *iwrrEntry
}

type iwrrQueue struct {
	head *iwrrEntry
	tail *iwrrEntry
}

func NewIwrrQueue() *iwrrQueue {
	return &iwrrQueue{}
}

func (item *iwrrQueue) push(entry *iwrrEntry) {
	entry.next = nil
	tail := item.tail
	item.tail = entry
	if tail == nil {
		item.head = entry
	} else {
		tail.next = entry
	}
}

func (item *iwrrQueue) pop() *iwrrEntry {
	head := item.head
	next := head.next
	head.next = nil
	item.head = next
	if next == nil {
		item.tail = nil
	}
	return head
}

func (item *iwrrQueue) empty() bool {
	return item.head == nil
}

// interleavedweightedRoundRobin picks from a fixed list of invokers.
type interleavedweightedRoundRobin struct {
	invokers  []base.Invoker
	scheduler iwrrScheduler
}

func NewInterleavedweightedRoundRobin(invokers []base.Invoker, invocation base.Invocation) *interleavedweightedRoundRobin {
	picker := &interleavedweightedRoundRobin{invokers: slices.Clone(invokers)}
	picker.scheduler.update(len(invokers), picker.invokerAt, invocation)
	return picker
}

func (iwrr *interleavedweightedRoundRobin) invokerAt(index int) base.Invoker {
	return iwrr.invokers[index]
}

func (iwrr *interleavedweightedRoundRobin) Pick(invocation base.Invocation) base.Invoker {
	return iwrr.scheduler.pick(len(iwrr.invokers), iwrr.invokerAt, invocation)
}

// The scheduler stores positions in the candidate list so cached state does
// not hold invokers that may have been replaced since the previous selection.
type iwrrScheduler struct {
	mu            sync.Mutex
	current, next iwrrQueue
	step          int64
	weights       []int64
	entries       []iwrrEntry
}

func (s *iwrrScheduler) pick(count int, invokerAt func(int) base.Invoker, invocation base.Invocation) base.Invoker {
	if count == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.update(count, invokerAt, invocation)
	if s.current.empty() {
		s.current, s.next = s.next, s.current
	}

	entry := s.current.pop()
	entry.weight -= s.step
	if entry.weight > 0 {
		s.current.push(entry)
	} else {
		// A queued zero-weight candidate belongs to the all-zero fallback.
		entry.weight = max(s.weights[entry.index], 1)
		s.next.push(entry)
	}
	return invokerAt(entry.index)
}

func (s *iwrrScheduler) update(count int, invokerAt func(int) base.Invoker, invocation base.Invocation) {
	changed := len(s.weights) != count
	if changed {
		s.weights = make([]int64, count)
		s.entries = make([]iwrrEntry, count)
	}

	now := time.Now().Unix()
	for index := range s.weights {
		weight := loadbalance.GetWeightAt(invokerAt(index), invocation, now)
		changed = changed || s.weights[index] != weight
		s.weights[index] = weight
	}
	if changed {
		s.reset()
	}
}

func (s *iwrrScheduler) reset() {
	s.current, s.next = iwrrQueue{}, iwrrQueue{}
	s.step = 0
	for _, weight := range s.weights {
		s.step = gcdInt(s.step, weight)
	}

	allZero := s.step == 0
	if allZero {
		s.step = 1
	}

	offset := rand.Intn(len(s.weights)) //NOSONAR
	for i := range s.entries {
		index := (i + offset) % len(s.entries)
		weight := s.weights[index]
		if allZero {
			weight = 1
		}
		if weight > 0 {
			entry := &s.entries[index]
			*entry = iwrrEntry{index: index, weight: weight}
			s.current.push(entry)
		}
	}
}

func gcdInt(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
