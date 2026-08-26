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
	"net/url"
	"path"
	"slices"
	"strings"
	"sync"
	"time"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
	"github.com/dubbogo/gost/log/logger"

	"github.com/go-zookeeper/zk"

	perrors "github.com/pkg/errors"

	uatomic "go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

var (
	defaultTTL     = 10 * time.Minute
	maxScheduleTTL = 20 * time.Second
)

type ZkEventListener struct {
	Client      *gxzookeeper.ZookeeperClient
	pathMapLock sync.Mutex
	pathMap     map[string]*uatomic.Int32
	wg          sync.WaitGroup
	exit        chan struct{}
}

// WatchRegistrationResult tells the configuration event loop how to continue
// after a listener validates a newly registered ZooKeeper watch.
type WatchRegistrationResult uint8

const (
	// WatchRegistrationAccepted keeps the watch and processes the current read.
	WatchRegistrationAccepted WatchRegistrationResult = iota
	// WatchRegistrationReload re-reads the node without registering another watch.
	WatchRegistrationReload
	// WatchRegistrationDiscarded stops processing a stale event.
	WatchRegistrationDiscarded
)

// configurationWatchStateListener lets the config center validate watch state
// while the remoting layer remains responsible for ZooKeeper reads and events.
type configurationWatchStateListener interface {
	WatchStateChanged(path string) bool
	WatchRegistered(path string, events <-chan zk.Event, beforeSessionID, afterSessionID int64) WatchRegistrationResult
	WatchStateChangeFailed(path string)
}

// NewZkEventListener returns a EventListener instance
func NewZkEventListener(client *gxzookeeper.ZookeeperClient) *ZkEventListener {
	return &ZkEventListener{
		Client:  client,
		pathMap: make(map[string]*uatomic.Int32),
		exit:    make(chan struct{}),
	}
}

// ListenServiceNodeEvent listen a path node event
func (l *ZkEventListener) ListenServiceNodeEvent(zkPath string, listener remoting.DataListener) {
	l.wg.Add(1)
	go func(zkPath string, listener remoting.DataListener) {
		defer l.wg.Done()
		if l.listenServiceNodeEvent(zkPath, listener) {
			listener.DataChange(remoting.Event{Path: zkPath, Action: remoting.EventTypeDel})
		}
		l.pathMapLock.Lock()
		delete(l.pathMap, zkPath)
		l.pathMapLock.Unlock()
		logger.Warnf("[Remoting][Zookeeper] listenServiceNodeEvent->listenSelf, zkPath=%s goroutine exit now", zkPath)
	}(zkPath, listener)
}

// ListenConfigurationEvent listen a path node event
func (l *ZkEventListener) ListenConfigurationEvent(zkPath string, listener remoting.DataListener) {
	l.wg.Add(1)
	go func(zkPath string, listener remoting.DataListener) {
		defer l.wg.Done()
		var eventChan = make(chan zk.Event, 16)
		l.Client.RegisterEvent(zkPath, eventChan)
		watchStateListener, tracksWatchState := listener.(configurationWatchStateListener)
		for {
			select {
			case event := <-eventChan:
				logger.Infof("[Remoting][Zookeeper]Receive configuration change event:%#v", event)
				l.processConfigurationEvent(event, listener, watchStateListener, tracksWatchState)
			case <-l.exit:
				return
			}
		}

	}(zkPath, listener)
}

type configurationEventReader struct {
	exists  func(registerWatch bool) (bool, <-chan zk.Event, int64, int64, error)
	content func() ([]byte, int64, int64, error)
}

// processConfigurationEvent renews a one-shot watch when requested before
// reading the latest value, then lets watch-aware listeners reject results from
// stale sessions.
func (l *ZkEventListener) processConfigurationEvent(
	event zk.Event,
	listener remoting.DataListener,
	watchStateListener configurationWatchStateListener,
	tracksWatchState bool,
) {
	reader := configurationEventReader{
		exists: func(registerWatch bool) (bool, <-chan zk.Event, int64, int64, error) {
			beforeSessionID := l.Client.Conn.SessionID()
			var (
				exists      bool
				watchEvents <-chan zk.Event
				err         error
			)
			if registerWatch {
				exists, _, watchEvents, err = l.Client.Conn.ExistsW(event.Path)
			} else {
				exists, _, err = l.Client.Conn.Exists(event.Path)
			}
			return exists, watchEvents, beforeSessionID, l.Client.Conn.SessionID(), err
		},
		content: func() ([]byte, int64, int64, error) {
			beforeSessionID := l.Client.Conn.SessionID()
			content, _, err := l.Client.Conn.Get(event.Path)
			return content, beforeSessionID, l.Client.Conn.SessionID(), err
		},
	}
	processConfigurationEvent(event, listener, watchStateListener, tracksWatchState, reader)
}

func processConfigurationEvent(
	event zk.Event,
	listener remoting.DataListener,
	watchStateListener configurationWatchStateListener,
	tracksWatchState bool,
	reader configurationEventReader,
) {
	if event.Type == zk.EventNotWatching || event.Type == zk.EventNodeChildrenChanged {
		return
	}
	registerWatch := true
	if tracksWatchState {
		registerWatch = watchStateListener.WatchStateChanged(event.Path)
	}

	// Re-set the watcher before reading the value so a concurrent update cannot
	// occur between the read and watch registration.
	exists, watchEvents, beforeSessionID, afterSessionID, err := reader.exists(registerWatch)
	if err != nil {
		if tracksWatchState {
			watchStateListener.WatchStateChangeFailed(event.Path)
		}
		logger.Warnf("[Remoting][Zookeeper]Re-set watcher error, err=%v", err)
		return
	}
	if tracksWatchState && registerWatch {
		registrationResult := watchStateListener.WatchRegistered(
			event.Path, watchEvents, beforeSessionID, afterSessionID,
		)
		if registrationResult == WatchRegistrationDiscarded {
			return
		}
		if registrationResult == WatchRegistrationAccepted &&
			!sessionStable(beforeSessionID, afterSessionID) {
			watchStateListener.WatchStateChangeFailed(event.Path)
			return
		}
		if registrationResult == WatchRegistrationReload {
			exists, _, beforeSessionID, afterSessionID, err = reader.exists(false)
			if err != nil {
				watchStateListener.WatchStateChangeFailed(event.Path)
				logger.Warnf("[Remoting][Zookeeper] reload config existence error, err=%v", err)
				return
			}
			if !sessionStable(beforeSessionID, afterSessionID) {
				watchStateListener.WatchStateChangeFailed(event.Path)
				return
			}
		}
	}
	if !tracksWatchState || !registerWatch {
		if !sessionStable(beforeSessionID, afterSessionID) {
			if tracksWatchState {
				watchStateListener.WatchStateChangeFailed(event.Path)
			}
			return
		}
	}

	action := remoting.EventTypeDel
	var content string
	if exists {
		action = remoting.EventTypeAdd
		contentBytes, beforeSessionID, afterSessionID, err := reader.content()
		if err != nil {
			if tracksWatchState {
				watchStateListener.WatchStateChangeFailed(event.Path)
			}
			logger.Warnf("[Remoting][Zookeeper] get config value error, err=%v", err)
			return
		}
		if !sessionStable(beforeSessionID, afterSessionID) {
			if tracksWatchState {
				watchStateListener.WatchStateChangeFailed(event.Path)
			}
			return
		}
		content = string(contentBytes)
		logger.Debugf("[Remoting][Zookeeper] successfully get new config value=%s", content)
	}

	listener.DataChange(remoting.Event{Path: event.Path, Action: action, Content: content})
}

// sessionStable reports whether a ZooKeeper operation completed in the session
// in which it started. A zero pair represents a reader without session data.
func sessionStable(beforeSessionID, afterSessionID int64) bool {
	return (beforeSessionID == 0 && afterSessionID == 0) ||
		(beforeSessionID != 0 && beforeSessionID == afterSessionID)
}

// listenServiceNodeEvent watches a single zk node and reports changes via listener.
func (l *ZkEventListener) listenServiceNodeEvent(zkPath string, listener ...remoting.DataListener) bool {
	l.pathMapLock.Lock()
	a, ok := l.pathMap[zkPath]
	if !ok || a.Load() > 1 {
		l.pathMapLock.Unlock()
		return false
	}
	a.Inc()
	l.pathMapLock.Unlock()
	defer a.Dec()
	var zkEvent zk.Event
	for {
		keyEventCh, err := l.Client.ExistW(zkPath)
		if err != nil {
			logger.Warnf("[Remoting][Zookeeper] existW, key=%s err=%v", zkPath, err)
			return false
		}
		select {
		case zkEvent = <-keyEventCh:
			logger.Warnf("[Remoting][Zookeeper] get a zookeeper keyEventCh, type=%s server=%s path=%s state=%d-%s err=%s",
				zkEvent.Type.String(), zkEvent.Server, zkEvent.Path, zkEvent.State, gxzookeeper.StateToString(zkEvent.State), zkEvent.Err)
			switch zkEvent.Type {
			case zk.EventNodeDataChanged:
				logger.Warnf("[Remoting][Zookeeper] zk.ExistW(key=%s) = event{EventNodeDataChanged}", zkPath)
				if len(listener) > 0 {
					content, _, err := l.Client.Conn.Get(zkEvent.Path)
					if err != nil {
						logger.Warnf("[Remoting][Zookeeper] zk.Conn.Get, key=%s err=%v", zkPath, err)
						return false
					}
					listener[0].DataChange(remoting.Event{Path: zkEvent.Path, Action: remoting.EventTypeUpdate, Content: string(content)})
				}
			case zk.EventNodeCreated:
				logger.Warnf("[Remoting][Zookeeper] get a EventNodeCreated event for path {%s}", zkPath)
				if len(listener) > 0 {
					content, _, err := l.Client.Conn.Get(zkEvent.Path)
					if err != nil {
						logger.Warnf("[Remoting][Zookeeper] zk.Conn.Get, key=%s err=%v", zkPath, err)
						return false
					}
					listener[0].DataChange(remoting.Event{Path: zkEvent.Path, Action: remoting.EventTypeAdd, Content: string(content)})
				}
			case zk.EventNotWatching:
				logger.Infof("[Remoting][Zookeeper] get a EventNotWatching event for path {%s}", zkPath)
			case zk.EventNodeDeleted:
				logger.Infof("[Remoting][Zookeeper] get a EventNodeDeleted event for path {%s}", zkPath)
				return true
			}
		case <-l.exit:
			return false
		}
	}
}

func (l *ZkEventListener) handleZkNodeEvent(zkPath string, children []string, listener remoting.DataListener) {
	contains := func(s []string, e string) bool {
		return slices.Contains(s, e)
	}
	newChildren, err := l.Client.GetChildren(zkPath)
	if err != nil {
		logger.Errorf("[Remoting][Zookeeper] path{%s} child nodes changed, zk.Children() = error{%v}", zkPath, perrors.WithStack(err))
		return
	}
	// a node was added -- listen the new node
	var (
		newNode string
	)
	for _, n := range newChildren {
		newNode = path.Join(zkPath, n)
		logger.Debugf("[Remoting][Zookeeper] add zkNode{%s}", newNode)
		content, _, connErr := l.Client.Conn.Get(newNode)
		if connErr != nil {
			logger.Errorf("[Remoting][Zookeeper] get new node path {%v} 's content error,message is  {%v}",
				newNode, perrors.WithStack(connErr))
		}

		if !listener.DataChange(remoting.Event{Path: newNode, Action: remoting.EventTypeAdd, Content: string(content)}) {
			continue
		}
		// listen l service node
		l.wg.Add(1)
		go func(node string, listener remoting.DataListener) {
			defer l.wg.Done()
			if l.listenServiceNodeEvent(node, listener) {
				logger.Warnf("[Remoting][Zookeeper] delete zkNode=%s", node)
				listener.DataChange(remoting.Event{Path: node, Action: remoting.EventTypeDel})
			}
			l.pathMapLock.Lock()
			delete(l.pathMap, zkPath)
			l.pathMapLock.Unlock()
			logger.Debugf("[Remoting][Zookeeper] handleZkNodeEvent->listenSelf, zkPath=%s goroutine exit now", node)
		}(newNode, listener)
	}

	// old node was deleted
	var oldNode string
	for _, n := range children {
		if contains(newChildren, n) {
			continue
		}
		oldNode = path.Join(zkPath, n)
		logger.Warnf("[Remoting][Zookeeper] delete oldNode=%s", oldNode)
		listener.DataChange(remoting.Event{Path: oldNode, Action: remoting.EventTypeDel})
	}
}

// listenerAllDirEvents listens all services when conf.InterfaceKey = "*"
func (l *ZkEventListener) listenAllDirEvents(conf *common.URL, listener remoting.DataListener) {
	var (
		failTimes int
		ttl       time.Duration
	)
	ttl = defaultTTL
	if conf != nil {
		if timeout, err := time.ParseDuration(conf.GetParam(constant.RegistryTTLKey, constant.DefaultRegTTL)); err == nil {
			ttl = timeout
		} else {
			logger.Warnf("[Remoting][Zookeeper] wrong configuration for registry.ttl, err=%v, using default value %v instead", err, defaultTTL)
		}
	}
	if ttl > maxScheduleTTL {
		ttl = maxScheduleTTL
	}

	rootPath := path.Join(constant.PathSeparator, constant.Dubbo)
	for {
		// get all interfaces
		children, childEventCh, err := l.Client.GetChildrenW(rootPath)
		if err != nil {
			failTimes++
			if MaxFailTimes <= failTimes {
				failTimes = MaxFailTimes
			}
			logger.Errorf("[Remoting][Zookeeper] get children of path {%s} with watcher failed, err=%v", rootPath, err)
			// Maybe the zookeeper does not ready yet, sleep failTimes * ConnDelay senconds to wait
			after := time.After(timeSecondDuration(failTimes * ConnDelay))
			select {
			case <-after:
				continue
			case <-l.exit:
				return
			}
		}
		failTimes = 0
		if len(children) == 0 {
			logger.Warnf("[Remoting][Zookeeper] can not get any children for the path \"%s\", please check if the provider does ready", rootPath)
		}
		for _, c := range children {
			// Build the child path
			zkRootPath := path.Join(rootPath, constant.PathSeparator, url.QueryEscape(c), constant.PathSeparator, constant.ProvidersCategory)
			// Save the path to avoid listen repeatedly
			l.pathMapLock.Lock()
			if _, ok := l.pathMap[zkRootPath]; ok {
				logger.Warnf("[Remoting][Zookeeper] the child with zk path {%s} has already been listened", zkRootPath)
				l.pathMapLock.Unlock()
				continue
			} else {
				l.pathMap[zkRootPath] = uatomic.NewInt32(0)
			}
			l.pathMapLock.Unlock()
			logger.Debugf("[Remoting][Zookeeper] listen dubbo interface key{%s}", zkRootPath)
			l.wg.Add(1)
			// listen every interface
			go l.listenDirEvent(conf, zkRootPath, listener, c)
		}

		ticker := time.NewTicker(ttl)
		select {
		case <-ticker.C:
			ticker.Stop()
		case zkEvent := <-childEventCh:
			logger.Debugf("[Remoting][Zookeeper] get a zookeeper childEventCh, type=%s server=%s path=%s state=%d-%s err=%v",
				zkEvent.Type.String(), zkEvent.Server, zkEvent.Path, zkEvent.State, gxzookeeper.StateToString(zkEvent.State), zkEvent.Err)
			ticker.Stop()
		case <-l.exit:
			logger.Warnf("[Remoting][Zookeeper] listen(path=%s) goroutine exit now", rootPath)
			ticker.Stop()
			return
		}
	}
}

func (l *ZkEventListener) listenDirEvent(conf *common.URL, zkRootPath string, listener remoting.DataListener, intf string) {
	defer l.wg.Done()
	if intf == constant.AnyValue {
		l.listenAllDirEvents(conf, listener)
		return
	}
	var (
		failTimes int
		ttl       time.Duration
	)
	ttl = defaultTTL
	if conf != nil {
		timeout, err := time.ParseDuration(conf.GetParam(constant.RegistryTTLKey, constant.DefaultRegTTL))
		if err == nil {
			ttl = timeout
		} else {
			logger.Warnf("[Remoting][Zookeeper] wrong configuration for registry.ttl, err=%v, using default value %v instead", err, defaultTTL)
		}
	}

	for {
		// Get current children with watcher for the zkRootPath
		children, childEventCh, err := l.Client.GetChildrenW(zkRootPath)
		if err != nil {
			failTimes++
			if MaxFailTimes <= failTimes {
				failTimes = MaxFailTimes
			}

			if !perrors.Is(err, zk.ErrNoNode) { // ignore if node not exist
				logger.Errorf("[Remoting][Zookeeper] get children of path {%s} with watcher failed, err=%v", zkRootPath, err)
			}
			// Maybe the provider does not ready yet, sleep failTimes * ConnDelay senconds to wait
			after := time.After(timeSecondDuration(failTimes * ConnDelay))
			select {
			case <-after:
				continue
			case <-l.exit:
				return
			}
		}
		failTimes = 0
		if len(children) == 0 {
			logger.Debugf("[Remoting][Zookeeper] can not get any children for the path {%s}, please check if the provider does ready", zkRootPath)
		}
		for _, c := range children {
			// Only need to compare Path when subscribing to provider
			if strings.LastIndex(zkRootPath, constant.ProviderCategory) != -1 {
				provider, _ := common.NewURL(c)
				if provider.Interface() != intf || !common.IsAnyCondition(constant.AnyValue, conf.Group(), conf.Version(), provider) {
					continue
				}
			}
			// Build the children path
			zkNodePath := path.Join(zkRootPath, c)
			// Save the path to avoid listen repeatedly
			l.pathMapLock.Lock()
			_, ok := l.pathMap[zkNodePath]
			if !ok {
				l.pathMap[zkNodePath] = uatomic.NewInt32(0)
			}
			l.pathMapLock.Unlock()
			if ok {
				logger.Warnf("[Remoting][Zookeeper] the child with zk path {%s} has already been listened", zkNodePath)
				l.Client.RLock()
				if l.Client.Conn == nil {
					l.Client.RUnlock()
					break
				}
				content, _, err := l.Client.Conn.Get(zkNodePath)
				l.Client.RUnlock()
				if err != nil {
					logger.Errorf("[Remoting][Zookeeper] get content of the child node {%v} failed, err=%v", zkNodePath, perrors.WithStack(err))
				}
				listener.DataChange(remoting.Event{Path: zkNodePath, Action: remoting.EventTypeAdd, Content: string(content)})
				continue
			}
			// When Zk disconnected, the Conn will be set to nil, so here need check the value of Conn
			l.Client.RLock()
			if l.Client.Conn == nil {
				l.Client.RUnlock()
				break
			}
			content, _, err := l.Client.Conn.Get(zkNodePath)
			l.Client.RUnlock()
			if err != nil {
				logger.Errorf("[Remoting][Zookeeper] get content of the child node {%v} failed, err=%v", zkNodePath, perrors.WithStack(err))
			}
			logger.Debugf("[Remoting][Zookeeper] get children!{%s}", zkNodePath)
			if !listener.DataChange(remoting.Event{Path: zkNodePath, Action: remoting.EventTypeAdd, Content: string(content)}) {
				continue
			}
			logger.Debugf("[Remoting][Zookeeper] listen dubbo service key{%s}", zkNodePath)
			l.wg.Add(1)
			go func(zkPath string, listener remoting.DataListener) {
				defer l.wg.Done()
				if l.listenServiceNodeEvent(zkPath, listener) {
					listener.DataChange(remoting.Event{Path: zkPath, Action: remoting.EventTypeDel})
				}
				l.pathMapLock.Lock()
				delete(l.pathMap, zkPath)
				l.pathMapLock.Unlock()
				logger.Warnf("[Remoting][Zookeeper] listenDirEvent->listenSelf(zkPath=%s) goroutine exit now", zkPath)
			}(zkNodePath, listener)
		}
		if l.startScheduleWatchTask(zkRootPath, children, ttl, listener, childEventCh) {
			return
		}
	}
}

// startScheduleWatchTask periodically update provider information, return true when receive exit signal
func (l *ZkEventListener) startScheduleWatchTask(
	zkRootPath string, children []string, ttl time.Duration,
	listener remoting.DataListener, childEventCh <-chan zk.Event) bool {
	tickerTTL := min(ttl, maxScheduleTTL)

	childrenNode, err := l.Client.GetChildren(zkRootPath)
	if err == nil {
		l.handleZkNodeEvent(zkRootPath, childrenNode, listener)
	}

	ticker := time.NewTicker(tickerTTL)
	for {
		select {
		case <-ticker.C:
			l.handleZkNodeEvent(zkRootPath, children, listener)
			if tickerTTL < ttl {
				tickerTTL *= 2
				if tickerTTL > ttl {
					tickerTTL = ttl
				}
				ticker.Stop()
				ticker = time.NewTicker(tickerTTL)
			}
		case zkEvent := <-childEventCh:
			logger.Debugf("[Remoting][Zookeeper] get a zookeeper childEventCh, type=%s server=%s path=%s state=%d-%s err=%v",
				zkEvent.Type.String(), zkEvent.Server, zkEvent.Path, zkEvent.State, gxzookeeper.StateToString(zkEvent.State), zkEvent.Err)
			ticker.Stop()
			if zkEvent.Type == zk.EventNodeChildrenChanged {
				l.handleZkNodeEvent(zkEvent.Path, children, listener)
			}
			return false
		case <-l.exit:
			logger.Warnf("[Remoting][Zookeeper] listen(path=%s) goroutine exit now", zkRootPath)
			ticker.Stop()
			return true
		}
	}
}

func timeSecondDuration(sec int) time.Duration {
	return time.Duration(sec) * time.Second
}

// ListenServiceEvent is invoked by ZkConsumerRegistry::Register/ZkConsumerRegistry::get/ZkConsumerRegistry::getListener
// registry.go:Listen -> listenServiceEvent -> listenDirEvent -> listenServiceNodeEvent
// registry.go:Listen -> listenServiceEvent -> listenServiceNodeEvent
func (l *ZkEventListener) ListenServiceEvent(conf *common.URL, zkPath string, listener remoting.DataListener) {
	logger.Infof("[Remoting][Zookeeper] listen dubbo path{%s}", zkPath)
	l.wg.Add(1)
	go func(zkPath string, listener remoting.DataListener) {
		intf := ""
		if conf != nil {
			intf = conf.Interface()
		}
		l.listenDirEvent(conf, zkPath, listener, intf)
		logger.Warnf("[Remoting][Zookeeper] listenServiceEvent->listenDirEvent, zkPath=%s goroutine exit now", zkPath)
	}(zkPath, listener)
}

// Close will let client listen exit
func (l *ZkEventListener) Close() {
	close(l.exit)
	l.wg.Wait()
}
