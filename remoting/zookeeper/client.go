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
	perrors "github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

const (
	ConnDelay    = 3
	MaxFailTimes = 15
)

var (
	errNilZkClientConn = perrors.New("zookeeperclient{conn} is nil")
)

type ZookeeperClient struct {
	name          string
	ZkAddrs       []string
	sync.Mutex    // for conn
	Conn          *zk.Conn
	Timeout       time.Duration
	exit          chan struct{}
	Wait          sync.WaitGroup
	eventRegistry map[string][]*chan struct{}
}

func StateToString(state zk.State) string {
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
		return "zookeeper sasl authenticated"
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

type Options struct {
	zkName string
	client *ZookeeperClient

	ts *zk.TestCluster
}

type Option func(*Options)

func WithZkName(name string) Option {
	return func(opt *Options) {
		opt.zkName = name
	}
}

func ValidateZookeeperClient(container zkClientFacade, opts ...Option) error {
	var (
		err error
	)
	opions := &Options{}
	for _, opt := range opts {
		opt(opions)
	}

	err = nil

	lock := container.ZkClientLock()
	url := container.GetUrl()

	lock.Lock()
	defer lock.Unlock()

	if container.ZkClient() == nil {
		//in dubbo ,every registry only connect one node ,so this is []string{r.Address}
		timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
		if err != nil {
			logger.Errorf("timeout config %v is invalid ,err is %v",
				url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT), err.Error())
			return perrors.WithMessagef(err, "newZookeeperClient(address:%+v)", url.Location)
		}
		zkAddresses := strings.Split(url.Location, ",")
		newClient, err := newZookeeperClient(opions.zkName, zkAddresses, timeout)
		if err != nil {
			logger.Warnf("newZookeeperClient(name{%s}, zk address{%v}, timeout{%d}) = error{%v}",
				opions.zkName, url.Location, timeout.String(), err)
			return perrors.WithMessagef(err, "newZookeeperClient(address:%+v)", url.Location)
		}
		container.SetZkClient(newClient)
	}

	if container.ZkClient().Conn == nil {
		var event <-chan zk.Event
		container.ZkClient().Conn, event, err = zk.Connect(container.ZkClient().ZkAddrs, container.ZkClient().Timeout)
		if err == nil {
			container.ZkClient().Wait.Add(1)
			go container.ZkClient().HandleZkEvent(event)
		}
	}

	return perrors.WithMessagef(err, "newZookeeperClient(address:%+v)", url.PrimitiveURL)
}

func newZookeeperClient(name string, zkAddrs []string, timeout time.Duration) (*ZookeeperClient, error) {
	var (
		err   error
		event <-chan zk.Event
		z     *ZookeeperClient
	)

	z = &ZookeeperClient{
		name:          name,
		ZkAddrs:       zkAddrs,
		Timeout:       timeout,
		exit:          make(chan struct{}),
		eventRegistry: make(map[string][]*chan struct{}),
	}
	// connect to zookeeper
	z.Conn, event, err = zk.Connect(zkAddrs, timeout)
	if err != nil {
		return nil, perrors.WithMessagef(err, "zk.Connect(zkAddrs:%+v)", zkAddrs)
	}

	z.Wait.Add(1)
	go z.HandleZkEvent(event)

	return z, nil
}

func WithTestCluster(ts *zk.TestCluster) Option {
	return func(opt *Options) {
		opt.ts = ts
	}
}

func NewMockZookeeperClient(name string, timeout time.Duration, opts ...Option) (*zk.TestCluster, *ZookeeperClient, <-chan zk.Event, error) {
	var (
		err   error
		event <-chan zk.Event
		z     *ZookeeperClient
		ts    *zk.TestCluster
	)

	z = &ZookeeperClient{
		name:          name,
		ZkAddrs:       []string{},
		Timeout:       timeout,
		exit:          make(chan struct{}),
		eventRegistry: make(map[string][]*chan struct{}),
	}

	opions := &Options{}
	for _, opt := range opts {
		opt(opions)
	}

	// connect to zookeeper
	if opions.ts != nil {
		ts = opions.ts
	} else {
		ts, err = zk.StartTestCluster(1, nil, nil)
		if err != nil {
			return nil, nil, nil, perrors.WithMessagef(err, "zk.Connect")
		}
	}

	//callbackChan := make(chan zk.Event)
	//f := func(event zk.Event) {
	//	callbackChan <- event
	//}

	z.Conn, event, err = ts.ConnectWithOptions(timeout)
	if err != nil {
		return nil, nil, nil, perrors.WithMessagef(err, "zk.Connect")
	}
	//z.wait.Add(1)

	return ts, z, event, nil
}

func (z *ZookeeperClient) HandleZkEvent(session <-chan zk.Event) {
	var (
		state int
		event zk.Event
	)

	defer func() {
		z.Wait.Done()
		logger.Infof("zk{path:%v, name:%s} connection goroutine game over.", z.ZkAddrs, z.name)
	}()

LOOP:
	for {
		select {
		case <-z.exit:
			break LOOP
		case event = <-session:
			logger.Warnf("client{%s} get a zookeeper event{type:%s, server:%s, path:%s, state:%d-%s, err:%v}",
				z.name, event.Type, event.Server, event.Path, event.State, StateToString(event.State), event.Err)
			switch (int)(event.State) {
			case (int)(zk.StateDisconnected):
				logger.Warnf("zk{addr:%s} state is StateDisconnected, so close the zk client{name:%s}.", z.ZkAddrs, z.name)
				z.stop()
				z.Lock()
				if z.Conn != nil {
					z.Conn.Close()
					z.Conn = nil
				}
				z.Unlock()
				break LOOP
			case (int)(zk.EventNodeDataChanged), (int)(zk.EventNodeChildrenChanged):
				logger.Infof("zkClient{%s} get zk node changed event{path:%s}", z.name, event.Path)
				z.Lock()
				for p, a := range z.eventRegistry {
					if strings.HasPrefix(p, event.Path) {
						logger.Infof("send event{state:zk.EventNodeDataChange, Path:%s} notify event to path{%s} related listener",
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

func (z *ZookeeperClient) RegisterEvent(zkPath string, event *chan struct{}) {
	if zkPath == "" || event == nil {
		return
	}

	z.Lock()
	a := z.eventRegistry[zkPath]
	a = append(a, event)
	z.eventRegistry[zkPath] = a
	logger.Debugf("zkClient{%s} register event{path:%s, ptr:%p}", z.name, zkPath, event)
	z.Unlock()
}

func (z *ZookeeperClient) UnregisterEvent(zkPath string, event *chan struct{}) {
	if zkPath == "" {
		return
	}
	z.Lock()
	defer z.Unlock()
	infoList, ok := z.eventRegistry[zkPath]
	if !ok {
		return
	}
	for i, e := range infoList {
		if e == event {
			arr := infoList
			infoList = append(arr[:i], arr[i+1:]...)
			logger.Infof("zkClient{%s} unregister event{path:%s, event:%p}", z.name, zkPath, event)
		}
	}
	logger.Debugf("after zkClient{%s} unregister event{path:%s, event:%p}, array length %d",
		z.name, zkPath, event, len(infoList))
	if len(infoList) == 0 {
		delete(z.eventRegistry, zkPath)
	} else {
		z.eventRegistry[zkPath] = infoList
	}
}

func (z *ZookeeperClient) Done() <-chan struct{} {
	return z.exit
}

func (z *ZookeeperClient) stop() bool {
	select {
	case <-z.exit:
		return true
	default:
		close(z.exit)
	}

	return false
}

func (z *ZookeeperClient) ZkConnValid() bool {
	select {
	case <-z.exit:
		return false
	default:
	}

	valid := true
	z.Lock()
	if z.Conn == nil {
		valid = false
	}
	z.Unlock()

	return valid
}

func (z *ZookeeperClient) Close() {
	if z == nil {
		return
	}

	z.stop()
	z.Wait.Wait()
	z.Lock()
	if z.Conn != nil {
		z.Conn.Close()
		z.Conn = nil
	}
	z.Unlock()
	logger.Warnf("zkClient{name:%s, zk addr:%s} exit now.", z.name, z.ZkAddrs)
}

func (z *ZookeeperClient) Create(basePath string) error {
	var (
		err     error
		tmpPath string
	)

	logger.Debugf("zookeeperClient.Create(basePath{%s})", basePath)
	for _, str := range strings.Split(basePath, "/")[1:] {
		tmpPath = path.Join(tmpPath, "/", str)
		err = errNilZkClientConn
		z.Lock()
		if z.Conn != nil {
			_, err = z.Conn.Create(tmpPath, []byte(""), 0, zk.WorldACL(zk.PermAll))
		}
		z.Unlock()
		if err != nil {
			if err == zk.ErrNodeExists {
				logger.Infof("zk.create(\"%s\") exists\n", tmpPath)
			} else {
				logger.Errorf("zk.create(\"%s\") error(%v)\n", tmpPath, perrors.WithStack(err))
				return perrors.WithMessagef(err, "zk.Create(path:%s)", basePath)
			}
		}
	}

	return nil
}

func (z *ZookeeperClient) Delete(basePath string) error {
	var (
		err error
	)

	err = errNilZkClientConn
	z.Lock()
	if z.Conn != nil {
		err = z.Conn.Delete(basePath, -1)
	}
	z.Unlock()

	return perrors.WithMessagef(err, "Delete(basePath:%s)", basePath)
}

func (z *ZookeeperClient) RegisterTemp(basePath string, node string) (string, error) {
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
	if z.Conn != nil {
		tmpPath, err = z.Conn.Create(zkPath, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	}
	z.Unlock()
	//if err != nil && err != zk.ErrNodeExists {
	if err != nil {
		logger.Warnf("conn.Create(\"%s\", zk.FlagEphemeral) = error(%v)\n", zkPath, perrors.WithStack(err))
		return "", perrors.WithStack(err)
	}
	logger.Debugf("zkClient{%s} create a temp zookeeper node:%s\n", z.name, tmpPath)

	return tmpPath, nil
}

func (z *ZookeeperClient) RegisterTempSeq(basePath string, data []byte) (string, error) {
	var (
		err     error
		tmpPath string
	)

	err = errNilZkClientConn
	z.Lock()
	if z.Conn != nil {
		tmpPath, err = z.Conn.Create(
			path.Join(basePath)+"/",
			data,
			zk.FlagEphemeral|zk.FlagSequence,
			zk.WorldACL(zk.PermAll),
		)
	}
	z.Unlock()
	logger.Debugf("zookeeperClient.RegisterTempSeq(basePath{%s}) = tempPath{%s}", basePath, tmpPath)
	if err != nil && err != zk.ErrNodeExists {
		logger.Errorf("zkClient{%s} conn.Create(\"%s\", \"%s\", zk.FlagEphemeral|zk.FlagSequence) error(%v)\n",
			z.name, basePath, string(data), err)
		return "", perrors.WithStack(err)
	}
	logger.Debugf("zkClient{%s} create a temp zookeeper node:%s\n", z.name, tmpPath)

	return tmpPath, nil
}

func (z *ZookeeperClient) GetChildrenW(path string) ([]string, <-chan zk.Event, error) {
	var (
		err      error
		children []string
		stat     *zk.Stat
		event    <-chan zk.Event
	)

	err = errNilZkClientConn
	z.Lock()
	if z.Conn != nil {
		children, stat, event, err = z.Conn.ChildrenW(path)
	}
	z.Unlock()
	if err != nil {
		if err == zk.ErrNoNode {
			return nil, nil, perrors.Errorf("path{%s} has none children", path)
		}
		logger.Errorf("zk.ChildrenW(path{%s}) = error(%v)", path, err)
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

func (z *ZookeeperClient) GetChildren(path string) ([]string, error) {
	var (
		err      error
		children []string
		stat     *zk.Stat
	)

	err = errNilZkClientConn
	z.Lock()
	if z.Conn != nil {
		children, stat, err = z.Conn.Children(path)
	}
	z.Unlock()
	if err != nil {
		if err == zk.ErrNoNode {
			return nil, perrors.Errorf("path{%s} has none children", path)
		}
		logger.Errorf("zk.Children(path{%s}) = error(%v)", path, perrors.WithStack(err))
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

func (z *ZookeeperClient) ExistW(zkPath string) (<-chan zk.Event, error) {
	var (
		exist bool
		err   error
		event <-chan zk.Event
	)

	err = errNilZkClientConn
	z.Lock()
	if z.Conn != nil {
		exist, _, event, err = z.Conn.ExistsW(zkPath)
	}
	z.Unlock()
	if err != nil {
		logger.Warnf("zkClient{%s}.ExistsW(path{%s}) = error{%v}.", z.name, zkPath, perrors.WithStack(err))
		return nil, perrors.WithMessagef(err, "zk.ExistsW(path:%s)", zkPath)
	}
	if !exist {
		logger.Warnf("zkClient{%s}'s App zk path{%s} does not exist.", z.name, zkPath)
		return nil, perrors.Errorf("zkClient{%s} App zk path{%s} does not exist.", z.name, zkPath)
	}

	return event, nil
}

func (z *ZookeeperClient) GetContent(zkPath string) ([]byte, *zk.Stat, error) {
	return z.Conn.Get(zkPath)
}
