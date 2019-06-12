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
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	perrors "github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/common/utils"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting/zookeeper"
	"github.com/apache/dubbo-go/version"
)

const (
	RegistryZkClient = "zk registry"
)

var (
	processID = ""
	localIP   = ""
)

func init() {
	processID = fmt.Sprintf("%d", os.Getpid())
	localIP, _ = utils.GetLocalIP()
	//plugins.PluggableRegistries["zookeeper"] = newZkRegistry
	extension.SetRegistry("zookeeper", newZkRegistry)
}

/////////////////////////////////////
// zookeeper registry
/////////////////////////////////////

type zkRegistry struct {
	context context.Context
	*common.URL
	birth int64          // time of file birth, seconds since Epoch; 0 if unknown
	wg    sync.WaitGroup // wg+done for zk restart
	done  chan struct{}

	cltLock  sync.Mutex
	client   *zookeeper.ZookeeperClient
	services map[string]common.URL // service name + protocol -> service config

	listenerLock   sync.Mutex
	listener       *zookeeper.ZkEventListener
	dataListener   *RegistryDataListener
	configListener *RegistryConfigurationListener
	//for provider
	zkPath map[string]int // key = protocol://ip:port/interface

}

func newZkRegistry(url *common.URL) (registry.Registry, error) {
	var (
		err error
		r   *zkRegistry
	)

	r = &zkRegistry{
		URL:      url,
		birth:    time.Now().UnixNano(),
		done:     make(chan struct{}),
		services: make(map[string]common.URL),
		zkPath:   make(map[string]int),
	}

	err = zookeeper.ValidateZookeeperClient(r, zookeeper.WithZkName(RegistryZkClient))
	if err != nil {
		return nil, err
	}

	r.wg.Add(1)
	go zookeeper.HandleClientRestart(r)

	r.listener = zookeeper.NewZkEventListener(r.client)
	r.configListener = NewRegistryConfigurationListener(r.client, r)
	r.dataListener = NewRegistryDataListener(r.configListener)

	return r, nil
}

type Options struct {
	client *zookeeper.ZookeeperClient
}

type Option func(*Options)

func newMockZkRegistry(url *common.URL, opts ...zookeeper.Option) (*zk.TestCluster, *zkRegistry, error) {
	var (
		err error
		r   *zkRegistry
		c   *zk.TestCluster
		//event <-chan zk.Event
	)

	r = &zkRegistry{
		URL:      url,
		birth:    time.Now().UnixNano(),
		done:     make(chan struct{}),
		services: make(map[string]common.URL),
		zkPath:   make(map[string]int),
	}

	c, r.client, _, err = zookeeper.NewMockZookeeperClient("test", 15*time.Second, opts...)
	if err != nil {
		return nil, nil, err
	}
	r.wg.Add(1)
	go zookeeper.HandleClientRestart(r)

	r.listener = zookeeper.NewZkEventListener(r.client)
	r.configListener = NewRegistryConfigurationListener(r.client, r)
	r.dataListener = NewRegistryDataListener(r.configListener)

	return c, r, nil
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

func (r *zkRegistry) WaitGroup() *sync.WaitGroup {
	return &r.wg
}

func (r *zkRegistry) GetDone() chan struct{} {
	return r.done
}

func (r *zkRegistry) GetUrl() common.URL {
	return *r.URL
}

func (r *zkRegistry) Destroy() {
	if r.configListener != nil {
		r.configListener.Close()
	}
	close(r.done)
	r.wg.Wait()
	r.closeRegisters()
}

func (r *zkRegistry) RestartCallBack() bool {

	// copy r.services
	services := []common.URL{}
	for _, confIf := range r.services {
		services = append(services, confIf)
	}

	flag := true
	for _, confIf := range services {
		err := r.register(confIf)
		if err != nil {
			logger.Errorf("(ZkProviderRegistry)register(conf{%#v}) = error{%#v}",
				confIf, perrors.WithStack(err))
			flag = false
			break
		}
		logger.Infof("success to re-register service :%v", confIf.Key())
	}
	return flag
}

func (r *zkRegistry) Register(conf common.URL) error {
	var (
		ok  bool
		err error
	)
	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	switch role {
	case common.CONSUMER:
		ok = false
		r.cltLock.Lock()
		_, ok = r.services[conf.Key()]
		r.cltLock.Unlock()
		if ok {
			return perrors.Errorf("Path{%s} has been registered", conf.Path)
		}

		err = r.register(conf)
		if err != nil {
			return perrors.WithStack(err)
		}

		r.cltLock.Lock()
		r.services[conf.Key()] = conf
		r.cltLock.Unlock()
		logger.Debugf("(consumerZkConsumerRegistry)Register(conf{%#v})", conf)

	case common.PROVIDER:

		// 检验服务是否已经注册过
		ok = false
		r.cltLock.Lock()
		// 注意此处与consumerZookeeperRegistry的差异，consumer用的是conf.Path，
		// 因为consumer要提供watch功能给selector使用, provider允许注册同一个service的多个group or version
		_, ok = r.services[conf.Key()]
		r.cltLock.Unlock()
		if ok {
			return perrors.Errorf("Path{%s} has been registered", conf.Key())
		}

		err = r.register(conf)
		if err != nil {
			return perrors.WithMessagef(err, "register(conf:%+v)", conf)
		}

		r.cltLock.Lock()
		r.services[conf.Key()] = conf
		r.cltLock.Unlock()

		logger.Debugf("(ZkProviderRegistry)Register(conf{%#v})", conf)
	}

	return nil
}

func (r *zkRegistry) register(c common.URL) error {
	var (
		err error
		//revision   string
		params     url.Values
		urlPath    string
		rawURL     string
		encodedURL string
		dubboPath  string
		//conf       config.URL
	)

	err = zookeeper.ValidateZookeeperClient(r, zookeeper.WithZkName(RegistryZkClient))
	if err != nil {
		return perrors.WithStack(err)
	}
	params = url.Values{}
	for k, v := range c.Params {
		params[k] = v
	}

	params.Add("pid", processID)
	params.Add("ip", localIP)
	//params.Add("timeout", fmt.Sprintf("%d", int64(r.Timeout)/1e6))

	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	switch role {

	case common.PROVIDER:

		if c.Path == "" || len(c.Methods) == 0 {
			return perrors.Errorf("conf{Path:%s, Methods:%s}", c.Path, c.Methods)
		}
		// 先创建服务下面的provider node
		dubboPath = fmt.Sprintf("/dubbo%s/%s", c.Path, common.DubboNodes[common.PROVIDER])
		r.cltLock.Lock()
		err = r.client.Create(dubboPath)
		r.cltLock.Unlock()
		if err != nil {
			logger.Errorf("zkClient.create(path{%s}) = error{%#v}", dubboPath, perrors.WithStack(err))
			return perrors.WithMessagef(err, "zkclient.Create(path:%s)", dubboPath)
		}
		params.Add("anyhost", "true")

		// dubbo java consumer来启动找provider url时，因为category不匹配，会找不到provider，导致consumer启动不了,所以使用consumers&providers
		// DubboRole               = [...]string{"consumer", "", "", "provider"}
		// params.Add("category", (RoleType(PROVIDER)).Role())
		params.Add("category", (common.RoleType(common.PROVIDER)).String())
		params.Add("dubbo", "dubbo-provider-golang-"+version.Version)

		params.Add("side", (common.RoleType(common.PROVIDER)).Role())

		if len(c.Methods) == 0 {
			params.Add("methods", strings.Join(c.Methods, ","))
		}
		logger.Debugf("provider zk url params:%#v", params)
		var host string
		if c.Ip == "" {
			host = localIP + ":" + c.Port
		} else {
			host = c.Ip + ":" + c.Port
		}

		urlPath = c.Path
		if r.zkPath[urlPath] != 0 {
			urlPath += strconv.Itoa(r.zkPath[urlPath])
		}
		r.zkPath[urlPath]++
		rawURL = fmt.Sprintf("%s://%s%s?%s", c.Protocol, host, urlPath, params.Encode())
		encodedURL = url.QueryEscape(rawURL)

		// 把自己注册service providers
		dubboPath = fmt.Sprintf("/dubbo%s/%s", c.Path, (common.RoleType(common.PROVIDER)).String())
		logger.Debugf("provider path:%s, url:%s", dubboPath, rawURL)

	case common.CONSUMER:
		dubboPath = fmt.Sprintf("/dubbo%s/%s", c.Path, common.DubboNodes[common.CONSUMER])
		r.cltLock.Lock()
		err = r.client.Create(dubboPath)
		r.cltLock.Unlock()
		if err != nil {
			logger.Errorf("zkClient.create(path{%s}) = error{%v}", dubboPath, perrors.WithStack(err))
			return perrors.WithStack(err)
		}
		dubboPath = fmt.Sprintf("/dubbo%s/%s", c.Path, common.DubboNodes[common.PROVIDER])
		r.cltLock.Lock()
		err = r.client.Create(dubboPath)
		r.cltLock.Unlock()
		if err != nil {
			logger.Errorf("zkClient.create(path{%s}) = error{%v}", dubboPath, perrors.WithStack(err))
			return perrors.WithStack(err)
		}

		params.Add("protocol", c.Protocol)

		params.Add("category", (common.RoleType(common.CONSUMER)).String())
		params.Add("dubbo", "dubbogo-consumer-"+version.Version)

		rawURL = fmt.Sprintf("consumer://%s%s?%s", localIP, c.Path, params.Encode())
		encodedURL = url.QueryEscape(rawURL)

		dubboPath = fmt.Sprintf("/dubbo%s/%s", c.Path, (common.RoleType(common.CONSUMER)).String())
		logger.Debugf("consumer path:%s, url:%s", dubboPath, rawURL)

	default:
		return perrors.Errorf("@c{%v} type is not referencer or provider", c)
	}

	err = r.registerTempZookeeperNode(dubboPath, encodedURL)

	if err != nil {
		return perrors.WithMessagef(err, "registerTempZookeeperNode(path:%s, url:%s)", dubboPath, rawURL)
	}
	return nil
}

func (r *zkRegistry) registerTempZookeeperNode(root string, node string) error {
	var (
		err    error
		zkPath string
	)

	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	err = r.client.Create(root)
	if err != nil {
		logger.Errorf("zk.Create(root{%s}) = err{%v}", root, perrors.WithStack(err))
		return perrors.WithStack(err)
	}
	zkPath, err = r.client.RegisterTemp(root, node)
	if err != nil {
		logger.Errorf("RegisterTempNode(root{%s}, node{%s}) = error{%v}", root, node, perrors.WithStack(err))
		return perrors.WithMessagef(err, "RegisterTempNode(root{%s}, node{%s})", root, node)
	}
	logger.Debugf("create a zookeeper node:%s", zkPath)

	return nil
}

func (r *zkRegistry) Subscribe(conf common.URL) (registry.Listener, error) {
	return r.getListener(conf)
}

func (r *zkRegistry) getListener(conf common.URL) (*RegistryConfigurationListener, error) {
	var (
		zkListener *RegistryConfigurationListener
	)

	r.listenerLock.Lock()
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

	//注册到dataconfig的interested
	r.dataListener.AddInterestedURL(&conf)

	go r.listener.ListenServiceEvent(fmt.Sprintf("/dubbo%s/providers", conf.Path), r.dataListener)

	return zkListener, nil
}

func (r *zkRegistry) closeRegisters() {
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	logger.Infof("begin to close provider zk client")
	// 先关闭旧client，以关闭tmp node
	r.client.Close()
	r.client = nil
	r.services = nil
}

func (r *zkRegistry) IsAvailable() bool {
	select {
	case <-r.done:
		return false
	default:
		return true
	}
}
