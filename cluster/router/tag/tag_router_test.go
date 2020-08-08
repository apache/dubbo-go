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
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/utils"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

const (
	tagRouterTestHangZhouUrl     = "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true&dubbo.tag=hangzhou"
	tagRouterTestShangHaiUrl     = "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true&dubbo.tag=shanghai"
	tagRouterTestBeijingUrl      = "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true&dubbo.tag=beijing"
	tagRouterTestUserConsumer    = "dubbo://127.0.0.1:20000/com.ikurento.user.UserConsumer?interface=com.ikurento.user.UserConsumer&group=&version=2.6.0&enabled=true"
	tagRouterTestUserConsumerTag = "dubbo://127.0.0.1:20000/com.ikurento.user.UserConsumer?interface=com.ikurento.user.UserConsumer&group=&version=2.6.0&enabled=true&dubbo.force.tag=true"

	tagRouterTestDubboTag      = "dubbo.tag"
	tagRouterTestDubboForceTag = "dubbo.force.tag"
	tagRouterTestHangZhou      = "hangzhou"
	tagRouterTestGuangZhou     = "guangzhou"
	tagRouterTestFalse         = "false"
	tagRouterTestTrue          = "true"
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
	invRst1 := tagRouter.Route(utils.ToBitmap(invokers), setUpAddrCache(tagRouter, invokers), &u1, inv)
	assert.Equal(t, 1, len(invRst1.ToArray()))
	assert.Equal(t, tagRouterTestHangZhou, invokers[invRst1.ToArray()[0]].GetUrl().GetParam(tagRouterTestDubboTag, ""))

	inv.SetAttachments(tagRouterTestDubboTag, tagRouterTestGuangZhou)
	invRst2 := tagRouter.Route(utils.ToBitmap(invokers), setUpAddrCache(tagRouter, invokers), &u1, inv)
	assert.Equal(t, 0, len(invRst2.ToArray()))
	inv.SetAttachments(tagRouterTestDubboForceTag, tagRouterTestFalse)
	inv.SetAttachments(tagRouterTestDubboTag, tagRouterTestGuangZhou)
	invRst3 := tagRouter.Route(utils.ToBitmap(invokers), setUpAddrCache(tagRouter, invokers), &u1, inv)
	assert.Equal(t, 3, len(invRst3.ToArray()))
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
	invRst := tagRouter.Route(utils.ToBitmap(invokers), setUpAddrCache(tagRouter, invokers), &u1, inv)
	assert.Equal(t, 1, len(invRst.ToArray()))
	assert.Equal(t, tagRouterTestHangZhou, invokers[invRst.ToArray()[0]].GetUrl().GetParam(tagRouterTestDubboTag, ""))

	inv.SetAttachments(tagRouterTestDubboTag, tagRouterTestGuangZhou)
	inv.SetAttachments(tagRouterTestDubboForceTag, tagRouterTestTrue)
	invRst1 := tagRouter.Route(utils.ToBitmap(invokers), setUpAddrCache(tagRouter, invokers), &u1, inv)
	assert.Equal(t, 0, len(invRst1.ToArray()))
	inv.SetAttachments(tagRouterTestDubboForceTag, tagRouterTestFalse)
	invRst2 := tagRouter.Route(utils.ToBitmap(invokers), setUpAddrCache(tagRouter, invokers), &u1, inv)
	assert.Equal(t, 3, len(invRst2.ToArray()))
}

func setUpAddrCache(r router.Poolable, addrs []protocol.Invoker) *router.AddrCache {
	pool, info := r.Pool(addrs)
	cache := &router.AddrCache{
		Invokers: addrs,
		AddrPool: make(map[string]router.RouterAddrPool),
		AddrMeta: make(map[string]router.AddrMetadata),
	}

	cache.AddrMeta[r.Name()] = info
	cache.AddrPool[r.Name()] = pool
	return cache
}
