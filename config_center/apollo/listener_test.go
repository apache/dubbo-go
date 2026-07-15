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
	"sync/atomic"
	"testing"
	"time"
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

type countingListener struct {
	count atomic.Int64
}

func (c *countingListener) Process(*config_center.ConfigChangeEvent) {
	c.count.Add(1)
}

type callbackListener struct {
	process func(*config_center.ConfigChangeEvent)
}

func (c *callbackListener) Process(event *config_center.ConfigChangeEvent) {
	c.process(event)
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

func TestApolloListenerConcurrentAccess(t *testing.T) {
	listener := newApolloListener()
	first := &countingListener{}
	second := &countingListener{}
	change := testFullChangeEvent()

	const iterations = 200
	var waitGroup sync.WaitGroup
	waitGroup.Add(4)
	go toggleListenerMembership(listener, first, iterations, &waitGroup)
	go toggleListenerMembership(listener, second, iterations, &waitGroup)
	go dispatchListenerChanges(listener, change, iterations, &waitGroup)
	go observeListenerEmpty(listener, iterations, &waitGroup)
	waitGroup.Wait()

	firstBefore := first.count.Load()
	secondBefore := second.count.Load()
	listener.AddListener(first)
	listener.AddListener(second)
	listener.OnNewestChange(change)
	if got := first.count.Load(); got != firstBefore+1 {
		t.Fatalf("first listener count = %d, want %d", got, firstBefore+1)
	}
	if got := second.count.Load(); got != secondBefore+1 {
		t.Fatalf("second listener count = %d, want %d", got, secondBefore+1)
	}
	listener.RemoveListener(first)
	listener.RemoveListener(second)
	if !listener.IsEmpty() {
		t.Fatal("listener should be empty after cleanup")
	}
}

func TestApolloListenerAllowsReentrantRemoval(t *testing.T) {
	listener := newApolloListener()
	var self *callbackListener
	self = &callbackListener{process: func(*config_center.ConfigChangeEvent) {
		listener.RemoveListener(self)
	}}
	listener.AddListener(self)

	returned := make(chan struct{})
	go func() {
		listener.OnNewestChange(testFullChangeEvent())
		close(returned)
	}()
	waitForSignal(t, returned)
	if !listener.IsEmpty() {
		t.Fatal("listener should be empty after removing itself")
	}
}

func toggleListenerMembership(listener *apolloListener, target config_center.ConfigurationListener, iterations int, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for range iterations {
		listener.AddListener(target)
		listener.RemoveListener(target)
	}
}

func dispatchListenerChanges(listener *apolloListener, change *storage.FullChangeEvent, iterations int, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for range iterations {
		listener.OnNewestChange(change)
	}
}

func observeListenerEmpty(listener *apolloListener, iterations int, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for range iterations {
		listener.IsEmpty()
	}
}

func testFullChangeEvent() *storage.FullChangeEvent {
	change := &storage.FullChangeEvent{
		Changes: map[string]any{"k": "v"},
	}
	change.Namespace = "application"
	return change
}

func waitForSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for listener operation")
	}
}
