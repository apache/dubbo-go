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
	"path"
	"strconv"
	"testing"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/config_center/parser"
)

func initZkDynamicConfiguration(t *testing.T) (*zk.TestCluster, *zookeeperDynamicConfiguration) {
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
	dubbo.application.name=dubbogo
`
	err = zreg.client.Create(path.Join(zreg.rootPath, dubboPropertyFileName))
	assert.NoError(t, err)
	_, err = zreg.client.Conn.Set(path.Join(zreg.rootPath, dubboPropertyFileName), []byte(data), 0)
	assert.NoError(t, err)

	return ts, zreg
}

func TestZookeeperDynamicConfigurationPathToKey(t *testing.T) {
	ts, reg := initZkDynamicConfiguration(t)
	defer func() {
		err := ts.Stop()
		assert.NoError(t, err)
	}()
	listener := &mockDataListener{}
	key := path.Join("dubbogoDemo" + constant.CONFIGURATORS_SUFFIX)
	reg.AddListener(key, listener)
	listener.wg.Add(1)

	data := `
scope: application
key: dubbogoDemo
enabled: true
configs:
  - addresses: [0.0.0.0:20880]
    side: provider
    parameters:
      weight: 60
  - addresses: [0.0.0.0:20881]
    side: provider
    parameters:
      weight: 40
`
	zkPath := path.Join(reg.rootPath, "dubbo", key)
	exists, _, err := reg.client.Conn.Exists(zkPath)
	assert.NoError(t, err)
	if !exists {
		err = reg.client.Create(zkPath)
		assert.NoError(t, err)
	}
	_, err = reg.client.SetContent(zkPath, []byte(data), 0)
	assert.NoError(t, err)
	listener.wg.Wait()
	assert.Equal(t, key, listener.event)
}
