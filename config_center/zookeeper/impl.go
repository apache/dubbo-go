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
	"errors"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
	"github.com/dubbogo/gost/log/logger"

	"github.com/go-zookeeper/zk"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
	"dubbo.apache.org/dubbo-go/v3/remoting/zookeeper"
)

const (
	pathSeparator         = "/"
	defaultConfigCacheTTL = 30 * time.Second
	minConfigCacheTTL     = time.Second
	maxConfigCacheTTL     = 10 * time.Minute
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
	cache         configCache
	parser        parser.ConfigurationParser

	base64Enabled bool
}

func newZookeeperDynamicConfiguration(url *common.URL) (*zookeeperDynamicConfiguration, error) {
	rootPath := url.GetParam(constant.ConfigRootPathParamKey, "/dubbo/config")
	cacheTTL, err := parseConfigCacheTTL(url.GetParam(constant.ConfigCacheTTLKey, ""))
	if err != nil {
		return nil, err
	}
	c := &zookeeperDynamicConfiguration{
		url:      url,
		rootPath: rootPath,
		cache:    newConfigCache(cacheTTL),
	}
	logger.Infof("[ConfigCenter][Zookeeper] new Zookeeper ConfigCenter with Configuration, zkConfig=%v url=%v", c, c.GetURL())
	if v := url.GetParam("base64", ""); v != "" {
		base64Enabled, parseErr := strconv.ParseBool(v)
		if parseErr != nil {
			panic("value of base64 must be bool, error=" + parseErr.Error())
		}
		c.base64Enabled = base64Enabled
	}

	err = zookeeper.ValidateZookeeperClient(c, url.Location)
	if err != nil {
		logger.Errorf("[ConfigCenter][Zookeeper] zookeeper client start error, err=%v", err)
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
	c.cacheListener = newCacheListener(c.rootPath, c.listener, &c.cache)
	c.listener.ListenConfigurationEvent(c.rootPath, c.cacheListener)
	return c, nil
}

// AddListener add listener for key
func (c *zookeeperDynamicConfiguration) AddListener(key string, listener config_center.ConfigurationListener, options ...config_center.Option) {
	qualifiedKey := c.getPropertiesPath(key, options...)
	c.cacheListener.AddListener(qualifiedKey, listener)
}

// buildPath build path and format
func buildPath(rootPath, subPath string) string {
	fullPath := strings.TrimRight(rootPath+pathSeparator+subPath, pathSeparator)
	if !strings.HasPrefix(fullPath, pathSeparator) {
		fullPath = pathSeparator + fullPath
	}

	return path.Clean(fullPath)
}

func (c *zookeeperDynamicConfiguration) RemoveListener(key string, listener config_center.ConfigurationListener, options ...config_center.Option) {
	qualifiedKey := c.getPropertiesPath(key, options...)
	c.cacheListener.RemoveListener(qualifiedKey, listener)
}

func (c *zookeeperDynamicConfiguration) GetProperties(key string, opts ...config_center.Option) (string, error) {
	path := c.getPropertiesPath(key, opts...)
	entry, err := c.cache.load(path, func(registerWatch bool) (configCacheEntry, watchRegistration, error) {
		return c.loadProperties(path, registerWatch)
	})
	if err != nil {
		return "", err
	}
	if !entry.exists {
		return "", nil
	}
	if !c.base64Enabled {
		return entry.content, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(entry.content)
	if err != nil {
		return "", perrors.WithStack(err)
	}
	return string(decoded), nil
}

func (c *zookeeperDynamicConfiguration) getPropertiesPath(key string, opts ...config_center.Option) string {
	tmpOpts := config_center.NewOptions(opts...)
	/**
	 * when group is not null, we are getting startup configs from Config Center, for example:
	 * group=dubbo, key=dubbo.properties
	 */
	if len(tmpOpts.Center.Group) != 0 {
		key = tmpOpts.Center.Group + "/" + key
	} else {
		key = c.GetURL().GetParam(constant.ConfigNamespaceKey, config_center.DefaultGroup) + "/" + key
	}
	return buildPath(c.rootPath, key)
}

func (c *zookeeperDynamicConfiguration) loadProperties(path string, registerWatch bool) (configCacheEntry, watchRegistration, error) {
	if !c.cache.enabled() || !registerWatch {
		beforeSessionID := c.client.Conn.SessionID()
		content, _, err := c.client.GetContent(path)
		afterSessionID := c.client.Conn.SessionID()
		registration := watchRegistration{
			beforeSessionID: beforeSessionID,
			afterSessionID:  afterSessionID,
		}
		if errors.Is(err, zk.ErrNoNode) {
			logger.Warnf("[ConfigCenter][Zookeeper] query rule fail, key=%s err=%v", path, err)
			return configCacheEntry{exists: false}, registration, nil
		}
		if err != nil {
			return configCacheEntry{}, registration, perrors.WithStack(err)
		}
		return configCacheEntry{content: string(content), exists: true}, registration, nil
	}

	for {
		beforeSessionID := c.client.Conn.SessionID()
		content, _, events, err := c.client.Conn.GetW(path)
		registration := watchRegistration{
			events:          events,
			beforeSessionID: beforeSessionID,
			afterSessionID:  c.client.Conn.SessionID(),
		}
		if err == nil {
			return configCacheEntry{content: string(content), exists: true}, registration, nil
		}
		if !errors.Is(err, zk.ErrNoNode) {
			return configCacheEntry{}, watchRegistration{}, perrors.WithStack(err)
		}

		beforeSessionID = c.client.Conn.SessionID()
		exists, _, events, watchErr := c.client.Conn.ExistsW(path)
		registration = watchRegistration{
			events:          events,
			beforeSessionID: beforeSessionID,
			afterSessionID:  c.client.Conn.SessionID(),
		}
		if watchErr != nil {
			return configCacheEntry{}, watchRegistration{}, perrors.WithStack(watchErr)
		}
		if !exists {
			logger.Warnf("[ConfigCenter][Zookeeper] query rule fail, key=%s err=%v", path, err)
			return configCacheEntry{exists: false}, registration, nil
		}

		readBeforeSessionID := c.client.Conn.SessionID()
		content, _, getErr := c.client.Conn.Get(path)
		readAfterSessionID := c.client.Conn.SessionID()
		registration.readBeforeSessionID = readBeforeSessionID
		registration.readAfterSessionID = readAfterSessionID
		if errors.Is(getErr, zk.ErrNoNode) {
			continue
		}
		if getErr != nil {
			return configCacheEntry{}, registration, perrors.WithStack(getErr)
		}
		return configCacheEntry{content: string(content), exists: true}, registration, nil
	}
}

func parseConfigCacheTTL(value string) (time.Duration, error) {
	if value == "" {
		return defaultConfigCacheTTL, nil
	}
	ttl, err := time.ParseDuration(value)
	if err != nil {
		return 0, perrors.Wrapf(err, "invalid %s value %q", constant.ConfigCacheTTLKey, value)
	}
	if ttl < 0 {
		return 0, perrors.Errorf("%s must not be negative", constant.ConfigCacheTTLKey)
	}
	if ttl > 0 && (ttl < minConfigCacheTTL || ttl > maxConfigCacheTTL) {
		return 0, perrors.Errorf("%s must be 0 or between %s and %s", constant.ConfigCacheTTLKey, minConfigCacheTTL, maxConfigCacheTTL)
	}
	return ttl, nil
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
		// try update value if node already exists
		if errors.Is(err, zk.ErrNodeExists) {
			_, stat, _ := c.client.GetContent(path)
			_, setErr := c.client.SetContent(path, valueBytes, stat.Version)
			if setErr != nil {
				return perrors.WithStack(setErr)
			}
			return nil
		}
		return perrors.WithStack(err)
	}
	return nil
}

// RemoveConfig will remove the config with the (key, group) pair
func (c *zookeeperDynamicConfiguration) RemoveConfig(key string, group string) error {
	fullPath := c.getPath(key, group)
	err := c.client.Delete(fullPath)
	if err != nil {
		return perrors.WithStack(err)
	}
	return nil
}

// GetConfigKeysByGroup will return all keys with the group
func (c *zookeeperDynamicConfiguration) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
	fullPath := c.getPath("", group)
	result, err := c.client.GetChildren(fullPath)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	if len(result) == 0 {
		return nil, errors.New("could not find keys with group: " + group)
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
	logger.Info("[ConfigCenter][Zookeeper] begin to close provider zk client")
	c.cltLock.Lock()
	defer c.cltLock.Unlock()
	c.client.Close()
	c.client = nil
}

func (c *zookeeperDynamicConfiguration) RestartCallBack() bool {
	var sessionID int64
	if c.client != nil && c.client.Conn != nil {
		sessionID = c.client.Conn.SessionID()
	}
	c.cache.reset(sessionID)
	if c.cacheListener != nil {
		c.cacheListener.restoreBusinessWatches()
	}
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
