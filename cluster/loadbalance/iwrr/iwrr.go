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
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type iwrrEntry struct {
	weight  int64
	invoker protocol.Invoker

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

// InterleavedweightedRoundRobin struct
type interleavedweightedRoundRobin struct {
	current *iwrrQueue
	next    *iwrrQueue
	step    int64
	mu      sync.Mutex
}

func NewInterleavedweightedRoundRobin(invokers []protocol.Invoker, invocation protocol.Invocation) *interleavedweightedRoundRobin {
	iwrrp := new(interleavedweightedRoundRobin)
	iwrrp.current = NewIwrrQueue()
	iwrrp.next = NewIwrrQueue()

	size := uint64(len(invokers))
	offset := rand.Uint64() % size
	step := int64(0)
	for idx := uint64(0); idx < size; idx++ {
		invoker := invokers[(idx+offset)%size]
		weight := loadbalance.GetWeight(invoker, invocation)
		step = gcdInt(step, weight)
		iwrrp.current.push(&iwrrEntry{
			invoker: invoker,
			weight:  weight,
		})
	}
	iwrrp.step = step

	return iwrrp
}

func (iwrr *interleavedweightedRoundRobin) Pick(invocation protocol.Invocation) protocol.Invoker {
	iwrr.mu.Lock()
	defer iwrr.mu.Unlock()

	if iwrr.current.empty() {
		iwrr.current, iwrr.next = iwrr.next, iwrr.current
	}

	entry := iwrr.current.pop()
	entry.weight -= iwrr.step

	if entry.weight > 0 {
		iwrr.current.push(entry)
	} else {
		weight := loadbalance.GetWeight(entry.invoker, invocation)
		if weight < 0 {
			weight = 0
		}
		entry.weight = weight
		iwrr.next.push(entry)
	}

	return entry.invoker
}

func gcdInt(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
