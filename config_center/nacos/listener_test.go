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

package nacos

import (
	"sync"
	"testing"
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

func TestCallback(t *testing.T) {
	l := &recordingListener{}
	var m sync.Map
	m.Store(l, struct{}{})

	callback(&m, "", "g", "data", "payload")

	if len(l.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(l.events))
	}
	if l.events[0].Key != "data" || l.events[0].Value != "payload" || l.events[0].ConfigType != remoting.EventTypeUpdate {
		t.Fatalf("unexpected event %+v", l.events[0])
	}
}

func TestRemoveListener(t *testing.T) {
	n := &nacosDynamicConfiguration{}
	key := "k"
	l := &recordingListener{}
	inner := &sync.Map{}
	inner.Store(l, struct{}{})
	n.keyListeners.Store(key, inner)

	n.removeListener(key, l)

	if _, ok := inner.Load(l); ok {
		t.Fatalf("listener should be removed")
	}
}
