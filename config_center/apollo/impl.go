/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apollo

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

import (
	"github.com/go-errors/errors"
	"github.com/zouyx/agollo"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	. "github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/config_center/parser"
	"github.com/apache/dubbo-go/remoting"
)

const (
	apolloProtocolPrefix = "http://"
	apolloConfigFormat   = "%s.%s"
)

type apolloConfiguration struct {
	url *common.URL

	listeners sync.Map
	appConf   *agollo.AppConfig
	parser    parser.ConfigurationParser
}

func newApolloConfiguration(url *common.URL) (*apolloConfiguration, error) {
	c := &apolloConfiguration{
		url: url,
	}
	configAddr := c.getAddressWithProtocolPrefix(url)
	configCluster := url.GetParam(constant.CONFIG_CLUSTER_KEY, "")

	appId := url.GetParam(constant.CONFIG_GROUP_KEY, DEFAULT_GROUP)
	namespaces := url.GetParam(constant.CONFIG_NAMESPACE_KEY, getProperties(DEFAULT_GROUP))
	readyConfig := &agollo.AppConfig{
		AppId:         appId,
		Cluster:       configCluster,
		NamespaceName: namespaces,
		Ip:            configAddr,
	}

	agollo.InitCustomConfig(func() (*agollo.AppConfig, error) {
		return readyConfig, nil
	})

	return c, agollo.Start()
}

func getChangeType(change agollo.ConfigChangeType) remoting.EventType {
	switch change {
	case agollo.ADDED:
		return remoting.EventTypeAdd
	case agollo.DELETED:
		return remoting.EventTypeDel
	case agollo.MODIFIED:
		return remoting.EventTypeUpdate
	default:
		panic("unknown type: " + strconv.Itoa(int(change)))
	}
}

func (c *apolloConfiguration) AddListener(key string, listener ConfigurationListener, opts ...Option) {
	k := &Options{}
	for _, opt := range opts {
		opt(k)
	}

	key = k.Group + key
	l, _ := c.listeners.LoadOrStore(key, NewApolloListener())
	l.(*apolloListener).AddListener(listener)
}

func (c *apolloConfiguration) RemoveListener(key string, listener ConfigurationListener, opts ...Option) {
	k := &Options{}
	for _, opt := range opts {
		opt(k)
	}

	key = k.Group + key
	l, ok := c.listeners.Load(key)
	if ok {
		l.(*apolloListener).RemoveListener(listener)
	}
}

func getProperties(namespace string) string {
	return getNamespaceName(namespace, agollo.Properties)
}

func getNamespaceName(namespace string, configFileFormat agollo.ConfigFileFormat) string {
	return fmt.Sprintf(apolloConfigFormat, namespace, configFileFormat)
}

func (c *apolloConfiguration) GetConfig(key string, opts ...Option) (string, error) {
	k := &Options{}
	for _, opt := range opts {
		opt(k)
	}
	config := agollo.GetConfig(key)
	if config == nil {
		return "", errors.New(fmt.Sprintf("nothiing in namespace:%s ", key))
	}
	return config.GetContent(agollo.Properties), nil
}

func (c *apolloConfiguration) getAddressWithProtocolPrefix(url *common.URL) string {
	address := url.Location
	converted := address
	if len(address) != 0 {
		reg := regexp.MustCompile("\\s+")
		address = reg.ReplaceAllString(address, "")
		parts := strings.Split(address, ",")
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

func (c *apolloConfiguration) GetConfigs(key string, opts ...Option) (string, error) {
	return c.GetConfig(key, opts...)
}
