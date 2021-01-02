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
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting/zookeeper"
)

const (
	// RegistryZkClient zk client name
	RegistryZkClient = "zk registry"
)

func init() {
	extension.SetRegistry("zookeeper", newZkRegistry)
}

/////////////////////////////////////
// zookeeper registry
/////////////////////////////////////

type zkRegistry struct {
	registry.BaseRegistry
	client         *zookeeper.ZookeeperClient
	listenerLock   sync.Mutex
	listener       *zookeeper.ZkEventListener
	dataListener   *RegistryDataListener
	configListener *RegistryConfigurationListener
	cltLock        sync.Mutex
	//for provider
	zkPath map[string]int // key = protocol://ip:port/interface
}

func newZkRegistry(url *common.URL) (registry.Registry, error) {
	var (
		err error
		r   *zkRegistry
	)
	r = &zkRegistry{
		zkPath: make(map[string]int),
	}
	r.InitBaseRegistry(url, r)

	err = zookeeper.ValidateZookeeperClient(r, zookeeper.WithZkName(RegistryZkClient))
	if err != nil {
		return nil, err
	}
	r.WaitGroup().Add(1) //zk client start successful, then wg +1
	go zookeeper.HandleClientRestart(r)
	r.listener = zookeeper.NewZkEventListener(r.client)
	r.configListener = NewRegistryConfigurationListener(r.client, r)
	r.dataListener = NewRegistryDataListener(r.configListener)

	return r, nil
}

// Options ...
type Options struct {
	client *zookeeper.ZookeeperClient
}

// Option ...
type Option func(*Options)

func newMockZkRegistry(url *common.URL, opts ...zookeeper.Option) (*zk.TestCluster, *zkRegistry, error) {
	var (
		err error
		r   *zkRegistry
		c   *zk.TestCluster
		//event <-chan zk.Event
	)

	r = &zkRegistry{
		zkPath: make(map[string]int),
	}
	r.InitBaseRegistry(url, r)
	c, r.client, _, err = zookeeper.NewMockZookeeperClient("test", 15*time.Second, opts...)
	if err != nil {
		return nil, nil, err
	}
	r.InitListeners()
	return c, r, nil
}

func (r *zkRegistry) InitListeners() {
	r.listener = zookeeper.NewZkEventListener(r.client)
	r.configListener = NewRegistryConfigurationListener(r.client, r)
	r.dataListener = NewRegistryDataListener(r.configListener)
}

func (r *zkRegistry) CreatePath(path string) error {
	return r.ZkClient().Create(path)
}

func (r *zkRegistry) DoRegister(root string, node string) error {
	return r.registerTempZookeeperNode(root, node)
}

func (r *zkRegistry) DoSubscribe(conf *common.URL) (registry.Listener, error) {
	return r.getListener(conf)
}

func (r *zkRegistry) CloseAndNilClient() {
	r.client = nil
}

func (r *zkRegistry) ZkClient() *zookeeper.ZookeeperClient {
	return r.client
}

func (r *zkRegistry) SetZkClient(client *zookeeper.ZookeeperClient) {
	r.client = client
}

func (r *zkRegistry) ZkClientLock() *sync.Mutex {
	return &r.cltLock
}

func (r *zkRegistry) CloseListener() {
	if r.configListener != nil {
		r.configListener.Close()
	}
}

func (r *zkRegistry) registerTempZookeeperNode(root string, node string) error {
	var (
		err    error
		zkPath string
	)

	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	if r.client == nil {
		return perrors.WithStack(perrors.New("zk client already been closed"))
	}
	err = r.client.Create(root)
	if err != nil {
		logger.Errorf("zk.Create(root{%s}) = err{%v}", root, perrors.WithStack(err))
		return perrors.WithStack(err)
	}
	zkPath, err = r.client.RegisterTemp(root, node)
	if err != nil {
		if err == zk.ErrNodeExists {
			logger.Warnf("RegisterTempNode(root{%s}, node{%s}) = error{%v}", root, node, perrors.WithStack(err))
		} else {
			logger.Errorf("RegisterTempNode(root{%s}, node{%s}) = error{%v}", root, node, perrors.WithStack(err))
		}
		return perrors.WithMessagef(err, "RegisterTempNode(root{%s}, node{%s})", root, node)
	}
	logger.Debugf("create a zookeeper node:%s", zkPath)

	return nil
}

func (r *zkRegistry) getListener(conf *common.URL) (*RegistryConfigurationListener, error) {
	var (
		zkListener *RegistryConfigurationListener
	)

	r.listenerLock.Lock()
	if r.configListener.isClosed {
		r.listenerLock.Unlock()
		return nil, perrors.New("configListener already been closed")
	}
	zkListener = r.configListener
	r.listenerLock.Unlock()
	if r.listener == nil {
		r.cltLock.Lock()
		client := r.client
		r.cltLock.Unlock()
		if client == nil {
			return nil, perrors.New("zk connection broken")
		}

		// new client & listener
		listener := zookeeper.NewZkEventListener(r.client)

		r.listenerLock.Lock()
		r.listener = listener
		r.listenerLock.Unlock()
	}

	//Interested register to dataconfig.
	r.dataListener.AddInterestedURL(conf)
	for _, v := range strings.Split(conf.GetParam(constant.CATEGORY_KEY, constant.DEFAULT_CATEGORY), ",") {
		go r.listener.ListenServiceEvent(fmt.Sprintf("/dubbo/%s/"+v, url.QueryEscape(conf.Service())), r.dataListener)
	}

	return zkListener, nil
}
