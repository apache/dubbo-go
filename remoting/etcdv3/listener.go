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

package etcdv3

import (
	"sync"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"

	"go.etcd.io/etcd/api/v3/mvccpb"

	clientv3 "go.etcd.io/etcd/client/v3"
)

import (
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// EventListener watches etcd keys and forwards events to DataListener.
type EventListener struct {
	client     *gxetcd.Client
	keyMapLock sync.RWMutex
	keyMap     map[string]struct{}
	wg         sync.WaitGroup
}

// NewEventListener returns a EventListener instance
func NewEventListener(client *gxetcd.Client) *EventListener {
	return &EventListener{
		client: client,
		keyMap: make(map[string]struct{}),
	}
}

// ListenServiceNodeEvent Listen on a spec key.It will return true when spec key deleted,
// and return false when deep layer connection lose
func (l *EventListener) ListenServiceNodeEvent(key string, listener ...remoting.DataListener) bool {
	defer l.wg.Done()
	for {
		wc, err := l.client.Watch(key)
		if err != nil {
			logger.Warnf("[Remoting][Etcdv3] WatchExist, key=%s err=%v", key, err)
			return false
		}

		select {

		// client stopped
		case <-l.client.Done():
			logger.Warn("[Remoting][Etcdv3] etcd client stopped")
			return false

		// client ctx stop
		case <-l.client.GetCtx().Done():
			logger.Warn("[Remoting][Etcdv3] etcd client ctx cancel")
			return false

		// handle etcd events
		case e, ok := <-wc:
			if !ok {
				logger.Warn("[Remoting][Etcdv3] etcd watch-chan closed")
				return false
			}

			if e.Err() != nil {
				logger.Errorf("[Remoting][Etcdv3] etcd watch error, err=%s", e.Err())
				continue
			}
			for _, event := range e.Events {
				if l.handleEvents(event, listener...) {
					// if event is delete
					return true
				}
			}
		}
	}
}

// return true means the event type is DELETE
// return false means the event type is CREATE || UPDATE
func (l *EventListener) handleEvents(event *clientv3.Event, listeners ...remoting.DataListener) bool {
	logger.Infof("[Remoting][Etcdv3] got a etcd event, type=%s key=%s", event.Type, event.Kv.Key)

	switch event.Type {
	// the etcdv3 event just include PUT && DELETE
	case mvccpb.PUT:
		for _, listener := range listeners {
			switch event.IsCreate() {
			case true:
				logger.Infof("[Remoting][Etcdv3] etcd get event, key=%s action=created", event.Kv.Key)
				listener.DataChange(remoting.Event{
					Path:    string(event.Kv.Key),
					Action:  remoting.EventTypeAdd,
					Content: string(event.Kv.Value),
				})
			case false:
				logger.Infof("[Remoting][Etcdv3] etcd get event, key=%s action=changed", event.Kv.Key)
				listener.DataChange(remoting.Event{
					Path:    string(event.Kv.Key),
					Action:  remoting.EventTypeUpdate,
					Content: string(event.Kv.Value),
				})
			}
		}
		return false
	case mvccpb.DELETE:
		logger.Warnf("[Remoting][Etcdv3] etcd get event, key=%s action=deleted", event.Kv.Key)
		return true

	default:
		return false
	}
}

// ListenServiceNodeEventWithPrefix listens on a set of key with spec prefix
func (l *EventListener) ListenServiceNodeEventWithPrefix(prefix string, listener ...remoting.DataListener) {
	defer l.wg.Done()
	for {
		wc, err := l.client.WatchWithPrefix(prefix)
		if err != nil {
			logger.Warnf("[Remoting][Etcdv3] listenDirEvent(key=%s) = err=%v", prefix, err)
		}

		select {

		// client stopped
		case <-l.client.Done():
			logger.Warn("[Remoting][Etcdv3] etcd client stopped")
			return

		// client ctx stop
		case <-l.client.GetCtx().Done():
			logger.Warn("[Remoting][Etcdv3] etcd client ctx cancel")
			return

		// etcd event stream
		case e, ok := <-wc:

			if !ok {
				logger.Warn("[Remoting][Etcdv3] etcd watch-chan closed")
				return
			}

			if e.Err() != nil {
				logger.Errorf("[Remoting][Etcdv3] etcd watch error, err=%s", e.Err())
				continue
			}
			for _, event := range e.Events {
				l.handleEvents(event, listener...)
			}
		}
	}
}

// ListenServiceEvent is invoked by etcdv3 ConsumerRegistry::Registe/ etcdv3 ConsumerRegistry::get/etcdv3 ConsumerRegistry::getListener
// registry.go:Listen -> listenServiceEvent -> listenDirEvent -> listenServiceNodeEvent
// registry.go:Listen -> listenServiceEvent -> listenServiceNodeEvent
func (l *EventListener) ListenServiceEvent(key string, listener remoting.DataListener) {
	l.keyMapLock.RLock()
	_, ok := l.keyMap[key]
	l.keyMapLock.RUnlock()
	if ok {
		logger.Warnf("[Remoting][Etcdv3] etcdv3 key %s has already been listened", key)
		return
	}

	l.keyMapLock.Lock()
	l.keyMap[key] = struct{}{}
	l.keyMapLock.Unlock()

	keyList, valueList, err := l.client.GetChildren(key)
	if err != nil {
		logger.Warnf("[Remoting][Etcdv3] get new node path={%v} 's content error,message is  {%v}", key, perrors.WithMessage(err, "get children"))
	}

	logger.Debugf("[Remoting][Etcdv3] get key children list, key=%s keys=%v values=%v", key, keyList, valueList)

	for i, k := range keyList {
		logger.Infof("[Remoting][Etcdv3] got children list key=%s", k)
		listener.DataChange(remoting.Event{
			Path:    k,
			Action:  remoting.EventTypeAdd,
			Content: valueList[i],
		})
	}

	logger.Debugf("[Remoting][Etcdv3] listen dubbo provider key{%s} event and wait to get all provider etcdv3 nodes", key)
	l.wg.Add(1)
	go func(key string, listener remoting.DataListener) {
		l.ListenServiceNodeEventWithPrefix(key, listener)
		logger.Warnf("[Remoting][Etcdv3] listenDirEvent(key=%s) goroutine exit now", key)
	}(key, listener)

	logger.Infof("[Remoting][Etcdv3] listen dubbo service key{%s}", key)
	l.wg.Add(1)
	go func(key string) {
		if l.ListenServiceNodeEvent(key) {
			listener.DataChange(remoting.Event{Path: key, Action: remoting.EventTypeDel})
		}
		logger.Warnf("[Remoting][Etcdv3] listenSelf(etcd key=%s) goroutine exit now", key)
	}(key)
}

// Close waits for all spawned goroutines to finish.
func (l *EventListener) Close() {
	l.wg.Wait()
}
