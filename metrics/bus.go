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

package metrics

import (
	"sync"
)

// eventListener is a struct that encapsulates the listener map and provides thread-safe access to it.
type eventListener struct {
	mu       sync.RWMutex
	listener map[string]chan MetricsEvent
}

var listener = &eventListener{
	listener: make(map[string]chan MetricsEvent),
}

// Publish publishes an event to all subscribers of the same type.
func Publish(event MetricsEvent) {
	listener.mu.RLock()
	defer listener.mu.RUnlock()

	if ch, ok := listener.listener[event.Type()]; ok {
		select {
		case ch <- event:
		default:
			// If the channel is full, drop the event to avoid blocking.
		}
	}
}

// Subscribe subscribes to events of the given type.
func Subscribe(typ string, ch chan MetricsEvent) {
	listener.mu.Lock()
	defer listener.mu.Unlock()

	listener.listener[typ] = ch
}

// Unsubscribe unsubscribes from events of the given type.
func Unsubscribe(typ string) {
	listener.mu.Lock()
	defer listener.mu.Unlock()

	if ch, ok := listener.listener[typ]; ok {
		close(ch)
		delete(listener.listener, typ)
	}
}
