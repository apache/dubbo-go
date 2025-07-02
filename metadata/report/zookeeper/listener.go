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
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/zookeeper"
)

// ListenerSet defines a thread-safe set of listeners
type ListenerSet struct {
	sync.RWMutex
	listeners map[mapping.MappingListener]struct{}
}

func NewListenerSet() *ListenerSet {
	return &ListenerSet{
		listeners: make(map[mapping.MappingListener]struct{}),
	}
}

// Add adds a listener to the set
func (s *ListenerSet) Add(listener mapping.MappingListener) {
	s.Lock()
	defer s.Unlock()
	s.listeners[listener] = struct{}{}
}

// Remove removes a listener from the set
func (s *ListenerSet) Remove(listener mapping.MappingListener) {
	s.Lock()
	defer s.Unlock()
	delete(s.listeners, listener)
}

// Has checks if a listener exists in the set
func (s *ListenerSet) Has(listener mapping.MappingListener) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.listeners[listener]
	return ok
}

// ForEach iterates over all listeners in the set
func (s *ListenerSet) ForEach(f func(mapping.MappingListener) error) error {
	s.RLock()
	defer s.RUnlock()
	for listener := range s.listeners {
		if err := f(listener); err != nil {
			return err
		}
	}
	return nil
}

// CacheListener defines keyListeners and rootPath
type CacheListener struct {
	// key is zkNode Path and value is set of listeners
	keyListeners    sync.Map
	zkEventListener *zookeeper.ZkEventListener
	rootPath        string
}

// NewCacheListener creates a new CacheListener
func NewCacheListener(rootPath string, listener *zookeeper.ZkEventListener) *CacheListener {
	return &CacheListener{zkEventListener: listener, rootPath: rootPath}
}

// AddListener will add a listener if loaded
func (l *CacheListener) AddListener(key string, listener mapping.MappingListener) {
	// FIXME do not use Client.ExistW, cause it has a bug(can not watch zk node that do not exist)
	_, _, _, err := l.zkEventListener.Client.Conn.ExistsW(key)
	// reference from https://stackoverflow.com/questions/34018908/golang-why-dont-we-have-a-set-datastructure
	// make a map[your type]struct{} like set in java
	if err != nil {
		return
	}
	// try to store the new set. If key exists, add listener to existing set
	listeners, _ := l.keyListeners.LoadOrStore(key, NewListenerSet())
	listeners.(*ListenerSet).Add(listener)
}

// RemoveListener will delete a listener if loaded
func (l *CacheListener) RemoveListener(key string, listener mapping.MappingListener) {
	listeners, loaded := l.keyListeners.Load(key)
	if loaded {
		listeners.(*ListenerSet).Remove(listener)
	}
}

// DataChange changes all listeners' event
func (l *CacheListener) DataChange(event remoting.Event) bool {
	if listeners, ok := l.keyListeners.Load(event.Path); ok {
		appNames := strings.Split(event.Content, constant.CommaSeparator)
		set := gxset.NewSet()
		for _, e := range appNames {
			set.Add(e)
		}
		err := listeners.(*ListenerSet).ForEach(func(listener mapping.MappingListener) error {
			return listener.OnEvent(registry.NewServiceMappingChangedEvent(l.pathToKey(event.Path), set))
		})
		if err != nil {
			logger.Error("Error notify mapping change event.", err)
			return false
		}
		return true
	}
	return false
}

func (l *CacheListener) pathToKey(path string) string {
	if len(path) == 0 {
		return path
	}
	groupKey := strings.ReplaceAll(strings.ReplaceAll(path, l.rootPath+constant.PathSeparator, ""), constant.PathSeparator, constant.DotSeparator)
	return groupKey[strings.Index(groupKey, constant.DotSeparator)+1:]
}
