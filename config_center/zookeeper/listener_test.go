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
	"sync"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type recListener struct {
	events []*config_center.ConfigChangeEvent
}

func (r *recListener) Process(e *config_center.ConfigChangeEvent) {
	r.events = append(r.events, e)
}

// safeCountingListener is a ConfigurationListener safe for concurrent Process.
type safeCountingListener struct {
	mu  sync.Mutex
	cnt int
}

func (s *safeCountingListener) Process(*config_center.ConfigChangeEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cnt++
}

func TestCacheListenerDataChange(t *testing.T) {
	l := &CacheListener{rootPath: "/dubbo/config"}
	path := "/dubbo/config/group/app"
	rec := &recListener{}
	set := newListenerSet()
	set.add(rec)
	l.keyListeners.Store(path, set)

	ok := l.DataChange(remoting.Event{Path: path, Action: remoting.EventTypeUpdate, Content: "val"})
	if !ok {
		t.Fatalf("expected listeners to be notified")
	}
	if len(rec.events) != 1 || rec.events[0].Value != "val" || rec.events[0].ConfigType != remoting.EventTypeUpdate {
		t.Fatalf("unexpected events %+v", rec.events)
	}
}

func TestCacheListenerDataChangeEmptyContent(t *testing.T) {
	l := &CacheListener{rootPath: "/dubbo/config"}
	path := "/dubbo/config/group/app"
	rec := &recListener{}
	set := newListenerSet()
	set.add(rec)
	l.keyListeners.Store(path, set)

	ok := l.DataChange(remoting.Event{Path: path, Action: remoting.EventTypeAdd})
	if !ok {
		t.Fatalf("expected listeners to be notified")
	}
	if len(rec.events) != 1 || rec.events[0].ConfigType != remoting.EventTypeDel {
		t.Fatalf("unexpected events %+v", rec.events)
	}
}

func TestCacheListenerPathToKeyGroup(t *testing.T) {
	l := &CacheListener{rootPath: "/dubbo/config"}
	key, group := l.pathToKeyGroup("/dubbo/config/g/app")
	if key != "app" || group != "g" {
		t.Fatalf("unexpected key/group %s %s", key, group)
	}
}

func TestCacheListenerRemoveListener(t *testing.T) {
	l := &CacheListener{}
	key := "k"
	rec := &recListener{}
	set := newListenerSet()
	set.add(rec)
	l.keyListeners.Store(key, set)
	l.RemoveListener(key, rec)
	if s, ok := l.keyListeners.Load(key); ok {
		if got := len(s.(*listenerSet).snapshot()); got != 0 {
			t.Fatalf("listener should be removed, got %d", got)
		}
	}
}

// TestListenerSetConcurrency verifies the mutex-guarded listenerSet is race-free
// under concurrent add/remove/snapshot (DataChange's read path). Run with -race.
// Regression for #3536 (inner map had no lock -> fatal concurrent map read+write).
func TestListenerSetConcurrency(t *testing.T) {
	s := newListenerSet()
	a := &safeCountingListener{}
	b := &safeCountingListener{}
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(4)
		go func() { defer wg.Done(); s.add(a) }()
		go func() { defer wg.Done(); s.remove(a) }()
		go func() { defer wg.Done(); s.add(b) }()
		go func() { defer wg.Done(); _ = s.snapshot() }()
	}
	wg.Wait()
	// snapshot must only ever reference the registered listeners and not panic.
	for _, l := range s.snapshot() {
		if l != a && l != b {
			t.Fatalf("unexpected listener %v", l)
		}
	}
}
