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

package graceful_shutdown

import (
	"reflect"
	"sync"

	clusterdirectory "dubbo.apache.org/dubbo-go/v3/cluster/directory"
)

type closingDirectoryRegistry struct {
	mu       sync.RWMutex
	removers map[string]map[uintptr]clusterdirectory.ClosingInstanceRemover
}

type closingEventHandler struct {
	registry ClosingDirectoryRegistry
}

var (
	defaultClosingDirectoryRegistry ClosingDirectoryRegistry = newClosingDirectoryRegistry()
	defaultClosingEventHandler      ClosingEventHandler      = &closingEventHandler{registry: defaultClosingDirectoryRegistry}
)

func newClosingDirectoryRegistry() ClosingDirectoryRegistry {
	return &closingDirectoryRegistry{
		removers: make(map[string]map[uintptr]clusterdirectory.ClosingInstanceRemover),
	}
}

func removerKey(remover clusterdirectory.ClosingInstanceRemover) uintptr {
	if remover == nil {
		return 0
	}
	value := reflect.ValueOf(remover)
	if value.Kind() != reflect.Pointer {
		return 0
	}
	return value.Pointer()
}

// DefaultClosingDirectoryRegistry returns the process-wide closing directory registry.
func DefaultClosingDirectoryRegistry() ClosingDirectoryRegistry {
	return defaultClosingDirectoryRegistry
}

// DefaultClosingEventHandler returns the default closing event handler.
func DefaultClosingEventHandler() ClosingEventHandler {
	return defaultClosingEventHandler
}

func (r *closingDirectoryRegistry) Register(serviceKey string, remover clusterdirectory.ClosingInstanceRemover) {
	key := removerKey(remover)
	if serviceKey == "" || key == 0 {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.removers[serviceKey]; !ok {
		r.removers[serviceKey] = make(map[uintptr]clusterdirectory.ClosingInstanceRemover)
	}
	r.removers[serviceKey][key] = remover
}

func (r *closingDirectoryRegistry) Unregister(serviceKey string, remover clusterdirectory.ClosingInstanceRemover) {
	key := removerKey(remover)
	if serviceKey == "" || key == 0 {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entries, ok := r.removers[serviceKey]
	if !ok {
		return
	}
	delete(entries, key)
	if len(entries) == 0 {
		delete(r.removers, serviceKey)
	}
}

func (r *closingDirectoryRegistry) Find(serviceKey string) []clusterdirectory.ClosingInstanceRemover {
	if serviceKey == "" {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, ok := r.removers[serviceKey]
	if !ok {
		return nil
	}

	result := make([]clusterdirectory.ClosingInstanceRemover, 0, len(entries))
	for _, remover := range entries {
		result = append(result, remover)
	}
	return result
}

func (h *closingEventHandler) HandleClosingEvent(event ClosingEvent) bool {
	if h.registry == nil || event.InstanceKey == "" || event.ServiceKey == "" {
		return false
	}

	removed := false
	for _, remover := range h.registry.Find(event.ServiceKey) {
		if remover.RemoveClosingInstance(event.InstanceKey) {
			removed = true
		}
	}
	return removed
}
