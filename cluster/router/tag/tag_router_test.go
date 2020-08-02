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

package tag

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/config"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config_center"
	_ "github.com/apache/dubbo-go/config_center/zookeeper"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/remoting"
	"github.com/apache/dubbo-go/remoting/zookeeper"
)

const (
	tagRouterTestHangZhouUrl       = "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true&dubbo.tag=hangzhou&remote.application=test-tag"
	tagRouterTestShangHaiUrl       = "dubbo://127.0.0.1:20002/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true&dubbo.tag=shanghai&remote.application=test-tag"
	tagRouterTestBeijingUrl        = "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true&dubbo.tag=beijing&remote.application=test-tag"
	tagRouterTestEnabledBeijingUrl = "dubbo://127.0.0.1:20004/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=false&dubbo.tag=beijing&remote.application=test-tag"
	tagRouterTestUserConsumer      = "dubbo://127.0.0.1:20005/com.ikurento.user.UserConsumer?interface=com.ikurento.user.UserConsumer&group=&version=2.6.0&enabled=true&remote.application=test-tag"
	tagRouterTestUserConsumerTag   = "dubbo://127.0.0.1:20000/com.ikurento.user.UserConsumer?interface=com.ikurento.user.UserConsumer&group=&version=2.6.0&enabled=true&dubbo.force.tag=true&remote.application=test-tag"

	tagRouterTestDynamicIpv4Provider1 = "dubbo://127.0.0.1:20001/com.ikurento.user.UserConsumer?interface=com.ikurento.user.UserConsumer&group=&version=2.6.0&enabled=true&remote.application=test-tag"
	tagRouterTestDynamicIpv4Provider2 = "dubbo://127.0.0.1:20002/com.ikurento.user.UserConsumer?interface=com.ikurento.user.UserConsumer&group=&version=2.6.0&enabled=true&remote.application=test-tag"
	tagRouterTestDynamicIpv4Provider3 = "dubbo://127.0.0.1:20003/com.ikurento.user.UserConsumer?interface=com.ikurento.user.UserConsumer&group=&version=2.6.0&enabled=true&remote.application=test-tag"
	tagRouterTestDynamicIpv4Provider4 = "dubbo://127.0.0.1:20004/com.ikurento.user.UserConsumer?interface=com.ikurento.user.UserConsumer&group=&version=2.6.0&enabled=true&remote.application=test-tag&dubbo.tag=tag4"
	tagRouterTestDynamicIpv4Provider5 = "dubbo://127.0.0.1:20005/com.ikurento.user.UserConsumer?interface=com.ikurento.user.UserConsumer&group=&version=2.6.0&enabled=true&remote.application=test-tag&dubbo.tag=tag5"

	tagRouterTestDubboTag      = "dubbo.tag"
	tagRouterTestDubboForceTag = "dubbo.force.tag"
	tagRouterTestHangZhou      = "hangzhou"
	tagRouterTestGuangZhou     = "guangzhou"
	tagRouterTestFalse         = "false"
	tagRouterTestTrue          = "true"

	routerPath    = "/dubbo/config/dubbo/test-tag.tag-router"
	routerLocalIP = "127.0.0.1"
	routerZk      = "zookeeper"
)

var (
	zkFormat        = "zookeeper://%s:%d"
	conditionFormat = "condition://%s/com.foo.BarService"
)

// MockInvoker is only mock the Invoker to support test tagRouter
type MockInvoker struct {
	url          common.URL
	available    bool
	destroyed    bool
	successCount int
}

func NewMockInvoker(url common.URL) *MockInvoker {
	return &MockInvoker{
		url:          url,
		available:    true,
		destroyed:    false,
		successCount: 0,
	}
}

func (bi *MockInvoker) GetUrl() common.URL {
	return bi.url
}

func (bi *MockInvoker) IsAvailable() bool {
	return bi.available
}

func (bi *MockInvoker) IsDestroyed() bool {
	return bi.destroyed
}

func (bi *MockInvoker) Invoke(_ context.Context, _ protocol.Invocation) protocol.Result {
	bi.successCount++

	result := &protocol.RPCResult{Err: nil}
	return result
}

func (bi *MockInvoker) Destroy() {
	bi.destroyed = true
	bi.available = false
}

func TestTagRouterPriority(t *testing.T) {
	u1, err := common.NewURL(tagRouterTestUserConsumerTag)
	assert.Nil(t, err)
	tagRouter, e := NewTagRouter(&u1)
	assert.Nil(t, e)
	p := tagRouter.Priority()
	assert.Equal(t, int64(0), p)
}

func TestTagRouterRouteForce(t *testing.T) {
	u1, e1 := common.NewURL(tagRouterTestUserConsumerTag)
	assert.Nil(t, e1)
	tagRouter, e := NewTagRouter(&u1)
	assert.Nil(t, e)

	u2, e2 := common.NewURL(tagRouterTestHangZhouUrl)
	u3, e3 := common.NewURL(tagRouterTestShangHaiUrl)
	u4, e4 := common.NewURL(tagRouterTestBeijingUrl)
	assert.Nil(t, e2)
	assert.Nil(t, e3)
	assert.Nil(t, e4)
	inv2 := NewMockInvoker(u2)
	inv3 := NewMockInvoker(u3)
	inv4 := NewMockInvoker(u4)
	var invokers []protocol.Invoker
	invokers = append(invokers, inv2, inv3, inv4)
	inv := &invocation.RPCInvocation{}
	inv.SetAttachments(tagRouterTestDubboTag, tagRouterTestHangZhou)
	invRst1 := tagRouter.Route(invokers, &u1, inv)
	assert.Equal(t, 1, len(invRst1))
	assert.Equal(t, tagRouterTestHangZhou, invRst1[0].GetUrl().GetParam(tagRouterTestDubboTag, ""))

	inv.SetAttachments(tagRouterTestDubboTag, tagRouterTestGuangZhou)
	invRst2 := tagRouter.Route(invokers, &u1, inv)
	assert.Equal(t, 0, len(invRst2))
	inv.SetAttachments(tagRouterTestDubboForceTag, tagRouterTestFalse)
	inv.SetAttachments(tagRouterTestDubboTag, tagRouterTestGuangZhou)
	invRst3 := tagRouter.Route(invokers, &u1, inv)
	assert.Equal(t, 3, len(invRst3))
}

func TestTagRouterRouteNoForce(t *testing.T) {
	u1, e1 := common.NewURL(tagRouterTestUserConsumer)
	assert.Nil(t, e1)
	tagRouter, e := NewTagRouter(&u1)
	assert.Nil(t, e)

	u2, e2 := common.NewURL(tagRouterTestHangZhouUrl)
	u3, e3 := common.NewURL(tagRouterTestShangHaiUrl)
	u4, e4 := common.NewURL(tagRouterTestShangHaiUrl)
	assert.Nil(t, e2)
	assert.Nil(t, e3)
	assert.Nil(t, e4)
	inv2 := NewMockInvoker(u2)
	inv3 := NewMockInvoker(u3)
	inv4 := NewMockInvoker(u4)
	var invokers []protocol.Invoker
	invokers = append(invokers, inv2, inv3, inv4)
	inv := &invocation.RPCInvocation{}
	inv.SetAttachments(tagRouterTestDubboTag, tagRouterTestHangZhou)
	invRst := tagRouter.Route(invokers, &u1, inv)
	assert.Equal(t, 1, len(invRst))
	assert.Equal(t, tagRouterTestHangZhou, invRst[0].GetUrl().GetParam(tagRouterTestDubboTag, ""))

	inv.SetAttachments(tagRouterTestDubboTag, tagRouterTestGuangZhou)
	inv.SetAttachments(tagRouterTestDubboForceTag, tagRouterTestTrue)
	invRst1 := tagRouter.Route(invokers, &u1, inv)
	assert.Equal(t, 0, len(invRst1))
	inv.SetAttachments(tagRouterTestDubboForceTag, tagRouterTestFalse)
	invRst2 := tagRouter.Route(invokers, &u1, inv)
	assert.Equal(t, 3, len(invRst2))
}

func TestFilterInvoker(t *testing.T) {
	u2, e2 := common.NewURL(tagRouterTestHangZhouUrl)
	u3, e3 := common.NewURL(tagRouterTestShangHaiUrl)
	u4, e4 := common.NewURL(tagRouterTestBeijingUrl)
	u5, e5 := common.NewURL(tagRouterTestEnabledBeijingUrl)
	assert.Nil(t, e2)
	assert.Nil(t, e3)
	assert.Nil(t, e4)
	assert.Nil(t, e5)
	inv2 := NewMockInvoker(u2)
	inv3 := NewMockInvoker(u3)
	inv4 := NewMockInvoker(u4)
	inv5 := NewMockInvoker(u5)
	var invokers []protocol.Invoker
	invokers = append(invokers, inv2, inv3, inv4, inv5)
	filterTag := func(invoker protocol.Invoker) bool {
		if invoker.GetUrl().GetParam(constant.Tagkey, "") == "beijing" {
			return true
		}
		return false
	}
	res := filterInvoker(invokers, filterTag)
	assert.Equal(t, []protocol.Invoker{inv4, inv5}, res)
	flag := true
	filterEnabled := func(invoker protocol.Invoker) bool {
		if invoker.GetUrl().GetParamBool(constant.RouterEnabled, false) == flag {
			return true
		}
		return false
	}
	res2 := filterInvoker(invokers, filterTag, filterEnabled)
	assert.Equal(t, []protocol.Invoker{inv4}, res2)
}

type DynamicTagRouter struct {
	suite.Suite
	rule *RouterRule

	route       *tagRouter
	zkClient    *zookeeper.ZookeeperClient
	testCluster *zk.TestCluster
	invokers    []protocol.Invoker
	url         *common.URL
}

func TestDynamicTagRouter(t *testing.T) {
	dtg := &DynamicTagRouter{}
	u1, _ := common.NewURL(tagRouterTestDynamicIpv4Provider1)
	u2, _ := common.NewURL(tagRouterTestDynamicIpv4Provider2)
	u3, _ := common.NewURL(tagRouterTestDynamicIpv4Provider3)
	u4, _ := common.NewURL(tagRouterTestDynamicIpv4Provider4)
	u5, _ := common.NewURL(tagRouterTestDynamicIpv4Provider5)
	inv1 := NewMockInvoker(u1)
	inv2 := NewMockInvoker(u2)
	inv3 := NewMockInvoker(u3)
	inv4 := NewMockInvoker(u4)
	inv5 := NewMockInvoker(u5)
	dtg.invokers = append(dtg.invokers, inv1, inv2, inv3, inv4, inv5)
	suite.Run(t, dtg)
}

func (suite *DynamicTagRouter) SetupTest() {
	var err error
	testYML := `enabled: true
scope: application
force: true
runtime: false
valid: true
priority: 1
key: demo-provider
tags:
  - name: tag1
    addresses: ["127.0.0.1:20001"]
  - name: tag2
    addresses: ["127.0.0.1:20002"]
  - name: tag3
    addresses: ["127.0.0.1:20003", "127.0.0.1:20004"]
`
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	suite.NoError(err)
	err = z.Create(routerPath)
	suite.NoError(err)

	suite.zkClient = z
	suite.testCluster = ts

	_, err = z.Conn.Set(routerPath, []byte(testYML), 0)
	suite.NoError(err)

	zkUrl, _ := common.NewURL(fmt.Sprintf(zkFormat, routerLocalIP, suite.testCluster.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory(routerZk).GetDynamicConfiguration(&zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	suite.Nil(err)
	suite.NotNil(configuration)

	url, e1 := common.NewURL(tagRouterTestUserConsumerTag)
	suite.Nil(e1)

	tagRouter, err := NewTagRouter(&url)
	suite.Nil(err)
	suite.NotNil(tagRouter)
	suite.route = tagRouter
	suite.url = &url
}

func (suite *DynamicTagRouter) TearDownTest() {
	suite.zkClient.Close()
	suite.testCluster.Stop()
}

func (suite *DynamicTagRouter) TestDynamicTagRouterSetByIPv4() {
	invokers := suite.invokers
	suite.route.Notify(invokers)
	suite.NotNil(suite.route.tagRouterRule)

	consumer := &invocation.RPCInvocation{}
	consumer.SetAttachments(tagRouterTestDubboTag, "tag1")
	targetInvokers := suite.route.Route(invokers, suite.url, consumer)
	suite.Equal(1, len(targetInvokers))
	suite.Equal(targetInvokers[0], suite.invokers[0])

	consumer.SetAttachments(tagRouterTestDubboTag, "tag3")
	targetInvokers = suite.route.Route(invokers, suite.url, consumer)
	suite.Equal(2, len(targetInvokers))
	suite.Equal(targetInvokers, []protocol.Invoker{suite.invokers[2], suite.invokers[3]})
}

func (suite *DynamicTagRouter) TestDynamicTagRouterStaticTag() {
	invokers := suite.invokers
	consumer := &invocation.RPCInvocation{}
	consumer.SetAttachments(tagRouterTestDubboTag, "tag4")
	targetInvokers := suite.route.Route(invokers, suite.url, consumer)
	suite.Equal(1, len(targetInvokers))
	suite.Equal(targetInvokers[0], suite.invokers[3])
}

// Teas no tag and return a address are not in dynamic tag group
func (suite *DynamicTagRouter) TestDynamicTagRouterByNoTagAndAddressMatch() {
	invokers := suite.invokers
	suite.route.Notify(invokers)
	suite.NotNil(suite.route.tagRouterRule)
	consumer := &invocation.RPCInvocation{}
	targetInvokers := suite.route.Route(invokers, suite.url, consumer)
	suite.Equal(1, len(targetInvokers))
	suite.Equal(targetInvokers[0], suite.invokers[4])
	// test if there are some addresses that are not in any dynamic tag group, continue to filter using the static tag group.
	consumer.SetAttachments(tagRouterTestDubboTag, "tag5")
	targetInvokers = suite.route.Route(invokers, suite.url, consumer)
	suite.Equal(1, len(targetInvokers))
	suite.Equal(targetInvokers[0], suite.invokers[4])
}

func (suite *DynamicTagRouter) TestTODO() {
	testYML := `enabled: true
scope: application
force: true
runtime: false
valid: true
priority: 1
key: demo-provider
tags:
  - name: tag1
    addresses: ["127.0.0.1:20001"]
  - name: tag2
    addresses: ["127.0.0.1:20002"]
  - name: tag3
    addresses: ["127.0.0.1:20003", "127.0.0.1:20004"]
`
	_, err := suite.zkClient.Conn.Set(routerPath, []byte(testYML), 1)
	suite.NoError(err)

	zkUrl, _ := common.NewURL(fmt.Sprintf(zkFormat, routerLocalIP, suite.testCluster.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory(routerZk).GetDynamicConfiguration(&zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)
}

func TestProcess(t *testing.T) {
	u1, err := common.NewURL(tagRouterTestUserConsumerTag)
	assert.Nil(t, err)
	tagRouter, e := NewTagRouter(&u1)
	assert.Nil(t, e)
	assert.NotNil(t, tagRouter)

	testYML := `
scope: application
force: true
runtime: false
enabled: true
valid: true
priority: 1
key: demo-provider
tags:
  - name: beijing
    addresses: [192.168.1.1, 192.168.1.2]
  - name: hangzhou
    addresses: [192.168.1.3, 192.168.1.4]
`
	tagRouter.Process(&config_center.ConfigChangeEvent{Value: testYML, ConfigType: remoting.EventTypeAdd})
	assert.NotNil(t, tagRouter.tagRouterRule)
	assert.Equal(t, []string{"beijing", "hangzhou"}, tagRouter.tagRouterRule.getTagNames())
	assert.Equal(t, []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"}, tagRouter.tagRouterRule.getAddresses())
	assert.Equal(t, []string{"192.168.1.3", "192.168.1.4"}, tagRouter.tagRouterRule.getTagNameToAddresses()["hangzhou"])
	assert.Equal(t, []string{"beijing"}, tagRouter.tagRouterRule.getAddressToTagNames()["192.168.1.1"])
	tagRouter.Process(&config_center.ConfigChangeEvent{ConfigType: remoting.EventTypeDel})
	assert.Nil(t, tagRouter.tagRouterRule)
}
