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
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/configurator"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
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
	config.GetEnvInstance().SetDynamicConfiguration(dc)

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
	config.GetEnvInstance().SetDynamicConfiguration(dc)

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

func TestConditionRoutePriority(t *testing.T) {
	ivks := buildInvokers()
	ar := NewApplicationRouter()
	ar.Process(&config_center.ConfigChangeEvent{Key: "", Value: `configVersion: v3.1
scope: service
force: false
runtime: true
enabled: true
key: shop
conditions:
  - from:
      match:
    to:
      - match: region=$region & version=v1
      - match: region=$region & version=v2
        weight: 200
      - match: region=$region & version=v3
        weight: 300
    force: false
    ratio: 20
    priority: 20
  - from: ## match here 
      match:
        region=beijing & version=v1
    to:
      - match: env=$env & region=beijing
    force: false
    ratio: 20 
    priority: 100
`, ConfigType: remoting.EventTypeUpdate})
	consumerUrl, err := common.NewURL("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing&version=v1")
	if err != nil {
		panic(err)
	}
	got := ar.Route(ivks, consumerUrl, invocation.NewRPCInvocation("getComment", nil, nil))
	expLen := 0
	for _, ivk := range ivks {
		if ivk.GetURL().GetParam("region", "") == "beijing" && "gray" == ivk.GetURL().GetParam("env", "") {
			expLen++
		}
	}
	if len(ivks)*100/expLen <= 20 {
		expLen = 0
	}
	assert.Equal(t, expLen, len(got))
}

func TestConditionRouteTrafficDisable(t *testing.T) {
	ivks := buildInvokers()
	ar := NewApplicationRouter()
	ar.Process(&config_center.ConfigChangeEvent{Key: "", Value: `
configVersion: v3.1
scope: service
force: true
runtime: true
enabled: true
key: org.apache.dubbo.samples.CommentService
conditionAction : true 
conditions:
  - rule: method=getComment & env=gray => region=Hangzhou & env=gray
    priority: 3
  - rule: method=getComment & env=gray => region=beijing & env=gray
    priority: 3
  - rule: method=getComment & env=gray => region=$region & env=gray 
    priority: 3
  - rule: method=getComment & env=normal => region=beijing 
    priority: 3
  - rule: method=getComment => region=$region 
    priority: 30
  - rule: method=echo =>
    force: true
  - rule: method=echo => region=$region 
`, ConfigType: remoting.EventTypeUpdate})
	consumerUrl, err := common.NewURL("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing")
	if err != nil {
		panic(err)
	}
	got := ar.Route(ivks, consumerUrl, invocation.NewRPCInvocation("echo", nil, nil))
	assert.Equal(t, 0, len(got))
}

func TestConditionRouteRegionPriority(t *testing.T) {
	ivks := buildInvokers()
	ar := NewApplicationRouter()
	ar.Process(&config_center.ConfigChangeEvent{Key: "", Value: `
configVersion: v3.1
scope: service
force: true
runtime: true
enabled: true
key: org.apache.dubbo.samples.CommentService
conditionAction : true 
conditions:
  - rule: => region=$region & env=$env
  - rule: method=getComment & env=gray => env=$env
  - rule: method=getComment & env=gray & region=beijing => region=beijing & env=gray
  - rule: method=getComment & env=gray => region=$region & env=gray 
  - rule: method=getComment & env=normal => region=beijing 
  - rule: method=echo =>
    force: true
  - rule: method=echo => region=$region 
`, ConfigType: remoting.EventTypeUpdate})
	consumerUrl, err := common.NewURL("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing")
	if err != nil {
		panic(err)
	}
	got := ar.Route(ivks, consumerUrl, invocation.NewRPCInvocation("getComment", nil, nil))
	expLen := 0
	for _, ivk := range ivks {
		if ivk.GetURL().GetRawParam("env") == consumerUrl.GetRawParam("env") &&
			ivk.GetURL().GetRawParam("region") == consumerUrl.GetRawParam("region") {
			expLen++
		}
	}
	assert.Equal(t, expLen, len(got))
	consumerUrl, err = common.NewURL("consumer://127.0.0.1/com.foo.BarService?env=gray&region=hangzhou")
	if err != nil {
		panic(err)
	}
	got = ar.Route(ivks, consumerUrl, invocation.NewRPCInvocation("getComment", nil, nil))
	expLen = 0
	for _, ivk := range ivks {
		if ivk.GetURL().GetRawParam("env") == consumerUrl.GetRawParam("env") &&
			ivk.GetURL().GetRawParam("region") == consumerUrl.GetRawParam("region") {
			expLen++
		}
	}
	assert.Equal(t, expLen, len(got))
	consumerUrl, err = common.NewURL("consumer://127.0.0.1/com.foo.BarService?env=normal&region=shanghai")
	if err != nil {
		panic(err)
	}
	got = ar.Route(ivks, consumerUrl, invocation.NewRPCInvocation("getComment", nil, nil))
	expLen = 0
	for _, ivk := range ivks {
		if ivk.GetURL().GetRawParam("region") == "beijing" {
			expLen++
		}
	}
	assert.Equal(t, expLen, len(got))
}

func TestConditionRouteMatchFail(t *testing.T) {
	ivks := buildInvokers()
	ar := NewApplicationRouter()
	ar.Process(&config_center.ConfigChangeEvent{Key: "", Value: `
configVersion: v3.1
scope: service
force: false
runtime: true
enabled: true
key: org.apache.dubbo.samples.CommentService
conditionAction : true 
conditions:
  - rule: => region=$region & env=$env & errTag=errTag
  - rule: method=getComment & env=gray => env=$env
  - rule: method=getComment & env=gray & region=beijing => region=beijing & env=gray
  - rule: method=getComment & env=gray => region=$region & env=gray 
  - rule: method=getComment & env=normal => region=beijing 
  - rule: method=echo =>
    force: true
  - rule: method=echo => region=$region 
`, ConfigType: remoting.EventTypeUpdate})
	consumerUrl, err := common.NewURL("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing")
	if err != nil {
		panic(err)
	}
	got := ar.Route(ivks, consumerUrl, invocation.NewRPCInvocation("errMethod", nil, nil))
	assert.Equal(t, len(ivks), len(got))
}
