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

const (
	host1 = "1.1.1.1"
	host2 = "10.20.3.3"

	conditionAddr = "condition://127.0.0.1/com.foo.BarService"

	localConsumerAddr  = "consumer://127.0.0.1/com.foo.BarService"
	remoteConsumerAddr = "consumer://" + host1 + "/com.foo.BarService"

	localProviderAddr  = "dubbo://127.0.0.1:20880/com.foo.BarService"
	remoteProviderAddr = "dubbo://" + host2 + ":20880/com.foo.BarService"
)

func TestRouteMatchWhen(t *testing.T) {

	rpcInvocation := invocation.NewRPCInvocation("getFoo", nil, nil)
	whenConsumerURL, _ := common.NewURL(remoteConsumerAddr)

	testData := []struct {
		name        string
		consumerUrl *common.URL
		rule        string

		wantVal bool
	}{
		{
			name:        " => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        " => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "host = 1.2.3.4 => ",
			consumerUrl: whenConsumerURL,
			rule:        "host = 1.2.3.4 => ",

			wantVal: false,
		},
		{
			name:        "host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        "host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "host = 2.2.2.2,1.1.1.1,3.3.3.3 & host !=1.1.1.1 => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        "host = 2.2.2.2,1.1.1.1,3.3.3.3 & host !=1.1.1.1 => host = 1.2.3.4",

			wantVal: false,
		},
		{
			name:        "host !=4.4.4.4 & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        "host !=4.4.4.4 & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "host !=4.4.4.* & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        "host !=4.4.4.* & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.1 => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        "host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.1 => host = 1.2.3.4",

			wantVal: false,
		},
		{
			name:        "host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.2 => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        "host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.2 => host = 1.2.3.4",

			wantVal: true,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			url, err := common.NewURL(conditionAddr)
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)
			resVal := router.matchWhen(data.consumerUrl, rpcInvocation)
			assert.Equal(t, data.wantVal, resVal)
		})
	}
}

// TestRouteMatchFilter also tests pattern_value.WildcardValuePattern's Match method
func TestRouteMatchFilter(t *testing.T) {

	consumerURL, _ := common.NewURL(localConsumerAddr)
	url1, _ := common.NewURL(remoteProviderAddr + "?serialization=fastjson")
	url2, _ := common.NewURL(localProviderAddr)
	url3, _ := common.NewURL(localProviderAddr)

	rpcInvocation := invocation.NewRPCInvocation("getFoo", nil, nil)

	ink1 := protocol.NewBaseInvoker(url1)
	ink2 := protocol.NewBaseInvoker(url2)
	ink3 := protocol.NewBaseInvoker(url3)

	invokerList := make([]protocol.Invoker, 0, 3)
	invokerList = append(invokerList, ink1)
	invokerList = append(invokerList, ink2)
	invokerList = append(invokerList, ink3)

	testData := []struct {
		name        string
		comsumerURL *common.URL
		rule        string

		wantVal int
	}{
		{
			name:        "host = 127.0.0.1 => host = 10.20.3.3",
			comsumerURL: consumerURL,
			rule:        "host = 127.0.0.1 => host = 10.20.3.3",

			wantVal: 1,
		},
		{
			name:        "host = 127.0.0.1 => host = 10.20.3.* & host != 10.20.3.3",
			comsumerURL: consumerURL,
			rule:        "host = 127.0.0.1 => host = 10.20.3.* & host != 10.20.3.3",

			wantVal: 0,
		},
		{
			name:        "host = 127.0.0.1 => host = 10.20.3.3  & host != 10.20.3.3",
			comsumerURL: consumerURL,
			rule:        "host = 127.0.0.1 => host = 10.20.3.3  & host != 10.20.3.3",

			wantVal: 0,
		},
		{
			name:        "host = 127.0.0.1 => host = 10.20.3.2,10.20.3.3,10.20.3.4",
			comsumerURL: consumerURL,
			rule:        "host = 127.0.0.1 => host = 10.20.3.2,10.20.3.3,10.20.3.4",

			wantVal: 1,
		},
		{
			name:        "host = 127.0.0.1 => host != 10.20.3.3",
			comsumerURL: consumerURL,
			rule:        "host = 127.0.0.1 => host != 10.20.3.3",

			wantVal: 2,
		},
		{
			name:        "host = 127.0.0.1 => serialization = fastjson",
			comsumerURL: consumerURL,
			rule:        "host = 127.0.0.1 => serialization = fastjson",

			wantVal: 1,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			url, err := common.NewURL(conditionAddr)
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			url.AddParam(constant.ForceKey, "true")

			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)

			filteredInvokers := router.Route(invokerList, data.comsumerURL, rpcInvocation)
			resVal := len(filteredInvokers)
			assert.Equal(t, data.wantVal, resVal)
		})
	}
}

func TestRouterMethodRoute(t *testing.T) {

	rpcInvocation := invocation.NewRPCInvocation("getFoo", nil, nil)

	testData := []struct {
		name        string
		consumerURL string
		rule        string

		wantVal bool
	}{
		{
			name:        "More than one methods, mismatch",
			consumerURL: remoteConsumerAddr + "?methods=setFoo,getFoo,findFoo",
			rule:        "methods=getFoo => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "Exactly one method, match",
			consumerURL: remoteConsumerAddr + "?methods=getFoo",
			rule:        "methods=getFoo => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "Method routing and Other condition routing can work together",
			consumerURL: remoteConsumerAddr + "?methods=getFoo",
			rule:        "methods=getFoo & host!=1.1.1.1 => host = 1.2.3.4",

			wantVal: false,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			url, err := common.NewURL(conditionAddr)
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)
			consumer, _ := common.NewURL(data.consumerURL)
			resVal := router.matchWhen(consumer, rpcInvocation)
			assert.Equal(t, data.wantVal, resVal)
		})
	}
}

func TestRouteReturn(t *testing.T) {

	rpcInvocation := invocation.NewRPCInvocation("getFoo", nil, nil)
	consumerURL, _ := common.NewURL(localConsumerAddr)

	testData := []struct {
		name string
		urls []string
		rule string

		wantUrls []string
		wantVal  int
	}{
		{
			name: "ReturnFalse",
			urls: []string{
				"",
				"",
				"",
			},
			rule: "host = 127.0.0.1 => false",

			wantUrls: []string{},
			wantVal:  0,
		},
		{
			name: "ReturnEmpty",
			urls: []string{
				"",
				"",
				"",
			},
			rule: "host = 127.0.0.1 => ",

			wantUrls: []string{},
			wantVal:  0,
		},
		{
			name: "ReturnAll",
			urls: []string{
				localProviderAddr,
				localProviderAddr,
				localProviderAddr,
			},
			rule: "host = 127.0.0.1 => host = 127.0.0.1",

			wantUrls: []string{
				localProviderAddr,
				localProviderAddr,
				localProviderAddr,
			},
			wantVal: 3,
		},
		{
			name: "HostFilter",
			urls: []string{
				remoteProviderAddr,
				localProviderAddr,
				localProviderAddr,
			},
			rule: "host = 127.0.0.1 => host = 127.0.0.1",

			wantUrls: []string{
				localProviderAddr,
				localProviderAddr,
			},
			wantVal: 2,
		},
		{
			name: "EmptyHostFilter",
			urls: []string{
				remoteProviderAddr,
				localProviderAddr,
				localProviderAddr,
			},
			rule: " => host = 127.0.0.1",

			wantUrls: []string{
				localProviderAddr,
				localProviderAddr,
			},
			wantVal: 2,
		},
		{
			name: "FalseHostFilter",
			urls: []string{
				remoteProviderAddr,
				localProviderAddr,
				localProviderAddr,
			},
			rule: "true => host = 127.0.0.1",

			wantUrls: []string{
				localProviderAddr,
				localProviderAddr,
			},
			wantVal: 2,
		},
		{
			name: "PlaceHolder",
			urls: []string{
				remoteProviderAddr,
				localProviderAddr,
				localProviderAddr,
			},
			rule: "host = 127.0.0.1 => host = $host",

			wantUrls: []string{
				localProviderAddr,
				localProviderAddr,
			},
			wantVal: 2,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {

			invokers := make([]protocol.Invoker, 0, len(data.urls))
			for _, urlStr := range data.urls {
				url, _ := common.NewURL(urlStr)
				invoker := protocol.NewBaseInvoker(url)
				invokers = append(invokers, invoker)
			}

			wantInvokers := make([]protocol.Invoker, 0, len(data.wantUrls))
			for _, wantUrlStr := range data.wantUrls {
				url, _ := common.NewURL(wantUrlStr)
				invoker := protocol.NewBaseInvoker(url)
				wantInvokers = append(wantInvokers, invoker)
			}

			url, err := common.NewURL(conditionAddr)
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)

			filterInvokers := router.Route(invokers, consumerURL, rpcInvocation)
			resVal := len(filterInvokers)

			assert.Equal(t, data.wantVal, resVal)
			assert.Equal(t, wantInvokers, filterInvokers)
		})
	}
}

// TestRouteArguments also tests matcher.ArgumentConditionMatcher's GetValue method
func TestRouteArguments(t *testing.T) {

	url1, _ := common.NewURL(remoteProviderAddr)
	url2, _ := common.NewURL(localProviderAddr)
	url3, _ := common.NewURL(localProviderAddr)

	ink1 := protocol.NewBaseInvoker(url1)
	ink2 := protocol.NewBaseInvoker(url2)
	ink3 := protocol.NewBaseInvoker(url3)

	invokerList := make([]protocol.Invoker, 0, 3)
	invokerList = append(invokerList, ink1)
	invokerList = append(invokerList, ink2)
	invokerList = append(invokerList, ink3)

	consumerURL, _ := common.NewURL(localConsumerAddr)

	testData := []struct {
		name     string
		argument interface{}
		rule     string

		wantVal int
	}{
		{
			name:     "Empty arguments",
			argument: nil,
			rule:     "arguments[0] = a => host = 1.2.3.4",

			wantVal: 3,
		},
		{
			name:     "String arguments",
			argument: "a",
			rule:     "arguments[0] = a => host = 1.2.3.4",

			wantVal: 0,
		},
		{
			name:     "Int arguments",
			argument: 1,
			rule:     "arguments[0] = 1 => host = 127.0.0.1",

			wantVal: 2,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {

			url, err := common.NewURL(conditionAddr)
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			url.AddParam(constant.ForceKey, "true")
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)

			arguments := make([]interface{}, 0, 1)
			arguments = append(arguments, data.argument)

			rpcInvocation := invocation.NewRPCInvocation("getBar", arguments, nil)

			filterInvokers := router.Route(invokerList, consumerURL, rpcInvocation)
			resVal := len(filterInvokers)
			assert.Equal(t, data.wantVal, resVal)

		})
	}
}
