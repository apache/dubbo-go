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
	"strconv"
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

func TestNewRouterChain(t *testing.T) {
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)
	err = z.Create("/dubbo/config/dubbo/test-condition.condition-router")
	assert.NoError(t, err)

	testyml := `enabled: true
force: true
runtime: false
conditions:
  - => host != 172.22.3.91
`

	_, err = z.Conn.Set("/dubbo/config/dubbo/test-condition.condition-router", []byte(testyml), 0)
	assert.NoError(t, err)
	defer ts.Stop()
	defer z.Close()

	zkUrl, _ := common.NewURL("zookeeper://127.0.0.1:" + strconv.Itoa(ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory("zookeeper").GetDynamicConfiguration(&zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	assert.Nil(t, err)
	assert.NotNil(t, configuration)

	chain, err := NewRouterChain(getRouteUrl("test-condition"))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(chain.routers))
	appRouter := chain.routers[0].(*condition.AppRouter)

	assert.NotNil(t, appRouter)
	assert.NotNil(t, appRouter.RouterRule())
	rule := appRouter.RouterRule()
	assert.Equal(t, "", rule.Scope)
	assert.True(t, rule.Force)
	assert.True(t, rule.Enabled)
	assert.True(t, rule.Valid)

	assert.Equal(t, testyml, rule.RawRule)
	assert.Equal(t, false, rule.Runtime)
	assert.Equal(t, false, rule.Dynamic)
	assert.Equal(t, "", rule.Key)
}

func TestNewRouterChainURLNil(t *testing.T) {
	chain, err := NewRouterChain(nil)
	assert.NoError(t, err)
	assert.NotNil(t, chain)
}

func TestRouterChain_AddRouters(t *testing.T) {
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)
	err = z.Create("/dubbo/config/dubbo/test-condition.condition-router")
	assert.NoError(t, err)

	testyml := `enabled: true
force: true
runtime: false
conditions:
  - => host != 172.22.3.91
`

	_, err = z.Conn.Set("/dubbo/config/dubbo/test-condition.condition-router", []byte(testyml), 0)
	assert.NoError(t, err)
	defer ts.Stop()
	defer z.Close()

	zkUrl, _ := common.NewURL("zookeeper://127.0.0.1:" + strconv.Itoa(ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory("zookeeper").GetDynamicConfiguration(&zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	chain, err := NewRouterChain(getConditionRouteUrl("test-condition"))
	assert.Nil(t, err)
	assert.Equal(t, 2, len(chain.routers))

	url := getConditionRouteUrl("test-condition")
	assert.NotNil(t, url)
	factory := extension.GetRouterFactory(url.Protocol)
	r, err := factory.NewRouter(url)
	assert.Nil(t, err)
	assert.NotNil(t, r)

	routers := make([]router.Router, 0)
	routers = append(routers, r)
	chain.AddRouters(routers)
	assert.Equal(t, 3, len(chain.routers))
}

func TestRouterChain_Route(t *testing.T) {
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	defer ts.Stop()
	defer z.Close()

	zkUrl, _ := common.NewURL("zookeeper://127.0.0.1:" + strconv.Itoa(ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory("zookeeper").GetDynamicConfiguration(&zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	chain, err := NewRouterChain(getConditionRouteUrl("test-condition"))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(chain.routers))

	url := getConditionRouteUrl("test-condition")
	assert.NotNil(t, url)

	invokers := []protocol.Invoker{}
	dubboURL, _ := common.NewURL(fmt.Sprintf("dubbo://1.2.3.4:20000/com.foo.BarService"))
	invokers = append(invokers, protocol.NewBaseInvoker(dubboURL))

	targetURL, _ := common.NewURL(fmt.Sprintf("consumer://1.1.1.1/com.foo.BarService"))
	inv := &invocation.RPCInvocation{}
	finalInvokers := chain.Route(invokers, &targetURL, inv)

	assert.Equal(t, 1, len(finalInvokers))
}

func TestRouterChain_Route_AppRouter(t *testing.T) {
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)
	err = z.Create("/dubbo/config/dubbo/test-condition.condition-router")
	assert.NoError(t, err)

	testyml := `enabled: true
force: true
runtime: false
conditions:
  - => host = 1.1.1.1 => host != 1.2.3.4
`

	_, err = z.Conn.Set("/dubbo/config/dubbo/test-condition.condition-router", []byte(testyml), 0)
	assert.NoError(t, err)
	defer ts.Stop()
	defer z.Close()

	zkUrl, _ := common.NewURL("zookeeper://127.0.0.1:" + strconv.Itoa(ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory("zookeeper").GetDynamicConfiguration(&zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	chain, err := NewRouterChain(getConditionRouteUrl("test-condition"))
	assert.Nil(t, err)
	assert.Equal(t, 2, len(chain.routers))

	invokers := []protocol.Invoker{}
	dubboURL, _ := common.NewURL(fmt.Sprintf("dubbo://1.2.3.4:20000/com.foo.BarService"))
	invokers = append(invokers, protocol.NewBaseInvoker(dubboURL))

	targetURL, _ := common.NewURL(fmt.Sprintf("consumer://1.1.1.1/com.foo.BarService"))
	inv := &invocation.RPCInvocation{}
	finalInvokers := chain.Route(invokers, &targetURL, inv)

	assert.Equal(t, 0, len(finalInvokers))
}

func TestRouterChain_Route_NoRoute(t *testing.T) {
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	defer ts.Stop()
	defer z.Close()

	zkUrl, _ := common.NewURL("zookeeper://127.0.0.1:" + strconv.Itoa(ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory("zookeeper").GetDynamicConfiguration(&zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	chain, err := NewRouterChain(getConditionNoRouteUrl("test-condition"))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(chain.routers))

	url := getConditionRouteUrl("test-condition")
	assert.NotNil(t, url)

	invokers := []protocol.Invoker{}
	dubboURL, _ := common.NewURL(fmt.Sprintf("dubbo://1.2.3.4:20000/com.foo.BarService"))
	invokers = append(invokers, protocol.NewBaseInvoker(dubboURL))

	targetURL, _ := common.NewURL(fmt.Sprintf("consumer://1.1.1.1/com.foo.BarService"))
	inv := &invocation.RPCInvocation{}
	finalInvokers := chain.Route(invokers, &targetURL, inv)

	assert.Equal(t, 0, len(finalInvokers))
}

func getConditionNoRouteUrl(applicationKey string) *common.URL {
	url, _ := common.NewURL("condition://0.0.0.0/com.foo.BarService")
	url.AddParam("application", applicationKey)
	url.AddParam("force", "true")
	rule := base64.URLEncoding.EncodeToString([]byte("host = 1.1.1.1 => host != 1.2.3.4"))
	url.AddParam(constant.RULE_KEY, rule)
	return &url
}

func getConditionRouteUrl(applicationKey string) *common.URL {
	url, _ := common.NewURL("condition://0.0.0.0/com.foo.BarService")
	url.AddParam("application", applicationKey)
	url.AddParam("force", "true")
	rule := base64.URLEncoding.EncodeToString([]byte("host = 1.1.1.1 => host = 1.2.3.4"))
	url.AddParam(constant.RULE_KEY, rule)
	return &url
}

func getRouteUrl(applicationKey string) *common.URL {
	url, _ := common.NewURL("condition://0.0.0.0/com.foo.BarService")
	url.AddParam("application", applicationKey)
	url.AddParam("force", "true")
	return &url
}
