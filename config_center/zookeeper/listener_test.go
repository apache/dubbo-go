/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * the Apache License, Version 2.0
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

func TestCacheListenerDataChange(t *testing.T) {
	l := &CacheListener{rootPath: "/dubbo/config"}
	path := "/dubbo/config/group/app"
	rec := &recListener{}
	l.keyListeners.Store(path, map[config_center.ConfigurationListener]struct{}{rec: {}})

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
	l.keyListeners.Store(path, map[config_center.ConfigurationListener]struct{}{rec: {}})

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
	l.keyListeners.Store(key, map[config_center.ConfigurationListener]struct{}{rec: {}})
	l.RemoveListener(key, rec)
	if m, ok := l.keyListeners.Load(key); ok {
		if _, exists := m.(map[config_center.ConfigurationListener]struct{})[rec]; exists {
			t.Fatalf("listener should be removed")
		}
	}
}
