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

package dubboutil

import (
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestLoadOrStoreSyncMapConcurrentFirstInitialization(t *testing.T) {
	const goroutines = 64

	type state struct {
		id int
	}

	var states sync.Map
	start := make(chan struct{})
	values := make(chan *state, goroutines)
	okValues := make(chan bool, goroutines)
	wg := sync.WaitGroup{}

	for range goroutines {
		wg.Go(func() {
			<-start
			value, _, ok := LoadOrStoreSyncMap(&states, "key", &state{id: 1})
			okValues <- ok
			values <- value
		})
	}

	close(start)
	wg.Wait()
	close(values)
	close(okValues)

	for ok := range okValues {
		assert.True(t, ok)
	}

	var first *state
	for value := range values {
		if first == nil {
			first = value
			continue
		}
		assert.Same(t, first, value)
	}

	count := 0
	states.Range(func(_, _ any) bool {
		count++
		return true
	})
	assert.Equal(t, 1, count)
}

func TestLoadSyncMapReturnsFalseForWrongType(t *testing.T) {
	var states sync.Map
	states.Store("key", "value")

	value, ok := LoadSyncMap[*struct{}](&states, "key")
	assert.False(t, ok)
	assert.Nil(t, value)
}
