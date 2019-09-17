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
)

const (
	RegistryZkClient  = "zk registry"
	RegistryConnDelay = 3
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

		// Check if the service has been registered
		r.cltLock.Lock()
		// Note the difference between consumer and consumerZookeeperRegistry (consumer use conf.Path).
		// Because the consumer wants to provide monitoring functions for the selector,
		// the provider allows multiple groups or versions of the same service to be registered.
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

	c.RangeParams(func(key, value string) bool {
		params.Add(key, value)
		return true
	})

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
		dubboPath = fmt.Sprintf("/dubbo/%s/%s", c.Service(), common.DubboNodes[common.PROVIDER])
		r.cltLock.Lock()
		err = r.client.Create(dubboPath)
		r.cltLock.Unlock()
		if err != nil {
			logger.Errorf("zkClient.create(path{%s}) = error{%#v}", dubboPath, perrors.WithStack(err))
			return perrors.WithMessagef(err, "zkclient.Create(path:%s)", dubboPath)
		}
		params.Add("anyhost", "true")

		// Dubbo java consumer to start looking for the provider url,because the category does not match,
		// the provider will not find, causing the consumer can not start, so we use consumers.
		// DubboRole               = [...]string{"consumer", "", "", "provider"}
		// params.Add("category", (RoleType(PROVIDER)).Role())
		params.Add("category", (common.RoleType(common.PROVIDER)).String())
		params.Add("dubbo", "dubbo-provider-golang-"+constant.Version)

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

		rawURL = fmt.Sprintf("%s://%s%s?%s", c.Protocol, host, c.Path, params.Encode())
		encodedURL = url.QueryEscape(rawURL)

		// Print your own registration service providers.
		dubboPath = fmt.Sprintf("/dubbo/%s/%s", c.Service(), (common.RoleType(common.PROVIDER)).String())
		logger.Debugf("provider path:%s, url:%s", dubboPath, rawURL)

	case common.CONSUMER:
		dubboPath = fmt.Sprintf("/dubbo/%s/%s", c.Service(), common.DubboNodes[common.CONSUMER])
		r.cltLock.Lock()
		err = r.client.Create(dubboPath)
		r.cltLock.Unlock()
		if err != nil {
			logger.Errorf("zkClient.create(path{%s}) = error{%v}", dubboPath, perrors.WithStack(err))
			return perrors.WithStack(err)
		}
		dubboPath = fmt.Sprintf("/dubbo/%s/%s", c.Service(), common.DubboNodes[common.PROVIDER])
		r.cltLock.Lock()
		err = r.client.Create(dubboPath)
		r.cltLock.Unlock()
		if err != nil {
			logger.Errorf("zkClient.create(path{%s}) = error{%v}", dubboPath, perrors.WithStack(err))
			return perrors.WithStack(err)
		}

		params.Add("protocol", c.Protocol)

		params.Add("category", (common.RoleType(common.CONSUMER)).String())
		params.Add("dubbo", "dubbogo-consumer-"+constant.Version)

		rawURL = fmt.Sprintf("consumer://%s%s?%s", localIP, c.Path, params.Encode())
		encodedURL = url.QueryEscape(rawURL)

		dubboPath = fmt.Sprintf("/dubbo/%s/%s", c.Service(), (common.RoleType(common.CONSUMER)).String())
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

func (r *zkRegistry) subscribe(conf *common.URL) (registry.Listener, error) {
	return r.getListener(conf)
}

//subscibe from registry
func (r *zkRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) {
	for {
		if !r.IsAvailable() {
			logger.Warnf("event listener game over.")
			return
		}

		listener, err := r.subscribe(url)
		if err != nil {
			if !r.IsAvailable() {
				logger.Warnf("event listener game over.")
				return
			}
			logger.Warnf("getListener() = err:%v", perrors.WithStack(err))
			time.Sleep(time.Duration(RegistryConnDelay) * time.Second)
			continue
		}

		for {
			if serviceEvent, err := listener.Next(); err != nil {
				logger.Warnf("Selector.watch() = error{%v}", perrors.WithStack(err))
				listener.Close()
				return
			} else {
				logger.Infof("update begin, service event: %v", serviceEvent.String())
				notifyListener.Notify(serviceEvent)
			}

		}

	}
}
func (r *zkRegistry) getListener(conf *common.URL) (*RegistryConfigurationListener, error) {
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

	//Interested register to dataconfig.
	r.dataListener.AddInterestedURL(conf)
	for _, v := range strings.Split(conf.GetParam(constant.CATEGORY_KEY, constant.DEFAULT_CATEGORY), ",") {
		go r.listener.ListenServiceEvent(fmt.Sprintf("/dubbo/%s/"+v, conf.Service()), r.dataListener)
	}

	return zkListener, nil
}

func (r *zkRegistry) closeRegisters() {
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	logger.Infof("begin to close provider zk client")
	// Close the old client first to close the tmp node.
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
