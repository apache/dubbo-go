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

package apollo

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
	gxset "github.com/dubbogo/gost/container/set"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	perrors "github.com/pkg/errors"

	cc "dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
	"github.com/dubbogo/gost/log/logger"
)

const (
	apolloProtocolPrefix = "http://"
)

type apolloConfiguration struct {
	cc.BaseDynamicConfiguration
	url *common.URL

	listeners sync.Map
	appConf   *config.AppConfig
	parser    parser.ConfigurationParser
	client    agollo.Client
}

func newApolloConfiguration(url *common.URL) (*apolloConfiguration, error) {
	c := &apolloConfiguration{
		url: url,
	}
	c.appConf = &config.AppConfig{
		AppID:            url.GetParam(constant.ConfigAppIDKey, ""),
		Cluster:          url.GetParam(constant.ConfigClusterKey, ""),
		NamespaceName:    url.GetParam(constant.ConfigNamespaceKey, cc.DefaultGroup),
		IP:               c.getAddressWithProtocolPrefix(url),
		Secret:           url.GetParam(constant.ConfigSecretKey, ""),
		IsBackupConfig:   url.GetParamBool(constant.ConfigBackupConfigKey, true),
		BackupConfigPath: url.GetParam(constant.ConfigBackupConfigPathKey, ""),
	}
	logger.Infof("[Apollo ConfigCenter] New Apollo ConfigCenter with Configuration: %+v, url = %+v", c.appConf, c.url)
	client, err := agollo.StartWithConfig(func() (*config.AppConfig, error) {
		return c.appConf, nil
	})
	if err != nil {
		return nil, perrors.Wrap(err, "start apollo client failed")
	}
	c.client = client
	return c, nil
}

func (c *apolloConfiguration) AddListener(key string, listener cc.ConfigurationListener, opts ...cc.Option) {
	tmpOpts := cc.NewOptions(opts...)
	key = tmpOpts.Center.Group + key
	al := newApolloListener()
	l, exist := c.listeners.LoadOrStore(key, al)
	if !exist {
		c.client.AddChangeListener(al)
	}
	l.(*apolloListener).AddListener(listener)
}

func (c *apolloConfiguration) RemoveListener(key string, listener cc.ConfigurationListener, opts ...cc.Option) {
	tmpOpts := cc.NewOptions(opts...)
	key = tmpOpts.Center.Group + key
	l, ok := c.listeners.Load(key)
	if ok {
		l.(*apolloListener).RemoveListener(listener)
	}
}

func (c *apolloConfiguration) GetInternalProperty(key string, opts ...cc.Option) (string, error) {
	if c.client == nil {
		return "", perrors.New("apollo client is not initialized")
	}
	cache := c.client.GetConfigCache(c.appConf.NamespaceName)
	if cache == nil {
		return "", perrors.New(fmt.Sprintf("nothing in namespace:%s ", key))
	}
	value, err := cache.Get(key)
	if err != nil {
		return "", perrors.Wrap(err, "get config value failed")
	}
	return value.(string), nil
}

func (c *apolloConfiguration) GetRule(key string, opts ...cc.Option) (string, error) {
	return c.GetInternalProperty(key, opts...)
}

// PublishConfig will publish the config with the (key, group, value) pair
func (c *apolloConfiguration) PublishConfig(string, string, string) error {
	return perrors.New("unsupport operation")
}

// GetConfigKeysByGroup will return all keys with the group
func (c *apolloConfiguration) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
	return nil, perrors.New("unsupport operation")
}

func (c *apolloConfiguration) GetProperties(key string, opts ...cc.Option) (string, error) {
	if c.client == nil {
		return "", perrors.New("apollo client is not initialized")
	}
	if key == "" {
		key = c.appConf.NamespaceName
	}
	tmpConfig := c.client.GetConfig(key)
	if tmpConfig == nil {
		return "", perrors.New(fmt.Sprintf("nothing in namespace:%s ", key))
	}
	content := tmpConfig.GetContent()
	b := []byte(content)
	if len(b) == 0 {
		return "", perrors.New(fmt.Sprintf("nothing in namespace:%s ", key))
	}
	return content, nil
}

func (c *apolloConfiguration) getAddressWithProtocolPrefix(url *common.URL) string {
	address := url.Location
	converted := address
	if len(address) != 0 {
		addr := regexp.MustCompile(`\s+`).ReplaceAllString(address, "")
		parts := strings.Split(addr, ",")
		addrs := make([]string, 0)
		for _, part := range parts {
			addr := part
			if !strings.HasPrefix(part, apolloProtocolPrefix) {
				addr = apolloProtocolPrefix + part
			}
			addrs = append(addrs, addr)
		}
		converted = strings.Join(addrs, ",")
	}
	return converted
}

func (c *apolloConfiguration) Parser() parser.ConfigurationParser {
	return c.parser
}

func (c *apolloConfiguration) SetParser(p parser.ConfigurationParser) {
	c.parser = p
}
