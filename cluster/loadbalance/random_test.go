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
	"net/url"
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

func Test_RandomlbSelect(t *testing.T) {
	randomlb := NewRandomLoadBalance()

	invokers := []protocol.Invoker{}

	url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", 0))
	invokers = append(invokers, protocol.NewBaseInvoker(url))
	i := randomlb.Select(invokers, &invocation.RPCInvocation{})
	assert.True(t, i.GetUrl().URLEqual(url))

	for i := 1; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}
	randomlb.Select(invokers, &invocation.RPCInvocation{})
}

func Test_RandomlbSelectWeight(t *testing.T) {
	randomlb := NewRandomLoadBalance()

	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}

	urlParams := url.Values{}
	urlParams.Set("methods.test."+constant.WEIGHT_KEY, "10000000000000")
	urll, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.100:20000/com.ikurento.user.UserProvider"), common.WithParams(urlParams))
	invokers = append(invokers, protocol.NewBaseInvoker(urll))
	ivc := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	var selectedInvoker []protocol.Invoker
	var selected float64
	for i := 0; i < 10000; i++ {
		s := randomlb.Select(invokers, ivc)
		if s.GetUrl().Ip == "192.168.1.100" {
			selected++
		}
		selectedInvoker = append(selectedInvoker, s)
	}

	assert.Condition(t, func() bool {
		//really is 0.9999999999999
		return selected/10000 > 0.9
	})
}

func Test_RandomlbSelectWarmup(t *testing.T) {
	randomlb := NewRandomLoadBalance()

	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}

	urlParams := url.Values{}
	urlParams.Set(constant.REMOTE_TIMESTAMP_KEY, strconv.FormatInt(time.Now().Add(time.Minute*(-9)).Unix(), 10))
	urll, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.100:20000/com.ikurento.user.UserProvider"), common.WithParams(urlParams))
	invokers = append(invokers, protocol.NewBaseInvoker(urll))
	ivc := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))

	var selectedInvoker []protocol.Invoker
	var selected float64
	for i := 0; i < 10000; i++ {
		s := randomlb.Select(invokers, ivc)
		if s.GetUrl().Ip == "192.168.1.100" {
			selected++
		}
		selectedInvoker = append(selectedInvoker, s)
	}
	assert.Condition(t, func() bool {
		return selected/10000 < 0.1
	})
}
