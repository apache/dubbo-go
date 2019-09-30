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

package config_center

import (
	"sync"
)

import (
	"gopkg.in/yaml.v2"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/config_center/parser"
	"github.com/apache/dubbo-go/remoting"
)

type MockDynamicConfigurationFactory struct {
	Content string
}

var (
	once                 sync.Once
	dynamicConfiguration *MockDynamicConfiguration
)

func (f *MockDynamicConfigurationFactory) GetDynamicConfiguration(url *common.URL) (DynamicConfiguration, error) {
	var err error
	once.Do(func() {
		dynamicConfiguration = &MockDynamicConfiguration{listener: map[string]ConfigurationListener{}}
		dynamicConfiguration.SetParser(&parser.DefaultConfigurationParser{})

		dynamicConfiguration.content = `
	dubbo.consumer.request_timeout=5s
	dubbo.consumer.connect_timeout=5s
	dubbo.application.organization=ikurento.com
	dubbo.application.name=BDTService
	dubbo.application.module=dubbogo user-info server
	dubbo.application.version=0.0.1
	dubbo.application.owner=ZX
	dubbo.application.environment=dev
	dubbo.registries.hangzhouzk.protocol=zookeeper
	dubbo.registries.hangzhouzk.timeout=3s
	dubbo.registries.hangzhouzk.address=127.0.0.1:2181
	dubbo.registries.shanghaizk.protocol=zookeeper
	dubbo.registries.shanghaizk.timeout=3s
	dubbo.registries.shanghaizk.address=127.0.0.1:2182
	dubbo.service.com.ikurento.user.UserProvider.protocol=dubbo
	dubbo.service.com.ikurento.user.UserProvider.interface=com.ikurento.user.UserProvider
	dubbo.service.com.ikurento.user.UserProvider.loadbalance=random
	dubbo.service.com.ikurento.user.UserProvider.warmup=100
	dubbo.service.com.ikurento.user.UserProvider.cluster=failover
	dubbo.protocols.jsonrpc1.name=jsonrpc
	dubbo.protocols.jsonrpc1.ip=127.0.0.1
	dubbo.protocols.jsonrpc1.port=20001
`
	})
	if len(f.Content) != 0 {
		dynamicConfiguration.content = f.Content
	}
	return dynamicConfiguration, err

}

type MockDynamicConfiguration struct {
	parser   parser.ConfigurationParser
	content  string
	listener map[string]ConfigurationListener
}

func (c *MockDynamicConfiguration) AddListener(key string, listener ConfigurationListener, opions ...Option) {
	c.listener[key] = listener
}

func (c *MockDynamicConfiguration) RemoveListener(key string, listener ConfigurationListener, opions ...Option) {
}

func (c *MockDynamicConfiguration) GetConfig(key string, opts ...Option) (string, error) {

	return c.content, nil
}

//For zookeeper, getConfig and getConfigs have the same meaning.
func (c *MockDynamicConfiguration) GetConfigs(key string, opts ...Option) (string, error) {
	return c.GetConfig(key, opts...)
}

func (c *MockDynamicConfiguration) Parser() parser.ConfigurationParser {
	return c.parser
}
func (c *MockDynamicConfiguration) SetParser(p parser.ConfigurationParser) {
	c.parser = p
}

func (c *MockDynamicConfiguration) MockServiceConfigEvent() {
	config := &parser.ConfiguratorConfig{
		ConfigVersion: "2.7.1",
		Scope:         parser.GeneralType,
		Key:           "org.apache.dubbo-go.mockService",
		Enabled:       true,
		Configs: []parser.ConfigItem{
			{Type: parser.GeneralType,
				Enabled:    true,
				Addresses:  []string{"0.0.0.0"},
				Services:   []string{"org.apache.dubbo-go.mockService"},
				Side:       "provider",
				Parameters: map[string]string{"cluster": "mock1"},
			},
		},
	}
	value, _ := yaml.Marshal(config)
	key := "group*org.apache.dubbo-go.mockService:1.0.0" + constant.CONFIGURATORS_SUFFIX
	c.listener[key].Process(&ConfigChangeEvent{Key: key, Value: string(value), ConfigType: remoting.EventTypeAdd})
}

func (c *MockDynamicConfiguration) MockApplicationConfigEvent() {
	config := &parser.ConfiguratorConfig{
		ConfigVersion: "2.7.1",
		Scope:         parser.ScopeApplication,
		Key:           "org.apache.dubbo-go.mockService",
		Enabled:       true,
		Configs: []parser.ConfigItem{
			{Type: parser.ScopeApplication,
				Enabled:    true,
				Addresses:  []string{"0.0.0.0"},
				Services:   []string{"org.apache.dubbo-go.mockService"},
				Side:       "provider",
				Parameters: map[string]string{"cluster": "mock1"},
			},
		},
	}
	value, _ := yaml.Marshal(config)
	key := "test-application" + constant.CONFIGURATORS_SUFFIX
	c.listener[key].Process(&ConfigChangeEvent{Key: key, Value: string(value), ConfigType: remoting.EventTypeAdd})
}
