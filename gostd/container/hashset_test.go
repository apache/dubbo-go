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
package container

import "testing"

func TestSetNew(t *testing.T) {
	set := NewSet(2, 1)

	if actualValue := Size(); actualValue != 2 {
		t.Errorf("Got %v expected %v", actualValue, 2)
	}
	if actualValue := Contains(1); actualValue != true {
		t.Errorf("Got %v expected %v", actualValue, true)
	}
	if actualValue := Contains(2); actualValue != true {
		t.Errorf("Got %v expected %v", actualValue, true)
	}
	if actualValue := Contains(3); actualValue != false {
		t.Errorf("Got %v expected %v", actualValue, true)
	}
}

func TestSetAdd(t *testing.T) {
	set := NewSet()
	Add()
	Add(1)
	Add(2)
	Add(2, 3)
	Add()
	if actualValue := Empty(); actualValue != false {
		t.Errorf("Got %v expected %v", actualValue, false)
	}
	if actualValue := Size(); actualValue != 3 {
		t.Errorf("Got %v expected %v", actualValue, 3)
	}
}

func TestSetContains(t *testing.T) {
	set := NewSet()
	Add(3, 1, 2)
	Add(2, 3)
	Add()
	if actualValue := Contains(); actualValue != true {
		t.Errorf("Got %v expected %v", actualValue, true)
	}
	if actualValue := Contains(1); actualValue != true {
		t.Errorf("Got %v expected %v", actualValue, true)
	}
	if actualValue := Contains(1, 2, 3); actualValue != true {
		t.Errorf("Got %v expected %v", actualValue, true)
	}
	if actualValue := Contains(1, 2, 3, 4); actualValue != false {
		t.Errorf("Got %v expected %v", actualValue, false)
	}
}

func TestSetRemove(t *testing.T) {
	set := NewSet()
	Add(3, 1, 2)
	Remove()
	if actualValue := Size(); actualValue != 3 {
		t.Errorf("Got %v expected %v", actualValue, 3)
	}
	Remove(1)
	if actualValue := Size(); actualValue != 2 {
		t.Errorf("Got %v expected %v", actualValue, 2)
	}
	Remove(3)
	Remove(3)
	Remove()
	Remove(2)
	if actualValue := Size(); actualValue != 0 {
		t.Errorf("Got %v expected %v", actualValue, 0)
	}
}

func benchmarkContains(b *testing.B, set *HashSet, size int) {
	for i := 0; i < b.N; i++ {
		for n := 0; n < size; n++ {
			Contains(n)
		}
	}
}

func benchmarkAdd(b *testing.B, set *HashSet, size int) {
	for i := 0; i < b.N; i++ {
		for n := 0; n < size; n++ {
			Add(n)
		}
	}
}

func benchmarkRemove(b *testing.B, set *HashSet, size int) {
	for i := 0; i < b.N; i++ {
		for n := 0; n < size; n++ {
			Remove(n)
		}
	}
}

func BenchmarkHashSetContains100(b *testing.B) {
	b.StopTimer()
	size := 100
	set := NewSet()
	for n := 0; n < size; n++ {
		Add(n)
	}
	b.StartTimer()
	benchmarkContains(b, set, size)
}

func BenchmarkHashSetContains1000(b *testing.B) {
	b.StopTimer()
	size := 1000
	set := NewSet()
	for n := 0; n < size; n++ {
		Add(n)
	}
	b.StartTimer()
	benchmarkContains(b, set, size)
}

func BenchmarkHashSetContains10000(b *testing.B) {
	b.StopTimer()
	size := 10000
	set := NewSet()
	for n := 0; n < size; n++ {
		Add(n)
	}
	b.StartTimer()
	benchmarkContains(b, set, size)
}

func BenchmarkHashSetContains100000(b *testing.B) {
	b.StopTimer()
	size := 100000
	set := NewSet()
	for n := 0; n < size; n++ {
		Add(n)
	}
	b.StartTimer()
	benchmarkContains(b, set, size)
}

func BenchmarkHashSetAdd100(b *testing.B) {
	b.StopTimer()
	size := 100
	set := NewSet()
	b.StartTimer()
	benchmarkAdd(b, set, size)
}

func BenchmarkHashSetAdd1000(b *testing.B) {
	b.StopTimer()
	size := 1000
	set := NewSet()
	for n := 0; n < size; n++ {
		Add(n)
	}
	b.StartTimer()
	benchmarkAdd(b, set, size)
}

func BenchmarkHashSetAdd10000(b *testing.B) {
	b.StopTimer()
	size := 10000
	set := NewSet()
	for n := 0; n < size; n++ {
		Add(n)
	}
	b.StartTimer()
	benchmarkAdd(b, set, size)
}

func BenchmarkHashSetAdd100000(b *testing.B) {
	b.StopTimer()
	size := 100000
	set := NewSet()
	for n := 0; n < size; n++ {
		Add(n)
	}
	b.StartTimer()
	benchmarkAdd(b, set, size)
}

func BenchmarkHashSetRemove100(b *testing.B) {
	b.StopTimer()
	size := 100
	set := NewSet()
	for n := 0; n < size; n++ {
		Add(n)
	}
	b.StartTimer()
	benchmarkRemove(b, set, size)
}

func BenchmarkHashSetRemove1000(b *testing.B) {
	b.StopTimer()
	size := 1000
	set := NewSet()
	for n := 0; n < size; n++ {
		Add(n)
	}
	b.StartTimer()
	benchmarkRemove(b, set, size)
}

func BenchmarkHashSetRemove10000(b *testing.B) {
	b.StopTimer()
	size := 10000
	set := NewSet()
	for n := 0; n < size; n++ {
		Add(n)
	}
	b.StartTimer()
	benchmarkRemove(b, set, size)
}

func BenchmarkHashSetRemove100000(b *testing.B) {
	b.StopTimer()
	size := 100000
	set := NewSet()
	for n := 0; n < size; n++ {
		Add(n)
	}
	b.StartTimer()
	benchmarkRemove(b, set, size)
}
