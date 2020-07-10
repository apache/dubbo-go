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

// BaseListener base listenable
type BaseListener struct {
	Listenable
	ListenersCache map[reflect.Type][]EventListener
	Mutex          sync.RWMutex
}

// NewBaseListener a constructor of base listenable
func NewBaseListener() BaseListener {
	return BaseListener{
		ListenersCache: make(map[reflect.Type][]EventListener, 8),
	}
}

// AddEventListener add event listener
func (bl *BaseListener) AddEventListener(listener EventListener) {
	eventType := listener.GetEventType()
	if eventType.Kind() == reflect.Ptr {
		eventType = eventType.Elem()
	}
	bl.Mutex.Lock()
	defer bl.Mutex.Unlock()
	listenersSlice, loaded := bl.ListenersCache[eventType]
	if !loaded {
		listenersSlice = make([]EventListener, 0, 8)
	}
	// return if listenersSlice already has this listener
	if loaded && containListener(listenersSlice, listener) {
		return
	}
	listenersSlice = append(listenersSlice, listener)
	sort.Slice(listenersSlice, func(i, j int) bool {
		return listenersSlice[i].GetPriority() < listenersSlice[j].GetPriority()
	})
	bl.ListenersCache[eventType] = listenersSlice
}

// AddEventListeners add the slice of event listener
func (bl *BaseListener) AddEventListeners(listenersSlice []EventListener) {
	for _, listener := range listenersSlice {
		bl.AddEventListener(listener)
	}
}

// RemoveEventListener remove the event listener
func (bl *BaseListener) RemoveEventListener(listener EventListener) {
	eventType := listener.GetEventType()
	if eventType.Kind() == reflect.Ptr {
		eventType = eventType.Elem()
	}
	bl.Mutex.Lock()
	defer bl.Mutex.Unlock()
	listenersSlice, loaded := bl.ListenersCache[eventType]
	if !loaded {
		return
	}
	for i, l := range listenersSlice {
		if l == listener {
			listenersSlice = append(listenersSlice[:i], listenersSlice[i+1:]...)
		}
	}
	bl.ListenersCache[eventType] = listenersSlice
}

// RemoveEventListeners remove the slice of event listener
// it will iterate all listener and remove it one by one
func (bl *BaseListener) RemoveEventListeners(listenersSlice []EventListener) {
	for _, listener := range listenersSlice {
		bl.RemoveEventListener(listener)
	}
}

// RemoveAllEventListeners remove all
// using Lock
func (bl *BaseListener) RemoveAllEventListeners() {
	bl.Mutex.Lock()
	defer bl.Mutex.Unlock()
	bl.ListenersCache = make(map[reflect.Type][]EventListener, 8)
}

// GetAllEventListeners get all listener
// using RLock
func (bl *BaseListener) GetAllEventListeners() []EventListener {
	allListenersSlice := make([]EventListener, 0, 16)

	bl.Mutex.RLock()
	defer bl.Mutex.RUnlock()
	for _, listenersSlice := range bl.ListenersCache {
		allListenersSlice = append(allListenersSlice, listenersSlice...)
	}
	sort.Slice(allListenersSlice, func(i, j int) bool {
		return allListenersSlice[i].GetPriority() < allListenersSlice[j].GetPriority()
	})
	return allListenersSlice
}

// containListener true if contain listener
// it's not thread safe
// usually it should be use in lock scope
func containListener(listenersSlice []EventListener, listener EventListener) bool {
	for _, loadListener := range listenersSlice {
		if loadListener == listener {
			return true
		}
	}
	return false
}
