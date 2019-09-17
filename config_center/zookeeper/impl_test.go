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
	"context"
	"fmt"
	"sync"
	"testing"
)

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/config_center/parser"
)

func initZkData(group string, t *testing.T) (*zk.TestCluster, *zookeeperDynamicConfiguration) {
	regurl, _ := common.NewURL(context.TODO(), "registry://127.0.0.1:1111")
	ts, reg, err := newMockZookeeperDynamicConfiguration(&regurl)
	reg.SetParser(&parser.DefaultConfigurationParser{})

	assert.NoError(t, err)

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
		err = reg.client.Create(reg.rootPath + "/dubbo/dubbo.properties")
		assert.NoError(t, err)

		_, err = reg.client.Conn.Set(reg.rootPath+"/dubbo/dubbo.properties", []byte(data), 0)
		assert.NoError(t, err)
	} else {
		err = reg.client.Create(reg.rootPath + "/dubbo.properties")
		assert.NoError(t, err)

		_, err = reg.client.Conn.Set(reg.rootPath+"/dubbo.properties", []byte(data), 0)
		assert.NoError(t, err)
	}

	return ts, reg
}
func Test_GetConfig(t *testing.T) {
	ts, reg := initZkData("dubbo", t)
	defer ts.Stop()
	configs, err := reg.GetConfig("dubbo.properties", config_center.WithGroup("dubbo"))
	assert.NoError(t, err)
	m, err := reg.Parser().Parse(configs)
	assert.NoError(t, err)
	assert.Equal(t, "5s", m["dubbo.consumer.request_timeout"])
}

func Test_AddListener(t *testing.T) {
	ts, reg := initZkData("", t)
	defer ts.Stop()
	listener := &mockDataListener{}
	reg.AddListener("dubbo.properties", listener)
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
	_, err := reg.client.Conn.Set(reg.rootPath+"/dubbo.properties", []byte(data), 1)
	assert.NoError(t, err)
	listener.wg.Wait()
	assert.Equal(t, "dubbo.properties", listener.event)
}

func Test_RemoveListener(t *testing.T) {
	ts, reg := initZkData("", t)
	defer ts.Stop()
	listener := &mockDataListener{}
	reg.AddListener("dubbo.properties", listener)
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
	reg.RemoveListener("dubbo.properties", listener)
	listener.wg.Done()
	_, err := reg.client.Conn.Set(reg.rootPath+"/dubbo.properties", []byte(data), 1)
	assert.NoError(t, err)
	listener.wg.Wait()
	assert.Equal(t, "", listener.event)
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
