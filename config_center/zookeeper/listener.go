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
	"github.com/go-zookeeper/zk"

	"github.com/dubbogo/gost/log/logger"
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

type watchEventState struct {
	generation uint64
	sessionID  int64
}

// NewCacheListener creates a new CacheListener
func NewCacheListener(rootPath string, listener *zookeeper.ZkEventListener) *CacheListener {
	return newCacheListener(rootPath, listener, nil)
}

func newCacheListener(rootPath string, listener *zookeeper.ZkEventListener, cache *configCache) *CacheListener {
	return &CacheListener{zkEventListener: listener, rootPath: rootPath, cache: cache}
}

func (l *CacheListener) registerWatcher(key string) (watchRegistration, error) {
	conn := l.zkEventListener.Client.Conn
	beforeSessionID := conn.SessionID()
	_, _, events, err := conn.ExistsW(key)
	return watchRegistration{
		events:          events,
		beforeSessionID: beforeSessionID,
		afterSessionID:  conn.SessionID(),
	}, err
}

// AddListener will add a listener if loaded
func (l *CacheListener) AddListener(key string, listener config_center.ConfigurationListener) {
	// FIXME do not use Client.ExistW, cause it has a bug(can not watch zk node that do not exist)
	register := func() (watchRegistration, error) {
		return l.registerWatcher(key)
	}
	var err error
	if l.cache == nil {
		_, err = register()
	} else {
		err = l.cache.ensureBusinessWatch(key, register)
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

func (l *CacheListener) restoreBusinessWatches() {
	if l.cache == nil || l.zkEventListener == nil ||
		l.zkEventListener.Client == nil || l.zkEventListener.Client.Conn == nil {
		return
	}

	l.keyListeners.Range(func(key, _ any) bool {
		path := key.(string)
		err := l.cache.ensureBusinessWatch(path, func() (watchRegistration, error) {
			return l.registerWatcher(path)
		})
		if err != nil {
			logger.Warnf("[ConfigCenter][Zookeeper] restore configuration watcher failed, path=%s err=%v", path, err)
			return true
		}
		if !l.hasListeners(path) {
			l.cache.releaseBusinessWatch(path)
		}
		return true
	})
}

// WatchStateChanged updates the cache's concrete-path watch state.
func (l *CacheListener) WatchStateChanged(path string) bool {
	if l.cache == nil {
		return true
	}
	generation, registerWatch := l.cache.beginWatchRenewal(path, l.hasListeners(path))
	_, watchState := l.cache.snapshot(path)
	l.eventGeneration.Store(path, watchEventState{
		generation: generation,
		sessionID:  watchState.sessionID,
	})
	return registerWatch
}

// WatchRegistered records a watch created while handling a configuration event.
func (l *CacheListener) WatchRegistered(path string, events <-chan zk.Event, beforeSessionID, afterSessionID int64) bool {
	if l.cache == nil {
		return true
	}
	state, ok := l.eventGeneration.Load(path)
	if !ok {
		return false
	}
	return l.cache.finishWatchRegistration(path, state.(watchEventState).generation, watchRegistration{
		events:          events,
		beforeSessionID: beforeSessionID,
		afterSessionID:  afterSessionID,
	})
}

func (l *CacheListener) WatchStateChangeFailed(path string) {
	if l.cache == nil {
		return
	}
	state, ok := l.eventGeneration.LoadAndDelete(path)
	if !ok {
		return
	}
	l.cache.cancelWatchRenewal(path)
	if !l.hasListeners(path) || l.zkEventListener == nil ||
		l.zkEventListener.Client == nil || l.zkEventListener.Client.Conn == nil {
		return
	}

	eventState := state.(watchEventState)
	conn := l.zkEventListener.Client.Conn
	if conn.State() != zk.StateHasSession || conn.SessionID() == eventState.sessionID {
		return
	}
	if err := l.cache.ensureBusinessWatch(path, func() (watchRegistration, error) {
		return l.registerWatcher(path)
	}); err != nil {
		logger.Warnf("[ConfigCenter][Zookeeper] retry configuration watcher failed, path=%s err=%v", path, err)
		return
	}
	if !l.hasListeners(path) {
		l.cache.releaseBusinessWatch(path)
	}
}

func (l *CacheListener) hasListeners(path string) bool {
	_, ok := l.keyListeners.Load(path)
	return ok
}

// RemoveListener will delete a listener if loaded
func (l *CacheListener) RemoveListener(key string, listener config_center.ConfigurationListener) {
	listeners, loaded := l.keyListeners.Load(key)
	if !loaded {
		return
	}
	listenerSet := listeners.(map[config_center.ConfigurationListener]struct{})
	delete(listenerSet, listener)
	if len(listenerSet) != 0 {
		return
	}
	l.keyListeners.Delete(key)
	if l.cache != nil {
		l.cache.releaseBusinessWatch(key)
	}
}

// DataChange changes all listeners' event
func (l *CacheListener) DataChange(event remoting.Event) bool {
	if l.cache != nil {
		entry := configCacheEntry{content: event.Content, exists: true}
		if event.Action == remoting.EventTypeDel {
			entry = configCacheEntry{exists: false}
		}
		if state, ok := l.eventGeneration.LoadAndDelete(event.Path); ok {
			l.cache.storeAtGeneration(event.Path, state.(watchEventState).generation, entry)
		} else {
			l.cache.store(event.Path, entry)
		}
	}

	key, group := l.pathToKeyGroup(event.Path)
	defer metrics.Publish(metricsConfigCenter.NewIncMetricEvent(key, group, event.Action, metricsConfigCenter.Zookeeper))
	if listeners, ok := l.keyListeners.Load(event.Path); ok {
		for listener := range listeners.(map[config_center.ConfigurationListener]struct{}) {
			listener.Process(&config_center.ConfigChangeEvent{
				Key:        key,
				Value:      event.Content,
				ConfigType: event.Action,
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
