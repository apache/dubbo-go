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
	"strings"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsConfigCenter "dubbo.apache.org/dubbo-go/v3/metrics/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/zookeeper"
)

// CacheListener defines keyListeners and rootPath
type CacheListener struct {
	// key is zkNode Path and value is *listenerSet (mutex-guarded set of listeners)
	keyListeners    sync.Map
	zkEventListener *zookeeper.ZkEventListener
	rootPath        string
}

// NewCacheListener creates a new CacheListener
func NewCacheListener(rootPath string, listener *zookeeper.ZkEventListener) *CacheListener {
	return &CacheListener{zkEventListener: listener, rootPath: rootPath}
}

// listenerSet is a mutex-guarded set of ConfigurationListeners. AddListener and
// RemoveListener run on router/config goroutines while DataChange runs on the
// zk event goroutine; guarding the inner map avoids a fatal concurrent map
// read+write. See #3536.
type listenerSet struct {
	mu        sync.Mutex
	listeners map[config_center.ConfigurationListener]struct{}
}

func newListenerSet() *listenerSet {
	return &listenerSet{listeners: make(map[config_center.ConfigurationListener]struct{})}
}

func (s *listenerSet) add(l config_center.ConfigurationListener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners[l] = struct{}{}
}

func (s *listenerSet) remove(l config_center.ConfigurationListener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.listeners, l)
}

// snapshot returns a slice copy of the listeners under the lock, safe to
// iterate outside the lock so listener.Process is not called while holding it.
func (s *listenerSet) snapshot() []config_center.ConfigurationListener {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]config_center.ConfigurationListener, 0, len(s.listeners))
	for l := range s.listeners {
		out = append(out, l)
	}
	return out
}

// AddListener will add a listener if loaded
func (l *CacheListener) AddListener(key string, listener config_center.ConfigurationListener) {
	// FIXME do not use Client.ExistW, cause it has a bug(can not watch zk node that do not exist)
	_, _, _, err := l.zkEventListener.Client.Conn.ExistsW(key)
	if err != nil {
		return
	}
	actual, _ := l.keyListeners.LoadOrStore(key, newListenerSet())
	actual.(*listenerSet).add(listener)
}

// RemoveListener will delete a listener if loaded
func (l *CacheListener) RemoveListener(key string, listener config_center.ConfigurationListener) {
	if listeners, ok := l.keyListeners.Load(key); ok {
		listeners.(*listenerSet).remove(listener)
	}
}

// DataChange changes all listeners' event
func (l *CacheListener) DataChange(event remoting.Event) bool {
	changeType := event.Action
	if event.Content == "" {
		changeType = remoting.EventTypeDel
	}

	key, group := l.pathToKeyGroup(event.Path)
	defer metrics.Publish(metricsConfigCenter.NewIncMetricEvent(key, group, changeType, metricsConfigCenter.Zookeeper))
	if listeners, ok := l.keyListeners.Load(event.Path); ok {
		for _, listener := range listeners.(*listenerSet).snapshot() {
			listener.Process(&config_center.ConfigChangeEvent{
				Key:        key,
				Value:      event.Content,
				ConfigType: changeType,
			})
		}
		return true
	}
	return false
}

func (l *CacheListener) pathToKeyGroup(path string) (string, string) {
	if len(path) == 0 {
		return path, ""
	}
	groupKey := strings.ReplaceAll(strings.ReplaceAll(path, l.rootPath+constant.PathSeparator, ""), constant.PathSeparator, constant.DotSeparator)
	index := strings.Index(groupKey, constant.DotSeparator)
	return groupKey[index+1:], groupKey[0:index]
}
