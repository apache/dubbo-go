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
	"github.com/dubbogo/gost/log/logger"

	"github.com/go-zookeeper/zk"
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
	auto       bool
	token      uint64
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
	l.addListenerWithRegister(key, listener, func() (watchRegistration, error) {
		return l.registerWatcher(key)
	})
}

func (l *CacheListener) addListenerWithRegister(
	key string,
	listener config_center.ConfigurationListener,
	register func() (watchRegistration, error),
) {
	if l.cache == nil {
		if _, err := register(); err != nil {
			return
		}
		l.storeListener(key, listener)
		return
	}

	pathLock := l.cache.pathLock(key)
	pathLock.Lock()
	added := l.storeListener(key, listener)
	pathLock.Unlock()
	stillNeeded := func() bool {
		pathLock.Lock()
		defer pathLock.Unlock()
		return l.hasListener(key, listener)
	}
	err := l.cache.ensureBusinessWatchWithRetryIf(key, register, 1, stillNeeded)
	pathLock.Lock()
	defer pathLock.Unlock()
	present := l.hasListener(key, listener)
	if err != nil && added && present {
		_, last := l.removeListener(key, listener)
		if last {
			l.cache.releaseBusinessWatchLocked(key)
		}
		return
	}
	if !present && !l.hasListeners(key) {
		l.cache.releaseBusinessWatchLocked(key)
	}
}

func (l *CacheListener) storeListener(key string, listener config_center.ConfigurationListener) bool {
	// reference from https://stackoverflow.com/questions/34018908/golang-why-dont-we-have-a-set-datastructure
	// make a map[your type]struct{} like set in java
	listeners, loaded := l.keyListeners.LoadOrStore(key, map[config_center.ConfigurationListener]struct{}{listener: {}})
	if !loaded {
		return true
	}
	listenerSet := listeners.(map[config_center.ConfigurationListener]struct{})
	if _, exists := listenerSet[listener]; exists {
		return false
	}
	listenerSet[listener] = struct{}{}
	l.keyListeners.Store(key, listenerSet)
	return true
}

func (l *CacheListener) restoreBusinessWatches() {
	if l.cache == nil || l.zkEventListener == nil ||
		l.zkEventListener.Client == nil || l.zkEventListener.Client.Conn == nil {
		return
	}

	l.keyListeners.Range(func(key, _ any) bool {
		path := key.(string)
		err := l.cache.ensureBusinessWatchWithRetry(path, func() (watchRegistration, error) {
			return l.registerWatcher(path)
		}, 1)
		if err != nil {
			logger.Warnf("[ConfigCenter][Zookeeper] restore configuration watcher failed, path=%s err=%v", path, err)
			return true
		}
		pathLock := l.cache.pathLock(path)
		pathLock.Lock()
		if !l.hasListeners(path) {
			l.cache.releaseBusinessWatchLocked(path)
		}
		pathLock.Unlock()
		return true
	})
}

// WatchStateChanged updates the cache's concrete-path watch state.
func (l *CacheListener) WatchStateChanged(path string) bool {
	if l.cache == nil {
		return true
	}
	pathLock := l.cache.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()
	generation, watchState, registerWatch := l.cache.beginWatchRenewalLocked(path, l.hasListeners(path))
	l.eventGeneration.Store(path, watchEventState{
		generation: generation,
		sessionID:  watchState.sessionID,
		auto:       watchState.auto,
		token:      watchState.pendingOpToken(),
	})
	return registerWatch
}

// WatchRegistered records a watch created while handling a configuration event.
func (l *CacheListener) WatchRegistered(path string, events <-chan zk.Event, beforeSessionID, afterSessionID int64) zookeeper.WatchRegistrationResult {
	if l.cache == nil {
		return zookeeper.WatchRegistrationAccepted
	}
	pathLock := l.cache.pathLock(path)
	pathLock.Lock()
	state, ok := l.eventGeneration.Load(path)
	if !ok {
		pathLock.Unlock()
		return zookeeper.WatchRegistrationDiscarded
	}
	eventState := state.(watchEventState)
	registration := watchRegistration{
		events:          events,
		beforeSessionID: beforeSessionID,
		afterSessionID:  afterSessionID,
	}
	stored := l.cache.finishWatchRegistrationLocked(path, eventState.generation, eventState.token, registration)
	currentGeneration := l.cache.isCurrentGeneration(eventState.generation)
	if stored && registration.sessionStable() && currentGeneration {
		pathLock.Unlock()
		return zookeeper.WatchRegistrationAccepted
	}
	reload := eventState.auto || (stored && !currentGeneration)
	if reload {
		l.eventGeneration.Store(path, eventState)
	} else {
		l.eventGeneration.Delete(path)
	}
	pathLock.Unlock()

	if reload {
		return zookeeper.WatchRegistrationReload
	}
	if l.retryBusinessWatch(path, eventState.sessionID, eventState.generation) {
		pathLock.Lock()
		if _, ok := l.eventGeneration.Load(path); !ok {
			l.eventGeneration.Store(path, eventState)
		}
		pathLock.Unlock()
		return zookeeper.WatchRegistrationReload
	}
	return zookeeper.WatchRegistrationDiscarded
}

func (l *CacheListener) WatchStateChangeFailed(path string) {
	if l.cache == nil {
		return
	}
	pathLock := l.cache.pathLock(path)
	pathLock.Lock()
	state, ok := l.eventGeneration.LoadAndDelete(path)
	if !ok {
		pathLock.Unlock()
		return
	}
	l.cache.cancelPendingLocked(path)
	pathLock.Unlock()
	eventState := state.(watchEventState)
	l.retryBusinessWatch(path, eventState.sessionID, eventState.generation)
}

func (l *CacheListener) retryBusinessWatch(path string, previousSessionID int64, previousGeneration uint64) bool {
	pathLock := l.cache.pathLock(path)
	pathLock.Lock()
	hasListeners := l.hasListeners(path)
	pathLock.Unlock()
	if !hasListeners || l.zkEventListener == nil ||
		l.zkEventListener.Client == nil || l.zkEventListener.Client.Conn == nil {
		return false
	}

	conn := l.zkEventListener.Client.Conn
	currentGeneration, _ := l.cache.snapshot(path)
	if conn.State() != zk.StateHasSession ||
		(conn.SessionID() == previousSessionID && currentGeneration == previousGeneration) {
		return false
	}
	if err := l.cache.ensureBusinessWatchWithRetry(path, func() (watchRegistration, error) {
		return l.registerWatcher(path)
	}, 1); err != nil {
		logger.Warnf("[ConfigCenter][Zookeeper] retry configuration watcher failed, path=%s err=%v", path, err)
		return false
	}
	pathLock = l.cache.pathLock(path)
	pathLock.Lock()
	defer pathLock.Unlock()
	if !l.hasListeners(path) {
		l.cache.releaseBusinessWatchLocked(path)
		return false
	}
	return true
}

func (l *CacheListener) hasListeners(path string) bool {
	_, ok := l.keyListeners.Load(path)
	return ok
}

func (l *CacheListener) hasListener(path string, listener config_center.ConfigurationListener) bool {
	listeners, ok := l.keyListeners.Load(path)
	if !ok {
		return false
	}
	_, ok = listeners.(map[config_center.ConfigurationListener]struct{})[listener]
	return ok
}

// RemoveListener will delete a listener if loaded
func (l *CacheListener) RemoveListener(key string, listener config_center.ConfigurationListener) {
	if l.cache == nil {
		l.removeListener(key, listener)
		return
	}

	pathLock := l.cache.pathLock(key)
	pathLock.Lock()
	defer pathLock.Unlock()
	removed, last := l.removeListener(key, listener)
	if removed && last {
		l.cache.releaseBusinessWatchLocked(key)
	}
}

func (l *CacheListener) removeListener(key string, listener config_center.ConfigurationListener) (bool, bool) {
	listeners, loaded := l.keyListeners.Load(key)
	if !loaded {
		return false, false
	}
	listenerSet := listeners.(map[config_center.ConfigurationListener]struct{})
	if _, exists := listenerSet[listener]; !exists {
		return false, false
	}
	delete(listenerSet, listener)
	if len(listenerSet) != 0 {
		l.keyListeners.Store(key, listenerSet)
		return true, false
	}
	l.keyListeners.Delete(key)
	return true, true
}

// DataChange changes all listeners' event
func (l *CacheListener) DataChange(event remoting.Event) bool {
	if l.cache != nil {
		entry := configCacheEntry{content: event.Content, exists: true}
		if event.Action == remoting.EventTypeDel {
			entry = configCacheEntry{exists: false}
		}
		pathLock := l.cache.pathLock(event.Path)
		pathLock.Lock()
		if state, ok := l.eventGeneration.LoadAndDelete(event.Path); ok {
			l.cache.storeAtGenerationLocked(event.Path, state.(watchEventState).generation, entry)
		} else {
			l.cache.storeLocked(event.Path, entry)
		}
		pathLock.Unlock()
	}

	key, group := l.pathToKeyGroup(event.Path)
	defer metrics.Publish(metricsConfigCenter.NewIncMetricEvent(key, group, event.Action, metricsConfigCenter.Zookeeper))
	listeners := l.snapshotListeners(event.Path)
	for _, listener := range listeners {
		listener.Process(&config_center.ConfigChangeEvent{
			Key:        key,
			Value:      event.Content,
			ConfigType: event.Action,
		})
	}
	return len(listeners) != 0
}

func (l *CacheListener) snapshotListeners(path string) []config_center.ConfigurationListener {
	if l.cache != nil {
		pathLock := l.cache.pathLock(path)
		pathLock.Lock()
		defer pathLock.Unlock()
	}

	listeners, ok := l.keyListeners.Load(path)
	if !ok {
		return nil
	}
	listenerSet := listeners.(map[config_center.ConfigurationListener]struct{})
	result := make([]config_center.ConfigurationListener, 0, len(listenerSet))
	for listener := range listenerSet {
		result = append(result, listener)
	}
	return result
}

func (l *CacheListener) pathToKeyGroup(path string) (string, string) {
	if len(path) == 0 {
		return path, ""
	}
	groupKey := strings.ReplaceAll(strings.ReplaceAll(path, l.rootPath+constant.PathSeparator, ""), constant.PathSeparator, constant.DotSeparator)
	before, after, _ := strings.Cut(groupKey, constant.DotSeparator)
	return after, before
}
