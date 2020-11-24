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
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/config_center/parser"
	"github.com/apache/dubbo-go/remoting/zookeeper"
)

const (
	// ZkClient
	// zookeeper client name
	ZkClient      = "zk config_center"
	pathSeparator = "/"
)

type zookeeperDynamicConfiguration struct {
	config_center.BaseDynamicConfiguration
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
	c.listener.ListenServiceEvent(url, c.rootPath, c.cacheListener)
	return c, err

}

func (c *zookeeperDynamicConfiguration) AddListener(key string, listener config_center.ConfigurationListener, opions ...config_center.Option) {
	c.cacheListener.AddListener(key, listener)
}

func (c *zookeeperDynamicConfiguration) RemoveListener(key string, listener config_center.ConfigurationListener, opions ...config_center.Option) {
	c.cacheListener.RemoveListener(key, listener)
}

func (c *zookeeperDynamicConfiguration) GetProperties(key string, opts ...config_center.Option) (string, error) {

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
	}

	return string(content), nil
}

// GetInternalProperty For zookeeper, getConfig and getConfigs have the same meaning.
func (c *zookeeperDynamicConfiguration) GetInternalProperty(key string, opts ...config_center.Option) (string, error) {
	return c.GetProperties(key, opts...)
}

// PublishConfig will put the value into Zk with specific path
func (c *zookeeperDynamicConfiguration) PublishConfig(key string, group string, value string) error {
	path := c.getPath(key, group)
	err := c.client.CreateWithValue(path, []byte(value))
	if err != nil {
		return perrors.WithStack(err)
	}
	return nil
}

// GetConfigKeysByGroup will return all keys with the group
func (c *zookeeperDynamicConfiguration) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
	path := c.getPath("", group)
	result, err := c.client.GetChildren(path)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	if len(result) == 0 {
		return nil, perrors.New("could not find keys with group: " + group)
	}
	set := gxset.NewSet()
	for _, e := range result {
		set.Add(e)
	}
	return set, nil
}

func (c *zookeeperDynamicConfiguration) GetRule(key string, opts ...config_center.Option) (string, error) {
	return c.GetProperties(key, opts...)
}

func (c *zookeeperDynamicConfiguration) Parser() parser.ConfigurationParser {
	return c.parser
}

func (c *zookeeperDynamicConfiguration) SetParser(p parser.ConfigurationParser) {
	c.parser = p
}

func (c *zookeeperDynamicConfiguration) ZkClient() *zookeeper.ZookeeperClient {
	return c.client
}

func (c *zookeeperDynamicConfiguration) SetZkClient(client *zookeeper.ZookeeperClient) {
	c.client = client
}

func (c *zookeeperDynamicConfiguration) ZkClientLock() *sync.Mutex {
	return &c.cltLock
}

func (c *zookeeperDynamicConfiguration) WaitGroup() *sync.WaitGroup {
	return &c.wg
}

func (c *zookeeperDynamicConfiguration) Done() chan struct{} {
	return c.done
}

func (c *zookeeperDynamicConfiguration) GetUrl() *common.URL {
	return c.url
}

func (c *zookeeperDynamicConfiguration) Destroy() {
	if c.listener != nil {
		c.listener.Close()
	}
	close(c.done)
	c.wg.Wait()
	c.closeConfigs()
}

func (c *zookeeperDynamicConfiguration) IsAvailable() bool {
	select {
	case <-c.done:
		return false
	default:
		return true
	}
}

func (c *zookeeperDynamicConfiguration) closeConfigs() {
	c.cltLock.Lock()
	defer c.cltLock.Unlock()
	logger.Infof("begin to close provider zk client")
	// Close the old client first to close the tmp node
	c.client.Close()
	c.client = nil
}

func (c *zookeeperDynamicConfiguration) RestartCallBack() bool {
	return true
}

func (c *zookeeperDynamicConfiguration) getPath(key string, group string) string {
	if len(key) == 0 {
		return c.buildPath(group)
	}
	return c.buildPath(group) + pathSeparator + key
}

func (c *zookeeperDynamicConfiguration) buildPath(group string) string {
	if len(group) == 0 {
		group = config_center.DEFAULT_GROUP
	}
	return c.rootPath + pathSeparator + group
}
