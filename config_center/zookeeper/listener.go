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
	// key is zkNode Path and value is set of listeners
	keyListeners    sync.Map
	eventGeneration sync.Map
	zkEventListener *zookeeper.ZkEventListener
	rootPath        string
	cache           *configCache
}

// NewCacheListener creates a new CacheListener
func NewCacheListener(rootPath string, listener *zookeeper.ZkEventListener) *CacheListener {
	return newCacheListener(rootPath, listener, nil)
}

func newCacheListener(rootPath string, listener *zookeeper.ZkEventListener, cache *configCache) *CacheListener {
	return &CacheListener{zkEventListener: listener, rootPath: rootPath, cache: cache}
}

// AddListener will add a listener if loaded
func (l *CacheListener) AddListener(key string, listener config_center.ConfigurationListener) {
	// FIXME do not use Client.ExistW, cause it has a bug(can not watch zk node that do not exist)
	register := func() error {
		_, _, _, err := l.zkEventListener.Client.Conn.ExistsW(key)
		return err
	}
	var err error
	if l.cache == nil {
		err = register()
	} else {
		err = l.cache.ensureWatch(key, register)
	}
	// reference from https://stackoverflow.com/questions/34018908/golang-why-dont-we-have-a-set-datastructure
	// make a map[your type]struct{} like set in java
	if err != nil {
		return
	}
	listeners, loaded := l.keyListeners.LoadOrStore(key, map[config_center.ConfigurationListener]struct{}{listener: {}})
	if loaded {
		listeners.(map[config_center.ConfigurationListener]struct{})[listener] = struct{}{}
		l.keyListeners.Store(key, listeners)
	}
}

// WatchStateChanged updates the cache's concrete-path watch state.
func (l *CacheListener) WatchStateChanged(path string, active bool) {
	if l.cache == nil {
		return
	}
	if !active {
		generation, _ := l.cache.snapshot(path)
		l.eventGeneration.Store(path, generation)
		l.cache.setWatchActiveAtGeneration(path, generation, false)
		return
	}
	if generation, ok := l.eventGeneration.Load(path); ok {
		l.cache.setWatchActiveAtGeneration(path, generation.(uint64), true)
		return
	}
	l.cache.setWatchActive(path, true)
}

// RemoveListener will delete a listener if loaded
func (l *CacheListener) RemoveListener(key string, listener config_center.ConfigurationListener) {
	listeners, loaded := l.keyListeners.Load(key)
	if loaded {
		delete(listeners.(map[config_center.ConfigurationListener]struct{}), listener)
	}
}

// DataChange changes all listeners' event
func (l *CacheListener) DataChange(event remoting.Event) bool {
	if l.cache != nil {
		entry := configCacheEntry{content: event.Content, exists: true}
		if event.Action == remoting.EventTypeDel {
			entry = configCacheEntry{exists: false}
		}
		if generation, ok := l.eventGeneration.LoadAndDelete(event.Path); ok {
			l.cache.storeAtGeneration(event.Path, generation.(uint64), entry)
		} else {
			l.cache.store(event.Path, entry)
		}
	}

	changeType := event.Action
	if event.Content == "" {
		changeType = remoting.EventTypeDel
	}

	key, group := l.pathToKeyGroup(event.Path)
	defer metrics.Publish(metricsConfigCenter.NewIncMetricEvent(key, group, changeType, metricsConfigCenter.Zookeeper))
	if listeners, ok := l.keyListeners.Load(event.Path); ok {
		for listener := range listeners.(map[config_center.ConfigurationListener]struct{}) {
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
	before, after, _ := strings.Cut(groupKey, constant.DotSeparator)
	return after, before
}
