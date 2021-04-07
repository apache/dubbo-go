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
	"path"
	"strings"
	"sync"
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"
	"github.com/dubbogo/go-zookeeper/zk"
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
)

var (
	defaultTTL = 15 * time.Minute
)

// nolint
type ZkEventListener struct {
	client      *gxzookeeper.ZookeeperClient
	pathMapLock sync.Mutex
	pathMap     map[string]struct{}
	wg          sync.WaitGroup
	exit        chan struct{}
}

// NewZkEventListener returns a EventListener instance
func NewZkEventListener(client *gxzookeeper.ZookeeperClient) *ZkEventListener {
	return &ZkEventListener{
		client:  client,
		pathMap: make(map[string]struct{}),
		exit:    make(chan struct{}),
	}
}

// nolint
func (l *ZkEventListener) SetClient(client *gxzookeeper.ZookeeperClient) {
	l.client = client
}

// ListenServiceNodeEvent listen a path node event
func (l *ZkEventListener) ListenServiceNodeEvent(zkPath string, listener remoting.DataListener) {
	// listen l service node
	l.wg.Add(1)
	go func(zkPath string, listener remoting.DataListener) {
		if l.listenServiceNodeEvent(zkPath, listener) {
			listener.DataChange(remoting.Event{Path: zkPath, Action: remoting.EventTypeDel})
			l.pathMapLock.Lock()
			delete(l.pathMap, zkPath)
			l.pathMapLock.Unlock()
		}
		logger.Warnf("ListenServiceNodeEvent->listenSelf(zk path{%s}) goroutine exit now", zkPath)
	}(zkPath, listener)
}

// nolint
func (l *ZkEventListener) listenServiceNodeEvent(zkPath string, listener ...remoting.DataListener) bool {
	defer l.wg.Done()
	var zkEvent zk.Event
	for {
		keyEventCh, err := l.client.ExistW(zkPath)
		if err != nil {
			logger.Warnf("existW{key:%s} = error{%v}", zkPath, err)
			return false
		}

		select {
		case zkEvent = <-keyEventCh:
			logger.Warnf("get a zookeeper keyEventCh{type:%s, server:%s, path:%s, state:%d-%s, err:%s}",
				zkEvent.Type.String(), zkEvent.Server, zkEvent.Path, zkEvent.State, gxzookeeper.StateToString(zkEvent.State), zkEvent.Err)
			switch zkEvent.Type {
			case zk.EventNodeDataChanged:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNodeDataChanged}", zkPath)
				if len(listener) > 0 {
					content, _, err := l.client.Conn.Get(zkEvent.Path)
					if err != nil {
						logger.Warnf("zk.Conn.Get{key:%s} = error{%v}", zkPath, err)
						return false
					}
					listener[0].DataChange(remoting.Event{Path: zkEvent.Path, Action: remoting.EventTypeUpdate, Content: string(content)})
				}
			case zk.EventNodeCreated:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNodeCreated}", zkPath)
				if len(listener) > 0 {
					content, _, err := l.client.Conn.Get(zkEvent.Path)
					if err != nil {
						logger.Warnf("zk.Conn.Get{key:%s} = error{%v}", zkPath, err)
						return false
					}
					listener[0].DataChange(remoting.Event{Path: zkEvent.Path, Action: remoting.EventTypeAdd, Content: string(content)})
				}
			case zk.EventNotWatching:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNotWatching}", zkPath)
			case zk.EventNodeDeleted:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNodeDeleted}", zkPath)
				return true
			}
		case <-l.exit:
			return false
		}
	}
}

func (l *ZkEventListener) handleZkNodeEvent(zkPath string, children []string, listener remoting.DataListener) {
	contains := func(s []string, e string) bool {
		for _, a := range s {
			if a == e {
				return true
			}
		}
		return false
	}

	newChildren, err := l.client.GetChildren(zkPath)
	if err != nil {
		if err == errNilChildren {
			content, _, connErr := l.client.Conn.Get(zkPath)
			if connErr != nil {
				logger.Errorf("Get new node path {%v} 's content error,message is  {%v}",
					zkPath, perrors.WithStack(connErr))
			} else {
				listener.DataChange(remoting.Event{Path: zkPath, Action: remoting.EventTypeUpdate, Content: string(content)})
			}

		} else {
			logger.Errorf("path{%s} child nodes changed, zk.Children() = error{%v}", zkPath, perrors.WithStack(err))
		}
		return
	}

	// a node was added -- listen the new node
	var (
		newNode string
	)
	for _, n := range newChildren {
		if contains(children, n) {
			continue
		}

		newNode = path.Join(zkPath, n)
		logger.Infof("add zkNode{%s}", newNode)
		content, _, connErr := l.client.Conn.Get(newNode)
		if connErr != nil {
			logger.Errorf("Get new node path {%v} 's content error,message is  {%v}",
				newNode, perrors.WithStack(connErr))
		}

		if !listener.DataChange(remoting.Event{Path: newNode, Action: remoting.EventTypeAdd, Content: string(content)}) {
			continue
		}
		// listen l service node
		l.wg.Add(1)
		go func(node string, listener remoting.DataListener) {
			if l.listenServiceNodeEvent(node, listener) {
				logger.Warnf("delete zkNode{%s}", node)
				listener.DataChange(remoting.Event{Path: node, Action: remoting.EventTypeDel})
				l.pathMapLock.Lock()
				delete(l.pathMap, node)
				l.pathMapLock.Unlock()
			}
			logger.Warnf("handleZkNodeEvent->listenSelf(zk path{%s}) goroutine exit now", node)
		}(newNode, listener)
	}

	// old node was deleted
	var oldNode string
	for _, n := range children {
		if contains(newChildren, n) {
			continue
		}

		oldNode = path.Join(zkPath, n)
		logger.Warnf("delete oldNode{%s}", oldNode)
		listener.DataChange(remoting.Event{Path: oldNode, Action: remoting.EventTypeDel})
		l.pathMapLock.Lock()
		delete(l.pathMap, oldNode)
		l.pathMapLock.Unlock()
	}
}

func (l *ZkEventListener) listenDirEvent(conf *common.URL, zkPath string, listener remoting.DataListener) {
	defer l.wg.Done()

	var (
		failTimes int
		ttl       time.Duration
		event     chan struct{}
		zkEvent   zk.Event
	)
	event = make(chan struct{}, 4)
	ttl = defaultTTL
	if conf != nil {
		timeout, err := time.ParseDuration(conf.GetParam(constant.REGISTRY_TTL_KEY, constant.DEFAULT_REG_TTL))
		if err == nil {
			ttl = timeout
		} else {
			logger.Warnf("wrong configuration for registry ttl, error:=%+v, using default value %v instead", err, defaultTTL)
		}
	}
	defer close(event)
	for {
		// get current children for a zkPath
		children, childEventCh, err := l.client.GetChildrenW(zkPath)
		if err != nil {
			failTimes++
			if MaxFailTimes <= failTimes {
				failTimes = MaxFailTimes
			}
			logger.Infof("listenDirEvent(path{%s}) = error{%v}", zkPath, err)
			// clear the event channel
		CLEAR:
			for {
				select {
				case <-event:
				default:
					break CLEAR
				}
			}
			l.client.RegisterEvent(zkPath, &event)
			if err == errNilNode {
				logger.Warnf("listenDirEvent(path{%s}) got errNilNode,so exit listen", zkPath)
				l.client.UnregisterEvent(zkPath, &event)
				return
			}
			select {
			case <-getty.GetTimeWheel().After(timeSecondDuration(failTimes * ConnDelay)):
				l.client.UnregisterEvent(zkPath, &event)
				continue
			case <-l.exit:
				l.client.UnregisterEvent(zkPath, &event)
				logger.Warnf("listen(path{%s}) goroutine exit now...", zkPath)
				return
			case <-event:
				logger.Infof("get zk.EventNodeDataChange notify event")
				l.client.UnregisterEvent(zkPath, &event)
				l.handleZkNodeEvent(zkPath, nil, listener)
				continue
			}
		}
		failTimes = 0
		for _, c := range children {

			// Only need to compare Path when subscribing to provider
			if strings.LastIndex(zkPath, constant.PROVIDER_CATEGORY) != -1 {
				provider, _ := common.NewURL(c)
				if provider.ServiceKey() != conf.ServiceKey() {
					continue
				}
			}

			// listen l service node
			dubboPath := path.Join(zkPath, c)

			// Save the path to avoid listen repeatedly
			l.pathMapLock.Lock()
			_, ok := l.pathMap[dubboPath]
			l.pathMapLock.Unlock()
			if ok {
				logger.Warnf("@zkPath %s has already been listened.", dubboPath)
				continue
			}

			l.pathMapLock.Lock()
			l.pathMap[dubboPath] = struct{}{}
			l.pathMapLock.Unlock()
			// When Zk disconnected, the Conn will be set to nil, so here need check the value of Conn
			l.client.RLock()
			if l.client.Conn == nil {
				l.client.RUnlock()
				break
			}
			content, _, err := l.client.Conn.Get(dubboPath)
			l.client.RUnlock()
			if err != nil {
				logger.Errorf("Get new node path {%v} 's content error,message is  {%v}", dubboPath, perrors.WithStack(err))
			}
			logger.Debugf("Get children!{%s}", dubboPath)
			if !listener.DataChange(remoting.Event{Path: dubboPath, Action: remoting.EventTypeAdd, Content: string(content)}) {
				continue
			}
			logger.Infof("listen dubbo service key{%s}", dubboPath)
			l.wg.Add(1)
			go func(zkPath string, listener remoting.DataListener) {
				if l.listenServiceNodeEvent(zkPath, listener) {
					listener.DataChange(remoting.Event{Path: zkPath, Action: remoting.EventTypeDel})
					l.pathMapLock.Lock()
					delete(l.pathMap, zkPath)
					l.pathMapLock.Unlock()
				}
				logger.Warnf("listenDirEvent->listenSelf(zk path{%s}) goroutine exit now", zkPath)
			}(dubboPath, listener)

			// listen sub path recursive
			// if zkPath is end of "providers/ & consumers/" we do not listen children dir
			if strings.LastIndex(zkPath, constant.PROVIDER_CATEGORY) == -1 &&
				strings.LastIndex(zkPath, constant.CONSUMER_CATEGORY) == -1 {
				l.wg.Add(1)
				go func(zkPath string, listener remoting.DataListener) {
					l.listenDirEvent(conf, zkPath, listener)
					logger.Warnf("listenDirEvent(zkPath{%s}) goroutine exit now", zkPath)
				}(dubboPath, listener)
			}
		}
		// Periodically update provider information
		ticker := time.NewTicker(ttl)
	WATCH:
		for {
			select {
			case <-ticker.C:
				l.handleZkNodeEvent(zkPath, children, listener)
			case zkEvent = <-childEventCh:
				logger.Warnf("get a zookeeper childEventCh{type:%s, server:%s, path:%s, state:%d-%s, err:%s}",
					zkEvent.Type.String(), zkEvent.Server, zkEvent.Path, zkEvent.State, gxzookeeper.StateToString(zkEvent.State), zkEvent.Err)
				ticker.Stop()
				if zkEvent.Type != zk.EventNodeChildrenChanged {
					break WATCH
				}
				l.handleZkNodeEvent(zkEvent.Path, children, listener)
				break WATCH
			case <-l.exit:
				logger.Warnf("listen(path{%s}) goroutine exit now...", zkPath)
				ticker.Stop()
				return
			}
		}

	}
}

func timeSecondDuration(sec int) time.Duration {
	return time.Duration(sec) * time.Second
}

// ListenServiceEvent is invoked by ZkConsumerRegistry::Register/ZkConsumerRegistry::get/ZkConsumerRegistry::getListener
// registry.go:Listen -> listenServiceEvent -> listenDirEvent -> listenServiceNodeEvent
//                            |
//                            --------> listenServiceNodeEvent
func (l *ZkEventListener) ListenServiceEvent(conf *common.URL, zkPath string, listener remoting.DataListener) {
	logger.Infof("listen dubbo path{%s}", zkPath)
	l.wg.Add(1)
	go func(zkPath string, listener remoting.DataListener) {
		l.listenDirEvent(conf, zkPath, listener)
		logger.Warnf("ListenServiceEvent->listenDirEvent(zkPath{%s}) goroutine exit now", zkPath)
	}(zkPath, listener)
}

//func (l *ZkEventListener) valid() bool {
//	return l.client.ZkConnValid()
//}

// Close will let client listen exit
func (l *ZkEventListener) Close() {
	close(l.exit)
	l.wg.Wait()
}
