// Copyright 2016-2019 hxmhlt
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zookeeper

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common/logger"
	perrors "github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/registry"
)

const (
	MaxFailTimes = 15
)

type zkEvent struct {
	res *registry.ServiceEvent
	err error
}

func (e zkEvent) String() string {
	return fmt.Sprintf("err:%s, res:%s", e.err, e.res)
}

type zkEventListener struct {
	client         *zookeeperClient
	events         chan zkEvent
	serviceMapLock sync.Mutex
	serviceMap     map[string]struct{}
	wg             sync.WaitGroup
	registry       *zkRegistry
}

func newZkEventListener(registry *zkRegistry, client *zookeeperClient) *zkEventListener {
	return &zkEventListener{
		client:     client,
		registry:   registry,
		events:     make(chan zkEvent, 32),
		serviceMap: make(map[string]struct{}),
	}
}

func (l *zkEventListener) listenServiceNodeEvent(zkPath string) bool {
	l.wg.Add(1)
	defer l.wg.Done()
	var zkEvent zk.Event
	for {
		keyEventCh, err := l.client.existW(zkPath)
		if err != nil {
			logger.Errorf("existW{key:%s} = error{%v}", zkPath, err)
			return false
		}

		select {
		case zkEvent = <-keyEventCh:
			logger.Warnf("get a zookeeper zkEvent{type:%s, server:%s, path:%s, state:%d-%s, err:%s}",
				zkEvent.Type.String(), zkEvent.Server, zkEvent.Path, zkEvent.State, stateToString(zkEvent.State), zkEvent.Err)
			switch zkEvent.Type {
			case zk.EventNodeDataChanged:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNodeDataChanged}", zkPath)
			case zk.EventNodeCreated:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNodeCreated}", zkPath)
			case zk.EventNotWatching:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNotWatching}", zkPath)
			case zk.EventNodeDeleted:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNodeDeleted}", zkPath)
				return true
			}
		case <-l.client.done():
			return false
		}
	}

	return false
}

func (l *zkEventListener) handleZkNodeEvent(zkPath string, children []string, conf common.URL) {
	contains := func(s []string, e string) bool {
		for _, a := range s {
			if a == e {
				return true
			}
		}

		return false
	}

	newChildren, err := l.client.getChildren(zkPath)
	if err != nil {
		logger.Errorf("path{%s} child nodes changed, zk.Children() = error{%v}", zkPath, perrors.WithStack(err))
		return
	}

	// a node was added -- listen the new node
	var (
		newNode    string
		serviceURL common.URL
	)
	for _, n := range newChildren {
		if contains(children, n) {
			continue
		}

		newNode = path.Join(zkPath, n)
		logger.Infof("add zkNode{%s}", newNode)
		//context.TODO
		serviceURL, err = common.NewURL(context.TODO(), n)
		if err != nil {
			logger.Errorf("NewURL(%s) = error{%v}", n, perrors.WithStack(err))
			continue
		}
		if !conf.URLEqual(serviceURL) {
			logger.Warnf("serviceURL{%s} is not compatible with SubURL{%#v}", serviceURL.Key(), conf.Key())
			continue
		}
		logger.Infof("add serviceURL{%s}", serviceURL)
		l.events <- zkEvent{&registry.ServiceEvent{Action: registry.ServiceAdd, Service: serviceURL}, nil}
		// listen l service node
		go func(node string, serviceURL common.URL) {
			logger.Infof("delete zkNode{%s}", node)
			if l.listenServiceNodeEvent(node) {
				logger.Infof("delete serviceURL{%s}", serviceURL)
				l.events <- zkEvent{&registry.ServiceEvent{Action: registry.ServiceDel, Service: serviceURL}, nil}
			}
			logger.Warnf("listenSelf(zk path{%s}) goroutine exit now", zkPath)
		}(newNode, serviceURL)
	}

	// old node was deleted
	var oldNode string
	for _, n := range children {
		if contains(newChildren, n) {
			continue
		}

		oldNode = path.Join(zkPath, n)
		logger.Warnf("delete zkPath{%s}", oldNode)
		serviceURL, err = common.NewURL(context.TODO(), n)
		if !conf.URLEqual(serviceURL) {
			logger.Warnf("serviceURL{%s} has been deleted is not compatible with SubURL{%#v}", serviceURL.Key(), conf.Key())
			continue
		}
		logger.Warnf("delete serviceURL{%s}", serviceURL)
		if err != nil {
			logger.Errorf("NewURL(i{%s}) = error{%v}", n, perrors.WithStack(err))
			continue
		}
		l.events <- zkEvent{&registry.ServiceEvent{Action: registry.ServiceDel, Service: serviceURL}, nil}
	}
}

func (l *zkEventListener) listenDirEvent(zkPath string, conf common.URL) {
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
		children, childEventCh, err := l.client.getChildrenW(zkPath)
		if err != nil {
			failTimes++
			if MaxFailTimes <= failTimes {
				failTimes = MaxFailTimes
			}
			logger.Errorf("listenDirEvent(path{%s}) = error{%v}", zkPath, err)
			// clear the event channel
		CLEAR:
			for {
				select {
				case <-event:
				default:
					break CLEAR
				}
			}
			l.client.registerEvent(zkPath, &event)
			select {
			case <-time.After(timeSecondDuration(failTimes * RegistryConnDelay)):
				l.client.unregisterEvent(zkPath, &event)
				continue
			case <-l.client.done():
				l.client.unregisterEvent(zkPath, &event)
				logger.Warnf("client.done(), listen(path{%s}, ReferenceConfig{%#v}) goroutine exit now...", zkPath, conf)
				return
			case <-event:
				logger.Infof("get zk.EventNodeDataChange notify event")
				l.client.unregisterEvent(zkPath, &event)
				l.handleZkNodeEvent(zkPath, nil, conf)
				continue
			}
		}
		failTimes = 0

		select {
		case zkEvent = <-childEventCh:
			logger.Warnf("get a zookeeper zkEvent{type:%s, server:%s, path:%s, state:%d-%s, err:%s}",
				zkEvent.Type.String(), zkEvent.Server, zkEvent.Path, zkEvent.State, stateToString(zkEvent.State), zkEvent.Err)
			if zkEvent.Type != zk.EventNodeChildrenChanged {
				continue
			}
			l.handleZkNodeEvent(zkEvent.Path, children, conf)
		case <-l.client.done():
			logger.Warnf("client.done(), listen(path{%s}, ReferenceConfig{%#v}) goroutine exit now...", zkPath, conf)
			return
		}
	}
}

// this func is invoked by ZkConsumerRegistry::Registe/ZkConsumerRegistry::get/ZkConsumerRegistry::getListener
// registry.go:Listen -> listenServiceEvent -> listenDirEvent -> listenServiceNodeEvent
//                            |
//                            --------> listenServiceNodeEvent
func (l *zkEventListener) listenServiceEvent(conf common.URL) {
	var (
		err        error
		zkPath     string
		dubboPath  string
		children   []string
		serviceURL common.URL
	)

	zkPath = fmt.Sprintf("/dubbo%s/providers", conf.Path)

	l.serviceMapLock.Lock()
	_, ok := l.serviceMap[zkPath]
	l.serviceMapLock.Unlock()
	if ok {
		logger.Warnf("@zkPath %s has already been listened.", zkPath)
		return
	}

	l.serviceMapLock.Lock()
	l.serviceMap[zkPath] = struct{}{}
	l.serviceMapLock.Unlock()

	logger.Infof("listen dubbo provider path{%s} event and wait to get all provider zk nodes", zkPath)
	children, err = l.client.getChildren(zkPath)
	if err != nil {
		children = nil
		logger.Errorf("fail to get children of zk path{%s}", zkPath)
	}

	for _, c := range children {
		serviceURL, err = common.NewURL(context.TODO(), c)
		if err != nil {
			logger.Errorf("NewURL(r{%s}) = error{%v}", c, err)
			continue
		}
		if !conf.URLEqual(serviceURL) {
			logger.Warnf("serviceURL %v is not compatible with SubURL %v", serviceURL.Key(), conf.Key())
			continue
		}
		logger.Debugf("add serviceUrl{%s}", serviceURL)
		l.events <- zkEvent{&registry.ServiceEvent{Action: registry.ServiceAdd, Service: serviceURL}, nil}

		// listen l service node
		dubboPath = path.Join(zkPath, c)
		logger.Infof("listen dubbo service key{%s}", dubboPath)
		go func(zkPath string, serviceURL common.URL) {
			if l.listenServiceNodeEvent(dubboPath) {
				logger.Debugf("delete serviceUrl{%s}", serviceURL)
				l.events <- zkEvent{&registry.ServiceEvent{Action: registry.ServiceDel, Service: serviceURL}, nil}
			}
			logger.Warnf("listenSelf(zk path{%s}) goroutine exit now", zkPath)
		}(dubboPath, serviceURL)
	}

	logger.Infof("listen dubbo path{%s}", zkPath)
	go func(zkPath string, conf common.URL) {
		l.listenDirEvent(zkPath, conf)
		logger.Warnf("listenDirEvent(zkPath{%s}) goroutine exit now", zkPath)
	}(zkPath, conf)
}

func (l *zkEventListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-l.client.done():
			logger.Warnf("listener's zk client connection is broken, so zk event listener exit now.")
			return nil, perrors.New("listener stopped")

		case <-l.registry.done:
			logger.Warnf("zk consumer register has quit, so zk event listener exit asap now.")
			return nil, perrors.New("listener stopped")

		case e := <-l.events:
			logger.Debugf("got zk event %s", e)
			if e.err != nil {
				return nil, perrors.WithStack(e.err)
			}
			if e.res.Action == registry.ServiceDel && !l.valid() {
				logger.Warnf("update @result{%s}. But its connection to registry is invalid", e.res)
				continue
			}
			//r.update(e.res)
			//write to invoker
			//r.outerEventCh <- e.res
			return e.res, nil
		}
	}
}

func (l *zkEventListener) valid() bool {
	return l.client.zkConnValid()
}

func (l *zkEventListener) Close() {
	l.registry.listenerLock.Lock()
	l.client.Close()
	l.registry.listenerLock.Unlock()
	l.registry.wg.Done()
	l.wg.Wait()
}
