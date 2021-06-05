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
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/observer"
)

func TestSetAndInitGlobalDispatcher(t *testing.T) {
	mock := &mockEventDispatcher{}
	SetEventDispatcher("mock", func() observer.EventDispatcher {
		return mock
	})

	SetAndInitGlobalDispatcher("mock")
	dispatcher := GetGlobalDispatcher()
	assert.NotNil(t, dispatcher)
	assert.Equal(t, mock, dispatcher)

	mock1 := &mockEventDispatcher{}

	SetEventDispatcher("mock1", func() observer.EventDispatcher {
		return mock1
	})

	SetAndInitGlobalDispatcher("mock1")
	dispatcher = GetGlobalDispatcher()
	assert.NotNil(t, dispatcher)
	assert.Equal(t, mock1, dispatcher)
}

func TestAddEventListener(t *testing.T) {
	AddEventListener(func() observer.EventListener {
		return &mockEventListener{}
	})

	AddEventListener(func() observer.EventListener {
		return &mockEventListener{}
	})

	assert.Equal(t, 2, len(initEventListeners))
}

type mockEventListener struct{}

func (m mockEventListener) GetPriority() int {
	panic("implement me")
}

func (m mockEventListener) OnEvent(e observer.Event) error {
	panic("implement me")
}

func (m mockEventListener) GetEventType() reflect.Type {
	panic("implement me")
}

type mockEventDispatcher struct{}

func (m mockEventDispatcher) AddEventListener(listener observer.EventListener) {
	panic("implement me")
}

func (m mockEventDispatcher) AddEventListeners(listenersSlice []observer.EventListener) {
	panic("implement me")
}

func (m mockEventDispatcher) RemoveEventListener(listener observer.EventListener) {
	panic("implement me")
}

func (m mockEventDispatcher) RemoveEventListeners(listenersSlice []observer.EventListener) {
	panic("implement me")
}

func (m mockEventDispatcher) GetAllEventListeners() []observer.EventListener {
	panic("implement me")
}

func (m mockEventDispatcher) RemoveAllEventListeners() {
	panic("implement me")
}

func (m mockEventDispatcher) Dispatch(event observer.Event) {
	panic("implement me")
}
