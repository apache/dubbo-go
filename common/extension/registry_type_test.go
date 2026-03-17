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

package extension

import (
	"strconv"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestRegistryBasicOps(t *testing.T) {
	r := NewRegistry[int]("test-registry")

	_, ok := r.Get("missing")
	assert.False(t, ok)

	r.Register("a", 1)
	v, ok := r.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, v)

	must := r.MustGet("a")
	assert.Equal(t, 1, must)

	snapshot := r.Snapshot()
	assert.Equal(t, map[string]int{"a": 1}, snapshot)
	snapshot["a"] = 99

	v, ok = r.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, v)

	names := r.Names()
	assert.Equal(t, 1, len(names))
	assert.Equal(t, "a", names[0])

	r.Unregister("a")
	_, ok = r.Get("a")
	assert.False(t, ok)
}

func TestRegistryConcurrentAccess(t *testing.T) {
	r := NewRegistry[int]("concurrent")

	var wg sync.WaitGroup
	workers := 32
	iterations := 200

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := "k-" + strconv.Itoa(worker) + "-" + strconv.Itoa(j)
				r.Register(key, j)
				_, _ = r.Get(key)
				_ = r.Snapshot()
				_ = r.Names()
				r.Unregister(key)
			}
		}(i)
	}

	wg.Wait()
}
