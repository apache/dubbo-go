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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

func TestGcdInt(t *testing.T) {
	tests := []struct {
		name     string
		a        int64
		b        int64
		expected int64
	}{
		{
			name:     "gcd of 12 and 8",
			a:        12,
			b:        8,
			expected: 4,
		},
		{
			name:     "gcd of 100 and 50",
			a:        100,
			b:        50,
			expected: 50,
		},
		{
			name:     "gcd of 17 and 13",
			a:        17,
			b:        13,
			expected: 1,
		},
		{
			name:     "gcd with zero",
			a:        10,
			b:        0,
			expected: 10,
		},
		{
			name:     "gcd of same numbers",
			a:        25,
			b:        25,
			expected: 25,
		},
		{
			name:     "gcd of prime numbers",
			a:        7,
			b:        11,
			expected: 1,
		},
		{
			name:     "gcd with 1",
			a:        100,
			b:        1,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gcdInt(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewIwrrQueue(t *testing.T) {
	queue := NewIwrrQueue()
	assert.NotNil(t, queue)
	assert.Nil(t, queue.head)
	assert.Nil(t, queue.tail)
	assert.True(t, queue.empty())
}

func TestIwrrQueuePush(t *testing.T) {
	queue := NewIwrrQueue()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(url)

	entry := &iwrrEntry{
		weight:  100,
		invoker: invoker,
	}

	queue.push(entry)

	assert.False(t, queue.empty())
	assert.Equal(t, entry, queue.head)
	assert.Equal(t, entry, queue.tail)
	assert.Nil(t, entry.next)
}

func TestIwrrQueuePushMultiple(t *testing.T) {
	queue := NewIwrrQueue()
	entries := make([]*iwrrEntry, 3)

	for i := 0; i < 3; i++ {
		url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
		require.NoError(t, err)
		invoker := base.NewBaseInvoker(url)
		entries[i] = &iwrrEntry{
			weight:  int64(100 * (i + 1)),
			invoker: invoker,
		}
		queue.push(entries[i])
	}

	assert.False(t, queue.empty())
	assert.Equal(t, entries[0], queue.head)
	assert.Equal(t, entries[2], queue.tail)

	// Check linked list structure
	assert.Equal(t, entries[1], entries[0].next)
	assert.Equal(t, entries[2], entries[1].next)
	assert.Nil(t, entries[2].next)
}

func TestIwrrQueuePop(t *testing.T) {
	queue := NewIwrrQueue()
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(url)

	entry := &iwrrEntry{
		weight:  100,
		invoker: invoker,
	}

	queue.push(entry)
	popped := queue.pop()

	assert.Equal(t, entry, popped)
	assert.True(t, queue.empty())
	assert.Nil(t, queue.head)
	assert.Nil(t, queue.tail)
	assert.Nil(t, popped.next)
}

func TestIwrrQueuePopMultiple(t *testing.T) {
	queue := NewIwrrQueue()
	entries := make([]*iwrrEntry, 3)

	for i := 0; i < 3; i++ {
		url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
		require.NoError(t, err)
		invoker := base.NewBaseInvoker(url)
		entries[i] = &iwrrEntry{
			weight:  int64(100 * (i + 1)),
			invoker: invoker,
		}
		queue.push(entries[i])
	}

	// Pop first entry
	popped1 := queue.pop()
	assert.Equal(t, entries[0], popped1)
	assert.False(t, queue.empty())
	assert.Equal(t, entries[1], queue.head)

	// Pop second entry
	popped2 := queue.pop()
	assert.Equal(t, entries[1], popped2)
	assert.False(t, queue.empty())
	assert.Equal(t, entries[2], queue.head)

	// Pop third entry
	popped3 := queue.pop()
	assert.Equal(t, entries[2], popped3)
	assert.True(t, queue.empty())
	assert.Nil(t, queue.head)
	assert.Nil(t, queue.tail)
}

func TestIwrrQueueEmpty(t *testing.T) {
	queue := NewIwrrQueue()
	assert.True(t, queue.empty())

	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	invoker := base.NewBaseInvoker(url)
	entry := &iwrrEntry{
		weight:  100,
		invoker: invoker,
	}

	queue.push(entry)
	assert.False(t, queue.empty())

	queue.pop()
	assert.True(t, queue.empty())
}

func TestIwrrQueuePopEmptyQueue(t *testing.T) {
	queue := NewIwrrQueue()
	assert.True(t, queue.empty())

	// Pop from empty queue should panic
	assert.Panics(t, func() {
		queue.pop()
	}, "pop from empty queue should panic")
}

func TestIwrrQueuePushPopSequence(t *testing.T) {
	queue := NewIwrrQueue()

	// Create multiple entries
	entries := make([]*iwrrEntry, 5)
	for i := 0; i < 5; i++ {
		url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
		require.NoError(t, err)
		invoker := base.NewBaseInvoker(url)
		entries[i] = &iwrrEntry{
			weight:  int64(100 * (i + 1)),
			invoker: invoker,
		}
	}

	// Push 3 entries
	queue.push(entries[0])
	queue.push(entries[1])
	queue.push(entries[2])

	// Pop 1 entry
	popped := queue.pop()
	assert.Equal(t, entries[0], popped)

	// Push 2 more entries
	queue.push(entries[3])
	queue.push(entries[4])

	// Pop remaining entries in FIFO order
	assert.Equal(t, entries[1], queue.pop())
	assert.Equal(t, entries[2], queue.pop())
	assert.Equal(t, entries[3], queue.pop())
	assert.Equal(t, entries[4], queue.pop())

	assert.True(t, queue.empty())
}

func TestGcdIntCommutative(t *testing.T) {
	// GCD should be commutative: gcd(a, b) == gcd(b, a)
	pairs := [][2]int64{
		{12, 8},
		{100, 50},
		{17, 13},
	}

	for _, pair := range pairs {
		result1 := gcdInt(pair[0], pair[1])
		result2 := gcdInt(pair[1], pair[0])
		assert.Equal(t, result1, result2, "gcd should be commutative")
	}
}

func TestGcdIntExtendedProperty(t *testing.T) {
	// GCD extended property: gcd(gcd(a, b), c) == gcd(a, gcd(b, c)) for three numbers
	a, b, c := int64(24), int64(18), int64(12)

	result1 := gcdInt(gcdInt(a, b), c)
	result2 := gcdInt(a, gcdInt(b, c))

	assert.Equal(t, result1, result2, "gcd extended property should hold")
}

func TestGcdIntBothZero(t *testing.T) {
	result := gcdInt(0, 0)
	assert.Equal(t, int64(0), result)
}

func TestGcdIntLargeNumbers(t *testing.T) {
	result := gcdInt(1000000, 500000)
	assert.Equal(t, int64(500000), result)

	result = gcdInt(123456789, 987654321)
	assert.Greater(t, result, int64(0))
}
