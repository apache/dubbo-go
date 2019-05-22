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
	"path"
	"strings"
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	perrors "github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
)

var (
	errNilZkClientConn = perrors.New("zookeeperclient{conn} is nil")
)

type zookeeperClient struct {
	name          string
	zkAddrs       []string
	sync.Mutex    // for conn
	conn          *zk.Conn
	timeout       time.Duration
	exit          chan struct{}
	wait          sync.WaitGroup
	eventRegistry map[string][]*chan struct{}
}

func stateToString(state zk.State) string {
	switch state {
	case zk.StateDisconnected:
		return "zookeeper disconnected"
	case zk.StateConnecting:
		return "zookeeper connecting"
	case zk.StateAuthFailed:
		return "zookeeper auth failed"
	case zk.StateConnectedReadOnly:
		return "zookeeper connect readonly"
	case zk.StateSaslAuthenticated:
		return "zookeeper sasl authenticaed"
	case zk.StateExpired:
		return "zookeeper connection expired"
	case zk.StateConnected:
		return "zookeeper connected"
	case zk.StateHasSession:
		return "zookeeper has session"
	case zk.StateUnknown:
		return "zookeeper unknown state"
	case zk.State(zk.EventNodeDeleted):
		return "zookeeper node deleted"
	case zk.State(zk.EventNodeDataChanged):
		return "zookeeper node data changed"
	default:
		return state.String()
	}

	return "zookeeper unknown state"
}

func timeSecondDuration(sec int) time.Duration {
	return time.Duration(sec) * time.Second
}

func newZookeeperClient(name string, zkAddrs []string, timeout time.Duration) (*zookeeperClient, error) {
	var (
		err   error
		event <-chan zk.Event
		z     *zookeeperClient
	)

	z = &zookeeperClient{
		name:          name,
		zkAddrs:       zkAddrs,
		timeout:       timeout,
		exit:          make(chan struct{}),
		eventRegistry: make(map[string][]*chan struct{}),
	}
	// connect to zookeeper
	z.conn, event, err = zk.Connect(zkAddrs, timeout)
	if err != nil {
		return nil, perrors.WithMessagef(err, "zk.Connect(zkAddrs:%+v)", zkAddrs)
	}

	z.wait.Add(1)
	go z.handleZkEvent(event)

	return z, nil
}
func newMockZookeeperClient(name string, timeout time.Duration) (*zk.TestCluster, *zookeeperClient, <-chan zk.Event, error) {
	var (
		err   error
		event <-chan zk.Event
		z     *zookeeperClient
	)

	z = &zookeeperClient{
		name:          name,
		zkAddrs:       []string{},
		timeout:       timeout,
		exit:          make(chan struct{}),
		eventRegistry: make(map[string][]*chan struct{}),
	}
	// connect to zookeeper

	ts, err := zk.StartTestCluster(1, nil, nil)
	if err != nil {
		return nil, nil, nil, perrors.WithMessagef(err, "zk.Connect")
	}

	//callbackChan := make(chan zk.Event)
	//f := func(event zk.Event) {
	//	callbackChan <- event
	//}

	z.conn, event, err = ts.ConnectWithOptions(timeout)
	if err != nil {
		return nil, nil, nil, perrors.WithMessagef(err, "zk.Connect")
	}
	//z.wait.Add(1)

	return ts, z, event, nil
}

func (z *zookeeperClient) handleZkEvent(session <-chan zk.Event) {
	var (
		state int
		event zk.Event
	)

	defer func() {
		z.wait.Done()
		log.Info("zk{path:%v, name:%s} connection goroutine game over.", z.zkAddrs, z.name)
	}()

LOOP:
	for {
		select {
		case <-z.exit:
			break LOOP
		case event = <-session:
			log.Warn("client{%s} get a zookeeper event{type:%s, server:%s, path:%s, state:%d-%s, err:%v}",
				z.name, event.Type, event.Server, event.Path, event.State, stateToString(event.State), event.Err)
			switch (int)(event.State) {
			case (int)(zk.StateDisconnected):
				log.Warn("zk{addr:%s} state is StateDisconnected, so close the zk client{name:%s}.", z.zkAddrs, z.name)
				z.stop()
				z.Lock()
				if z.conn != nil {
					z.conn.Close()
					z.conn = nil
				}
				z.Unlock()
				break LOOP
			case (int)(zk.EventNodeDataChanged), (int)(zk.EventNodeChildrenChanged):
				log.Info("zkClient{%s} get zk node changed event{path:%s}", z.name, event.Path)
				z.Lock()
				for p, a := range z.eventRegistry {
					if strings.HasPrefix(p, event.Path) {
						log.Info("send event{state:zk.EventNodeDataChange, Path:%s} notify event to path{%s} related listener",
							event.Path, p)
						for _, e := range a {
							*e <- struct{}{}
						}
					}
				}
				z.Unlock()
			case (int)(zk.StateConnecting), (int)(zk.StateConnected), (int)(zk.StateHasSession):
				if state == (int)(zk.StateHasSession) {
					continue
				}
				if a, ok := z.eventRegistry[event.Path]; ok && 0 < len(a) {
					for _, e := range a {
						*e <- struct{}{}
					}
				}
			}
			state = (int)(event.State)
		}
	}
}

func (z *zookeeperClient) registerEvent(zkPath string, event *chan struct{}) {
	if zkPath == "" || event == nil {
		return
	}

	z.Lock()
	a := z.eventRegistry[zkPath]
	a = append(a, event)
	z.eventRegistry[zkPath] = a
	log.Debug("zkClient{%s} register event{path:%s, ptr:%p}", z.name, zkPath, event)
	z.Unlock()
}

func (z *zookeeperClient) unregisterEvent(zkPath string, event *chan struct{}) {
	if zkPath == "" {
		return
	}

	z.Lock()
	for {
		a, ok := z.eventRegistry[zkPath]
		if !ok {
			break
		}
		for i, e := range a {
			if e == event {
				arr := a
				a = append(arr[:i], arr[i+1:]...)
				log.Debug("zkClient{%s} unregister event{path:%s, event:%p}", z.name, zkPath, event)
			}
		}
		log.Debug("after zkClient{%s} unregister event{path:%s, event:%p}, array length %d",
			z.name, zkPath, event, len(a))
		if len(a) == 0 {
			delete(z.eventRegistry, zkPath)
		} else {
			z.eventRegistry[zkPath] = a
		}
	}
	z.Unlock()
}

func (z *zookeeperClient) done() <-chan struct{} {
	return z.exit
}

func (z *zookeeperClient) stop() bool {
	select {
	case <-z.exit:
		return true
	default:
		close(z.exit)
	}

	return false
}

func (z *zookeeperClient) zkConnValid() bool {
	select {
	case <-z.exit:
		return false
	default:
	}

	valid := true
	z.Lock()
	if z.conn == nil {
		valid = false
	}
	z.Unlock()

	return valid
}

func (z *zookeeperClient) Close() {
	if z == nil {
		return
	}

	z.stop()
	z.wait.Wait()
	z.Lock()
	if z.conn != nil {
		z.conn.Close()
		z.conn = nil
	}
	z.Unlock()
	log.Warn("zkClient{name:%s, zk addr:%s} exit now.", z.name, z.zkAddrs)
}

func (z *zookeeperClient) Create(basePath string) error {
	var (
		err     error
		tmpPath string
	)

	log.Debug("zookeeperClient.Create(basePath{%s})", basePath)
	for _, str := range strings.Split(basePath, "/")[1:] {
		tmpPath = path.Join(tmpPath, "/", str)
		err = errNilZkClientConn
		z.Lock()
		if z.conn != nil {
			_, err = z.conn.Create(tmpPath, []byte(""), 0, zk.WorldACL(zk.PermAll))
		}
		z.Unlock()
		if err != nil {
			if err == zk.ErrNodeExists {
				log.Error("zk.create(\"%s\") exists\n", tmpPath)
			} else {
				log.Error("zk.create(\"%s\") error(%v)\n", tmpPath, perrors.WithStack(err))
				return perrors.WithMessagef(err, "zk.Create(path:%s)", basePath)
			}
		}
	}

	return nil
}

func (z *zookeeperClient) Delete(basePath string) error {
	var (
		err error
	)

	err = errNilZkClientConn
	z.Lock()
	if z.conn != nil {
		err = z.conn.Delete(basePath, -1)
	}
	z.Unlock()

	return perrors.WithMessagef(err, "Delete(basePath:%s)", basePath)
}

func (z *zookeeperClient) RegisterTemp(basePath string, node string) (string, error) {
	var (
		err     error
		data    []byte
		zkPath  string
		tmpPath string
	)

	err = errNilZkClientConn
	data = []byte("")
	zkPath = path.Join(basePath) + "/" + node
	z.Lock()
	if z.conn != nil {
		tmpPath, err = z.conn.Create(zkPath, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	}
	z.Unlock()
	//if err != nil && err != zk.ErrNodeExists {
	if err != nil {
		log.Warn("conn.Create(\"%s\", zk.FlagEphemeral) = error(%v)\n", zkPath, perrors.WithStack(err))
		return "", perrors.WithStack(err)
	}
	log.Debug("zkClient{%s} create a temp zookeeper node:%s\n", z.name, tmpPath)

	return tmpPath, nil
}

func (z *zookeeperClient) RegisterTempSeq(basePath string, data []byte) (string, error) {
	var (
		err     error
		tmpPath string
	)

	err = errNilZkClientConn
	z.Lock()
	if z.conn != nil {
		tmpPath, err = z.conn.Create(
			path.Join(basePath)+"/",
			data,
			zk.FlagEphemeral|zk.FlagSequence,
			zk.WorldACL(zk.PermAll),
		)
	}
	z.Unlock()
	log.Debug("zookeeperClient.RegisterTempSeq(basePath{%s}) = tempPath{%s}", basePath, tmpPath)
	if err != nil && err != zk.ErrNodeExists {
		log.Error("zkClient{%s} conn.Create(\"%s\", \"%s\", zk.FlagEphemeral|zk.FlagSequence) error(%v)\n",
			z.name, basePath, string(data), err)
		return "", perrors.WithStack(err)
	}
	log.Debug("zkClient{%s} create a temp zookeeper node:%s\n", z.name, tmpPath)

	return tmpPath, nil
}

func (z *zookeeperClient) getChildrenW(path string) ([]string, <-chan zk.Event, error) {
	var (
		err      error
		children []string
		stat     *zk.Stat
		event    <-chan zk.Event
	)

	err = errNilZkClientConn
	z.Lock()
	if z.conn != nil {
		children, stat, event, err = z.conn.ChildrenW(path)
	}
	z.Unlock()
	if err != nil {
		if err == zk.ErrNoNode {
			return nil, nil, perrors.Errorf("path{%s} has none children", path)
		}
		log.Error("zk.ChildrenW(path{%s}) = error(%v)", path, err)
		return nil, nil, perrors.WithMessagef(err, "zk.ChildrenW(path:%s)", path)
	}
	if stat == nil {
		return nil, nil, perrors.Errorf("path{%s} has none children", path)
	}
	if len(children) == 0 {
		return nil, nil, perrors.Errorf("path{%s} has none children", path)
	}

	return children, event, nil
}

func (z *zookeeperClient) getChildren(path string) ([]string, error) {
	var (
		err      error
		children []string
		stat     *zk.Stat
	)

	err = errNilZkClientConn
	z.Lock()
	if z.conn != nil {
		children, stat, err = z.conn.Children(path)
	}
	z.Unlock()
	if err != nil {
		if err == zk.ErrNoNode {
			return nil, perrors.Errorf("path{%s} has none children", path)
		}
		log.Error("zk.Children(path{%s}) = error(%v)", path, perrors.WithStack(err))
		return nil, perrors.WithMessagef(err, "zk.Children(path:%s)", path)
	}
	if stat == nil {
		return nil, perrors.Errorf("path{%s} has none children", path)
	}
	if len(children) == 0 {
		return nil, perrors.Errorf("path{%s} has none children", path)
	}

	return children, nil
}

func (z *zookeeperClient) existW(zkPath string) (<-chan zk.Event, error) {
	var (
		exist bool
		err   error
		event <-chan zk.Event
	)

	err = errNilZkClientConn
	z.Lock()
	if z.conn != nil {
		exist, _, event, err = z.conn.ExistsW(zkPath)
	}
	z.Unlock()
	if err != nil {
		log.Error("zkClient{%s}.ExistsW(path{%s}) = error{%v}.", z.name, zkPath, perrors.WithStack(err))
		return nil, perrors.WithMessagef(err, "zk.ExistsW(path:%s)", zkPath)
	}
	if !exist {
		log.Warn("zkClient{%s}'s App zk path{%s} does not exist.", z.name, zkPath)
		return nil, perrors.Errorf("zkClient{%s} App zk path{%s} does not exist.", z.name, zkPath)
	}

	return event, nil
}
