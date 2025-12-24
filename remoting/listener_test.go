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

package remoting

import (
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

const (
	testPath    = "/dubbo/services/test"
	testContent = "test-content"
)

type mockDataListener struct {
	mu     sync.Mutex
	events []Event
}

func (m *mockDataListener) DataChange(event Event) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return true
}

// Compile-time check
var _ DataListener = (*mockDataListener)(nil)

func TestEventType(t *testing.T) {
	assert.Equal(t, "add", EventTypeAdd.String())
	assert.Equal(t, "delete", EventTypeDel.String())
	assert.Equal(t, "update", EventTypeUpdate.String())

	assert.Equal(t, EventType(0), EventTypeAdd)
	assert.Equal(t, EventType(1), EventTypeDel)
	assert.Equal(t, EventType(2), EventTypeUpdate)
}

func TestEvent(t *testing.T) {
	event := Event{Path: testPath, Action: EventTypeAdd, Content: testContent}

	assert.Equal(t, testPath, event.Path)
	assert.Equal(t, EventTypeAdd, event.Action)
	assert.Equal(t, testContent, event.Content)

	str := event.String()
	assert.Contains(t, str, "add")
	assert.Contains(t, str, testContent)
}

func TestDataListener(t *testing.T) {
	listener := &mockDataListener{}

	events := []Event{
		{Path: testPath, Action: EventTypeAdd, Content: testContent},
		{Path: testPath, Action: EventTypeDel, Content: ""},
		{Path: testPath, Action: EventTypeUpdate, Content: "updated"},
	}

	for _, event := range events {
		assert.True(t, listener.DataChange(event))
	}

	assert.Equal(t, 3, len(listener.events))
	assert.Equal(t, events, listener.events)
}

func TestDataListenerConcurrent(t *testing.T) {
	listener := &mockDataListener{}
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			listener.DataChange(Event{Path: testPath, Action: EventType(idx % 3), Content: testContent})
		}(i)
	}

	wg.Wait()
	assert.Equal(t, 100, len(listener.events))
}
