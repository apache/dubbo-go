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

package impl

import (
	"math/rand"
	"sync"
	"time"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

const (
	// randomMask = ^(int64(1) << uint64(63))
	randomMask                  int64 = 9223372036854775807
	defaultUniformReservoirSize int64 = 1024
)

// A random sampling reservoir of a stream of int64s.
// Uses Vitter's Algorithm R to produce a statistically representative sample.
// Please read http://www.cs.umd.edu/~samir/498/vitter.pdf" Random Sampling with a Reservoir
type UniformReservoir struct {
	count  int64
	values []int64
	mutex  sync.RWMutex
}

func (u *UniformReservoir) Size() int {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	size := len(u.values)
	if u.count > int64(size) {
		return size
	}
	return int(u.count)
}

func (u *UniformReservoir) UpdateN(value int64) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	u.count++
	size := int64(len(u.values))
	if u.count <= size {
		u.values[u.count-1] = value
	} else {
		index := u.nextRandomPos(u.count)
		if index < size {
			u.values[index] = value
		}
	}
}

func (u *UniformReservoir) nextRandomPos(n int64) int64 {
	source := rand.NewSource(time.Now().UnixNano())
	var random int64
	var val int64
	for {
		random = source.Int63() & randomMask
		val = random % n
		if random-val+n-1 >= 0 {
			break
		}
	}

	return val
}

func (u *UniformReservoir) GetSnapshot() metrics.Snapshot {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	return NewUniformSnapshot(u.values)
}

func NewUniformReservoir(size int64) metrics.Reservoir {
	return &UniformReservoir{
		values: make([]int64, size),
	}
}
