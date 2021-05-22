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

package kubernetes

import (
	"sync"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type EventListener struct {
	client     *Client
	keyMapLock sync.RWMutex
	keyMap     map[string]struct{}
	wg         sync.WaitGroup
}

func NewEventListener(client *Client) *EventListener {
	return &EventListener{
		client: client,
		keyMap: make(map[string]struct{}, 8),
	}
}

// Listen on a spec key
// this method returns true when spec key deleted,
// this method returns false when deep layer connection lose
func (l *EventListener) ListenServiceNodeEvent(key string, listener ...remoting.DataListener) bool {
	defer l.wg.Done()
	for {
		wc, done, err := l.client.Watch(key)
		if err != nil {
			logger.Warnf("watch exist{key:%s} = error{%v}", key, err)
			return false
		}

		select {

		// client stopped
		case <-l.client.Done():
			logger.Warnf("kubernetes client stopped")
			return false

		// watcherSet watcher stopped
		case <-done:
			logger.Warnf("kubernetes watcherSet watcher stopped")
			return false

		// handle kubernetes-watcherSet events
		case e, ok := <-wc:
			if !ok {
				logger.Warnf("kubernetes-watcherSet watch-chan closed")
				return false
			}

			if l.handleEvents(e, listener...) {
				// if event is delete
				return true
			}
		}
	}
}

// return true means the event type is DELETE
// return false means the event type is CREATE || UPDATE
func (l *EventListener) handleEvents(event *WatcherEvent, listeners ...remoting.DataListener) bool {
	logger.Infof("got a kubernetes-watcherSet event {type: %d, key: %s}", event.EventType, event.Key)

	switch event.EventType {
	case Create:
		for _, listener := range listeners {
			logger.Infof("kubernetes-watcherSet get event (key{%s}) = event{EventNodeDataCreated}", event.Key)
			listener.DataChange(remoting.Event{
				Path:    string(event.Key),
				Action:  remoting.EventTypeAdd,
				Content: string(event.Value),
			})
		}
		return false
	case Update:
		for _, listener := range listeners {
			logger.Infof("kubernetes-watcherSet get event (key{%s}) = event{EventNodeDataChanged}", event.Key)
			listener.DataChange(remoting.Event{
				Path:    string(event.Key),
				Action:  remoting.EventTypeUpdate,
				Content: string(event.Value),
			})
		}
		return false
	case Delete:
		logger.Warnf("kubernetes-watcherSet get event (key{%s}) = event{EventNodeDeleted}", event.Key)
		return true
	default:
		return false
	}
}

// Listen on a set of key with spec prefix
func (l *EventListener) ListenServiceNodeEventWithPrefix(prefix string, listener ...remoting.DataListener) {
	defer l.wg.Done()
	for {
		wc, done, err := l.client.WatchWithPrefix(prefix)
		if err != nil {
			logger.Warnf("listenDirEvent(key{%s}) = error{%v}", prefix, err)
		}

		select {
		// client stopped
		case <-l.client.Done():
			logger.Warnf("kubernetes client stopped")
			return

		// watcher stopped
		case <-done:
			logger.Warnf("kubernetes watcherSet watcher stopped")
			return

			// kuberentes-watcherSet event stream
		case e, ok := <-wc:

			if !ok {
				logger.Warnf("kubernetes-watcherSet watch-chan closed")
				return
			}

			l.handleEvents(e, listener...)
		}
	}
}

// this func is invoked by kubernetes ConsumerRegistry::Registry/ kubernetes ConsumerRegistry::get/kubernetes ConsumerRegistry::getListener
// registry.go:Listen -> listenServiceEvent -> listenDirEvent -> ListenServiceNodeEvent
//                            |
//                            --------> ListenServiceNodeEvent
func (l *EventListener) ListenServiceEvent(key string, listener remoting.DataListener) {
	l.keyMapLock.RLock()
	_, ok := l.keyMap[key]
	l.keyMapLock.RUnlock()
	if ok {
		logger.Warnf("kubernetes-watcherSet key %s has already been listened.", key)
		return
	}

	l.keyMapLock.Lock()
	// double check
	if _, ok := l.keyMap[key]; ok {
		// another goroutine already set it
		l.keyMapLock.Unlock()
		return
	}
	l.keyMap[key] = struct{}{}
	l.keyMapLock.Unlock()

	keyList, valueList, err := l.client.GetChildren(key)
	if err != nil {
		logger.Warnf("Get new node path {%v} 's content error,message is  {%v}", key, perrors.WithMessage(err, "get children"))
	}

	logger.Infof("get key children list %s, keys %v values %v", key, keyList, valueList)

	for i, k := range keyList {
		logger.Infof("got children list key -> %s", k)
		listener.DataChange(remoting.Event{
			Path:    k,
			Action:  remoting.EventTypeAdd,
			Content: valueList[i],
		})
	}

	logger.Infof("listen dubbo provider key{%s} event and wait to get all provider from kubernetes-watcherSet", key)

	l.wg.Add(1)
	go func(key string, listener remoting.DataListener) {
		l.ListenServiceNodeEventWithPrefix(key, listener)
		logger.Warnf("listenDirEvent(key{%s}) goroutine exit now", key)
	}(key, listener)

	logger.Infof("listen dubbo service key{%s}", key)
	l.wg.Add(1)
	go func(key string) {
		if l.ListenServiceNodeEvent(key) {
			listener.DataChange(remoting.Event{Path: key, Action: remoting.EventTypeDel})
		}
		logger.Warnf("listenSelf(kubernetes key{%s}) goroutine exit now", key)
	}(key)
}

func (l *EventListener) Close() {
	l.wg.Wait()
}
