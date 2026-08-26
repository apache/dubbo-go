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

package atomic

import (
	"encoding/json"
	"math"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBool(t *testing.T) {
	var value Bool
	assert.False(t, value.Load())
	assert.Equal(t, "false", value.String())

	value.Store(true)
	assert.True(t, value.Load())
	assert.True(t, value.Swap(false))
	assert.True(t, value.CAS(false, true))
	assert.False(t, value.CompareAndSwap(false, true))
	assert.True(t, value.Toggle())
	assert.False(t, value.Load())

	data, err := json.Marshal(&value)
	require.NoError(t, err)
	assert.JSONEq(t, "false", string(data))
	require.NoError(t, json.Unmarshal([]byte("true"), &value))
	assert.True(t, value.Load())
}

func TestInt32(t *testing.T) {
	value := NewInt32(2)
	assert.Equal(t, int32(5), value.Add(3))
	assert.Equal(t, int32(4), value.Dec())
	assert.Equal(t, int32(5), value.Inc())
	assert.Equal(t, int32(3), value.Sub(2))
	assert.Equal(t, int32(3), value.Swap(7))
	assert.True(t, value.CAS(7, 8))
	assert.False(t, value.CompareAndSwap(7, 9))
	assert.Equal(t, "8", value.String())

	data, err := json.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, "8", string(data))
	require.NoError(t, json.Unmarshal([]byte("10"), value))
	assert.Equal(t, int32(10), value.Load())
}

func TestInt64(t *testing.T) {
	value := NewInt64(2)
	assert.Equal(t, int64(5), value.Add(3))
	assert.Equal(t, int64(4), value.Dec())
	assert.Equal(t, int64(5), value.Inc())
	assert.Equal(t, int64(3), value.Sub(2))
	assert.Equal(t, int64(3), value.Swap(7))
	assert.True(t, value.CAS(7, 8))
	assert.False(t, value.CompareAndSwap(7, 9))
	assert.Equal(t, "8", value.String())

	data, err := json.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, "8", string(data))
	require.NoError(t, json.Unmarshal([]byte("10"), value))
	assert.Equal(t, int64(10), value.Load())
}

func TestUint32(t *testing.T) {
	value := NewUint32(2)
	assert.Equal(t, uint32(5), value.Add(3))
	assert.Equal(t, uint32(4), value.Dec())
	assert.Equal(t, uint32(5), value.Inc())
	assert.Equal(t, uint32(3), value.Sub(2))
	assert.Equal(t, uint32(3), value.Swap(7))
	assert.True(t, value.CAS(7, 8))
	assert.False(t, value.CompareAndSwap(7, 9))
	assert.Equal(t, "8", value.String())

	data, err := json.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, "8", string(data))
	require.NoError(t, json.Unmarshal([]byte("10"), value))
	assert.Equal(t, uint32(10), value.Load())
}

func TestUint64(t *testing.T) {
	value := NewUint64(2)
	assert.Equal(t, uint64(5), value.Add(3))
	assert.Equal(t, uint64(4), value.Dec())
	assert.Equal(t, uint64(5), value.Inc())
	assert.Equal(t, uint64(3), value.Sub(2))
	assert.Equal(t, uint64(3), value.Swap(7))
	assert.True(t, value.CAS(7, 8))
	assert.False(t, value.CompareAndSwap(7, 9))
	assert.Equal(t, "8", value.String())

	data, err := json.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, "8", string(data))
	require.NoError(t, json.Unmarshal([]byte("10"), value))
	assert.Equal(t, uint64(10), value.Load())
}

func TestPointer(t *testing.T) {
	first, second, third := 1, 2, 3
	value := NewPointer(&first)
	assert.Same(t, &first, value.Load())
	assert.Same(t, &first, value.Swap(&second))
	assert.True(t, value.CompareAndSwap(&second, &third))
	assert.False(t, value.CompareAndSwap(&second, &first))
	assert.Same(t, &third, value.Load())

	value.Store(nil)
	assert.Nil(t, value.Load())
}

func TestDuration(t *testing.T) {
	value := NewDuration(2 * time.Second)
	assert.Equal(t, 5*time.Second, value.Add(3*time.Second))
	assert.Equal(t, 4*time.Second, value.Sub(time.Second))
	assert.Equal(t, 4*time.Second, value.Swap(7*time.Second))
	assert.True(t, value.CAS(7*time.Second, 8*time.Second))
	assert.False(t, value.CompareAndSwap(7*time.Second, 9*time.Second))
	assert.Equal(t, "8s", value.String())

	data, err := json.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, "8000000000", string(data))
	require.NoError(t, json.Unmarshal([]byte("10000000000"), value))
	assert.Equal(t, 10*time.Second, value.Load())
}

func TestFloat64(t *testing.T) {
	value := NewFloat64(2.5)
	assert.InDelta(t, 5.5, value.Add(3), 1e-9)
	assert.InDelta(t, 4.5, value.Sub(1), 1e-9)
	assert.InDelta(t, 4.5, value.Swap(7.5), 1e-9)
	assert.True(t, value.CAS(7.5, 8.5))
	assert.False(t, value.CompareAndSwap(7.5, 9.5))
	assert.Equal(t, "8.5", value.String())

	data, err := json.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, "8.5", string(data))
	require.NoError(t, json.Unmarshal([]byte("10.5"), value))
	assert.InDelta(t, 10.5, value.Load(), 1e-9)

	negativeZero := math.Copysign(0, -1)
	value.Store(negativeZero)
	assert.Equal(t, math.Float64bits(negativeZero), math.Float64bits(value.Load()))

	nan := math.Float64frombits(0x7ff8000000000001)
	value.Store(nan)
	assert.True(t, value.CompareAndSwap(nan, 1))
}

func TestTime(t *testing.T) {
	var value Time
	assert.True(t, value.Load().IsZero())

	now := time.Now()
	value.Store(now)
	assert.Equal(t, now, value.Load())
	assert.Equal(t, now.Add(time.Second), NewTime(now.Add(time.Second)).Load())
}

func TestCopyAfterFirstUseSharesState(t *testing.T) {
	var original Int64
	original.Store(1)
	copied := original
	assert.Equal(t, int64(2), copied.Inc())
	assert.Equal(t, int64(2), original.Load())

	first, second := 1, 2
	pointer := NewPointer(&first)
	pointerCopy := *pointer
	pointerCopy.Store(&second)
	assert.Same(t, &second, pointer.Load())
}

func TestConcurrentZeroValueInitialization(t *testing.T) {
	var value Uint64
	const goroutines = 100
	const increments = 100

	var waitGroup sync.WaitGroup
	for range goroutines {
		waitGroup.Go(func() {
			for range increments {
				value.Inc()
			}
		})
	}
	waitGroup.Wait()

	assert.Equal(t, uint64(goroutines*increments), value.Load())
}
