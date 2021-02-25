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
	"sync/atomic"
	"time"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

const (
	// ConnDelay connection delay interval
	ConnDelay = 3
	// MaxFailTimes max fail times
	MaxFailTimes = 3
)

var (
	errNilZkClientConn = perrors.New("zookeeper client{conn} is nil")
	errNilChildren     = perrors.Errorf("has none children")
	errNilNode         = perrors.Errorf("node does not exist")
)

var (
	zkClientPool ZookeeperClientPool
	once         sync.Once
)

type ZookeeperClientPool struct {
	sync.Mutex
	zkClient map[string]*ZookeeperClient
}

// ZookeeperClient represents zookeeper client Configuration
type ZookeeperClient struct {
	name              string
	ZkAddrs           []string
	sync.RWMutex      // for conn
	Conn              *zk.Conn
	activeNumber      uint32
	Timeout           time.Duration
	Wait              sync.WaitGroup
	valid             uint32
	reconnectCh       chan struct{}
	eventRegistry     map[string][]*chan struct{}
	eventRegistryLock sync.RWMutex
}

// nolint
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
}

// nolint
type Options struct {
	zkName string
	client *ZookeeperClient
	ts     *zk.TestCluster
}

// Option will define a function of handling Options
type Option func(*Options)

// WithZkName sets zk client name
func WithZkName(name string) Option {
	return func(opt *Options) {
		opt.zkName = name
	}
}

// ValidateZookeeperClient validates client and sets options
func ValidateZookeeperClient(container ZkClientFacade, opts ...Option) error {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	lock := container.ZkClientLock()
	url := container.GetUrl()

	lock.Lock()
	defer lock.Unlock()

	if container.ZkClient() == nil {
		// in dubbo, every registry only connect one node, so this is []string{r.Address}
		timeout, paramErr := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
		if paramErr != nil {
			logger.Errorf("timeout config %v is invalid, err is %v",
				url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT), paramErr.Error())
			return perrors.WithMessagef(paramErr, "newZookeeperClient(address:%+v)", url.Location)
		}
		zkAddresses := strings.Split(url.Location, ",")
		newClient, cltErr := getZookeeperClient(options.zkName, zkAddresses, timeout)
		if cltErr != nil {
			logger.Warnf("newZookeeperClient(name{%s}, zk address{%v}, timeout{%d}) = error{%v}",
				options.zkName, url.Location, timeout.String(), cltErr)
			return perrors.WithMessagef(cltErr, "newZookeeperClient(address:%+v)", url.Location)
		}
		container.SetZkClient(newClient)
	}
	return nil
}

func initZookeeperClientPool() {
	zkClientPool.zkClient = make(map[string]*ZookeeperClient)
}
func getZookeeperClient(name string, zkAddrs []string, timeout time.Duration) (*ZookeeperClient, error) {
	once.Do(initZookeeperClientPool)
	zkClientPool.Lock()
	defer zkClientPool.Unlock()
	if zkClient, ok := zkClientPool.zkClient[name]; ok {
		zkClient.activeNumber++
		return zkClient, nil
	}
	zkClient, err := NewZookeeperClient(name, zkAddrs, timeout)
	if err != nil {
		return nil, err
	}
	zkClientPool.zkClient[name] = zkClient
	zkClient.activeNumber++
	return zkClient, nil
}

// nolint
func NewZookeeperClient(name string, zkAddrs []string, timeout time.Duration) (*ZookeeperClient, error) {
	var (
		err   error
		event <-chan zk.Event
		z     *ZookeeperClient
	)

	z = &ZookeeperClient{
		name:          name,
		ZkAddrs:       zkAddrs,
		Timeout:       timeout,
		activeNumber:  0,
		reconnectCh:   make(chan struct{}),
		eventRegistry: make(map[string][]*chan struct{}),
	}
	// connect to zookeeper
	z.Conn, event, err = zk.Connect(zkAddrs, timeout)
	if err != nil {
		return nil, perrors.WithMessagef(err, "zk.Connect(zkAddrs:%+v)", zkAddrs)
	}

	atomic.StoreUint32(&z.valid, 1)
	go z.HandleZkEvent(event)
	return z, nil
}

// WithTestCluster sets test cluster for zk client
func WithTestCluster(ts *zk.TestCluster) Option {
	return func(opt *Options) {
		opt.ts = ts
	}
}

// NewMockZookeeperClient returns a mock client instance
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
		reconnectCh:   make(chan struct{}),
		eventRegistry: make(map[string][]*chan struct{}),
	}

	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	// connect to zookeeper
	if options.ts != nil {
		ts = options.ts
	} else {
		ts, err = zk.StartTestCluster(1, nil, nil)
		if err != nil {
			return nil, nil, nil, perrors.WithMessagef(err, "zk.Connect")
		}
	}

	z.Conn, event, err = ts.ConnectWithOptions(timeout)
	if err != nil {
		return nil, nil, nil, perrors.WithMessagef(err, "zk.Connect")
	}
	atomic.StoreUint32(&z.valid, 1)
	z.activeNumber++
	return ts, z, event, nil
}

// HandleZkEvent handles zookeeper events
func (z *ZookeeperClient) HandleZkEvent(session <-chan zk.Event) {
	var (
		state int
		event zk.Event
	)

	defer func() {
		logger.Infof("zk{path:%v, name:%s} connection goroutine game over.", z.ZkAddrs, z.name)
	}()

	for {
		select {
		case event = <-session:
			logger.Infof("client{%s} get a zookeeper event{type:%s, server:%s, path:%s, state:%d-%s, err:%v}",
				z.name, event.Type, event.Server, event.Path, event.State, StateToString(event.State), event.Err)
			switch (int)(event.State) {
			case (int)(zk.StateDisconnected):
				atomic.StoreUint32(&z.valid, 0)
				logger.Warnf("zk{addr:%s} state is StateDisconnected, so close the zk client{name:%s}.", z.ZkAddrs, z.name)
			case (int)(zk.EventNodeDataChanged), (int)(zk.EventNodeChildrenChanged):
				logger.Infof("zkClient{%s} get zk node changed event{path:%s}", z.name, event.Path)
				z.eventRegistryLock.RLock()
				for p, a := range z.eventRegistry {
					if strings.HasPrefix(p, event.Path) {
						logger.Infof("send event{state:zk.EventNodeDataChange, Path:%s} notify event to path{%s} related listener",
							event.Path, p)
						for _, e := range a {
							*e <- struct{}{}
						}
					}
				}
				z.eventRegistryLock.RUnlock()
			case (int)(zk.StateConnecting), (int)(zk.StateConnected), (int)(zk.StateHasSession):
				if state == (int)(zk.StateHasSession) {
					continue
				}
				if event.State == zk.StateHasSession {
					atomic.StoreUint32(&z.valid, 1)
					close(z.reconnectCh)
					z.reconnectCh = make(chan struct{})
				}
				z.eventRegistryLock.RLock()
				if a, ok := z.eventRegistry[event.Path]; ok && 0 < len(a) {
					for _, e := range a {
						*e <- struct{}{}
					}
				}
				z.eventRegistryLock.RUnlock()
			}
			state = (int)(event.State)
		}
	}
}

// RegisterEvent registers zookeeper events
func (z *ZookeeperClient) RegisterEvent(zkPath string, event *chan struct{}) {
	if zkPath == "" || event == nil {
		return
	}

	z.eventRegistryLock.Lock()
	defer z.eventRegistryLock.Unlock()
	a := z.eventRegistry[zkPath]
	a = append(a, event)
	z.eventRegistry[zkPath] = a
	logger.Debugf("zkClient{%s} register event{path:%s, ptr:%p}", z.name, zkPath, event)
}

// UnregisterEvent unregisters zookeeper events
func (z *ZookeeperClient) UnregisterEvent(zkPath string, event *chan struct{}) {
	if zkPath == "" {
		return
	}

	z.eventRegistryLock.Lock()
	defer z.eventRegistryLock.Unlock()
	infoList, ok := z.eventRegistry[zkPath]
	if !ok {
		return
	}
	for i, e := range infoList {
		if e == event {
			infoList = append(infoList[:i], infoList[i+1:]...)
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

// ZkConnValid validates zookeeper connection
func (z *ZookeeperClient) ZkConnValid() bool {
	if atomic.LoadUint32(&z.valid) == 1 {
		return true
	}
	return false
}

// Create will create the node recursively, which means that if the parent node is absent,
// it will create parent node first.
// And the value for the basePath is ""
func (z *ZookeeperClient) Create(basePath string) error {
	return z.CreateWithValue(basePath, []byte(""))
}

// CreateWithValue will create the node recursively, which means that if the parent node is absent,
// it will create parent node first.
func (z *ZookeeperClient) CreateWithValue(basePath string, value []byte) error {
	var (
		err     error
		tmpPath string
	)

	logger.Debugf("zookeeperClient.Create(basePath{%s})", basePath)
	conn := z.getConn()
	err = errNilZkClientConn
	if conn == nil {
		return perrors.WithMessagef(err, "zk.Create(path:%s)", basePath)
	}

	for _, str := range strings.Split(basePath, "/")[1:] {
		tmpPath = path.Join(tmpPath, "/", str)
		_, err = conn.Create(tmpPath, value, 0, zk.WorldACL(zk.PermAll))

		if err != nil {
			if err == zk.ErrNodeExists {
				logger.Debugf("zk.create(\"%s\") exists", tmpPath)
			} else {
				logger.Errorf("zk.create(\"%s\") error(%v)", tmpPath, perrors.WithStack(err))
				return perrors.WithMessagef(err, "zk.Create(path:%s)", basePath)
			}
		}
	}

	return nil
}

// CreateTempWithValue will create the node recursively, which means that if the parent node is absent,
// it will create parent node first，and set value in last child path
// If the path exist, it will update data
func (z *ZookeeperClient) CreateTempWithValue(basePath string, value []byte) error {
	var (
		err     error
		tmpPath string
	)

	logger.Debugf("zookeeperClient.Create(basePath{%s})", basePath)
	conn := z.getConn()
	err = errNilZkClientConn
	if conn == nil {
		return perrors.WithMessagef(err, "zk.Create(path:%s)", basePath)
	}

	pathSlice := strings.Split(basePath, "/")[1:]
	length := len(pathSlice)
	for i, str := range pathSlice {
		tmpPath = path.Join(tmpPath, "/", str)
		// last child need be ephemeral
		if i == length-1 {
			_, err = conn.Create(tmpPath, value, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
			if err == zk.ErrNodeExists {
				return err
			}
		} else {
			_, err = conn.Create(tmpPath, []byte{}, 0, zk.WorldACL(zk.PermAll))
		}
		if err != nil {
			if err == zk.ErrNodeExists {
				logger.Debugf("zk.create(\"%s\") exists", tmpPath)
			} else {
				logger.Errorf("zk.create(\"%s\") error(%v)", tmpPath, perrors.WithStack(err))
				return perrors.WithMessagef(err, "zk.Create(path:%s)", basePath)
			}
		}
	}

	return nil
}

// nolint
func (z *ZookeeperClient) Delete(basePath string) error {
	err := errNilZkClientConn
	conn := z.getConn()
	if conn != nil {
		err = conn.Delete(basePath, -1)
	}

	return perrors.WithMessagef(err, "Delete(basePath:%s)", basePath)
}

// RegisterTemp registers temporary node by @basePath and @node
func (z *ZookeeperClient) RegisterTemp(basePath string, node string) (string, error) {
	var (
		err     error
		zkPath  string
		tmpPath string
	)

	err = errNilZkClientConn
	zkPath = path.Join(basePath) + "/" + node
	conn := z.getConn()
	if conn != nil {
		tmpPath, err = conn.Create(zkPath, []byte(""), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	}

	if err != nil {
		logger.Warnf("conn.Create(\"%s\", zk.FlagEphemeral) = error(%v)", zkPath, perrors.WithStack(err))
		return zkPath, perrors.WithStack(err)
	}
	logger.Debugf("zkClient{%s} create a temp zookeeper node:%s", z.name, tmpPath)

	return tmpPath, nil
}

// RegisterTempSeq register temporary sequence node by @basePath and @data
func (z *ZookeeperClient) RegisterTempSeq(basePath string, data []byte) (string, error) {
	var (
		err     error
		tmpPath string
	)

	err = errNilZkClientConn
	conn := z.getConn()
	if conn != nil {
		tmpPath, err = conn.Create(
			path.Join(basePath)+"/",
			data,
			zk.FlagEphemeral|zk.FlagSequence,
			zk.WorldACL(zk.PermAll),
		)
	}

	logger.Debugf("zookeeperClient.RegisterTempSeq(basePath{%s}) = tempPath{%s}", basePath, tmpPath)
	if err != nil && err != zk.ErrNodeExists {
		logger.Errorf("zkClient{%s} conn.Create(\"%s\", \"%s\", zk.FlagEphemeral|zk.FlagSequence) error(%v)",
			z.name, basePath, string(data), err)
		return "", perrors.WithStack(err)
	}
	logger.Debugf("zkClient{%s} create a temp zookeeper node:%s", z.name, tmpPath)

	return tmpPath, nil
}

// GetChildrenW gets children watch by @path
func (z *ZookeeperClient) GetChildrenW(path string) ([]string, <-chan zk.Event, error) {
	var (
		err      error
		children []string
		stat     *zk.Stat
		watcher  *zk.Watcher
	)

	err = errNilZkClientConn
	conn := z.getConn()
	if conn != nil {
		children, stat, watcher, err = conn.ChildrenW(path)
	}

	if err != nil {
		if err == zk.ErrNoChildrenForEphemerals {
			return nil, nil, errNilChildren
		}
		if err == zk.ErrNoNode {
			return nil, nil, errNilNode
		}
		logger.Warnf("zk.ChildrenW(path{%s}) = error(%v)", path, err)
		return nil, nil, perrors.WithMessagef(err, "zk.ChildrenW(path:%s)", path)
	}
	if stat == nil {
		return nil, nil, perrors.Errorf("path{%s} get stat is nil", path)
	}
	if len(children) == 0 {
		return nil, nil, errNilChildren
	}

	return children, watcher.EvtCh, nil
}

// GetChildren gets children by @path
func (z *ZookeeperClient) GetChildren(path string) ([]string, error) {
	var (
		err      error
		children []string
		stat     *zk.Stat
	)

	err = errNilZkClientConn
	conn := z.getConn()
	if conn != nil {
		children, stat, err = conn.Children(path)
	}

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
		return nil, errNilChildren
	}

	return children, nil
}

// ExistW to judge watch whether it exists or not by @zkPath
func (z *ZookeeperClient) ExistW(zkPath string) (<-chan zk.Event, error) {
	var (
		exist   bool
		err     error
		watcher *zk.Watcher
	)

	err = errNilZkClientConn
	conn := z.getConn()
	if conn != nil {
		exist, _, watcher, err = conn.ExistsW(zkPath)
	}

	if err != nil {
		logger.Warnf("zkClient{%s}.ExistsW(path{%s}) = error{%v}.", z.name, zkPath, perrors.WithStack(err))
		return nil, perrors.WithMessagef(err, "zk.ExistsW(path:%s)", zkPath)
	}
	if !exist {
		logger.Warnf("zkClient{%s}'s App zk path{%s} does not exist.", z.name, zkPath)
		return nil, perrors.Errorf("zkClient{%s} App zk path{%s} does not exist.", z.name, zkPath)
	}

	return watcher.EvtCh, nil
}

// GetContent gets content by @zkPath
func (z *ZookeeperClient) GetContent(zkPath string) ([]byte, *zk.Stat, error) {
	return z.Conn.Get(zkPath)
}

// nolint
func (z *ZookeeperClient) SetContent(zkPath string, content []byte, version int32) (*zk.Stat, error) {
	return z.Conn.Set(zkPath, content, version)
}

// getConn gets zookeeper connection safely
func (z *ZookeeperClient) getConn() *zk.Conn {
	if z == nil {
		return nil
	}
	z.RLock()
	defer z.RUnlock()
	return z.Conn
}

// Reconnect gets zookeeper reconnect event
func (z *ZookeeperClient) Reconnect() <-chan struct{} {
	return z.reconnectCh
}

// In my opinion, this method should never called by user, here just for lint
func (z *ZookeeperClient) Close() {
	zkClientPool.Lock()
	defer zkClientPool.Unlock()
	z.activeNumber--
	if z.activeNumber == 0 {
		z.Conn.Close()
		delete(zkClientPool.zkClient, z.name)
	}
}
