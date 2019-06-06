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
	"github.com/apache/dubbo-go/common/logger"
	perrors "github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/utils"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/version"
)

const (
	defaultTimeout    = int64(10e9)
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
	client   *zookeeperClient
	services map[string]common.URL // service name + protocol -> service config

	listenerLock sync.Mutex
	listener     *zkEventListener

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

	//if r.SubURL.Name == "" {
	//	r.SubURL.Name = RegistryZkClient
	//}
	//if r.Version == "" {
	//	r.Version = version.Version
	//}

	err = r.validateZookeeperClient()
	if err != nil {
		return nil, err
	}

	r.wg.Add(1)
	go r.handleZkRestart()

	//if r.RoleType == registry.CONSUMER {
	//	r.wg.Add(1)
	//	go r.listen()
	//}

	return r, nil
}

func newMockZkRegistry(url *common.URL) (*zk.TestCluster, *zkRegistry, error) {
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

	c, r.client, _, err = newMockZookeeperClient("test", 15*time.Second)
	if err != nil {
		return nil, nil, err
	}

	r.wg.Add(1)
	go r.handleZkRestart()

	//if r.RoleType == registry.CONSUMER {
	//	r.wg.Add(1)
	//	go r.listen()
	//}

	return c, r, nil
}
func (r *zkRegistry) GetUrl() common.URL {
	return *r.URL
}

func (r *zkRegistry) Destroy() {
	if r.listener != nil {
		r.listener.Close()
	}
	close(r.done)
	r.wg.Wait()
	r.closeRegisters()
}

func (r *zkRegistry) validateZookeeperClient() error {
	var (
		err error
	)

	err = nil
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	if r.client == nil {
		//in dubbp ,every registry only connect one node ,so this is []string{r.Address}
		timeout, err := time.ParseDuration(r.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
		if err != nil {
			logger.Errorf("timeout config %v is invalid ,err is %v",
				r.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT), err.Error())
			return perrors.WithMessagef(err, "newZookeeperClient(address:%+v)", r.Location)
		}
		r.client, err = newZookeeperClient(RegistryZkClient, []string{r.Location}, timeout)
		if err != nil {
			logger.Warnf("newZookeeperClient(name{%s}, zk addresss{%v}, timeout{%d}) = error{%v}",
				RegistryZkClient, r.Location, timeout.String(), err)
			return perrors.WithMessagef(err, "newZookeeperClient(address:%+v)", r.Location)
		}
	}
	if r.client.conn == nil {
		var event <-chan zk.Event
		r.client.conn, event, err = zk.Connect(r.client.zkAddrs, r.client.timeout)
		if err == nil {
			r.client.wait.Add(1)
			go r.client.handleZkEvent(event)
		}
	}

	return perrors.WithMessagef(err, "newZookeeperClient(address:%+v)", r.PrimitiveURL)
}

func (r *zkRegistry) handleZkRestart() {
	var (
		err       error
		flag      bool
		failTimes int
		confIf    common.URL
	)

	defer r.wg.Done()
LOOP:
	for {
		select {
		case <-r.done:
			logger.Warnf("(ZkProviderRegistry)reconnectZkRegistry goroutine exit now...")
			break LOOP
			// re-register all services
		case <-r.client.done():
			r.cltLock.Lock()
			r.client.Close()
			r.client = nil
			r.cltLock.Unlock()

			// 接zk，直至成功
			failTimes = 0
			for {
				select {
				case <-r.done:
					logger.Warnf("(ZkProviderRegistry)reconnectZkRegistry goroutine exit now...")
					break LOOP
				case <-time.After(time.Duration(1e9 * failTimes * RegistryConnDelay)): // 防止疯狂重连zk
				}
				err = r.validateZookeeperClient()
				logger.Infof("ZkProviderRegistry.validateZookeeperClient(zkAddr{%s}) = error{%#v}",
					r.client.zkAddrs, perrors.WithStack(err))
				if err == nil {
					// copy r.services
					services := []common.URL{}
					for _, confIf = range r.services {
						services = append(services, confIf)
					}

					flag = true
					for _, confIf = range services {
						err = r.register(confIf)
						if err != nil {
							logger.Errorf("(ZkProviderRegistry)register(conf{%#v}) = error{%#v}",
								confIf, perrors.WithStack(err))
							flag = false
							break
						}
						logger.Infof("success to re-register service :%v", confIf.Key())
					}
					if flag {
						break
					}
				}
				failTimes++
				if MaxFailTimes <= failTimes {
					failTimes = MaxFailTimes
				}
			}
		}
	}
}

func (r *zkRegistry) Register(conf common.URL) error {
	var (
		ok       bool
		err      error
		listener *zkEventListener
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

		r.listenerLock.Lock()
		listener = r.listener
		r.listenerLock.Unlock()
		if listener != nil {
			go listener.listenServiceEvent(conf)
		}
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

	err = r.validateZookeeperClient()
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
	r.wg.Add(1)
	return r.getListener(conf)
}

func (r *zkRegistry) getListener(conf common.URL) (*zkEventListener, error) {
	var (
		zkListener *zkEventListener
	)

	r.listenerLock.Lock()
	zkListener = r.listener
	r.listenerLock.Unlock()
	if zkListener != nil {
		return zkListener, nil
	}

	r.cltLock.Lock()
	client := r.client
	r.cltLock.Unlock()
	if client == nil {
		return nil, perrors.New("zk connection broken")
	}

	// new client & listener
	zkListener = newZkEventListener(r, client)

	r.listenerLock.Lock()
	r.listener = zkListener
	r.listenerLock.Unlock()

	// listen
	r.cltLock.Lock()
	for _, svs := range r.services {
		if svs.URLEqual(conf) {
			go zkListener.listenServiceEvent(svs)
		}
	}
	r.cltLock.Unlock()

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
