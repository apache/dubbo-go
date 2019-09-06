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
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/remoting"
)

type MockDynamicConfigurationFactory struct{}

var (
	once                 sync.Once
	dynamicConfiguration *mockDynamicConfiguration
)

func (f *MockDynamicConfigurationFactory) GetDynamicConfiguration(url *common.URL) (DynamicConfiguration, error) {
	var err error
	once.Do(func() {
		dynamicConfiguration = &mockDynamicConfiguration{}
		dynamicConfiguration.SetParser(&DefaultConfigurationParser{})
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
	return dynamicConfiguration, err

}

type mockDynamicConfiguration struct {
	parser  ConfigurationParser
	content string
}

func (c *mockDynamicConfiguration) AddListener(key string, listener remoting.ConfigurationListener, opions ...Option) {
}

func (c *mockDynamicConfiguration) RemoveListener(key string, listener remoting.ConfigurationListener, opions ...Option) {
}

func (c *mockDynamicConfiguration) GetConfig(key string, opts ...Option) (string, error) {

	return c.content, nil
}

func (c *mockDynamicConfiguration) SetConfig(group string, key string, value string) error {
	return nil
}

//For zookeeper, getConfig and getConfigs have the same meaning.
func (c *mockDynamicConfiguration) GetConfigs(key string, opts ...Option) (string, error) {
	return c.GetConfig(key, opts...)
}

func (c *mockDynamicConfiguration) Parser() ConfigurationParser {
	return c.parser
}
func (c *mockDynamicConfiguration) SetParser(p ConfigurationParser) {
	c.parser = p
}
