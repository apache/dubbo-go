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

package condition

import (
	"fmt"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/config"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config_center"
	_ "github.com/apache/dubbo-go/config_center/zookeeper"
	"github.com/apache/dubbo-go/remoting"
	"github.com/apache/dubbo-go/remoting/zookeeper"
)

const (
	routerPath    = "/dubbo/config/dubbo/test-condition.condition-router"
	routerLocalIP = "127.0.0.1"
	routerZk      = "zookeeper"
	routerKey     = "test-condition"
)

var (
	zkFormat        = "zookeeper://%s:%d"
	conditionFormat = "condition://%s/com.foo.BarService"
)

func TestNewAppRouter(t *testing.T) {

	testYML := `scope: application
key: mock-app
enabled: true
force: true
runtime: false
conditions:
  - => host != 172.22.3.91
`
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)
	err = z.Create(routerPath)
	assert.NoError(t, err)

	_, err = z.Conn.Set(routerPath, []byte(testYML), 0)
	assert.NoError(t, err)
	//defer func() {
	//	err = ts.Stop()
	//	assert.NoError(t, err)
	//	z.Close()
	//}()

	zkUrl, _ := common.NewURL(fmt.Sprintf(zkFormat, routerLocalIP, ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory(routerZk).GetDynamicConfiguration(zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	assert.Nil(t, err)
	assert.NotNil(t, configuration)

	appRouteURL := getAppRouteURL(routerKey)
	appRouter, err := NewAppRouter(appRouteURL)
	assert.Nil(t, err)
	assert.NotNil(t, appRouter)

	assert.NotNil(t, appRouter)
	assert.NotNil(t, appRouter.RouterRule())
	rule := appRouter.RouterRule()
	assert.Equal(t, "application", rule.Scope)
	assert.True(t, rule.Force)
	assert.True(t, rule.Enabled)
	assert.True(t, rule.Valid)

	assert.Equal(t, testYML, rule.RawRule)
	assert.Equal(t, false, rule.Runtime)
	assert.Equal(t, false, rule.Dynamic)
	assert.Equal(t, "mock-app", rule.Key)
	assert.Equal(t, 0, rule.Priority)
}

func TestGenerateConditions(t *testing.T) {

	testYML := `scope: application
key: mock-app
enabled: true
force: true
runtime: false
conditions:
  - => host != 172.22.3.91
  - host = 192.168.199.208 => host = 192.168.199.208 
`
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)
	err = z.Create(routerPath)
	assert.NoError(t, err)

	_, err = z.Conn.Set(routerPath, []byte(testYML), 0)
	assert.NoError(t, err)
	//defer func() {
	//	err = ts.Stop()
	//	assert.NoError(t, err)
	//	z.Close()
	//}()

	zkUrl, _ := common.NewURL(fmt.Sprintf(zkFormat, routerLocalIP, ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory(routerZk).GetDynamicConfiguration(zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	assert.Nil(t, err)
	assert.NotNil(t, configuration)

	appRouteURL := getAppRouteURL(routerKey)
	appRouter, err := NewAppRouter(appRouteURL)
	assert.Nil(t, err)
	assert.NotNil(t, appRouter)

	rule, err := getRule(testYML)
	assert.Nil(t, err)
	appRouter.generateConditions(rule)

	assert.Equal(t, 2, len(appRouter.conditionRouters))
}

func TestProcess(t *testing.T) {

	testYML := `scope: application
key: mock-app
enabled: true
force: true
runtime: false
conditions:
  - => host != 172.22.3.91
`
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)
	err = z.Create(routerPath)
	assert.NoError(t, err)

	_, err = z.Conn.Set(routerPath, []byte(testYML), 0)
	assert.NoError(t, err)
	defer func() {
		err = ts.Stop()
		assert.NoError(t, err)
		z.Close()
	}()

	zkUrl, _ := common.NewURL(fmt.Sprintf(zkFormat, routerLocalIP, ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory(routerZk).GetDynamicConfiguration(zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	assert.Nil(t, err)
	assert.NotNil(t, configuration)

	appRouteURL := getAppRouteURL(routerKey)
	appRouter, err := NewAppRouter(appRouteURL)
	assert.Nil(t, err)
	assert.NotNil(t, appRouter)

	assert.Equal(t, 1, len(appRouter.conditionRouters))

	testNewYML := `scope: application
key: mock-app
enabled: true
force: true
runtime: false
conditions:
  - => host != 172.22.3.91
  - host = 192.168.199.208 => host = 192.168.199.208
`

	appRouter.Process(&config_center.ConfigChangeEvent{ConfigType: remoting.EventTypeDel})

	assert.Equal(t, 0, len(appRouter.conditionRouters))

	appRouter.Process(&config_center.ConfigChangeEvent{Value: testNewYML, ConfigType: remoting.EventTypeAdd})

	assert.Equal(t, 2, len(appRouter.conditionRouters))
}

func getAppRouteURL(applicationKey string) *common.URL {
	url, _ := common.NewURL(fmt.Sprintf(conditionFormat, constant.ANYHOST_VALUE))
	url.AddParam("application", applicationKey)
	url.AddParam("force", "true")
	return url
}
