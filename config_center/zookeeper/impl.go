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
	"encoding/base64"
	"strconv"
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"

	gxset "github.com/dubbogo/gost/container/set"
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
	"dubbo.apache.org/dubbo-go/v3/remoting/zookeeper"
)

const (
	pathSeparator = "/"
)

type zookeeperDynamicConfiguration struct {
	config_center.BaseDynamicConfiguration
	url      *common.URL
	rootPath string
	wg       sync.WaitGroup
	cltLock  sync.Mutex
	done     chan struct{}
	client   *gxzookeeper.ZookeeperClient

	// listenerLock  sync.Mutex
	listener      *zookeeper.ZkEventListener
	cacheListener *CacheListener
	parser        parser.ConfigurationParser

	base64Enabled bool
}

func newZookeeperDynamicConfiguration(url *common.URL) (*zookeeperDynamicConfiguration, error) {
	c := &zookeeperDynamicConfiguration{
		url: url,
		// TODO adapt config center config
		rootPath: "/dubbo/config",
	}
	logger.Infof("[Zookeeper ConfigCenter] New Zookeeper ConfigCenter with Configuration: %+v, url = %+v", c, c.GetURL())
	if v, ok := config.GetRootConfig().ConfigCenter.Params["base64"]; ok {
		base64Enabled, err := strconv.ParseBool(v)
		if err != nil {
			panic("value of base64 must be bool, error=" + err.Error())
		}
		c.base64Enabled = base64Enabled
	}

	err := zookeeper.ValidateZookeeperClient(c, url.Location)
	if err != nil {
		logger.Errorf("zookeeper client start error ,error message is %v", err)
		return nil, err
	}
	err = c.client.Create(c.rootPath)
	if err != nil && err != zk.ErrNodeExists {
		return nil, err
	}

	// Before handle client restart, we need to ensure that the zk dynamic configuration successfully start and create the configuration directory
	c.wg.Add(1)
	go zookeeper.HandleClientRestart(c)

	// Start listener
	c.listener = zookeeper.NewZkEventListener(c.client)
	c.cacheListener = NewCacheListener(c.rootPath, c.listener)
	c.listener.ListenConfigurationEvent(c.rootPath, c.cacheListener)
	return c, nil
}

// AddListener add listener for key
// TODO this method should has a parameter 'group', and it does not now, so we should concat group and key with '/' manually
func (c *zookeeperDynamicConfiguration) AddListener(key string, listener config_center.ConfigurationListener, options ...config_center.Option) {
	qualifiedKey := buildPath(c.rootPath, key)
	c.cacheListener.AddListener(qualifiedKey, listener)
}

// buildPath build path and format
func buildPath(rootPath, subPath string) string {
	path := strings.TrimRight(rootPath+pathSeparator+subPath, pathSeparator)
	if !strings.HasPrefix(path, pathSeparator) {
		path = pathSeparator + path
	}
	path = strings.ReplaceAll(path, "//", "/")
	return path
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
		key = c.GetURL().GetParam(constant.ConfigNamespaceKey, config_center.DefaultGroup) + "/" + key
	}
	content, _, err := c.client.GetContent(c.rootPath + "/" + key)
	if err != nil {
		return "", perrors.WithStack(err)
	}
	if !c.base64Enabled {
		return string(content), nil
	}

	decoded, err := base64.StdEncoding.DecodeString(string(content))
	if err != nil {
		return "", perrors.WithStack(err)
	}
	return string(decoded), nil
}

// GetInternalProperty For zookeeper, getConfig and getConfigs have the same meaning.
func (c *zookeeperDynamicConfiguration) GetInternalProperty(key string, opts ...config_center.Option) (string, error) {
	return c.GetProperties(key, opts...)
}

// PublishConfig will put the value into Zk with specific path
func (c *zookeeperDynamicConfiguration) PublishConfig(key string, group string, value string) error {
	path := c.getPath(key, group)
	valueBytes := []byte(value)
	if c.base64Enabled {
		valueBytes = []byte(base64.StdEncoding.EncodeToString(valueBytes))
	}
	// FIXME this method need to be fixed, because it will recursively
	// create every node in the path with given value which we may not expected.
	err := c.client.CreateWithValue(path, valueBytes)
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

func (c *zookeeperDynamicConfiguration) ZkClient() *gxzookeeper.ZookeeperClient {
	return c.client
}

func (c *zookeeperDynamicConfiguration) SetZkClient(client *gxzookeeper.ZookeeperClient) {
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

func (c *zookeeperDynamicConfiguration) GetURL() *common.URL {
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
	logger.Infof("begin to close provider zk client")
	c.cltLock.Lock()
	defer c.cltLock.Unlock()
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
		group = config_center.DefaultGroup
	}
	return c.rootPath + pathSeparator + group
}
