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

package cluster_impl

import (
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

const (
	baseClusterInvokerMethodName = "getUser"
	baseClusterInvokerFormat     = "dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider"
)

func TestStickyNormal(t *testing.T) {
	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf(baseClusterInvokerFormat, i))
		url.SetParam("sticky", "true")
		invokers = append(invokers, NewMockInvoker(url, 1))
	}
	base := &baseClusterInvoker{}
	base.availablecheck = true
	invoked := []protocol.Invoker{}

	tmpRandomBalance := loadbalance.NewRandomLoadBalance()
	tmpInvocation := invocation.NewRPCInvocation(baseClusterInvokerMethodName, nil, nil)
	result := base.doSelect(tmpRandomBalance, tmpInvocation, invokers, invoked)
	result1 := base.doSelect(tmpRandomBalance, tmpInvocation, invokers, invoked)
	assert.Equal(t, result, result1)
}

func TestStickyNormalWhenError(t *testing.T) {
	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf(baseClusterInvokerFormat, i))
		url.SetParam("sticky", "true")
		invokers = append(invokers, NewMockInvoker(url, 1))
	}
	base := &baseClusterInvoker{}
	base.availablecheck = true

	invoked := []protocol.Invoker{}
	result := base.doSelect(loadbalance.NewRandomLoadBalance(), invocation.NewRPCInvocation(baseClusterInvokerMethodName, nil, nil), invokers, invoked)
	invoked = append(invoked, result)
	result1 := base.doSelect(loadbalance.NewRandomLoadBalance(), invocation.NewRPCInvocation(baseClusterInvokerMethodName, nil, nil), invokers, invoked)
	assert.NotEqual(t, result, result1)
}
