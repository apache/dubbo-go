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

package apollo

import (
	"sync"
	"testing"
)

import (
	"github.com/apolloconfig/agollo/v4/storage"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type recordingListener struct {
	events []*config_center.ConfigChangeEvent
}

func (r *recordingListener) Process(e *config_center.ConfigChangeEvent) {
	r.events = append(r.events, e)
}

func TestApolloListener(t *testing.T) {
	l := newApolloListener()
	rec := &recordingListener{}

	// add/remove idempotent
	l.AddListener(rec)
	l.AddListener(rec)
	l.RemoveListener(rec)
	l.AddListener(rec)

	change := &storage.FullChangeEvent{
		Changes: map[string]any{"k": "v"},
	}
	change.Namespace = "application"
	l.OnNewestChange(change)

	if len(rec.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(rec.events))
	}
	ev := rec.events[0]
	if ev.Key != "application" || ev.ConfigType != remoting.EventTypeUpdate {
		t.Fatalf("unexpected event %+v", ev)
	}
	if l.IsEmpty() {
		t.Fatalf("listener should not be empty after add")
	}

	l.RemoveListener(rec)
	if !l.IsEmpty() {
		t.Fatalf("listener should be empty after remove")
	}
}

// safeRecordingListener is a ConfigurationListener safe for concurrent Process.
type safeRecordingListener struct {
	mu  sync.Mutex
	cnt int
}

func (s *safeRecordingListener) Process(*config_center.ConfigChangeEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cnt++
}

// TestApolloListenerConcurrency verifies the mutex-guarded listener set is
// race-free under concurrent OnNewestChange (agollo goroutine) vs Add/Remove
// (router goroutines). Run with -race. Regression for #3541 (unlocked map ->
// fatal concurrent map read+write).
func TestApolloListenerConcurrency(t *testing.T) {
	l := newApolloListener()
	a := &safeRecordingListener{}
	b := &safeRecordingListener{}
	change := &storage.FullChangeEvent{
		Changes: map[string]any{"k": "v"},
	}
	change.Namespace = "application"
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(4)
		go func() { defer wg.Done(); l.AddListener(a) }()
		go func() { defer wg.Done(); l.RemoveListener(a) }()
		go func() { defer wg.Done(); l.AddListener(b) }()
		go func() { defer wg.Done(); l.OnNewestChange(change) }()
	}
	wg.Wait()
	// no panic, no race; IsEmpty reads under the lock without racing writers.
	_ = l.IsEmpty()
}
