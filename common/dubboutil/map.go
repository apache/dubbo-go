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
)

// LoadSyncMap loads a typed value from m.
func LoadSyncMap[T any](m *sync.Map, key any) (T, bool) {
	var zero T
	if m == nil {
		return zero, false
	}

	value, ok := m.Load(key)
	if !ok {
		return zero, false
	}

	typedValue, ok := value.(T)
	if !ok {
		return zero, false
	}
	return typedValue, true
}

// LoadOrStoreSyncMap loads a typed value from m or stores value when key is absent.
func LoadOrStoreSyncMap[T any](m *sync.Map, key any, value T) (T, bool, bool) {
	var zero T
	if m == nil {
		return zero, false, false
	}

	actual, loaded := m.LoadOrStore(key, value)
	typedValue, ok := actual.(T)
	if !ok {
		return zero, loaded, false
	}
	return typedValue, loaded, true
}
