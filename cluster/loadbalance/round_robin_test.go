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
	"fmt"
	"strconv"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestRoundRobinSelect(t *testing.T) {
	loadBalance := NewRoundRobinLoadBalance()

	var invokers []protocol.Invoker

	url, _ := common.NewURL(fmt.Sprintf("dubbo://%s:%d/org.apache.demo.HelloService",
		constant.LOCAL_HOST_VALUE, constant.DEFAULT_PORT))
	invokers = append(invokers, protocol.NewBaseInvoker(url))
	i := loadBalance.Select(invokers, &invocation.RPCInvocation{})
	assert.True(t, i.GetURL().URLEqual(url))

	for i := 1; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/org.apache.demo.HelloService", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}
	loadBalance.Select(invokers, &invocation.RPCInvocation{})
}

func TestRoundRobinByWeight(t *testing.T) {
	loadBalance := NewRoundRobinLoadBalance()

	var invokers []protocol.Invoker
	loop := 10
	for i := 1; i <= loop; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/org.apache.demo.HelloService?weight=%v", i, i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}

	loop = (1 + loop) * loop / 2
	selected := make(map[protocol.Invoker]int)

	for i := 1; i <= loop; i++ {
		invoker := loadBalance.Select(invokers, &invocation.RPCInvocation{})
		selected[invoker]++
	}

	for _, i := range invokers {
		w, _ := strconv.Atoi(i.GetURL().GetParam("weight", "-1"))
		assert.True(t, selected[i] == w)
	}
}
