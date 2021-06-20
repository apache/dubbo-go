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

package nacos

import (
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
)

func getNacosConfig(t *testing.T) config_center.DynamicConfiguration {
	registryUrl, err := common.NewURL("registry://console.nacos.io:80")
	assert.Nil(t, err)
	nacosConfig, err := newNacosDynamicConfiguration(registryUrl)
	assert.Nil(t, err)
	return nacosConfig
}

func TestPublishConfig(t *testing.T) {
	nacosConfig := getNacosConfig(t)
	data := `dubbo.protocol.name=dubbo`
	err := nacosConfig.PublishConfig("dubbo.properties", "dubbo-go", data)
	assert.Nil(t, err)
}

func TestGetConfig(t *testing.T) {
	nacosConfig := getNacosConfig(t)
	nacosConfig.SetParser(&parser.DefaultConfigurationParser{})

	config, err := nacosConfig.GetProperties("dubbo.properties", config_center.WithGroup("dubbo-go"))
	assert.NotEmpty(t, config)
	assert.NoError(t, err)

	parse, err := nacosConfig.Parser().Parse(config)
	assert.NoError(t, err)
	assert.Equal(t, parse["dubbo.protocol.name"], "dubbo")
}

func TestGetConfigKeysByGroup(t *testing.T) {
	nacosConfig := getNacosConfig(t)
	config, err := nacosConfig.GetConfigKeysByGroup("dubbo-go")
	assert.NoError(t, err)
	assert.True(t, config.Contains("dubbo.properties"))
}

func TestAddListener(t *testing.T) {
	nacosConfig := getNacosConfig(t)
	listener := &mockDataListener{}
	time.Sleep(time.Second * 2)
	nacosConfig.AddListener("dubbo.properties", listener)
}

func TestRemoveListener(_ *testing.T) {
	// TODO not supported in current go_nacos_sdk version
}

type mockDataListener struct {
	wg    sync.WaitGroup
	event string
}

func (l *mockDataListener) Process(configType *config_center.ConfigChangeEvent) {
	l.wg.Done()
	l.event = configType.Key
}
