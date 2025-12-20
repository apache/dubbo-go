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

package zookeeper

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

import (
	"github.com/dubbogo/gost/gof/observer"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// ============================================
// Mock Listener Implementation
// ============================================

type mockMappingListener struct {
	eventCount atomic.Int32
	lastEvent  observer.Event
	mu         sync.Mutex
	onEventErr error
	stopped    bool
}

func newMockMappingListener() *mockMappingListener {
	return &mockMappingListener{}
}

func (m *mockMappingListener) OnEvent(e observer.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventCount.Add(1)
	m.lastEvent = e
	return m.onEventErr
}

func (m *mockMappingListener) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
}

func (m *mockMappingListener) GetEventCount() int32 {
	return m.eventCount.Load()
}

func (m *mockMappingListener) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onEventErr = err
}

func (m *mockMappingListener) IsStopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

// ============================================
// ListenerSet Tests
// ============================================

func TestNewListenerSet(t *testing.T) {
	set := NewListenerSet()
	assert.NotNil(t, set)
	assert.NotNil(t, set.listeners)
	assert.Equal(t, 0, len(set.listeners))
}

func TestListenerSetAdd(t *testing.T) {
	tests := []struct {
		name          string
		addCount      int
		expectedCount int
	}{
		{"add single listener", 1, 1},
		{"add multiple listeners", 3, 3},
		{"add no listener", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := NewListenerSet()
			for i := 0; i < tt.addCount; i++ {
				listener := newMockMappingListener()
				set.Add(listener)
			}
			assert.Equal(t, tt.expectedCount, len(set.listeners))
		})
	}
}

func TestListenerSetAddDuplicate(t *testing.T) {
	set := NewListenerSet()
	listener := newMockMappingListener()

	set.Add(listener)
	set.Add(listener) // add same listener again

	assert.Equal(t, 1, len(set.listeners))
}

func TestListenerSetRemove(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(*ListenerSet) mapping.MappingListener
		removeExisting bool
		expectedCount  int
	}{
		{
			name: "remove existing listener",
			setup: func(s *ListenerSet) mapping.MappingListener {
				l := newMockMappingListener()
				s.Add(l)
				return l
			},
			removeExisting: true,
			expectedCount:  0,
		},
		{
			name: "remove non-existing listener",
			setup: func(s *ListenerSet) mapping.MappingListener {
				s.Add(newMockMappingListener())
				return newMockMappingListener() // return different listener
			},
			removeExisting: true,
			expectedCount:  1,
		},
		{
			name: "remove from empty set",
			setup: func(s *ListenerSet) mapping.MappingListener {
				return newMockMappingListener()
			},
			removeExisting: true,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := NewListenerSet()
			listener := tt.setup(set)
			if tt.removeExisting {
				set.Remove(listener)
			}
			assert.Equal(t, tt.expectedCount, len(set.listeners))
		})
	}
}

func TestListenerSetHas(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*ListenerSet) mapping.MappingListener
		expected bool
	}{
		{
			name: "has existing listener",
			setup: func(s *ListenerSet) mapping.MappingListener {
				l := newMockMappingListener()
				s.Add(l)
				return l
			},
			expected: true,
		},
		{
			name: "has non-existing listener",
			setup: func(s *ListenerSet) mapping.MappingListener {
				s.Add(newMockMappingListener())
				return newMockMappingListener()
			},
			expected: false,
		},
		{
			name: "has in empty set",
			setup: func(s *ListenerSet) mapping.MappingListener {
				return newMockMappingListener()
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := NewListenerSet()
			listener := tt.setup(set)
			assert.Equal(t, tt.expected, set.Has(listener))
		})
	}
}

func TestListenerSetForEach(t *testing.T) {
	t.Run("iterate all listeners", func(t *testing.T) {
		set := NewListenerSet()
		listeners := make([]*mockMappingListener, 3)
		for i := 0; i < 3; i++ {
			listeners[i] = newMockMappingListener()
			set.Add(listeners[i])
		}

		count := 0
		err := set.ForEach(func(l mapping.MappingListener) error {
			count++
			return nil
		})

		assert.Nil(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("stop on error", func(t *testing.T) {
		set := NewListenerSet()
		for i := 0; i < 3; i++ {
			set.Add(newMockMappingListener())
		}

		expectedErr := errors.New("test error")
		count := 0
		err := set.ForEach(func(l mapping.MappingListener) error {
			count++
			if count == 2 {
				return expectedErr
			}
			return nil
		})

		assert.Equal(t, expectedErr, err)
		assert.LessOrEqual(t, count, 3)
	})

	t.Run("empty set", func(t *testing.T) {
		set := NewListenerSet()
		count := 0
		err := set.ForEach(func(l mapping.MappingListener) error {
			count++
			return nil
		})

		assert.Nil(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestListenerSetConcurrency(t *testing.T) {
	set := NewListenerSet()
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent adds
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			set.Add(newMockMappingListener())
		}()
	}
	wg.Wait()

	assert.Equal(t, numGoroutines, len(set.listeners))

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			set.Has(newMockMappingListener())
		}()
	}
	wg.Wait()
}

// ============================================
// CacheListener Tests
// ============================================

func TestNewCacheListener(t *testing.T) {
	tests := []struct {
		name     string
		rootPath string
	}{
		{"normal root path", "/dubbo/mapping"},
		{"root path with trailing slash", "/dubbo/"},
		{"empty root path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener := NewCacheListener(tt.rootPath, nil)
			assert.NotNil(t, listener)
			assert.Equal(t, tt.rootPath, listener.rootPath)
		})
	}
}

func TestCacheListenerRemoveListener(t *testing.T) {
	t.Run("remove existing listener", func(t *testing.T) {
		cacheListener := NewCacheListener("/dubbo", nil)
		mockListener := newMockMappingListener()
		key := "/dubbo/mapping/test.service"

		// Manually add listener to keyListeners
		listenerSet := NewListenerSet()
		listenerSet.Add(mockListener)
		cacheListener.keyListeners.Store(key, listenerSet)

		// Remove listener
		cacheListener.RemoveListener(key, mockListener)

		// Verify listener is removed
		if listeners, ok := cacheListener.keyListeners.Load(key); ok {
			assert.False(t, listeners.(*ListenerSet).Has(mockListener))
		}
	})

	t.Run("remove from non-existing key", func(t *testing.T) {
		cacheListener := NewCacheListener("/dubbo", nil)
		mockListener := newMockMappingListener()

		// Should not panic
		cacheListener.RemoveListener("/non/existing/key", mockListener)
	})
}

func TestCacheListenerDataChange(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(*CacheListener) string
		event          remoting.Event
		expectedResult bool
		validateFunc   func(*testing.T, *CacheListener)
	}{
		{
			name: "data change with registered listener",
			setup: func(cl *CacheListener) string {
				key := "/dubbo/mapping/test.service"
				listenerSet := NewListenerSet()
				listenerSet.Add(newMockMappingListener())
				cl.keyListeners.Store(key, listenerSet)
				return key
			},
			event: remoting.Event{
				Path:    "/dubbo/mapping/test.service",
				Action:  remoting.EventTypeUpdate,
				Content: "app1,app2",
			},
			expectedResult: true,
		},
		{
			name: "data change without registered listener",
			setup: func(cl *CacheListener) string {
				return "/dubbo/mapping/test.service"
			},
			event: remoting.Event{
				Path:    "/dubbo/mapping/other.service",
				Action:  remoting.EventTypeUpdate,
				Content: "app1",
			},
			expectedResult: false,
		},
		{
			name: "data change with single app",
			setup: func(cl *CacheListener) string {
				key := "/dubbo/mapping/single.service"
				listenerSet := NewListenerSet()
				listenerSet.Add(newMockMappingListener())
				cl.keyListeners.Store(key, listenerSet)
				return key
			},
			event: remoting.Event{
				Path:    "/dubbo/mapping/single.service",
				Action:  remoting.EventTypeAdd,
				Content: "singleApp",
			},
			expectedResult: true,
		},
		{
			name: "data change with empty content",
			setup: func(cl *CacheListener) string {
				key := "/dubbo/mapping/empty.service"
				listenerSet := NewListenerSet()
				listenerSet.Add(newMockMappingListener())
				cl.keyListeners.Store(key, listenerSet)
				return key
			},
			event: remoting.Event{
				Path:    "/dubbo/mapping/empty.service",
				Action:  remoting.EventTypeUpdate,
				Content: "",
			},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheListener := NewCacheListener("/dubbo", nil)
			tt.setup(cacheListener)

			result := cacheListener.DataChange(tt.event)
			assert.Equal(t, tt.expectedResult, result)

			if tt.validateFunc != nil {
				tt.validateFunc(t, cacheListener)
			}
		})
	}
}

func TestCacheListenerDataChangeWithError(t *testing.T) {
	cacheListener := NewCacheListener("/dubbo", nil)
	key := "/dubbo/mapping/error.service"

	mockListener := newMockMappingListener()
	mockListener.SetError(errors.New("listener error"))

	listenerSet := NewListenerSet()
	listenerSet.Add(mockListener)
	cacheListener.keyListeners.Store(key, listenerSet)

	event := remoting.Event{
		Path:    key,
		Action:  remoting.EventTypeUpdate,
		Content: "app1",
	}

	result := cacheListener.DataChange(event)
	assert.False(t, result)
}

func TestCacheListenerPathToKey(t *testing.T) {
	tests := []struct {
		name     string
		rootPath string
		path     string
		expected string
	}{
		{
			name:     "normal path",
			rootPath: "/dubbo",
			path:     "/dubbo/mapping/com.example.Service",
			expected: "com.example.Service",
		},
		{
			name:     "path with multiple segments",
			rootPath: "/dubbo/metadata",
			path:     "/dubbo/metadata/group/com.example.Service",
			expected: "com.example.Service",
		},
		{
			name:     "empty path",
			rootPath: "/dubbo",
			path:     "",
			expected: "",
		},
		{
			name:     "root path only",
			rootPath: "/dubbo",
			path:     "/dubbo/mapping",
			expected: "mapping", // pathToKey returns the last segment after rootPath
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheListener := NewCacheListener(tt.rootPath, nil)
			result := cacheListener.pathToKey(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================
// Integration-like Tests
// ============================================

func TestListenerSetAddRemoveSequence(t *testing.T) {
	set := NewListenerSet()
	listeners := make([]*mockMappingListener, 5)

	// Add all listeners
	for i := 0; i < 5; i++ {
		listeners[i] = newMockMappingListener()
		set.Add(listeners[i])
	}
	assert.Equal(t, 5, len(set.listeners))

	// Remove odd indexed listeners
	for i := 1; i < 5; i += 2 {
		set.Remove(listeners[i])
	}
	assert.Equal(t, 3, len(set.listeners))

	// Verify remaining listeners
	for i := 0; i < 5; i++ {
		if i%2 == 0 {
			assert.True(t, set.Has(listeners[i]))
		} else {
			assert.False(t, set.Has(listeners[i]))
		}
	}
}

func TestCacheListenerMultipleKeys(t *testing.T) {
	cacheListener := NewCacheListener("/dubbo", nil)
	keys := []string{
		"/dubbo/mapping/service1",
		"/dubbo/mapping/service2",
		"/dubbo/mapping/service3",
	}

	// Add listeners for each key
	for _, key := range keys {
		listenerSet := NewListenerSet()
		listenerSet.Add(newMockMappingListener())
		cacheListener.keyListeners.Store(key, listenerSet)
	}

	// Verify each key has listeners
	for _, key := range keys {
		event := remoting.Event{
			Path:    key,
			Action:  remoting.EventTypeUpdate,
			Content: "app1",
		}
		assert.True(t, cacheListener.DataChange(event))
	}

	// Verify non-existing key returns false
	event := remoting.Event{
		Path:    "/dubbo/mapping/nonexistent",
		Action:  remoting.EventTypeUpdate,
		Content: "app1",
	}
	assert.False(t, cacheListener.DataChange(event))
}
