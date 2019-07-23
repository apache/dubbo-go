package etcdv3

import (
	"gx/ipfs/QmZErC2Ay6WuGi96CPg316PwitdwgLo6RxZRqVjJjRj2MR/go-path"
	pathlib "path"
	"sync"
	"time"
)

import (
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/dubbogo/getty"
	perrors "github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
)

import (
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
)

type EventListener struct {
	client      *Client
	pathMapLock sync.Mutex
	pathMap     map[string]struct{}
	wg          sync.WaitGroup
}

func NewEventListener(client *Client) *EventListener {
	return &EventListener{
		client:  client,
		pathMap: make(map[string]struct{}),
	}
}
func (l *EventListener) SetClient(client *Client) {
	l.client = client
}

// this method will return true when spec path deleted,
// this method will return false when deep layer connection lose
func (l *EventListener) ListenServiceNodeEvent(path string, listener ...remoting.DataListener) bool {
	l.wg.Add(1)
	defer l.wg.Done()
	for {
		keyEventCh, err := l.client.ExistW(path)
		if err != nil {
			logger.Warnf("existW{key:%s} = error{%v}", path, err)
			return false
		}

		select {
		// client watch ctx stop
		// server stopped
		case <-l.client.cs.ctx.Done():
			return false

		// client stopped
		case <-l.client.Done():
			return false

		// etcd event stream
		case e := <-keyEventCh:

			if e.Err() != nil{
				logger.Warnf("get a etcd event {err: %s}", e.Err())
			}
			for _, event := range e.Events{
				logger.Warnf("get a etcd Event{type:%s,  path:%s,}",
					event.Type.String(), event.Kv.Key )
				switch event.Type {
				case mvccpb.PUT:
					if len(listener) > 0 {
						if event.IsCreate(){
							logger.Warnf("etcdV3.ExistW(key{%s}) = event{EventNodeDataCreated}", event.Kv.Key)
							listener[0].DataChange(remoting.Event{Path: string(event.Kv.Key), Action: remoting.EventTypeAdd, Content: string(event.Kv.Value)})
						}else{
							logger.Warnf("etcdV3.ExistW(key{%s}) = event{EventNodeDataChanged}", event.Kv.Key)
							listener[0].DataChange(remoting.Event{Path: string(event.Kv.Key), Action: remoting.EvnetTypeUpdate, Content: string(event.Kv.Value)})
						}
					}
				case mvccpb.DELETE:
					logger.Warnf("etcdV3.ExistW(key{%s}) = event{EventNodeDeleted}", event.Kv.Key)
					return true
				}
			}
		}
	}

	return false
}


func (l *EventListener) handleNodeEvent(path string, children []string, listener remoting.DataListener) {
	contains := func(s []string, e string) bool {
		for _, a := range s {
			if a == e {
				return true
			}
		}

		return false
	}

	newChildren, err := l.client.GetChildren(path)
	if err != nil {
		logger.Errorf("path{%s} child nodes changed, etcdV3.Children() = error{%v}", path, perrors.WithStack(err))
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

		newNode = pathlib.Join(path, n)
		logger.Infof("add zkNode{%s}", newNode)
		content, _, err := l.client.Conn.Get(newNode)
		if err != nil {
			logger.Errorf("Get new node path {%v} 's content error,message is  {%v}", newNode, perrors.WithStack(err))
		}

		if !listener.DataChange(remoting.Event{Path: zkPath, Action: remoting.EventTypeAdd, Content: string(content)}) {
			continue
		}
		// listen l service node
		go func(node, childNode string) {
			logger.Infof("delete zkNode{%s}", node)
			if l.ListenServiceNodeEvent(node, listener) {
				logger.Infof("delete content{%s}", childNode)
				listener.DataChange(remoting.Event{Path: zkPath, Action: remoting.EventTypeDel})
			}
			logger.Warnf("listenSelf(zk path{%s}) goroutine exit now", zkPath)
		}(newNode, n)
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

//
//func (l *ZkEventListener) listenFileEvent(zkPath string, listener remoting.DataListener) {
//	l.wg.EventTypeAdd(1)
//	defer l.wg.Done()
//
//	var (
//		failTimes int
//		event     chan struct{}
//		zkEvent   zk.Event
//	)
//	event = make(chan struct{}, 4)
//	defer close(event)
//	for {
//		// get current children for a zkPath
//		content,_, eventCh, err := l.client.Conn.GetW(zkPath)
//		if err != nil {
//			failTimes++
//			if MaxFailTimes <= failTimes {
//				failTimes = MaxFailTimes
//			}
//			logger.Errorf("listenFileEvent(path{%s}) = error{%v}", zkPath, err)
//			// clear the event channel
//		CLEAR:
//			for {
//				select {
//				case <-event:
//				default:
//					break CLEAR
//				}
//			}
//			l.client.RegisterEvent(zkPath, &event)
//			select {
//			case <-time.After(timeSecondDuration(failTimes * ConnDelay)):
//				l.client.UnregisterEvent(zkPath, &event)
//				continue
//			case <-l.client.Done():
//				l.client.UnregisterEvent(zkPath, &event)
//				logger.Warnf("client.done(), listen(path{%s}) goroutine exit now...", zkPath)
//				return
//			case <-event:
//				logger.Infof("get zk.EventNodeDataChange notify event")
//				l.client.UnregisterEvent(zkPath, &event)
//				l.handleZkNodeEvent(zkPath, nil, listener)
//				continue
//			}
//		}
//		failTimes = 0
//
//		select {
//		case zkEvent = <-eventCh:
//			logger.Warnf("get a zookeeper zkEvent{type:%s, server:%s, path:%s, state:%d-%s, err:%s}",
//				zkEvent.Type.String(), zkEvent.Server, zkEvent.Path, zkEvent.State, StateToString(zkEvent.State), zkEvent.Err)
//
//			l.handleZkNodeEvent(zkEvent.Path, children, listener)
//		case <-l.client.Done():
//			logger.Warnf("client.done(), listen(path{%s}) goroutine exit now...", zkPath)
//			return
//		}
//	}
//}

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
