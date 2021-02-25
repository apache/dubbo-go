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
	"fmt"
	"path"
	"strconv"
	"sync"
	"testing"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/config_center/parser"
)

const (
	dubboPropertyFileName = "dubbo.properties"
)

func initZkData(group string, t *testing.T) (*zk.TestCluster, *zookeeperDynamicConfiguration) {
	ts, err := zk.StartTestCluster(1, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, ts.Servers[0])
	urlString := "registry://127.0.0.1:" + strconv.Itoa(ts.Servers[0].Port)
	regurl, err := common.NewURL(urlString)
	assert.NoError(t, err)
	regurl.AddParam(constant.REGISTRY_TIMEOUT_KEY, "15s")
	zkFactory := &zookeeperDynamicConfigurationFactory{}
	reg, err := zkFactory.GetDynamicConfiguration(regurl)
	zreg, ok := reg.(*zookeeperDynamicConfiguration)
	assert.True(t, ok)
	assert.NoError(t, err)
	assert.True(t, zreg.IsAvailable())
	assert.Equal(t, zreg.GetUrl(), regurl)
	assert.True(t, zreg.RestartCallBack())
	zreg.SetParser(&parser.DefaultConfigurationParser{})

	data := `
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
`
	if group != "" {
		err = zreg.client.Create(path.Join(zreg.rootPath, group, dubboPropertyFileName))
		assert.NoError(t, err)

		_, err = zreg.client.Conn.Set(path.Join(zreg.rootPath, group, dubboPropertyFileName), []byte(data), 0)
		assert.NoError(t, err)
	} else {
		err = zreg.client.Create(path.Join(zreg.rootPath, dubboPropertyFileName))
		assert.NoError(t, err)

		_, err = zreg.client.Conn.Set(path.Join(zreg.rootPath, dubboPropertyFileName), []byte(data), 0)
		assert.NoError(t, err)
	}

	return ts, zreg
}

func TestGetConfig(t *testing.T) {
	ts, reg := initZkData("dubbo", t)
	defer func() {
		reg.client.Close()
		err := ts.Stop()
		assert.NoError(t, err)
	}()
	configs, err := reg.GetProperties(dubboPropertyFileName, config_center.WithGroup("dubbo"))
	assert.NoError(t, err)
	m, err := reg.Parser().Parse(configs)
	assert.NoError(t, err)
	assert.Equal(t, "5s", m["dubbo.consumer.request_timeout"])
	configs, err = reg.GetProperties(dubboPropertyFileName)
	assert.Error(t, err)
	assert.Equal(t, "", configs)
	configs, err = reg.GetInternalProperty(dubboPropertyFileName)
	assert.Error(t, err)
	assert.Equal(t, "", configs)
	configs, err = reg.GetRule(dubboPropertyFileName)
	assert.Error(t, err)
	assert.Equal(t, "", configs)
}

func TestAddListener(t *testing.T) {
	ts, reg := initZkData("", t)
	defer func() {
		reg.client.Close()
		err := ts.Stop()
		assert.NoError(t, err)
	}()
	listener := &mockDataListener{}
	reg.AddListener(dubboPropertyFileName, listener)

	listener.wg.Add(1)
	data := `
	dubbo.consumer.request_timeout=3s
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
`
	_, err := reg.client.Conn.Set(path.Join(reg.rootPath, dubboPropertyFileName), []byte(data), 1)
	assert.NoError(t, err)
	listener.wg.Wait()
	assert.Equal(t, dubboPropertyFileName, listener.event)
}

func TestRemoveListener(t *testing.T) {
	ts, reg := initZkData("", t)
	defer func() {
		reg.client.Close()
		err := ts.Stop()
		assert.NoError(t, err)
	}()
	listener := &mockDataListener{}
	reg.AddListener(dubboPropertyFileName, listener)
	listener.wg.Add(1)
	data := `
	dubbo.consumer.request_timeout=3s
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
`
	reg.RemoveListener(dubboPropertyFileName, listener)
	listener.wg.Done()
	_, err := reg.client.Conn.Set(path.Join(reg.rootPath, dubboPropertyFileName), []byte(data), 1)
	assert.NoError(t, err)
	listener.wg.Wait()
	assert.Equal(t, "", listener.event)
}

func TestZookeeperDynamicConfigurationPublishConfig(t *testing.T) {
	value := "Test Data"
	customGroup := "Custom Group"
	key := "myKey"
	ts, reg := initZkData(config_center.DEFAULT_GROUP, t)
	defer func() {
		reg.client.Close()
		err := ts.Stop()
		assert.NoError(t, err)
	}()
	err := reg.PublishConfig(key, customGroup, value)
	assert.Nil(t, err)
	result, err := reg.GetInternalProperty("myKey", config_center.WithGroup(customGroup))
	assert.Nil(t, err)
	assert.Equal(t, value, result)

	var keys *gxset.HashSet
	keys, err = reg.GetConfigKeysByGroup(customGroup)
	assert.Nil(t, err)
	assert.Equal(t, 1, keys.Size())
	assert.True(t, keys.Contains(key))

}

type mockDataListener struct {
	wg    sync.WaitGroup
	event string
}

func (l *mockDataListener) Process(configType *config_center.ConfigChangeEvent) {
	fmt.Println("process!!!!!")
	l.wg.Done()
	l.event = configType.Key
}
