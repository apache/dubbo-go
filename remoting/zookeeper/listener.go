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
	"sync"
	"time"
)

import (
	"github.com/dubbogo/getty"
	perrors "github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
)

import (
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
)

type ZkEventListener struct {
	client      *ZookeeperClient
	pathMapLock sync.Mutex
	pathMap     map[string]struct{}
	wg          sync.WaitGroup
}

func NewZkEventListener(client *ZookeeperClient) *ZkEventListener {
	return &ZkEventListener{
		client:  client,
		pathMap: make(map[string]struct{}),
	}
}
func (l *ZkEventListener) SetClient(client *ZookeeperClient) {
	l.client = client
}
func (l *ZkEventListener) ListenServiceNodeEvent(zkPath string, listener ...remoting.DataListener) bool {
	l.wg.Add(1)
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
			logger.Warnf("get a zookeeper zkEvent{type:%s, server:%s, path:%s, state:%d-%s, err:%s}",
				zkEvent.Type.String(), zkEvent.Server, zkEvent.Path, zkEvent.State, StateToString(zkEvent.State), zkEvent.Err)
			switch zkEvent.Type {
			case zk.EventNodeDataChanged:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNodeDataChanged}", zkPath)
				if len(listener) > 0 {
					content, _, _ := l.client.Conn.Get(zkEvent.Path)
					listener[0].DataChange(remoting.Event{Path: zkEvent.Path, Action: remoting.EvnetTypeUpdate, Content: string(content)})
				}

			case zk.EventNodeCreated:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNodeCreated}", zkPath)
				if len(listener) > 0 {
					content, _, _ := l.client.Conn.Get(zkEvent.Path)
					listener[0].DataChange(remoting.Event{Path: zkEvent.Path, Action: remoting.EventTypeAdd, Content: string(content)})
				}
			case zk.EventNotWatching:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNotWatching}", zkPath)
			case zk.EventNodeDeleted:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNodeDeleted}", zkPath)
				return true
			}
		case <-l.client.Done():
			return false
		}
	}

	return false
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
		logger.Errorf("path{%s} child nodes changed, zk.Children() = error{%v}", zkPath, perrors.WithStack(err))
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
		content, _, err := l.client.Conn.Get(newNode)
		if err != nil {
			logger.Errorf("Get new node path {%v} 's content error,message is  {%v}", newNode, perrors.WithStack(err))
		}

		if !listener.DataChange(remoting.Event{Path: zkPath, Action: remoting.EventTypeAdd, Content: string(content)}) {
			continue
		}
		// listen l service node
		go func(node string) {
			logger.Infof("delete zkNode{%s}", node)
			if l.ListenServiceNodeEvent(node, listener) {
				logger.Infof("delete content{%s}", node)
				listener.DataChange(remoting.Event{Path: zkPath, Action: remoting.EventTypeDel})
			}
			logger.Warnf("listenSelf(zk path{%s}) goroutine exit now", zkPath)
		}(newNode)
	}

	// old node was deleted
	var oldNode string
	for _, n := range children {
		if contains(newChildren, n) {
			continue
		}

		oldNode = path.Join(zkPath, n)
		logger.Warnf("delete zkPath{%s}", oldNode)

		if err != nil {
			logger.Errorf("NewURL(i{%s}) = error{%v}", n, perrors.WithStack(err))
			continue
		}
		listener.DataChange(remoting.Event{Path: oldNode, Action: remoting.EventTypeDel})
	}
}

func (l *ZkEventListener) listenDirEvent(zkPath string, listener remoting.DataListener) {
	l.wg.Add(1)
	defer l.wg.Done()

	var (
		failTimes int
		event     chan struct{}
		zkEvent   zk.Event
	)
	event = make(chan struct{}, 4)
	defer close(event)
	for {
		// get current children for a zkPath
		children, childEventCh, err := l.client.GetChildrenW(zkPath)
		if err != nil {
			failTimes++
			if MaxFailTimes <= failTimes {
				failTimes = MaxFailTimes
			}
			logger.Warnf("listenDirEvent(path{%s}) = error{%v}", zkPath, err)
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
			select {
			case <-getty.GetTimeWheel().After(timeSecondDuration(failTimes * ConnDelay)):
				l.client.UnregisterEvent(zkPath, &event)
				continue
			case <-l.client.Done():
				l.client.UnregisterEvent(zkPath, &event)
				logger.Warnf("client.done(), listen(path{%s}) goroutine exit now...", zkPath)
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

			// listen l service node
			dubboPath := path.Join(zkPath, c)

			//Save the path to avoid listen repeatly
			l.pathMapLock.Lock()
			_, ok := l.pathMap[dubboPath]
			l.pathMapLock.Unlock()
			if ok {
				logger.Warnf("@zkPath %s has already been listened.", zkPath)
				continue
			}

			l.pathMapLock.Lock()
			l.pathMap[dubboPath] = struct{}{}
			l.pathMapLock.Unlock()

			content, _, err := l.client.Conn.Get(dubboPath)
			if err != nil {
				logger.Errorf("Get new node path {%v} 's content error,message is  {%v}", dubboPath, perrors.WithStack(err))
			}
			logger.Infof("Get children!{%s}", dubboPath)
			if !listener.DataChange(remoting.Event{Path: dubboPath, Action: remoting.EventTypeAdd, Content: string(content)}) {
				continue
			}
			logger.Infof("listen dubbo service key{%s}", dubboPath)
			go func(zkPath string) {
				if l.ListenServiceNodeEvent(dubboPath) {
					listener.DataChange(remoting.Event{Path: dubboPath, Action: remoting.EventTypeDel})
				}
				logger.Warnf("listenSelf(zk path{%s}) goroutine exit now", zkPath)
			}(dubboPath)

			//liten sub path recursive
			go func(zkPath string, listener remoting.DataListener) {
				l.listenDirEvent(zkPath, listener)
				logger.Warnf("listenDirEvent(zkPath{%s}) goroutine exit now", zkPath)
			}(dubboPath, listener)
		}
		select {
		case zkEvent = <-childEventCh:
			logger.Warnf("get a zookeeper zkEvent{type:%s, server:%s, path:%s, state:%d-%s, err:%s}",
				zkEvent.Type.String(), zkEvent.Server, zkEvent.Path, zkEvent.State, StateToString(zkEvent.State), zkEvent.Err)
			if zkEvent.Type != zk.EventNodeChildrenChanged {
				continue
			}
			l.handleZkNodeEvent(zkEvent.Path, children, listener)
		case <-l.client.Done():
			logger.Warnf("client.done(), listen(path{%s}) goroutine exit now...", zkPath)
			return
		}
	}
}

func timeSecondDuration(sec int) time.Duration {
	return time.Duration(sec) * time.Second
}

// this func is invoked by ZkConsumerRegistry::Registe/ZkConsumerRegistry::get/ZkConsumerRegistry::getListener
// registry.go:Listen -> listenServiceEvent -> listenDirEvent -> ListenServiceNodeEvent
//                            |
//                            --------> ListenServiceNodeEvent
func (l *ZkEventListener) ListenServiceEvent(zkPath string, listener remoting.DataListener) {
	var (
		err       error
		dubboPath string
		children  []string
	)

	l.pathMapLock.Lock()
	_, ok := l.pathMap[zkPath]
	l.pathMapLock.Unlock()
	if ok {
		logger.Warnf("@zkPath %s has already been listened.", zkPath)
		return
	}

	l.pathMapLock.Lock()
	l.pathMap[zkPath] = struct{}{}
	l.pathMapLock.Unlock()

	logger.Infof("listen dubbo provider path{%s} event and wait to get all provider zk nodes", zkPath)
	children, err = l.client.GetChildren(zkPath)
	if err != nil {
		children = nil
		logger.Warnf("fail to get children of zk path{%s}", zkPath)
	}

	for _, c := range children {

		// listen l service node
		dubboPath = path.Join(zkPath, c)
		content, _, err := l.client.Conn.Get(dubboPath)
		if err != nil {
			logger.Errorf("Get new node path {%v} 's content error,message is  {%v}", dubboPath, perrors.WithStack(err))
		}
		if !listener.DataChange(remoting.Event{Path: dubboPath, Action: remoting.EventTypeAdd, Content: string(content)}) {
			continue
		}
		logger.Infof("listen dubbo service key{%s}", dubboPath)
		go func(zkPath string) {
			if l.ListenServiceNodeEvent(dubboPath) {
				listener.DataChange(remoting.Event{Path: dubboPath, Action: remoting.EventTypeDel})
			}
			logger.Warnf("listenSelf(zk path{%s}) goroutine exit now", zkPath)
		}(dubboPath)
	}

	logger.Infof("listen dubbo path{%s}", zkPath)
	go func(zkPath string, listener remoting.DataListener) {
		l.listenDirEvent(zkPath, listener)
		logger.Warnf("listenDirEvent(zkPath{%s}) goroutine exit now", zkPath)
	}(zkPath, listener)
}

func (l *ZkEventListener) valid() bool {
	return l.client.ZkConnValid()
}

func (l *ZkEventListener) Close() {
	l.wg.Wait()
}
