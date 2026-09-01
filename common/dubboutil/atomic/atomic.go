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

// Package atomic provides atomic types backed by sync/atomic. Its API mirrors
// the go.uber.org/atomic types used by dubbo-go.
package atomic

import (
	"encoding/json"
	"math"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"
)

type noCompare [0]func()

// loadOrCreate initializes an atomic value on first use. The field itself is a
// pointer so a wrapper copied after first use keeps sharing the same atomic
// state instead of copying a sync/atomic value.
func loadOrCreate[T any](target **T) *T {
	pointer := (*unsafe.Pointer)(unsafe.Pointer(target))
	if value := (*T)(atomic.LoadPointer(pointer)); value != nil {
		return value
	}

	value := new(T)
	if atomic.CompareAndSwapPointer(pointer, nil, unsafe.Pointer(value)) {
		return value
	}
	return (*T)(atomic.LoadPointer(pointer))
}

// Bool is an atomic wrapper around bool.
type Bool struct {
	_     noCompare
	value *atomic.Bool
}

func (b *Bool) atomicValue() *atomic.Bool { return loadOrCreate(&b.value) }

// NewBool creates a new Bool.
func NewBool(value bool) *Bool {
	result := &Bool{}
	result.Store(value)
	return result
}

// Load atomically loads the wrapped value.
func (b *Bool) Load() bool { return b.atomicValue().Load() }

// Store atomically stores the passed value.
func (b *Bool) Store(value bool) { b.atomicValue().Store(value) }

// Swap atomically stores the passed value and returns the old value.
func (b *Bool) Swap(value bool) bool { return b.atomicValue().Swap(value) }

// CAS is an atomic compare-and-swap.
//
// Deprecated: Use CompareAndSwap.
func (b *Bool) CAS(old, new bool) bool { return b.CompareAndSwap(old, new) }

// CompareAndSwap is an atomic compare-and-swap.
func (b *Bool) CompareAndSwap(old, new bool) bool {
	return b.atomicValue().CompareAndSwap(old, new)
}

// Toggle atomically negates the value and returns the old value.
func (b *Bool) Toggle() (old bool) {
	for {
		old = b.Load()
		if b.CompareAndSwap(old, !old) {
			return old
		}
	}
}

// String encodes the wrapped value as a string.
func (b *Bool) String() string { return strconv.FormatBool(b.Load()) }

// MarshalJSON encodes the wrapped value into JSON.
func (b *Bool) MarshalJSON() ([]byte, error) { return json.Marshal(b.Load()) }

// UnmarshalJSON decodes JSON into the wrapped value.
func (b *Bool) UnmarshalJSON(data []byte) error {
	var value bool
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	b.Store(value)
	return nil
}

// Int32 is an atomic wrapper around int32.
type Int32 struct {
	_     noCompare
	value *atomic.Int32
}

func (i *Int32) atomicValue() *atomic.Int32 { return loadOrCreate(&i.value) }

// NewInt32 creates a new Int32.
func NewInt32(value int32) *Int32 {
	result := &Int32{}
	result.Store(value)
	return result
}

// Load atomically loads the wrapped value.
func (i *Int32) Load() int32 { return i.atomicValue().Load() }

// Add atomically adds delta and returns the new value.
func (i *Int32) Add(delta int32) int32 { return i.atomicValue().Add(delta) }

// Sub atomically subtracts delta and returns the new value.
func (i *Int32) Sub(delta int32) int32 { return i.Add(-delta) }

// Inc atomically increments the value and returns the new value.
func (i *Int32) Inc() int32 { return i.Add(1) }

// Dec atomically decrements the value and returns the new value.
func (i *Int32) Dec() int32 { return i.Add(-1) }

// CAS is an atomic compare-and-swap.
//
// Deprecated: Use CompareAndSwap.
func (i *Int32) CAS(old, new int32) bool { return i.CompareAndSwap(old, new) }

// CompareAndSwap is an atomic compare-and-swap.
func (i *Int32) CompareAndSwap(old, new int32) bool {
	return i.atomicValue().CompareAndSwap(old, new)
}

// Store atomically stores the passed value.
func (i *Int32) Store(value int32) { i.atomicValue().Store(value) }

// Swap atomically stores the passed value and returns the old value.
func (i *Int32) Swap(value int32) int32 { return i.atomicValue().Swap(value) }

// MarshalJSON encodes the wrapped value into JSON.
func (i *Int32) MarshalJSON() ([]byte, error) { return json.Marshal(i.Load()) }

// UnmarshalJSON decodes JSON into the wrapped value.
func (i *Int32) UnmarshalJSON(data []byte) error {
	var value int32
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	i.Store(value)
	return nil
}

// String encodes the wrapped value as a string.
func (i *Int32) String() string { return strconv.FormatInt(int64(i.Load()), 10) }

// Int64 is an atomic wrapper around int64.
type Int64 struct {
	_     noCompare
	value *atomic.Int64
}

func (i *Int64) atomicValue() *atomic.Int64 { return loadOrCreate(&i.value) }

// NewInt64 creates a new Int64.
func NewInt64(value int64) *Int64 {
	result := &Int64{}
	result.Store(value)
	return result
}

// Load atomically loads the wrapped value.
func (i *Int64) Load() int64 { return i.atomicValue().Load() }

// Add atomically adds delta and returns the new value.
func (i *Int64) Add(delta int64) int64 { return i.atomicValue().Add(delta) }

// Sub atomically subtracts delta and returns the new value.
func (i *Int64) Sub(delta int64) int64 { return i.Add(-delta) }

// Inc atomically increments the value and returns the new value.
func (i *Int64) Inc() int64 { return i.Add(1) }

// Dec atomically decrements the value and returns the new value.
func (i *Int64) Dec() int64 { return i.Add(-1) }

// CAS is an atomic compare-and-swap.
//
// Deprecated: Use CompareAndSwap.
func (i *Int64) CAS(old, new int64) bool { return i.CompareAndSwap(old, new) }

// CompareAndSwap is an atomic compare-and-swap.
func (i *Int64) CompareAndSwap(old, new int64) bool {
	return i.atomicValue().CompareAndSwap(old, new)
}

// Store atomically stores the passed value.
func (i *Int64) Store(value int64) { i.atomicValue().Store(value) }

// Swap atomically stores the passed value and returns the old value.
func (i *Int64) Swap(value int64) int64 { return i.atomicValue().Swap(value) }

// MarshalJSON encodes the wrapped value into JSON.
func (i *Int64) MarshalJSON() ([]byte, error) { return json.Marshal(i.Load()) }

// UnmarshalJSON decodes JSON into the wrapped value.
func (i *Int64) UnmarshalJSON(data []byte) error {
	var value int64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	i.Store(value)
	return nil
}

// String encodes the wrapped value as a string.
func (i *Int64) String() string { return strconv.FormatInt(i.Load(), 10) }

// Uint32 is an atomic wrapper around uint32.
type Uint32 struct {
	_     noCompare
	value *atomic.Uint32
}

func (i *Uint32) atomicValue() *atomic.Uint32 { return loadOrCreate(&i.value) }

// NewUint32 creates a new Uint32.
func NewUint32(value uint32) *Uint32 {
	result := &Uint32{}
	result.Store(value)
	return result
}

// Load atomically loads the wrapped value.
func (i *Uint32) Load() uint32 { return i.atomicValue().Load() }

// Add atomically adds delta and returns the new value.
func (i *Uint32) Add(delta uint32) uint32 { return i.atomicValue().Add(delta) }

// Sub atomically subtracts delta and returns the new value.
func (i *Uint32) Sub(delta uint32) uint32 { return i.Add(^(delta - 1)) }

// Inc atomically increments the value and returns the new value.
func (i *Uint32) Inc() uint32 { return i.Add(1) }

// Dec atomically decrements the value and returns the new value.
func (i *Uint32) Dec() uint32 { return i.Sub(1) }

// CAS is an atomic compare-and-swap.
//
// Deprecated: Use CompareAndSwap.
func (i *Uint32) CAS(old, new uint32) bool { return i.CompareAndSwap(old, new) }

// CompareAndSwap is an atomic compare-and-swap.
func (i *Uint32) CompareAndSwap(old, new uint32) bool {
	return i.atomicValue().CompareAndSwap(old, new)
}

// Store atomically stores the passed value.
func (i *Uint32) Store(value uint32) { i.atomicValue().Store(value) }

// Swap atomically stores the passed value and returns the old value.
func (i *Uint32) Swap(value uint32) uint32 { return i.atomicValue().Swap(value) }

// MarshalJSON encodes the wrapped value into JSON.
func (i *Uint32) MarshalJSON() ([]byte, error) { return json.Marshal(i.Load()) }

// UnmarshalJSON decodes JSON into the wrapped value.
func (i *Uint32) UnmarshalJSON(data []byte) error {
	var value uint32
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	i.Store(value)
	return nil
}

// String encodes the wrapped value as a string.
func (i *Uint32) String() string { return strconv.FormatUint(uint64(i.Load()), 10) }

// Uint64 is an atomic wrapper around uint64.
type Uint64 struct {
	_     noCompare
	value *atomic.Uint64
}

func (i *Uint64) atomicValue() *atomic.Uint64 { return loadOrCreate(&i.value) }

// NewUint64 creates a new Uint64.
func NewUint64(value uint64) *Uint64 {
	result := &Uint64{}
	result.Store(value)
	return result
}

// Load atomically loads the wrapped value.
func (i *Uint64) Load() uint64 { return i.atomicValue().Load() }

// Add atomically adds delta and returns the new value.
func (i *Uint64) Add(delta uint64) uint64 { return i.atomicValue().Add(delta) }

// Sub atomically subtracts delta and returns the new value.
func (i *Uint64) Sub(delta uint64) uint64 { return i.Add(^(delta - 1)) }

// Inc atomically increments the value and returns the new value.
func (i *Uint64) Inc() uint64 { return i.Add(1) }

// Dec atomically decrements the value and returns the new value.
func (i *Uint64) Dec() uint64 { return i.Sub(1) }

// CAS is an atomic compare-and-swap.
//
// Deprecated: Use CompareAndSwap.
func (i *Uint64) CAS(old, new uint64) bool { return i.CompareAndSwap(old, new) }

// CompareAndSwap is an atomic compare-and-swap.
func (i *Uint64) CompareAndSwap(old, new uint64) bool {
	return i.atomicValue().CompareAndSwap(old, new)
}

// Store atomically stores the passed value.
func (i *Uint64) Store(value uint64) { i.atomicValue().Store(value) }

// Swap atomically stores the passed value and returns the old value.
func (i *Uint64) Swap(value uint64) uint64 { return i.atomicValue().Swap(value) }

// MarshalJSON encodes the wrapped value into JSON.
func (i *Uint64) MarshalJSON() ([]byte, error) { return json.Marshal(i.Load()) }

// UnmarshalJSON decodes JSON into the wrapped value.
func (i *Uint64) UnmarshalJSON(data []byte) error {
	var value uint64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	i.Store(value)
	return nil
}

// String encodes the wrapped value as a string.
func (i *Uint64) String() string { return strconv.FormatUint(i.Load(), 10) }

// Pointer is an atomic pointer of type *T.
type Pointer[T any] struct {
	_     noCompare
	value *atomic.Pointer[T]
}

func (p *Pointer[T]) atomicValue() *atomic.Pointer[T] { return loadOrCreate(&p.value) }

// NewPointer creates a new Pointer.
func NewPointer[T any](value *T) *Pointer[T] {
	result := &Pointer[T]{}
	result.Store(value)
	return result
}

// Load atomically loads the wrapped value.
func (p *Pointer[T]) Load() *T { return p.atomicValue().Load() }

// Store atomically stores the passed value.
func (p *Pointer[T]) Store(value *T) { p.atomicValue().Store(value) }

// Swap atomically stores the passed value and returns the old value.
func (p *Pointer[T]) Swap(value *T) *T { return p.atomicValue().Swap(value) }

// CompareAndSwap is an atomic compare-and-swap.
func (p *Pointer[T]) CompareAndSwap(old, new *T) bool {
	return p.atomicValue().CompareAndSwap(old, new)
}

// Duration is an atomic wrapper around time.Duration.
type Duration struct {
	_     noCompare
	value *atomic.Int64
}

func (d *Duration) atomicValue() *atomic.Int64 { return loadOrCreate(&d.value) }

// NewDuration creates a new Duration.
func NewDuration(value time.Duration) *Duration {
	result := &Duration{}
	result.Store(value)
	return result
}

// Load atomically loads the wrapped value.
func (d *Duration) Load() time.Duration { return time.Duration(d.atomicValue().Load()) }

// Store atomically stores the passed value.
func (d *Duration) Store(value time.Duration) { d.atomicValue().Store(int64(value)) }

// Add atomically adds delta and returns the new value.
func (d *Duration) Add(delta time.Duration) time.Duration {
	return time.Duration(d.atomicValue().Add(int64(delta)))
}

// Sub atomically subtracts delta and returns the new value.
func (d *Duration) Sub(delta time.Duration) time.Duration { return d.Add(-delta) }

// CAS is an atomic compare-and-swap.
//
// Deprecated: Use CompareAndSwap.
func (d *Duration) CAS(old, new time.Duration) bool { return d.CompareAndSwap(old, new) }

// CompareAndSwap is an atomic compare-and-swap.
func (d *Duration) CompareAndSwap(old, new time.Duration) bool {
	return d.atomicValue().CompareAndSwap(int64(old), int64(new))
}

// Swap atomically stores the passed value and returns the old value.
func (d *Duration) Swap(value time.Duration) time.Duration {
	return time.Duration(d.atomicValue().Swap(int64(value)))
}

// String encodes the wrapped value as a string.
func (d *Duration) String() string { return d.Load().String() }

// MarshalJSON encodes the wrapped value into JSON.
func (d *Duration) MarshalJSON() ([]byte, error) { return json.Marshal(d.Load()) }

// UnmarshalJSON decodes JSON into the wrapped value.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var value time.Duration
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	d.Store(value)
	return nil
}

// Float64 is an atomic wrapper around float64.
type Float64 struct {
	_     noCompare
	value *atomic.Uint64
}

func (f *Float64) atomicValue() *atomic.Uint64 { return loadOrCreate(&f.value) }

// NewFloat64 creates a new Float64.
func NewFloat64(value float64) *Float64 {
	result := &Float64{}
	result.Store(value)
	return result
}

// Load atomically loads the wrapped value.
func (f *Float64) Load() float64 { return math.Float64frombits(f.atomicValue().Load()) }

// Store atomically stores the passed value.
func (f *Float64) Store(value float64) { f.atomicValue().Store(math.Float64bits(value)) }

// Add atomically adds delta and returns the new value.
func (f *Float64) Add(delta float64) float64 {
	for {
		old := f.Load()
		newValue := old + delta
		if f.CompareAndSwap(old, newValue) {
			return newValue
		}
	}
}

// Sub atomically subtracts delta and returns the new value.
func (f *Float64) Sub(delta float64) float64 { return f.Add(-delta) }

// CAS is an atomic compare-and-swap.
//
// Deprecated: Use CompareAndSwap.
func (f *Float64) CAS(old, new float64) bool { return f.CompareAndSwap(old, new) }

// CompareAndSwap is an atomic compare-and-swap.
func (f *Float64) CompareAndSwap(old, new float64) bool {
	return f.atomicValue().CompareAndSwap(math.Float64bits(old), math.Float64bits(new))
}

// Swap atomically stores the passed value and returns the old value.
func (f *Float64) Swap(value float64) float64 {
	return math.Float64frombits(f.atomicValue().Swap(math.Float64bits(value)))
}

// String encodes the wrapped value as a string.
func (f *Float64) String() string { return strconv.FormatFloat(f.Load(), 'g', -1, 64) }

// MarshalJSON encodes the wrapped value into JSON.
func (f *Float64) MarshalJSON() ([]byte, error) { return json.Marshal(f.Load()) }

// UnmarshalJSON decodes JSON into the wrapped value.
func (f *Float64) UnmarshalJSON(data []byte) error {
	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	f.Store(value)
	return nil
}

// Time is an atomic wrapper around time.Time.
type Time struct {
	_     noCompare
	value *atomic.Pointer[time.Time]
}

func (t *Time) atomicValue() *atomic.Pointer[time.Time] { return loadOrCreate(&t.value) }

// NewTime creates a new Time.
func NewTime(value time.Time) *Time {
	result := &Time{}
	result.Store(value)
	return result
}

// Load atomically loads the wrapped value.
func (t *Time) Load() time.Time {
	value := t.atomicValue().Load()
	if value == nil {
		return time.Time{}
	}
	return *value
}

// Store atomically stores the passed value.
func (t *Time) Store(value time.Time) { t.atomicValue().Store(&value) }
