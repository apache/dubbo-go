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

package dispatcher

import (
	"dubbo.apache.org/dubbo-go/v3/common/observer"
)

// MockEventDispatcher will do nothing.
// It is only used by tests
// Now the implementation doing nothing,
// But you can modify this if needed
type MockEventDispatcher struct {
	Notify chan struct{}
	Event  observer.Event
}

func NewMockEventDispatcher() *MockEventDispatcher {
	return &MockEventDispatcher{Notify: make(chan struct{}, 1)}
}

// AddEventListener do nothing
func (m *MockEventDispatcher) AddEventListener(listener observer.EventListener) {
}

// AddEventListeners do nothing
func (m *MockEventDispatcher) AddEventListeners(listenersSlice []observer.EventListener) {
}

// RemoveEventListener do nothing
func (m *MockEventDispatcher) RemoveEventListener(listener observer.EventListener) {
}

// RemoveEventListeners do nothing
func (m *MockEventDispatcher) RemoveEventListeners(listenersSlice []observer.EventListener) {
}

// GetAllEventListeners return empty list
func (m *MockEventDispatcher) GetAllEventListeners() []observer.EventListener {
	return make([]observer.EventListener, 0)
}

// RemoveAllEventListeners do nothing
func (m *MockEventDispatcher) RemoveAllEventListeners() {
}

// Dispatch do nothing
func (m *MockEventDispatcher) Dispatch(event observer.Event) {
	m.Event = event
	m.Notify <- struct{}{}
}
