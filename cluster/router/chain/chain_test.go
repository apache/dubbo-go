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

package chain

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/condition"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/config"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	_ "github.com/apache/dubbo-go/config_center/zookeeper"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/remoting/zookeeper"
)

const (
	localIP    = "127.0.0.1"
	test1234IP = "1.2.3.4"
	test1111IP = "1.1.1.1"
	test0000IP = "0.0.0.0"
	port20000  = 20000

	path             = "/dubbo/config/dubbo/test-condition.condition-router"
	zkFormat         = "zookeeper://%s:%d"
	consumerFormat   = "consumer://%s/com.foo.BarService"
	dubboForamt      = "dubbo://%s:%d/com.foo.BarService"
	anyUrlFormat     = "condition://%s/com.foo.BarService"
	zk               = "zookeeper"
	applicationKey   = "test-condition"
	applicationField = "application"
	forceField       = "force"
	forceValue       = "true"
)

func TestNewRouterChain(t *testing.T) {
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)
	err = z.Create(path)
	assert.NoError(t, err)
	testyml := `scope: application
key: mock-app
enabled: true
force: true
runtime: false
conditions:
  - => host != 172.22.3.91
`

	_, err = z.Conn.Set(path, []byte(testyml), 0)
	assert.NoError(t, err)
	defer ts.Stop()
	defer z.Close()

	zkUrl, _ := common.NewURL(fmt.Sprintf(zkFormat, localIP, ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory(zk).GetDynamicConfiguration(zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	assert.Nil(t, err)
	assert.NotNil(t, configuration)

	chain, err := NewRouterChain(getRouteUrl(applicationKey))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(chain.routers))
	appRouter := chain.routers[0].(*condition.AppRouter)

	assert.NotNil(t, appRouter)
	assert.NotNil(t, appRouter.RouterRule())
	rule := appRouter.RouterRule()
	assert.Equal(t, "application", rule.Scope)
	assert.True(t, rule.Force)
	assert.True(t, rule.Enabled)
	assert.True(t, rule.Valid)

	assert.Equal(t, testyml, rule.RawRule)
	assert.Equal(t, false, rule.Runtime)
	assert.Equal(t, false, rule.Dynamic)
	assert.Equal(t, "mock-app", rule.Key)
}

func TestNewRouterChainURLNil(t *testing.T) {
	chain, err := NewRouterChain(nil)
	assert.NoError(t, err)
	assert.NotNil(t, chain)
}

func TestRouterChainAddRouters(t *testing.T) {
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)
	err = z.Create(path)
	assert.NoError(t, err)

	testyml := `scope: application
key: mock-app
enabled: true
force: true
runtime: false
conditions:
  - => host != 172.22.3.91
`

	_, err = z.Conn.Set(path, []byte(testyml), 0)
	assert.NoError(t, err)
	defer ts.Stop()
	defer z.Close()

	zkUrl, _ := common.NewURL(fmt.Sprintf(zkFormat, localIP, ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory(zk).GetDynamicConfiguration(zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	chain, err := NewRouterChain(getConditionRouteUrl(applicationKey))
	assert.Nil(t, err)
	assert.Equal(t, 2, len(chain.routers))

	url := getConditionRouteUrl(applicationKey)
	assert.NotNil(t, url)
	factory := extension.GetRouterFactory(url.Protocol)
	r, err := factory.NewPriorityRouter(url)
	assert.Nil(t, err)
	assert.NotNil(t, r)

	routers := make([]router.PriorityRouter, 0)
	routers = append(routers, r)
	chain.AddRouters(routers)
	assert.Equal(t, 3, len(chain.routers))
}

func TestRouterChainRoute(t *testing.T) {
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	defer ts.Stop()
	defer z.Close()

	zkUrl, _ := common.NewURL(fmt.Sprintf(zkFormat, localIP, ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory(zk).GetDynamicConfiguration(zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	chain, err := NewRouterChain(getConditionRouteUrl(applicationKey))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(chain.routers))

	url := getConditionRouteUrl(applicationKey)
	assert.NotNil(t, url)

	var invokers []protocol.Invoker
	dubboURL, _ := common.NewURL(fmt.Sprintf(dubboForamt, test1234IP, port20000))
	invokers = append(invokers, protocol.NewBaseInvoker(dubboURL))
	chain.SetInvokers(invokers)
	chain.buildCache()

	targetURL, _ := common.NewURL(fmt.Sprintf(consumerFormat, test1111IP))
	inv := &invocation.RPCInvocation{}
	finalInvokers := chain.Route(targetURL, inv)

	assert.Equal(t, 1, len(finalInvokers))
}

func TestRouterChainRouteAppRouter(t *testing.T) {
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)
	err = z.Create(path)
	assert.NoError(t, err)

	testyml := `scope: application
key: mock-app
enabled: true
force: true
runtime: false
conditions:
  - => host = 1.1.1.1 => host != 1.2.3.4
`

	_, err = z.Conn.Set(path, []byte(testyml), 0)
	assert.NoError(t, err)
	defer ts.Stop()
	defer z.Close()

	zkUrl, _ := common.NewURL(fmt.Sprintf(zkFormat, localIP, ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory(zk).GetDynamicConfiguration(zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	chain, err := NewRouterChain(getConditionRouteUrl(applicationKey))
	assert.Nil(t, err)
	assert.Equal(t, 2, len(chain.routers))

	var invokers []protocol.Invoker
	dubboURL, _ := common.NewURL(fmt.Sprintf(dubboForamt, test1234IP, port20000))
	invokers = append(invokers, protocol.NewBaseInvoker(dubboURL))
	chain.SetInvokers(invokers)
	chain.buildCache()

	targetURL, _ := common.NewURL(fmt.Sprintf(consumerFormat, test1111IP))
	inv := &invocation.RPCInvocation{}
	finalInvokers := chain.Route(targetURL, inv)

	assert.Equal(t, 0, len(finalInvokers))
}

func TestRouterChainRouteNoRoute(t *testing.T) {
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	defer ts.Stop()
	defer z.Close()

	zkUrl, _ := common.NewURL(fmt.Sprintf(zkFormat, localIP, ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory(zk).GetDynamicConfiguration(zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	chain, err := NewRouterChain(getConditionNoRouteUrl(applicationKey))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(chain.routers))

	url := getConditionRouteUrl(applicationKey)
	assert.NotNil(t, url)

	var invokers []protocol.Invoker
	dubboURL, _ := common.NewURL(fmt.Sprintf(dubboForamt, test1234IP, port20000))
	invokers = append(invokers, protocol.NewBaseInvoker(dubboURL))
	chain.SetInvokers(invokers)
	chain.buildCache()

	targetURL, _ := common.NewURL(fmt.Sprintf(consumerFormat, test1111IP))
	inv := &invocation.RPCInvocation{}
	finalInvokers := chain.Route(targetURL, inv)

	assert.Equal(t, 0, len(finalInvokers))
}

func getConditionNoRouteUrl(applicationKey string) *common.URL {
	url, _ := common.NewURL(fmt.Sprintf(anyUrlFormat, test0000IP))
	url.AddParam(applicationField, applicationKey)
	url.AddParam(forceField, forceValue)
	rule := base64.URLEncoding.EncodeToString([]byte("host = 1.1.1.1 => host != 1.2.3.4"))
	url.AddParam(constant.RULE_KEY, rule)
	return url
}

func getConditionRouteUrl(applicationKey string) *common.URL {
	url, _ := common.NewURL(fmt.Sprintf(anyUrlFormat, test0000IP))
	url.AddParam(applicationField, applicationKey)
	url.AddParam(forceField, forceValue)
	rule := base64.URLEncoding.EncodeToString([]byte("host = 1.1.1.1 => host = 1.2.3.4"))
	url.AddParam(constant.RULE_KEY, rule)
	return url
}

func getRouteUrl(applicationKey string) *common.URL {
	url, _ := common.NewURL(fmt.Sprintf(anyUrlFormat, test0000IP))
	url.AddParam(applicationField, applicationKey)
	url.AddParam(forceField, forceValue)
	return url
}
