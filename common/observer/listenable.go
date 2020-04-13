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

package observer

import (
	"reflect"
	"sort"
	"sync"
)

// Listenable could add and remove the event listener
type Listenable interface {
	AddEventListener(listener EventListener)
	AddEventListeners(listenersSlice []EventListener)
	RemoveEventListener(listener EventListener)
	RemoveEventListeners(listenersSlice []EventListener)
	GetAllEventListeners() []EventListener
	RemoveAllEventListeners()
}

// BaseListenable base listenable
type BaseListenable struct {
	Listenable
	ListenersCache sync.Map
	Mutex          sync.Mutex
}

// NewBaseListenable a constructor of base listenable
func NewBaseListenable() Listenable {
	return &BaseListenable{}
}

// AddEventListener add event listener
func (bl *BaseListenable) AddEventListener(listener EventListener) {
	eventType := listener.GetEventType()
	if eventType.Kind() == reflect.Ptr {
		eventType = eventType.Elem()
	}
	bl.Mutex.Lock()
	defer bl.Mutex.Unlock()
	value, loaded := bl.ListenersCache.LoadOrStore(eventType, make([]EventListener, 0, 8))
	listenersSlice := value.([]EventListener)
	// return if listenersSlice already has this listener
	if loaded && containListener(listenersSlice, listener) {
		return
	}
	listenersSlice = append(listenersSlice, listener)
	sort.Slice(listenersSlice, func(i, j int) bool {
		return listenersSlice[i].GetPriority() < listenersSlice[j].GetPriority()
	})
	bl.ListenersCache.Store(eventType, listenersSlice)
}

// AddEventListeners add the slice of event listener
func (bl *BaseListenable) AddEventListeners(listenersSlice []EventListener) {
	for _, listener := range listenersSlice {
		bl.AddEventListener(listener)
	}
}

// RemoveEventListener remove the event listener
func (bl *BaseListenable) RemoveEventListener(listener EventListener) {
	eventType := listener.GetEventType()
	if eventType.Kind() == reflect.Ptr {
		eventType = eventType.Elem()
	}
	bl.Mutex.Lock()
	defer bl.Mutex.Unlock()
	value, loaded := bl.ListenersCache.Load(eventType)
	if !loaded {
		return
	}
	listenersSlice := value.([]EventListener)
	for i, l := range listenersSlice {
		if l == listener {
			listenersSlice = append(listenersSlice[:i], listenersSlice[i+1:]...)
		}
	}
	bl.ListenersCache.Store(eventType, listenersSlice)
}

// RemoveEventListeners remove the slice of event listener
func (bl *BaseListenable) RemoveEventListeners(listenersSlice []EventListener) {
	for _, listener := range listenersSlice {
		bl.RemoveEventListener(listener)
	}
}

// RemoveAllEventListeners remove all
func (bl *BaseListenable) RemoveAllEventListeners() {
	bl.Mutex.Lock()
	defer bl.Mutex.Unlock()
	bl.ListenersCache = sync.Map{}
}

// GetAllEventListeners get all
func (bl *BaseListenable) GetAllEventListeners() []EventListener {
	allListenersSlice := make([]EventListener, 0, 16)
	bl.ListenersCache.Range(func(_, value interface{}) bool {
		listenersSlice := value.([]EventListener)
		allListenersSlice = append(allListenersSlice, listenersSlice...)
		return true
	})
	sort.Slice(allListenersSlice, func(i, j int) bool {
		return allListenersSlice[i].GetPriority() < allListenersSlice[j].GetPriority()
	})
	return allListenersSlice
}

// containListener true if contain listener
func containListener(listenersSlice []EventListener, listener EventListener) bool {
	for _, loadListener := range listenersSlice {
		if loadListener == listener {
			return true
		}
	}
	return false
}
