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
	"github.com/go-errors/errors"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/config_center/parser"
	"github.com/apache/dubbo-go/remoting"

	"github.com/zouyx/agollo"
)

const (
	apolloEnvKey         = "env"
	apolloAddrKey        = "apollo.meta"
	apolloClusterKey     = "apollo.cluster"
	apolloProtocolPrefix = "http://"
	apolloConfigFormat = "%s.%s"
)

type apolloDynamicConfiguration struct {
	url *common.URL

	listeners sync.Map
	appConf   *agollo.AppConfig
	parser    parser.ConfigurationParser
}

func newApolloDynamicConfiguration(url *common.URL) (*apolloDynamicConfiguration, error) {
	c := &apolloDynamicConfiguration{
		url: url,
	}
	configEnv := url.GetParam(apolloEnvKey, "")
	configAddr := c.getAddressWithProtocolPrefix(url)
	configCluster := url.GetParam(constant.CONFIG_GROUP_KEY, "")
	if len(configEnv) != 0 {
		os.Setenv(apolloEnvKey, configEnv)
	}

	key := os.Getenv(apolloEnvKey)
	if len(key) != 0 || constant.ANYHOST_VALUE == configAddr {
		configAddr = key
	}

	appId := os.Getenv("app.id")
	namespace := url.GetParam(constant.CONFIG_NAMESPACE_KEY, config_center.DEFAULT_GROUP)
	readyConfig := &agollo.AppConfig{
		AppId:         appId,
		Cluster:       configCluster,
		NamespaceName: getNamespaceName(namespace,agollo.YML),
		Ip:            configAddr,
	}

	agollo.InitCustomConfig(func() (*agollo.AppConfig, error) {
		return readyConfig, nil
	})

	return c, agollo.Start()
}

type apolloChangeListener struct {
	c *apolloDynamicConfiguration
}

func (a *apolloChangeListener) OnChange(event *agollo.ChangeEvent) {
	for name, change := range event.Changes {
		cfgChangeEvent := &config_center.ConfigChangeEvent{
			Key:        name,
			Value:      change.NewValue,
			ConfigType: a.c.getChangeType(change.ChangeType),
		}
		a.c.listeners.Range(func(key, value interface{}) bool {
			for listener, _ := range value.(apolloListener).listeners {
				listener.Process(cfgChangeEvent)
			}
			return true
		})
	}
}

func (c *apolloDynamicConfiguration) start() {
	agollo.AddChangeListener(&apolloChangeListener{})
}

func (c *apolloDynamicConfiguration) getChangeType(change agollo.ConfigChangeType) remoting.EventType {
	switch change {
	case agollo.ADDED:
		return remoting.EventTypeAdd
	case agollo.DELETED:
		return remoting.EventTypeDel
	case agollo.MODIFIED:
		return remoting.EventTypeUpdate
	default:
		panic("unknow type: " + strconv.Itoa(int(change)))
	}
}

func (c *apolloDynamicConfiguration) AddListener(key string, listener config_center.ConfigurationListener, opts ...config_center.Option) {
	k := &config_center.Options{}
	for _, opt := range opts {
		opt(k)
	}

	key = k.Group + key
	l, _ := c.listeners.LoadOrStore(key, NewApolloListener())
	l.(apolloListener).AddListener(listener)
}

func (c *apolloDynamicConfiguration) RemoveListener(key string, listener config_center.ConfigurationListener, opts ...config_center.Option) {
	k := &config_center.Options{}
	for _, opt := range opts {
		opt(k)
	}

	key = k.Group + key
	l, ok := c.listeners.Load(key)
	if ok {
		l.(apolloListener).RemoveListener(listener)
	}
}

func getNamespaceName(namespace string,configFileFormat agollo.ConfigFileFormat ) string{
	return fmt.Sprintf(apolloConfigFormat, namespace, configFileFormat)
}

func (c *apolloDynamicConfiguration) GetConfig(key string, opts ...config_center.Option) (string, error) {
	k := &config_center.Options{}
	for _, opt := range opts {
		opt(k)
	}
	group := k.Group
	if len(group) != 0 && c.url.GetParam(constant.CONFIG_GROUP_KEY, config_center.DEFAULT_GROUP) != group {
		namespace := c.url.GetParam(constant.CONFIG_GROUP_KEY, config_center.DEFAULT_GROUP)
		fileNamespace := getNamespaceName(namespace, agollo.Properties)
		config := agollo.GetConfig(fileNamespace)
		if config==nil{
			return "",errors.New(fmt.Sprintf("nothiing in namespace:%s ",fileNamespace))
		}
		return config.GetContent(),nil
	}
	return agollo.GetStringValue(key, ""), nil
}

func (c *apolloDynamicConfiguration) getAddressWithProtocolPrefix(url *common.URL) string {
	address := ""
	converted := address
	if len(address) != 0 {
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

func (c *apolloDynamicConfiguration) Parser() parser.ConfigurationParser {
	return c.parser
}
func (c *apolloDynamicConfiguration) SetParser(p parser.ConfigurationParser) {
	c.parser = p
}

func (c *apolloDynamicConfiguration) GetConfigs(key string, opts ...config_center.Option) (string, error) {
	return c.GetConfig(key, opts...)
}
