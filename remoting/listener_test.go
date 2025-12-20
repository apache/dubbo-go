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

// ============================================
// Mock DataListener Implementation
// ============================================

type mockDataListener struct {
	mu            sync.Mutex
	events        []Event
	returnValue   bool
	dataChangeCnt int
}

func newMockDataListener() *mockDataListener {
	return &mockDataListener{
		events:      make([]Event, 0),
		returnValue: true,
	}
}

func (m *mockDataListener) DataChange(event Event) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	m.dataChangeCnt++
	return m.returnValue
}

func (m *mockDataListener) getEvents() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.events
}

func (m *mockDataListener) getDataChangeCnt() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dataChangeCnt
}

func (m *mockDataListener) setReturnValue(val bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.returnValue = val
}

// ============================================
// EventType Tests
// ============================================

func TestEventType(t *testing.T) {
	tests := []struct {
		name       string
		eventType  EventType
		expectedStr string
	}{
		{
			name:       "add event",
			eventType:  EventTypeAdd,
			expectedStr: "add",
		},
		{
			name:       "delete event",
			eventType:  EventTypeDel,
			expectedStr: "delete",
		},
		{
			name:       "update event",
			eventType:  EventTypeUpdate,
			expectedStr: "update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedStr, tt.eventType.String())
		})
	}
}

func TestEventTypeValues(t *testing.T) {
	assert.Equal(t, EventType(0), EventTypeAdd)
	assert.Equal(t, EventType(1), EventTypeDel)
	assert.Equal(t, EventType(2), EventTypeUpdate)
}

// ============================================
// Event Tests
// ============================================

func TestEvent(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		action  EventType
		content string
	}{
		{
			name:    "add event",
			path:    "/dubbo/services/test",
			action:  EventTypeAdd,
			content: "new-content",
		},
		{
			name:    "delete event",
			path:    "/dubbo/services/test",
			action:  EventTypeDel,
			content: "",
		},
		{
			name:    "update event",
			path:    "/dubbo/services/test",
			action:  EventTypeUpdate,
			content: "updated-content",
		},
		{
			name:    "empty path",
			path:    "",
			action:  EventTypeAdd,
			content: "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := Event{
				Path:    tt.path,
				Action:  tt.action,
				Content: tt.content,
			}
			assert.Equal(t, tt.path, event.Path)
			assert.Equal(t, tt.action, event.Action)
			assert.Equal(t, tt.content, event.Content)
		})
	}
}

func TestEventString(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		contains []string
	}{
		{
			name: "add event string",
			event: Event{
				Path:    "/test/path",
				Action:  EventTypeAdd,
				Content: "test-content",
			},
			contains: []string{"add", "test-content"},
		},
		{
			name: "delete event string",
			event: Event{
				Path:    "/test/path",
				Action:  EventTypeDel,
				Content: "",
			},
			contains: []string{"delete"},
		},
		{
			name: "update event string",
			event: Event{
				Path:    "/test/path",
				Action:  EventTypeUpdate,
				Content: "updated",
			},
			contains: []string{"update", "updated"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.event.String()
			for _, substr := range tt.contains {
				assert.Contains(t, str, substr)
			}
		})
	}
}

// ============================================
// DataListener Interface Tests
// ============================================

func TestDataListenerInterface(t *testing.T) {
	listener := newMockDataListener()

	// Verify mockDataListener implements DataListener interface
	var _ DataListener = listener

	event := Event{
		Path:    "/test/path",
		Action:  EventTypeAdd,
		Content: "test-content",
	}

	result := listener.DataChange(event)
	assert.True(t, result)
	assert.Equal(t, 1, listener.getDataChangeCnt())

	events := listener.getEvents()
	assert.Equal(t, 1, len(events))
	assert.Equal(t, event, events[0])
}

func TestDataListenerReturnValue(t *testing.T) {
	listener := newMockDataListener()

	// Default return value is true
	result := listener.DataChange(Event{})
	assert.True(t, result)

	// Change return value to false
	listener.setReturnValue(false)
	result = listener.DataChange(Event{})
	assert.False(t, result)
}

func TestDataListenerMultipleEvents(t *testing.T) {
	listener := newMockDataListener()

	events := []Event{
		{Path: "/path1", Action: EventTypeAdd, Content: "content1"},
		{Path: "/path2", Action: EventTypeDel, Content: ""},
		{Path: "/path3", Action: EventTypeUpdate, Content: "content3"},
	}

	for _, event := range events {
		listener.DataChange(event)
	}

	assert.Equal(t, 3, listener.getDataChangeCnt())
	receivedEvents := listener.getEvents()
	assert.Equal(t, events, receivedEvents)
}

// ============================================
// Concurrent Access Tests
// ============================================

func TestDataListenerConcurrentAccess(t *testing.T) {
	listener := newMockDataListener()
	var wg sync.WaitGroup

	// Concurrent DataChange calls
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			event := Event{
				Path:    "/concurrent/path",
				Action:  EventType(idx % 3),
				Content: "concurrent-content",
			}
			listener.DataChange(event)
		}(i)
	}

	wg.Wait()
	assert.Equal(t, 100, listener.getDataChangeCnt())
}

// ============================================
// Edge Cases Tests
// ============================================

func TestEventEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		validate func(*testing.T, Event)
	}{
		{
			name: "event with special characters in path",
			event: Event{
				Path:    "/dubbo/services/com.example.Service:1.0.0",
				Action:  EventTypeAdd,
				Content: "content",
			},
			validate: func(t *testing.T, e Event) {
				str := e.String()
				assert.Contains(t, str, "add")
			},
		},
		{
			name: "event with unicode content",
			event: Event{
				Path:    "/test",
				Action:  EventTypeUpdate,
				Content: "中文内容",
			},
			validate: func(t *testing.T, e Event) {
				str := e.String()
				assert.Contains(t, str, "中文内容")
			},
		},
		{
			name: "event with very long content",
			event: Event{
				Path:    "/test",
				Action:  EventTypeAdd,
				Content: string(make([]byte, 1000)),
			},
			validate: func(t *testing.T, e Event) {
				str := e.String()
				assert.Contains(t, str, "add")
			},
		},
		{
			name: "event with empty content",
			event: Event{
				Path:    "/test",
				Action:  EventTypeDel,
				Content: "",
			},
			validate: func(t *testing.T, e Event) {
				str := e.String()
				assert.Contains(t, str, "delete")
			},
		},
		{
			name: "event with newlines in content",
			event: Event{
				Path:    "/test",
				Action:  EventTypeUpdate,
				Content: "line1\nline2\nline3",
			},
			validate: func(t *testing.T, e Event) {
				assert.Contains(t, e.Content, "\n")
			},
		},
		{
			name: "event with tabs in path",
			event: Event{
				Path:    "/test\t/path",
				Action:  EventTypeAdd,
				Content: "content",
			},
			validate: func(t *testing.T, e Event) {
				assert.Contains(t, e.Path, "\t")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.event)
		})
	}
}

// ============================================
// DataListener Behavior Tests
// ============================================

func TestDataListenerBehavior(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *mockDataListener
		events   []Event
		validate func(*testing.T, *mockDataListener)
	}{
		{
			name: "single event",
			setup: func() *mockDataListener {
				return newMockDataListener()
			},
			events: []Event{
				{Path: "/path1", Action: EventTypeAdd, Content: "content1"},
			},
			validate: func(t *testing.T, l *mockDataListener) {
				assert.Equal(t, 1, l.getDataChangeCnt())
			},
		},
		{
			name: "multiple events same path",
			setup: func() *mockDataListener {
				return newMockDataListener()
			},
			events: []Event{
				{Path: "/path1", Action: EventTypeAdd, Content: "content1"},
				{Path: "/path1", Action: EventTypeUpdate, Content: "content2"},
				{Path: "/path1", Action: EventTypeDel, Content: ""},
			},
			validate: func(t *testing.T, l *mockDataListener) {
				assert.Equal(t, 3, l.getDataChangeCnt())
				events := l.getEvents()
				assert.Equal(t, EventTypeAdd, events[0].Action)
				assert.Equal(t, EventTypeUpdate, events[1].Action)
				assert.Equal(t, EventTypeDel, events[2].Action)
			},
		},
		{
			name: "listener returns false",
			setup: func() *mockDataListener {
				l := newMockDataListener()
				l.setReturnValue(false)
				return l
			},
			events: []Event{
				{Path: "/path1", Action: EventTypeAdd, Content: "content1"},
			},
			validate: func(t *testing.T, l *mockDataListener) {
				assert.Equal(t, 1, l.getDataChangeCnt())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener := tt.setup()
			for _, event := range tt.events {
				listener.DataChange(event)
			}
			tt.validate(t, listener)
		})
	}
}

// ============================================
// Event Comparison Tests
// ============================================

func TestEventEquality(t *testing.T) {
	event1 := Event{
		Path:    "/test/path",
		Action:  EventTypeAdd,
		Content: "content",
	}

	event2 := Event{
		Path:    "/test/path",
		Action:  EventTypeAdd,
		Content: "content",
	}

	event3 := Event{
		Path:    "/test/path",
		Action:  EventTypeDel, // Different action
		Content: "content",
	}

	assert.Equal(t, event1, event2)
	assert.NotEqual(t, event1, event3)
}

// ============================================
// EventType Iteration Tests
// ============================================

func TestEventTypeIteration(t *testing.T) {
	eventTypes := []EventType{EventTypeAdd, EventTypeDel, EventTypeUpdate}
	expectedStrings := []string{"add", "delete", "update"}

	for i, et := range eventTypes {
		assert.Equal(t, expectedStrings[i], et.String())
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkEventString(b *testing.B) {
	event := Event{
		Path:    "/dubbo/services/test",
		Action:  EventTypeAdd,
		Content: "test-content",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = event.String()
	}
}

func BenchmarkEventTypeString(b *testing.B) {
	et := EventTypeUpdate

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = et.String()
	}
}

func BenchmarkDataListenerDataChange(b *testing.B) {
	listener := newMockDataListener()
	event := Event{
		Path:    "/test/path",
		Action:  EventTypeAdd,
		Content: "content",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		listener.DataChange(event)
	}
}

func BenchmarkEventCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Event{
			Path:    "/test/path",
			Action:  EventTypeAdd,
			Content: "content",
		}
	}
}
