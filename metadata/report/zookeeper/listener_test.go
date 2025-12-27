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
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

import (
	"github.com/dubbogo/gost/gof/observer"

	"github.com/stretchr/testify/assert"

	"golang.org/x/sync/errgroup"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type mockMappingListener struct {
	eventCount atomic.Int32
	mu         sync.Mutex
	onEventErr error
}

func newMockMappingListener() *mockMappingListener { return &mockMappingListener{} }

func (m *mockMappingListener) OnEvent(e observer.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventCount.Add(1)
	return m.onEventErr
}

func (m *mockMappingListener) Stop() {}

func (m *mockMappingListener) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onEventErr = err
}

// Ensure mockMappingListener implements mapping.MappingListener
var _ mapping.MappingListener = (*mockMappingListener)(nil)

func TestListenerSet(t *testing.T) {
	set := NewListenerSet()
	assert.NotNil(t, set)
	assert.Equal(t, 0, len(set.listeners))

	// Add listeners
	l1, l2 := newMockMappingListener(), newMockMappingListener()
	set.Add(l1)
	set.Add(l2)
	set.Add(l1) // duplicate
	assert.Equal(t, 2, len(set.listeners))

	// Has
	assert.True(t, set.Has(l1))
	assert.False(t, set.Has(newMockMappingListener()))

	// Remove
	set.Remove(l1)
	assert.False(t, set.Has(l1))
	assert.Equal(t, 1, len(set.listeners))

	// ForEach
	count := 0
	err := set.ForEach(func(l mapping.MappingListener) error { count++; return nil })
	assert.Nil(t, err)
	assert.Equal(t, 1, count)

	// ForEach with error
	set.Add(newMockMappingListener())
	expectedErr := errors.New("test")
	err = set.ForEach(func(l mapping.MappingListener) error { return expectedErr })
	assert.Equal(t, expectedErr, err)
}

func TestListenerSetConcurrency(t *testing.T) {
	set := NewListenerSet()
	g, _ := errgroup.WithContext(context.Background())

	for i := 0; i < 50; i++ {
		g.Go(func() error {
			set.Add(newMockMappingListener())
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 50, len(set.listeners))

}

func TestCacheListener(t *testing.T) {
	cl := NewCacheListener("/dubbo", nil)
	assert.NotNil(t, cl)
	assert.Equal(t, "/dubbo", cl.rootPath)

	// Setup listener
	key := "/dubbo/mapping/test.service"
	listenerSet := NewListenerSet()
	listenerSet.Add(newMockMappingListener())
	cl.keyListeners.Store(key, listenerSet)

	// DataChange with registered listener
	event := remoting.Event{Path: key, Action: remoting.EventTypeUpdate, Content: "app1"}
	assert.True(t, cl.DataChange(event))

	// DataChange without registered listener
	event.Path = "/dubbo/mapping/other"
	assert.False(t, cl.DataChange(event))

	// RemoveListener
	ml := newMockMappingListener()
	listenerSet.Add(ml)
	cl.RemoveListener(key, ml)
	listeners, _ := cl.keyListeners.Load(key)
	assert.False(t, listeners.(*ListenerSet).Has(ml))
}

func TestCacheListenerDataChangeWithError(t *testing.T) {
	cl := NewCacheListener("/dubbo", nil)
	key := "/dubbo/mapping/error.service"

	ml := newMockMappingListener()
	ml.SetError(errors.New("listener error"))

	listenerSet := NewListenerSet()
	listenerSet.Add(ml)
	cl.keyListeners.Store(key, listenerSet)

	event := remoting.Event{Path: key, Action: remoting.EventTypeUpdate, Content: "app1"}
	assert.False(t, cl.DataChange(event))
}

func TestCacheListenerPathToKey(t *testing.T) {
	cl := NewCacheListener("/dubbo", nil)
	assert.Equal(t, "com.example.Service", cl.pathToKey("/dubbo/mapping/com.example.Service"))
	assert.Equal(t, "", cl.pathToKey(""))
}
