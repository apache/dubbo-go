package etcdv3

import (
	"sync"
	"time"
)

import (
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
)

type EventListener struct {
	client     *Client
	keyMapLock sync.Mutex
	keyMap     map[string]struct{}
	wg         sync.WaitGroup
}

func NewEventListener(client *Client) *EventListener {
	return &EventListener{
		client: client,
		keyMap: make(map[string]struct{}),
	}
}

// Listen on a spec key
// this method will return true when spec key deleted,
// this method will return false when deep layer connection lose
func (l *EventListener) ListenServiceNodeEvent(key string, listener ...remoting.DataListener) bool {
	l.wg.Add(1)
	defer l.wg.Done()
	for {
		wc, err := l.client.Watch(key)
		if err != nil {
			logger.Warnf("WatchExist{key:%s} = error{%v}", key, err)
			return false
		}

		select {

		// client stopped
		case <-l.client.Done():
			logger.Warnf("etcd client stopped")
			return false

		// client ctx stop
		case <-l.client.ctx.Done():
			logger.Warnf("etcd client ctx cancel")
			return false

		// handle etcd events
		case e, ok := <-wc:
			if !ok {
				logger.Warnf("etcd watch-chan closed")
				return false
			}

			if e.Err() != nil {
				logger.Errorf("etcd watch ERR {err: %s}", e.Err())
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

	return false
}

// return true mean the event type is DELETE
// return false mean the event type is CREATE || UPDATE
func (l *EventListener) handleEvents(event *clientv3.Event, listeners ...remoting.DataListener) bool {

	logger.Infof("got a etcd event {type: %s, key: %s}", event.Type, event.Kv.Key)

	switch event.Type {
	// the etcdv3 event just include PUT && DELETE
	case mvccpb.PUT:
		for _, listener := range listeners {
			switch event.IsCreate() {
			case true:
				logger.Infof("etcd get event (key{%s}) = event{EventNodeDataCreated}", event.Kv.Key)
				listener.DataChange(remoting.Event{
					Path:    string(event.Kv.Key),
					Action:  remoting.EventTypeAdd,
					Content: string(event.Kv.Value),
				})
			case false:
				logger.Infof("etcd get event (key{%s}) = event{EventNodeDataChanged}", event.Kv.Key)
				listener.DataChange(remoting.Event{
					Path:    string(event.Kv.Key),
					Action:  remoting.EventTypeUpdate,
					Content: string(event.Kv.Value),
				})
			}
		}
		return false
	case mvccpb.DELETE:
		logger.Warnf("etcd get event (key{%s}) = event{EventNodeDeleted}", event.Kv.Key)
		return true

	default:
		return false
	}

	panic("unreachable")
}

// Listen on a set of key with spec prefix
func (l *EventListener) ListenServiceNodeEventWithPrefix(prefix string, listener ...remoting.DataListener) {

	l.wg.Add(1)
	defer l.wg.Done()
	for {
		wc, err := l.client.WatchWithPrefix(prefix)
		if err != nil {
			logger.Warnf("listenDirEvent(key{%s}) = error{%v}", prefix, err)
		}

		select {

		// client stopped
		case <-l.client.Done():
			logger.Warnf("etcd client stopped")
			return

			// client ctx stop
		case <-l.client.ctx.Done():
			logger.Warnf("etcd client ctx cancel")
			return

			// etcd event stream
		case e, ok := <-wc:

			if !ok {
				logger.Warnf("etcd watch-chan closed")
				return
			}

			if e.Err() != nil {
				logger.Errorf("etcd watch ERR {err: %s}", e.Err())
				continue
			}
			for _, event := range e.Events {
				l.handleEvents(event, listener...)
			}
		}
	}
}

func timeSecondDuration(sec int) time.Duration {
	return time.Duration(sec) * time.Second
}

// this func is invoked by etcdv3 ConsumerRegistry::Registe/ etcdv3 ConsumerRegistry::get/etcdv3 ConsumerRegistry::getListener
// registry.go:Listen -> listenServiceEvent -> listenDirEvent -> ListenServiceNodeEvent
//                            |
//                            --------> ListenServiceNodeEvent
func (l *EventListener) ListenServiceEvent(key string, listener remoting.DataListener) {

	l.keyMapLock.Lock()
	_, ok := l.keyMap[key]
	l.keyMapLock.Unlock()
	if ok {
		logger.Warnf("etcdv3 key %s has already been listened.", key)
		return
	}

	l.keyMapLock.Lock()
	l.keyMap[key] = struct{}{}
	l.keyMapLock.Unlock()

	keyList, valueList, err := l.client.getChildren(key)
	if err != nil {
		logger.Errorf("Get new node path {%v} 's content error,message is  {%v}", key, perrors.WithMessage(err, "get children"))
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

	logger.Infof("listen dubbo provider key{%s} event and wait to get all provider etcdv3 nodes", key)
	go func(key string, listener remoting.DataListener) {
		l.ListenServiceNodeEventWithPrefix(key, listener)
		logger.Warnf("listenDirEvent(key{%s}) goroutine exit now", key)
	}(key, listener)

	logger.Infof("listen dubbo service key{%s}", key)
	go func(key string) {
		if l.ListenServiceNodeEvent(key) {
			listener.DataChange(remoting.Event{Path: key, Action: remoting.EventTypeDel})
		}
		logger.Warnf("listenSelf(etcd key{%s}) goroutine exit now", key)
	}(key)
}

func (l *EventListener) Close() {
	l.wg.Wait()
}
