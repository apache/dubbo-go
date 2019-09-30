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
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/config_center/parser"
	"github.com/apache/dubbo-go/remoting/zookeeper"
)

const ZkClient = "zk config_center"

type zookeeperDynamicConfiguration struct {
	url      *common.URL
	rootPath string
	wg       sync.WaitGroup
	cltLock  sync.Mutex
	done     chan struct{}
	client   *zookeeper.ZookeeperClient

	listenerLock  sync.Mutex
	listener      *zookeeper.ZkEventListener
	cacheListener *CacheListener
	parser        parser.ConfigurationParser
}

func newZookeeperDynamicConfiguration(url *common.URL) (*zookeeperDynamicConfiguration, error) {
	c := &zookeeperDynamicConfiguration{
		url:      url,
		rootPath: "/" + url.GetParam(constant.CONFIG_NAMESPACE_KEY, config_center.DEFAULT_GROUP) + "/config",
	}
	err := zookeeper.ValidateZookeeperClient(c, zookeeper.WithZkName(ZkClient))
	if err != nil {
		logger.Errorf("zookeeper client start error ,error message is %v", err)
		return nil, err
	}
	c.wg.Add(1)
	go zookeeper.HandleClientRestart(c)

	c.listener = zookeeper.NewZkEventListener(c.client)
	c.cacheListener = NewCacheListener(c.rootPath)

	err = c.client.Create(c.rootPath)
	c.listener.ListenServiceEvent(c.rootPath, c.cacheListener)
	return c, err

}

func newMockZookeeperDynamicConfiguration(url *common.URL, opts ...zookeeper.Option) (*zk.TestCluster, *zookeeperDynamicConfiguration, error) {
	c := &zookeeperDynamicConfiguration{
		url:      url,
		rootPath: "/" + url.GetParam(constant.CONFIG_NAMESPACE_KEY, config_center.DEFAULT_GROUP) + "/config",
	}
	var (
		tc  *zk.TestCluster
		err error
	)
	tc, c.client, _, err = zookeeper.NewMockZookeeperClient("test", 15*time.Second, opts...)
	if err != nil {
		logger.Errorf("mock zookeeper client start error ,error message is %v", err)
		return tc, c, err
	}
	c.wg.Add(1)
	go zookeeper.HandleClientRestart(c)

	c.listener = zookeeper.NewZkEventListener(c.client)
	c.cacheListener = NewCacheListener(c.rootPath)

	err = c.client.Create(c.rootPath)
	go c.listener.ListenServiceEvent(c.rootPath, c.cacheListener)
	return tc, c, err

}

func (c *zookeeperDynamicConfiguration) AddListener(key string, listener config_center.ConfigurationListener, opions ...config_center.Option) {
	c.cacheListener.AddListener(key, listener)
}

func (c *zookeeperDynamicConfiguration) RemoveListener(key string, listener config_center.ConfigurationListener, opions ...config_center.Option) {
	c.cacheListener.RemoveListener(key, listener)
}

func (c *zookeeperDynamicConfiguration) GetConfig(key string, opts ...config_center.Option) (string, error) {

	tmpOpts := &config_center.Options{}
	for _, opt := range opts {
		opt(tmpOpts)
	}
	/**
	 * when group is not null, we are getting startup configs from Config Center, for example:
	 * group=dubbo, key=dubbo.properties
	 */
	if len(tmpOpts.Group) != 0 {
		key = tmpOpts.Group + "/" + key
	} else {

		/**
		 * when group is null, we are fetching governance rules, for example:
		 * 1. key=org.apache.dubbo.DemoService.configurators
		 * 2. key = org.apache.dubbo.DemoService.condition-router
		 */
		i := strings.LastIndex(key, ".")
		key = key[0:i] + "/" + key[i+1:]
	}
	content, _, err := c.client.GetContent(c.rootPath + "/" + key)
	if err != nil {
		return "", perrors.WithStack(err)
	} else {
		return string(content), nil
	}

}

//For zookeeper, getConfig and getConfigs have the same meaning.
func (c *zookeeperDynamicConfiguration) GetConfigs(key string, opts ...config_center.Option) (string, error) {
	return c.GetConfig(key, opts...)
}

func (c *zookeeperDynamicConfiguration) Parser() parser.ConfigurationParser {
	return c.parser
}
func (c *zookeeperDynamicConfiguration) SetParser(p parser.ConfigurationParser) {
	c.parser = p
}

func (r *zookeeperDynamicConfiguration) ZkClient() *zookeeper.ZookeeperClient {
	return r.client
}

func (r *zookeeperDynamicConfiguration) SetZkClient(client *zookeeper.ZookeeperClient) {
	r.client = client
}

func (r *zookeeperDynamicConfiguration) ZkClientLock() *sync.Mutex {
	return &r.cltLock
}

func (r *zookeeperDynamicConfiguration) WaitGroup() *sync.WaitGroup {
	return &r.wg
}

func (r *zookeeperDynamicConfiguration) GetDone() chan struct{} {
	return r.done
}

func (r *zookeeperDynamicConfiguration) GetUrl() common.URL {
	return *r.url
}

func (r *zookeeperDynamicConfiguration) Destroy() {
	if r.listener != nil {
		r.listener.Close()
	}
	close(r.done)
	r.wg.Wait()
	r.closeConfigs()
}

func (r *zookeeperDynamicConfiguration) IsAvailable() bool {
	select {
	case <-r.done:
		return false
	default:
		return true
	}
}

func (r *zookeeperDynamicConfiguration) closeConfigs() {
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	logger.Infof("begin to close provider zk client")
	// Close the old client first to close the tmp node
	r.client.Close()
	r.client = nil
}

func (r *zookeeperDynamicConfiguration) RestartCallBack() bool {
	return true
}
