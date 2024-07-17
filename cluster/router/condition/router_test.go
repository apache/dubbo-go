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
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	commonConfig "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/configurator"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

const (
	conditionAddr = "condition://127.0.0.1/com.foo.BarService"

	localConsumerAddr  = "consumer://127.0.0.1/com.foo.BarService"
	remoteConsumerAddr = "consumer://dubbo.apache.org/com.foo.BarService"

	localProviderAddr  = "dubbo://127.0.0.1:20880/com.foo.BarService"
	remoteProviderAddr = "dubbo://dubbo.apache.org:20880/com.foo.BarService"
	emptyProviderAddr  = ""

	region = "?region=hangzhou"
	method = "getFoo"
)

func TestRouteMatchWhen(t *testing.T) {

	rpcInvocation := invocation.NewRPCInvocation(method, nil, nil)
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
			name:        "host = 2.2.2.2,dubbo.apache.org,3.3.3.3 => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        "host = 2.2.2.2,dubbo.apache.org,3.3.3.3 => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "host = 2.2.2.2,dubbo.apache.org,3.3.3.3 & host !=dubbo.apache.org => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        "host = 2.2.2.2,dubbo.apache.org,3.3.3.3 & host !=dubbo.apache.org => host = 1.2.3.4",

			wantVal: false,
		},
		{
			name:        "host !=4.4.4.4 & host = 2.2.2.2,dubbo.apache.org,3.3.3.3 => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        "host !=4.4.4.4 & host = 2.2.2.2,dubbo.apache.org,3.3.3.3 => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "host !=4.4.4.* & host = 2.2.2.2,dubbo.apache.org,3.3.3.3 => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        "host !=4.4.4.* & host = 2.2.2.2,dubbo.apache.org,3.3.3.3 => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "host = 2.2.2.2,dubbo.apache.*,3.3.3.3 & host != dubbo.apache.org => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        "host = 2.2.2.2,dubbo.apache.*,3.3.3.3 & host != dubbo.apache.org => host = 1.2.3.4",

			wantVal: false,
		},
		{
			name:        "host = 2.2.2.2,dubbo.apache.*,3.3.3.3 & host != 1.1.1.2 => host = 1.2.3.4",
			consumerUrl: whenConsumerURL,
			rule:        "host = 2.2.2.2,dubbo.apache.*,3.3.3.3 & host != 1.1.1.2 => host = 1.2.3.4",

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

	rpcInvocation := invocation.NewRPCInvocation(method, nil, nil)

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
			name:        "host = 127.0.0.1 => host = dubbo.apache.org",
			comsumerURL: consumerURL,
			rule:        "host = 127.0.0.1 => host = dubbo.apache.org",

			wantVal: 1,
		},
		{
			name:        "host = 127.0.0.1 => host = 10.20.3.* & host != dubbo.apache.org",
			comsumerURL: consumerURL,
			rule:        "host = 127.0.0.1 => host = 10.20.3.* & host != dubbo.apache.org",

			wantVal: 0,
		},
		{
			name:        "host = 127.0.0.1 => host = dubbo.apache.org  & host != dubbo.apache.org",
			comsumerURL: consumerURL,
			rule:        "host = 127.0.0.1 => host = dubbo.apache.org  & host != dubbo.apache.org",

			wantVal: 0,
		},
		{
			name:        "host = 127.0.0.1 => host = 10.20.3.2,dubbo.apache.org,10.20.3.4",
			comsumerURL: consumerURL,
			rule:        "host = 127.0.0.1 => host = 10.20.3.2,dubbo.apache.org,10.20.3.4",

			wantVal: 1,
		},
		{
			name:        "host = 127.0.0.1 => host != dubbo.apache.org",
			comsumerURL: consumerURL,
			rule:        "host = 127.0.0.1 => host != dubbo.apache.org",

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

	rpcInvocation := invocation.NewRPCInvocation(method, nil, nil)

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
			name:        "Exactly one method, Route",
			consumerURL: remoteConsumerAddr + "?methods=getFoo",
			rule:        "methods=getFoo => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "Method routing and Other condition routing can work together",
			consumerURL: remoteConsumerAddr + "?methods=getFoo",
			rule:        "methods=getFoo & host!=dubbo.apache.org => host = 1.2.3.4",

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

	rpcInvocation := invocation.NewRPCInvocation(method, nil, nil)
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
				emptyProviderAddr,
				emptyProviderAddr,
				emptyProviderAddr,
			},
			rule: "host = 127.0.0.1 => false",

			wantUrls: []string{},
			wantVal:  0,
		},
		{
			name: "ReturnEmpty",
			urls: []string{
				emptyProviderAddr,
				emptyProviderAddr,
				emptyProviderAddr,
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

// TestRouteAttachments also tests matcher.AttachmentConditionMatcher's GetValue method
func TestRouteAttachments(t *testing.T) {
	consumerURL, _ := common.NewURL(localConsumerAddr)

	url1, _ := common.NewURL(remoteProviderAddr + region)
	url2, _ := common.NewURL(localProviderAddr)
	url3, _ := common.NewURL(localProviderAddr)

	ink1 := protocol.NewBaseInvoker(url1)
	ink2 := protocol.NewBaseInvoker(url2)
	ink3 := protocol.NewBaseInvoker(url3)

	invokerList := make([]protocol.Invoker, 0, 3)
	invokerList = append(invokerList, ink1)
	invokerList = append(invokerList, ink2)
	invokerList = append(invokerList, ink3)

	testData := []struct {
		name            string
		attachmentKey   string
		attachmentValue string
		rule            string

		wantVal int
	}{
		{
			name:            "Empty attachments",
			attachmentKey:   "",
			attachmentValue: "",
			rule:            "attachments[foo] = a => host = 1.2.3.4",

			wantVal: 3,
		},
		{
			name:            "Yes attachments and no host",
			attachmentKey:   "foo",
			attachmentValue: "a",
			rule:            "attachments[foo] = a => host = 1.2.3.4",

			wantVal: 0,
		},
		{
			name:            "No attachments and no host",
			attachmentKey:   "foo",
			attachmentValue: "a",
			rule:            "attachments = a => host = 1.2.3.4",

			wantVal: 3,
		},
		{
			name:            "Yes attachments and region",
			attachmentKey:   "foo",
			attachmentValue: "a",
			rule:            "attachments[foo] = a => region = hangzhou",

			wantVal: 1,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {

			rpcInvocation := invocation.NewRPCInvocation(method, nil, nil)
			rpcInvocation.SetAttachment(data.attachmentKey, data.attachmentValue)

			url, err := common.NewURL(conditionAddr)
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			url.AddParam(constant.ForceKey, "true")
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)

			filterInvokers := router.Route(invokerList, consumerURL, rpcInvocation)

			resVal := len(filterInvokers)
			assert.Equal(t, data.wantVal, resVal)
		})
	}
}

// TestRouteRangePattern also tests pattern_value.ScopeValuePattern's Match method
func TestRouteRangePattern(t *testing.T) {

	consumerURL, _ := common.NewURL(localConsumerAddr)

	url1, _ := common.NewURL(remoteProviderAddr + region)
	url2, _ := common.NewURL(localProviderAddr)
	url3, _ := common.NewURL(localProviderAddr)

	ink1 := protocol.NewBaseInvoker(url1)
	ink2 := protocol.NewBaseInvoker(url2)
	ink3 := protocol.NewBaseInvoker(url3)

	invokerList := make([]protocol.Invoker, 0, 3)
	invokerList = append(invokerList, ink1)
	invokerList = append(invokerList, ink2)
	invokerList = append(invokerList, ink3)

	testData := []struct {
		name            string
		attachmentKey   string
		attachmentValue string
		rule            string

		wantVal int
	}{
		{
			name:            "Empty attachment",
			attachmentKey:   "",
			attachmentValue: "",
			rule:            "attachments[user_id] = 1~100 => region=hangzhou",

			wantVal: 3,
		},
		{
			name:            "In the range",
			attachmentKey:   "user_id",
			attachmentValue: "80",
			rule:            "attachments[user_id] = 1~100 => region=hangzhou",

			wantVal: 1,
		},
		{
			name:            "Out of range",
			attachmentKey:   "user_id",
			attachmentValue: "101",
			rule:            "attachments[user_id] = 1~100 => region=hangzhou",

			wantVal: 3,
		},
		{
			name:            "In the single interval range",
			attachmentKey:   "user_id",
			attachmentValue: "1",
			rule:            "attachments[user_id] = ~100 => region=hangzhou",

			wantVal: 1,
		},
		{
			name:            "Not in the single interval range",
			attachmentKey:   "user_id",
			attachmentValue: "101",
			rule:            "attachments[user_id] = ~100 => region=hangzhou",

			wantVal: 3,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {

			rpcInvocation := invocation.NewRPCInvocation(method, nil, nil)
			rpcInvocation.SetAttachment(data.attachmentKey, data.attachmentValue)

			url, err := common.NewURL(conditionAddr)
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			url.AddParam(constant.ForceKey, "true")
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)

			filterInvokers := router.Route(invokerList, consumerURL, rpcInvocation)

			resVal := len(filterInvokers)
			assert.Equal(t, data.wantVal, resVal)
		})
	}
}

func TestRouteMultipleConditions(t *testing.T) {
	url1, _ := common.NewURL(remoteProviderAddr + region)
	url2, _ := common.NewURL(localProviderAddr)
	url3, _ := common.NewURL(localProviderAddr)

	ink1 := protocol.NewBaseInvoker(url1)
	ink2 := protocol.NewBaseInvoker(url2)
	ink3 := protocol.NewBaseInvoker(url3)

	invokerList := make([]protocol.Invoker, 0, 3)
	invokerList = append(invokerList, ink1)
	invokerList = append(invokerList, ink2)
	invokerList = append(invokerList, ink3)

	testData := []struct {
		name        string
		argument    string
		consumerURL string
		rule        string

		wantVal int
	}{
		{
			name:        "All conditions Route",
			argument:    "a",
			consumerURL: localConsumerAddr + "?application=consumer_app",
			rule:        "application=consumer_app&arguments[0]=a => host = 127.0.0.1",

			wantVal: 2,
		},
		{
			name:        "One of the conditions does not Route",
			argument:    "a",
			consumerURL: localConsumerAddr + "?application=another_consumer_app",
			rule:        "application=consumer_app&arguments[0]=a => host = 127.0.0.1",

			wantVal: 3,
		},
	}
	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			consumerUrl, err := common.NewURL(data.consumerURL)
			assert.Nil(t, err)

			url, err := common.NewURL(conditionAddr)
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			url.AddParam(constant.ForceKey, "true")
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)

			arguments := make([]interface{}, 0, 1)
			arguments = append(arguments, data.argument)

			rpcInvocation := invocation.NewRPCInvocation(method, arguments, nil)

			filterInvokers := router.Route(invokerList, consumerUrl, rpcInvocation)
			resVal := len(filterInvokers)
			assert.Equal(t, data.wantVal, resVal)
		})
	}
}

func TestServiceRouter(t *testing.T) {

	consumerURL, _ := common.NewURL(remoteConsumerAddr)

	url1, _ := common.NewURL(remoteProviderAddr)
	url2, _ := common.NewURL(remoteProviderAddr + region)
	url3, _ := common.NewURL(localProviderAddr)

	ink1 := protocol.NewBaseInvoker(url1)
	ink2 := protocol.NewBaseInvoker(url2)
	ink3 := protocol.NewBaseInvoker(url3)

	invokerList := make([]protocol.Invoker, 0, 3)
	invokerList = append(invokerList, ink1)
	invokerList = append(invokerList, ink2)
	invokerList = append(invokerList, ink3)

	extension.SetDefaultConfigurator(configurator.NewMockConfigurator)
	ccURL, _ := common.NewURL("mock://127.0.0.1:1111")
	mockFactory := &config_center.MockDynamicConfigurationFactory{
		Content: `
configVersion: v3.0
scope: service
force: true
enabled: true
runtime: true
key: com.foo.BarService
conditions:
 - 'method=sayHello => region=hangzhou'`,
	}
	dc, _ := mockFactory.GetDynamicConfiguration(ccURL)
	commonConfig.GetEnvInstance().SetDynamicConfiguration(dc)

	router := NewServiceRouter()
	router.Notify(invokerList)

	rpcInvocation := invocation.NewRPCInvocation("sayHello", nil, nil)
	invokers := router.Route(invokerList, consumerURL, rpcInvocation)
	assert.Equal(t, 1, len(invokers))

	rpcInvocation = invocation.NewRPCInvocation("sayHi", nil, nil)
	invokers = router.Route(invokerList, consumerURL, rpcInvocation)
	assert.Equal(t, 3, len(invokers))
}

func TestApplicationRouter(t *testing.T) {

	consumerURL, _ := common.NewURL(remoteConsumerAddr)

	url1, _ := common.NewURL(remoteProviderAddr + "?application=demo-provider")
	url2, _ := common.NewURL(localProviderAddr + "?application=demo-provider&region=hangzhou")
	url3, _ := common.NewURL(localProviderAddr + "?application=demo-provider")

	ink1 := protocol.NewBaseInvoker(url1)
	ink2 := protocol.NewBaseInvoker(url2)
	ink3 := protocol.NewBaseInvoker(url3)

	invokerList := make([]protocol.Invoker, 0, 3)
	invokerList = append(invokerList, ink1)
	invokerList = append(invokerList, ink2)
	invokerList = append(invokerList, ink3)

	extension.SetDefaultConfigurator(configurator.NewMockConfigurator)
	ccURL, _ := common.NewURL("mock://127.0.0.1:1111")
	mockFactory := &config_center.MockDynamicConfigurationFactory{
		Content: `
configVersion: V3.0
scope: application
force: true
enabled: true
runtime: true
key: demo-provider
conditions:
 - 'method=sayHello => region=hangzhou'`,
	}
	dc, _ := mockFactory.GetDynamicConfiguration(ccURL)
	commonConfig.GetEnvInstance().SetDynamicConfiguration(dc)

	router := NewApplicationRouter()
	router.Notify(invokerList)

	rpcInvocation := invocation.NewRPCInvocation("sayHello", nil, nil)
	invokers := router.Route(invokerList, consumerURL, rpcInvocation)
	assert.Equal(t, 1, len(invokers))

	rpcInvocation = invocation.NewRPCInvocation("sayHi", nil, nil)
	invokers = router.Route(invokerList, consumerURL, rpcInvocation)
	assert.Equal(t, 3, len(invokers))
}

var providerUrls = []string{
	"dubbo://127.0.0.1/com.foo.BarService",
	"dubbo://127.0.0.1/com.foo.BarService",
	"dubbo://127.0.0.1/com.foo.BarService?env=normal",
	"dubbo://127.0.0.1/com.foo.BarService?env=normal",
	"dubbo://127.0.0.1/com.foo.BarService?env=normal",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing&env=gray",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing&env=gray",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing&env=gray",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing&env=gray",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing&env=normal",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou&env=gray",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou&env=gray",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou&env=normal",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou&env=normal",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou&env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService",
	"dubbo://dubbo.apache.org/com.foo.BarService",
	"dubbo://dubbo.apache.org/com.foo.BarService?env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService?env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService?env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing&env=gray",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing&env=gray",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing&env=gray",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing&env=gray",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing&env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou&env=gray",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou&env=gray",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou&env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou&env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou&env=normal",
}

func buildInvokers() []protocol.Invoker {
	res := make([]protocol.Invoker, 0, len(providerUrls))
	for _, url := range providerUrls {
		u, err := common.NewURL(url)
		if err != nil {
			panic(err)
		}
		res = append(res, protocol.NewBaseInvoker(u))
	}
	return res
}

func Test_parseMultiConditionRoute(t *testing.T) {
	type args struct {
		routeContent string
	}
	tests := []struct {
		name    string
		args    args
		want    *config.ConditionRouter
		wantErr assert.ErrorAssertionFunc
	}{
		{name: "testParseConfig", args: args{`configVersion: v3.1
scope: service
key: org.apache.dubbo.samples.CommentService
force: false
runtime: true
enabled: true

#######
conditions:
  - from:
      match: tag=tag1     # disable traffic
  - from:
      match: tag=gray
    to:
      - match: tag!=gray
        weight: 100
      - match: tag=gray
        weight: 900
  - from:
      match: version=v1
    to:
      - match: version=v1`}, want: &config.ConditionRouter{
			Scope:   "service",
			Key:     "org.apache.dubbo.samples.CommentService",
			Force:   false,
			Runtime: true,
			Enabled: true,
			Conditions: []*config.ConditionRule{
				{
					From: config.ConditionRuleFrom{Match: "tag=tag1"},
					To:   nil,
				}, {
					From: config.ConditionRuleFrom{
						Match: `tag=gray`,
					},
					To: []config.ConditionRuleTo{{
						Match:  `tag!=gray`,
						Weight: 100,
					}, {
						Match:  `tag=gray`,
						Weight: 900,
					}},
				}, {
					From: config.ConditionRuleFrom{
						Match: `version=v1`,
					},
					To: []config.ConditionRuleTo{{
						Match:  `version=v1`,
						Weight: 0,
					}},
				}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMultiConditionRoute(tt.args.routeContent)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "parseMultiConditionRoute(%v)", tt.args.routeContent)
		})
	}
}

func genMatcher(rule string) FieldMatcher {
	cond, err := parseRule(rule)
	if err != nil {
		panic(err)
	}
	m := FieldMatcher{
		rule:  rule,
		match: cond,
	}
	return m
}

type InvokersFilters []FieldMatcher

func NewINVOKERS_FILTERS() InvokersFilters {
	return []FieldMatcher{}
}

func (INV InvokersFilters) add(rule string) InvokersFilters {
	m := genMatcher(rule)
	return append(INV, m)
}

func (INV InvokersFilters) filtrate(inv []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	for _, cond := range INV {
		tmpInv := make([]protocol.Invoker, 0)
		for _, invoker := range inv {
			if cond.MatchInvoker(url, invoker, invocation) {
				tmpInv = append(tmpInv, invoker)
			}
		}
		inv = tmpInv
	}
	return inv
}

func newUrl(url string) *common.URL {
	res, err := common.NewURL(url)
	if err != nil {
		panic(err)
	}
	return res
}

func Test_multiplyConditionRoute_route(t *testing.T) {
	type args struct {
		invokers   []protocol.Invoker
		url        *common.URL
		invocation protocol.Invocation
	}
	d := DynamicRouter{
		mu:              sync.RWMutex{},
		force:           false,
		enable:          false,
		conditionRouter: nil,
	}
	tests := []struct {
		name             string
		content          string
		args             args
		invokers_filters InvokersFilters
		expResLen        int
		multiDestination []struct {
			invokers_filters InvokersFilters
			weight           float32
		}
	}{
		{
			name: "test base condition",
			content: `configVersion: v3.1 
scope: service      
key: org.apache.dubbo.samples.CommentService 
force: false        
runtime: true       
enabled: true       
conditions:
  - from:
      match: env=gray
    to:
      - match: env!=gray
        weight: 100
`,
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("echo", nil, nil),
			},
			invokers_filters: NewINVOKERS_FILTERS().add("env!=gray"),
		}, {
			name: "test removeDuplicates condition",
			content: `configVersion: v3.1 
scope: service      
key: org.apache.dubbo.samples.CommentService 
force: false        
runtime: true       
enabled: true       
conditions:
  - from:
      match: env=gray
    to:
      - match: env!=gray
        weight: 100
  - from:
      match: env=gray
    to:
      - match: env!=gray
        weight: 100
`,
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("echo", nil, nil),
			},
			invokers_filters: NewINVOKERS_FILTERS().add("env!=gray"),
		}, {
			name: "test consequent condition",
			content: `configVersion: v3.1 
scope: service      
key: org.apache.dubbo.samples.CommentService 
force: false        
runtime: true       
enabled: true       
conditions:
  - from:
      match: env=gray
    to:
      - match: env!=gray
        weight: 100
  - from:
      match: region=beijing
    to:
      - match: region=beijing
        weight: 100
  - from:
    to:
      - match: host!=127.0.0.1
`,
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("echo", nil, nil),
			},
			invokers_filters: NewINVOKERS_FILTERS().add("env!=gray").add(`region=beijing`).add(`host!=127.0.0.1`),
		}, {
			name: "test unMatch condition",
			content: `configVersion: v3.1 
scope: service      
key: org.apache.dubbo.samples.CommentService 
force: false        
runtime: true       
enabled: true       
conditions:
  - from:
      match: env!=gray
    to:
      - match: env=gray
        weight: 100
  - from:
      match: region!=beijing
    to:
      - match: region=beijing
        weight: 100
  - from:
    to:
      - match: host!=127.0.0.1
`,
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("echo", nil, nil),
			},
			invokers_filters: NewINVOKERS_FILTERS().add(`host!=127.0.0.1`),
		}, {
			name: "test Match and Route zero",
			content: `configVersion: v3.1 
scope: service      
key: org.apache.dubbo.samples.CommentService
force: true # <---
runtime: true       
enabled: true       
conditions:
  - from:
      match: env=gray     # match success here
    to:
      - match: env=ErrTag # all invoker can't match this
        weight: 100
  - from:
      match: region!=beijing
    to:
      - match: region=beijing
        weight: 100
  - from:
    to:
      - match: host!=127.0.0.1
`,
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("echo", nil, nil),
			},
			expResLen: 0,
		}, {
			name: "test Match, Route zero and ignore ",
			content: `configVersion: v3.1 
scope: service      
key: org.apache.dubbo.samples.CommentService 
force: false # <--- to ignore bad result
runtime: true       
enabled: true       
conditions:
  - from:
      match: region=beijing
    to:
      - match: region!=beijing
        weight: 100
  - from:
    to:
      - match: host!=127.0.0.1
  - from:
      match: env=gray     # match success here
    to:
      - match: env=ErrTag # all invoker can't match this
        weight: 100
`,
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("echo", nil, nil),
			},
			invokers_filters: NewINVOKERS_FILTERS(),
		}, {
			name: "test traffic disabled and ignore condition-route.force",
			content: `configVersion: v3.1 
scope: service      
key: org.apache.dubbo.samples.CommentService 
force: false 
runtime: true       
enabled: true     
conditions:
  - from:
      match: host=127.0.0.1 # <--- disabled 
  - from:
      match: env=gray     
    to:
      - match: env!=gray     
        weight: 100
  - to:
      - match: region!=beijing
`,
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("echo", nil, nil),
			},
			expResLen: 0,
		}, {
			name: "test multiply destination",
			content: `configVersion: v3.1 
scope: service      
key: org.apache.dubbo.samples.CommentService 
force: false 
runtime: true       
enabled: true     
conditions:
  - from:
      match: env=gray     
    to:
      - match: env!=gray     
        weight: 100
      - match: env=gray     
        weight: 900
  - from:
      match: region=beijing
    to:
      - match: region!=beijing
        weight: 100
      - match: region=beijing
        weight: 200
`,
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("echo", nil, nil),
			},
			multiDestination: []struct {
				invokers_filters InvokersFilters
				weight           float32
			}{{
				invokers_filters: NewINVOKERS_FILTERS().add(`env=gray`).add(`region=beijing`),
				weight:           float32(900) / float32(1000) * float32(200) / float32(300),
			}, {
				invokers_filters: NewINVOKERS_FILTERS().add(`env!=gray`).add(`region=beijing`),
				weight:           float32(100) / float32(1000) * float32(200) / float32(300),
			}, {
				invokers_filters: NewINVOKERS_FILTERS().add(`env=gray`).add(`region!=beijing`),
				weight:           float32(900) / float32(1000) * float32(100) / float32(300),
			}, {
				invokers_filters: NewINVOKERS_FILTERS().add(`env!=gray`).add(`region!=beijing`),
				weight:           float32(100) / float32(1000) * float32(100) / float32(300),
			}},
		}, {
			name: "test multiply destination with ignore some condition",
			content: `configVersion: v3.1 
scope: service      
key: org.apache.dubbo.samples.CommentService 
force: false 
runtime: true       
enabled: true     
conditions:
  - from:
      match: env=gray     
    to:
      - match: env!=gray     
        weight: 100
      - match: env=gray----error # will ignore this subset    
        weight: 900
  - from:
      match: region=beijing
    to:
      - match: region!=beijing
        weight: 100
      - match: region=beijing
        weight: 200
`,
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("echo", nil, nil),
			},
			multiDestination: []struct {
				invokers_filters InvokersFilters
				weight           float32
			}{{
				invokers_filters: NewINVOKERS_FILTERS().add(`env!=gray`).add(`region=beijing`),
				weight:           float32(200) / float32(300),
			}, {
				invokers_filters: NewINVOKERS_FILTERS().add(`env!=gray`).add(`region!=beijing`),
				weight:           float32(100) / float32(300),
			}},
		}, {
			name: "test multiply destination with ignore some condition node",
			content: `configVersion: v3.1 
scope: service      
key: org.apache.dubbo.samples.CommentService 
force: false 
runtime: true       
enabled: true     
conditions:
  - from:
      match: env=gray     
    to:
      - match: env!=gray     
        weight: 100
      - match: env=gray     
        weight: 900
  - from:  # <-- will ignore this condition 
      match: region!=beijing
    to:
      - match: env=normal
        weight: 100
      - match: env=gray
        weight: 200
  - from:
      match: region=beijing
    to:
      - match: region!=beijing
        weight: 100
      - match: region=beijing
        weight: 200
`,
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("echo", nil, nil),
			},
			multiDestination: []struct {
				invokers_filters InvokersFilters
				weight           float32
			}{{
				invokers_filters: NewINVOKERS_FILTERS().add(`env=gray`).add(`region=beijing`),
				weight:           float32(900) / float32(1000) * float32(200) / float32(300),
			}, {
				invokers_filters: NewINVOKERS_FILTERS().add(`env!=gray`).add(`region=beijing`),
				weight:           float32(100) / float32(1000) * float32(200) / float32(300),
			}, {
				invokers_filters: NewINVOKERS_FILTERS().add(`env=gray`).add(`region!=beijing`),
				weight:           float32(900) / float32(1000) * float32(100) / float32(300),
			}, {
				invokers_filters: NewINVOKERS_FILTERS().add(`env!=gray`).add(`region!=beijing`),
				weight:           float32(100) / float32(1000) * float32(100) / float32(300),
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d.Process(&config_center.ConfigChangeEvent{
				Value:      tt.content,
				ConfigType: remoting.EventTypeUpdate,
			})
			if tt.multiDestination == nil {
				res := d.Route(tt.args.invokers, tt.args.url, tt.args.invocation)
				if tt.invokers_filters != nil {
					// check expect filtrate path
					ans := tt.invokers_filters.filtrate(tt.args.invokers, tt.args.url, tt.args.invocation)
					assert.Equalf(t, ans, res, "route(%v, %v, %v)", tt.args.invokers, tt.args.url, tt.args.invocation)
				} else {
					// check expect result.length
					assert.Equalf(t, tt.expResLen, len(res), "route(%v, %v, %v)", tt.args.invokers, tt.args.url, tt.args.invocation)
				}
			} else {
				// check multiply destination route successfully or not
				ans := map[interface{}]float32{}
				for _, s := range tt.multiDestination {
					args := struct {
						invokers   []protocol.Invoker
						url        *common.URL
						invocation protocol.Invocation
					}{tt.args.invokers[:], tt.args.url.Clone(), tt.args.invocation}
					ans[len(s.invokers_filters.filtrate(args.invokers, tt.args.url, tt.args.invocation))] = s.weight * 1000
				}
				res := map[interface{}]int{}
				for i := 0; i < 1000; i++ {
					args := struct {
						invokers   []protocol.Invoker
						url        *common.URL
						invocation protocol.Invocation
					}{tt.args.invokers[:], tt.args.url.Clone(), tt.args.invocation}
					res[len(d.Route(args.invokers, args.url, args.invocation))]++
				}
				for k, v := range ans {
					if float32(res[k]+50) > v && float32(res[k]-50) < v {
					} else {
						assert.Fail(t, "out of range")
					}
				}
			}
		})
	}
}
