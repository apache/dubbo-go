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

package loadbalance

import (
	"context"
	"fmt"
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

func TestLeastActiveSelect(t *testing.T) {
	loadBalance := NewLeastActiveLoadBalance()

	var invokers []protocol.Invoker

	url, _ := common.NewURL(context.TODO(), "dubbo://192.168.1.0:20000/org.apache.demo.HelloService")
	invokers = append(invokers, protocol.NewBaseInvoker(url))
	i := loadBalance.Select(invokers, &invocation.RPCInvocation{})
	assert.True(t, i.GetUrl().URLEqual(url))

	for i := 1; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/org.apache.demo.HelloService", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}
	loadBalance.Select(invokers, &invocation.RPCInvocation{})
}

func TestLeastActiveByWeight(t *testing.T) {
	loadBalance := NewLeastActiveLoadBalance()

	var invokers []protocol.Invoker
	loop := 3
	for i := 1; i <= loop; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("test%v://192.168.1.%v:20000/org.apache.demo.HelloService?weight=%v", i, i, i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))
	protocol.BeginCount(invokers[2].GetUrl(), inv.MethodName())

	loop = 10000

	var (
		firstCount  int
		secondCount int
	)

	for i := 1; i <= loop; i++ {
		invoker := loadBalance.Select(invokers, inv)
		if invoker.GetUrl().Protocol == "test1" {
			firstCount++
		} else if invoker.GetUrl().Protocol == "test2" {
			secondCount++
		}
	}

	assert.Equal(t, firstCount+secondCount, loop)
}
