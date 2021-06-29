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

package utils

import (
	"sync"
	"testing"
)

import (
	"github.com/RoaringBitmap/roaring"
	"github.com/stretchr/testify/assert"
)

func TestJoinIfNotEqual(t *testing.T) {
	l := roaring.NewBitmap()
	l.Add(uint32(1))
	l.Add(uint32(2))
	l.Add(uint32(3))
	r := roaring.NewBitmap()
	r.AddRange(1, 4)

	// this is for race condition test
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		JoinIfNotEqual(l, r)
	}()

	go func() {
		defer wg.Done()
		JoinIfNotEqual(l, r)
	}()
	wg.Wait()

	assert.True(t, l.Equals(JoinIfNotEqual(l, r)))
}

func BenchmarkJoinIfNotEqual10(b *testing.B) {
	benchmarkJoinIfNotEqual(b, 10)
}

func BenchmarkJoinIfNotEqual1000(b *testing.B) {
	benchmarkJoinIfNotEqual(b, 1000)
}

func BenchmarkJoinIfNotEqual100000(b *testing.B) {
	benchmarkJoinIfNotEqual(b, 100000)
}

func benchmarkJoinIfNotEqual(b *testing.B, size int) {
	l := roaring.NewBitmap()
	for i := 0; i < size; i++ {
		l.Add(uint32(i))
	}
	r := roaring.NewBitmap()
	r.AddRange(0, uint64(size+1))

	for i := 0; i < b.N; i++ {
		JoinIfNotEqual(l, r)
	}
}
