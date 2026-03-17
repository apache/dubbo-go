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

import "sync"

// Registry is a thread-safe generic container for extension registrations.
type Registry[T any] struct {
	mu    sync.RWMutex
	items map[string]T
	name  string
}

func NewRegistry[T any](name string) *Registry[T] {
	return &Registry[T]{
		items: make(map[string]T),
		name:  name,
	}
}

func (r *Registry[T]) Register(name string, v T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[name] = v
}

func (r *Registry[T]) Get(name string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.items[name]
	return v, ok
}

func (r *Registry[T]) MustGet(name string) T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.items[name]
	if !ok {
		panic(r.name + " for [" + name + "] is not existing, make sure you have import the package.")
	}
	return v
}

func (r *Registry[T]) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, name)
}

func (r *Registry[T]) Snapshot() map[string]T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := make(map[string]T, len(r.items))
	for k, v := range r.items {
		m[k] = v
	}
	return m
}

func (r *Registry[T]) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.items))
	for k := range r.items {
		names = append(names, k)
	}
	return names
}
