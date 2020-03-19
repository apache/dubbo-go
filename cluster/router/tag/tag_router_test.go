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

func TestTagRouter_Priority(t *testing.T) {
	u1, err := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserConsumer?interface=com.ikurento.user.UserConsumer&group=&version=2.6.0&enabled=true&dubbo.force.tag=true")
	assert.Nil(t, err)
	tagRouter, e := NewTagRouter(&u1)
	assert.Nil(t, e)
	p := tagRouter.Priority()
	assert.Equal(t, int64(0), p)
}
func TestTagRouter_Route_force(t *testing.T) {
	u1, e1 := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserConsumer?interface=com.ikurento.user.UserConsumer&group=&version=2.6.0&enabled=true&dubbo.force.tag=true")
	assert.Nil(t, e1)
	tagRouter, e := NewTagRouter(&u1)
	assert.Nil(t, e)

	u2, e2 := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true&dubbo.tag=hangzhou")
	u3, e3 := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true&dubbo.tag=shanghai")
	u4, e4 := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true&dubbo.tag=beijing")
	assert.Nil(t, e2)
	assert.Nil(t, e3)
	assert.Nil(t, e4)
	inv2 := NewMockInvoker(u2)
	inv3 := NewMockInvoker(u3)
	inv4 := NewMockInvoker(u4)
	var invokers []protocol.Invoker
	invokers = append(invokers, inv2, inv3, inv4)
	inv := &invocation.RPCInvocation{}
	inv.SetAttachments("dubbo.tag", "hangzhou")
	invRst1 := tagRouter.Route(invokers, &u1, inv)
	assert.Equal(t, 1, len(invRst1))
	assert.Equal(t, "hangzhou", invRst1[0].GetUrl().GetParam("dubbo.tag", ""))

	inv.SetAttachments("dubbo.tag", "guangzhou")
	invRst2 := tagRouter.Route(invokers, &u1, inv)
	assert.Equal(t, 0, len(invRst2))
	inv.SetAttachments("dubbo.force.tag", "false")
	inv.SetAttachments("dubbo.tag", "guangzhou")
	invRst3 := tagRouter.Route(invokers, &u1, inv)
	assert.Equal(t, 3, len(invRst3))
}

func TestTagRouter_Route_noForce(t *testing.T) {
	u1, e1 := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserConsumer?interface=com.ikurento.user.UserConsumer&group=&version=2.6.0&enabled=true")
	assert.Nil(t, e1)
	tagRouter, e := NewTagRouter(&u1)
	assert.Nil(t, e)

	u2, e2 := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true&dubbo.tag=hangzhou")
	u3, e3 := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true&dubbo.tag=shanghai")
	u4, e4 := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true&dubbo.tag=beijing")
	assert.Nil(t, e2)
	assert.Nil(t, e3)
	assert.Nil(t, e4)
	inv2 := NewMockInvoker(u2)
	inv3 := NewMockInvoker(u3)
	inv4 := NewMockInvoker(u4)
	var invokers []protocol.Invoker
	invokers = append(invokers, inv2, inv3, inv4)
	inv := &invocation.RPCInvocation{}
	inv.SetAttachments("dubbo.tag", "hangzhou")
	invRst := tagRouter.Route(invokers, &u1, inv)
	assert.Equal(t, 1, len(invRst))
	assert.Equal(t, "hangzhou", invRst[0].GetUrl().GetParam("dubbo.tag", ""))

	inv.SetAttachments("dubbo.tag", "guangzhou")
	inv.SetAttachments("dubbo.force.tag", "true")
	invRst1 := tagRouter.Route(invokers, &u1, inv)
	assert.Equal(t, 0, len(invRst1))
	inv.SetAttachments("dubbo.force.tag", "false")
	invRst2 := tagRouter.Route(invokers, &u1, inv)
	assert.Equal(t, 3, len(invRst2))
}
