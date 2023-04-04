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
	LocalHost = "127.0.0.1"
)

func TestRoute_matchWhen(t *testing.T) {

	rpcInvocation := invocation.NewRPCInvocation("getFoo", nil, nil)
	whenConsumerUrl, _ := common.NewURL("consumer://1.1.1.1/com.foo.BarService")

	testData := []struct {
		name        string
		consumerUrl *common.URL
		rule        string

		wantVal bool
	}{
		{
			name:        " => host = 1.2.3.4",
			consumerUrl: whenConsumerUrl,
			rule:        " => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "host = 1.2.3.4 => ",
			consumerUrl: whenConsumerUrl,
			rule:        "host = 1.2.3.4 => ",

			wantVal: false,
		},
		{
			name:        "host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4",
			consumerUrl: whenConsumerUrl,
			rule:        "host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "host = 2.2.2.2,1.1.1.1,3.3.3.3 & host !=1.1.1.1 => host = 1.2.3.4",
			consumerUrl: whenConsumerUrl,
			rule:        "host = 2.2.2.2,1.1.1.1,3.3.3.3 & host !=1.1.1.1 => host = 1.2.3.4",

			wantVal: false,
		},
		{
			name:        "host !=4.4.4.4 & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4",
			consumerUrl: whenConsumerUrl,
			rule:        "host !=4.4.4.4 & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "host !=4.4.4.* & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4",
			consumerUrl: whenConsumerUrl,
			rule:        "host !=4.4.4.* & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.1 => host = 1.2.3.4",
			consumerUrl: whenConsumerUrl,
			rule:        "host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.1 => host = 1.2.3.4",

			wantVal: false,
		},
		{
			name:        "host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.2 => host = 1.2.3.4",
			consumerUrl: whenConsumerUrl,
			rule:        "host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.2 => host = 1.2.3.4",

			wantVal: true,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			url, err := common.NewURL("condition://0.0.0.0/com.foo.BarService")
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)
			resVal := router.matchWhen(data.consumerUrl, rpcInvocation)
			assert.Equal(t, data.wantVal, resVal)
		})
	}
}

func TestRoute_matchFilter(t *testing.T) {

	consumerUrl, _ := common.NewURL("consumer://" + LocalHost + "/com.foo.BarService")
	url1, _ := common.NewURL("dubbo://10.20.3.3:20880/com.foo.BarService?serialization=fastjson")
	url2, _ := common.NewURL("dubbo://" + LocalHost + ":20880/com.foo.BarService")
	url3, _ := common.NewURL("dubbo://" + LocalHost + ":20880/com.foo.BarService")

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
		comsumerUrl *common.URL
		rule        string

		wantVal int
	}{
		{
			name:        "host = " + LocalHost + " => " + " host = 10.20.3.3",
			comsumerUrl: consumerUrl,
			rule:        "host = " + LocalHost + " => " + " host = 10.20.3.3",

			wantVal: 1,
		},
		{
			name:        "host = " + LocalHost + " => " + " host = 10.20.3.* & host != 10.20.3.3",
			comsumerUrl: consumerUrl,
			rule:        "host = " + LocalHost + " => " + " host = 10.20.3.* & host != 10.20.3.3",

			wantVal: 0,
		},
		{
			name:        "host = " + LocalHost + " => " + " host = 10.20.3.3  & host != 10.20.3.3",
			comsumerUrl: consumerUrl,
			rule:        "host = " + LocalHost + " => " + " host = 10.20.3.3  & host != 10.20.3.3",

			wantVal: 0,
		},
		{
			name:        "host = " + LocalHost + " => " + " host = 10.20.3.2,10.20.3.3,10.20.3.4",
			comsumerUrl: consumerUrl,
			rule:        "host = " + LocalHost + " => " + " host = 10.20.3.2,10.20.3.3,10.20.3.4",

			wantVal: 1,
		},
		{
			name:        "host = " + LocalHost + " => " + " host != 10.20.3.3",
			comsumerUrl: consumerUrl,
			rule:        "host = " + LocalHost + " => " + " host != 10.20.3.3",

			wantVal: 2,
		},
		{
			name:        "host = " + LocalHost + " => " + " serialization = fastjson",
			comsumerUrl: consumerUrl,
			rule:        "host = " + LocalHost + " => " + " serialization = fastjson",

			wantVal: 1,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			url, err := common.NewURL("condition://" + LocalHost + "/com.foo.BarService")
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			url.AddParam(constant.ForceKey, "true")

			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)

			filteredInvokers := router.Route(invokerList, data.comsumerUrl, rpcInvocation)
			resVal := len(filteredInvokers)
			assert.Equal(t, data.wantVal, resVal)
		})
	}
}

func TestRoute_methodRoute(t *testing.T) {

	rpcInvocation := invocation.NewRPCInvocation("getFoo", nil, nil)

	testData := []struct {
		name        string
		consumerUrl string
		rule        string

		wantVal bool
	}{
		{
			name:        "More than one methods, mismatch",
			consumerUrl: "consumer://1.1.1.1/com.foo.BarService?methods=setFoo,getFoo,findFoo",
			rule:        "methods=getFoo => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "Exactly one method, match",
			consumerUrl: "consumer://1.1.1.1/com.foo.BarService?methods=getFoo",
			rule:        "methods=getFoo => host = 1.2.3.4",

			wantVal: true,
		},
		{
			name:        "Method routing and Other condition routing can work together",
			consumerUrl: "consumer://1.1.1.1/com.foo.BarService?methods=getFoo",
			rule:        "methods=getFoo & host!=1.1.1.1 => host = 1.2.3.4",

			wantVal: false,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			url, err := common.NewURL("condition://0.0.0.0/com.foo.BarService")
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)
			consumer, _ := common.NewURL(data.consumerUrl)
			resVal := router.matchWhen(consumer, rpcInvocation)
			assert.Equal(t, data.wantVal, resVal)
		})
	}
}

func TestRoute_return(t *testing.T) {

	rpcInvocation := invocation.NewRPCInvocation("getFoo", nil, nil)
	consumerUrl, _ := common.NewURL("consumer://" + LocalHost + "/com.foo.BarService")

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
			rule: "host = " + LocalHost + " => false",

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
			rule: "host = " + LocalHost + " => ",

			wantUrls: []string{},
			wantVal:  0,
		},
		{
			name: "ReturnAll",
			urls: []string{
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
			},
			rule: "host = " + LocalHost + " => " + " host = " + LocalHost,

			wantUrls: []string{
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
			},
			wantVal: 3,
		},
		{
			name: "HostFilter",
			urls: []string{
				"dubbo://10.20.3.3:20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
			},
			rule: "host = " + LocalHost + " => " + " host = " + LocalHost,

			wantUrls: []string{
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
			},
			wantVal: 2,
		},
		{
			name: "EmptyHostFilter",
			urls: []string{
				"dubbo://10.20.3.3:20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
			},
			rule: " => " + " host = " + LocalHost,

			wantUrls: []string{
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
			},
			wantVal: 2,
		},
		{
			name: "FalseHostFilter",
			urls: []string{
				"dubbo://10.20.3.3:20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
			},
			rule: "true => " + " host = " + LocalHost,

			wantUrls: []string{
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
			},
			wantVal: 2,
		},
		{
			name: "PlaceHolder",
			urls: []string{
				"dubbo://10.20.3.3:20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
			},
			rule: "host = " + LocalHost + " => " + " host = $host",

			wantUrls: []string{
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
				"dubbo://" + LocalHost + ":20880/com.foo.BarService",
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

			url, err := common.NewURL("condition://" + LocalHost + "/com.foo.BarService")
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)

			filterInvokers := router.Route(invokers, consumerUrl, rpcInvocation)
			resVal := len(filterInvokers)

			assert.Equal(t, data.wantVal, resVal)
			assert.Equal(t, wantInvokers, filterInvokers)
		})
	}
}

func TestRoute_arguments(t *testing.T) {

	url1, _ := common.NewURL("dubbo://10.20.3.3:20880/com.foo.BarService")
	url2, _ := common.NewURL("dubbo://" + LocalHost + ":20880/com.foo.BarService")
	url3, _ := common.NewURL("dubbo://" + LocalHost + ":20880/com.foo.BarService")

	ink1 := protocol.NewBaseInvoker(url1)
	ink2 := protocol.NewBaseInvoker(url2)
	ink3 := protocol.NewBaseInvoker(url3)

	invokerList := make([]protocol.Invoker, 0, 3)
	invokerList = append(invokerList, ink1)
	invokerList = append(invokerList, ink2)
	invokerList = append(invokerList, ink3)

	consumerUrl, _ := common.NewURL("consumer://" + LocalHost + "/com.foo.BarService")

	testData := []struct {
		name     string
		argument interface{}
		rule     string

		wantVal int
	}{
		{
			name:     "Empty arguments",
			argument: nil,
			rule:     "arguments[0] = a " + " => " + " host = 1.2.3.4",

			wantVal: 3,
		},
		{
			name:     "String arguments",
			argument: "a",
			rule:     "arguments[0] = a " + " => " + " host = 1.2.3.4",

			wantVal: 0,
		},
		{
			name:     "Int arguments",
			argument: 1,
			rule:     "arguments[0] = 1 " + " => " + " host = 127.0.0.1",

			wantVal: 2,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {

			url, err := common.NewURL("condition://" + LocalHost + "/com.foo.BarService")
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			url.AddParam(constant.ForceKey, "true")
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)

			arguments := make([]interface{}, 0, 1)
			arguments = append(arguments, data.argument)

			rpcInvocation := invocation.NewRPCInvocation("getBar", arguments, nil)

			filterInvokers := router.Route(invokerList, consumerUrl, rpcInvocation)
			resVal := len(filterInvokers)
			assert.Equal(t, data.wantVal, resVal)

		})
	}
}

func TestRoute_attachments(t *testing.T) {
	consumerUrl, _ := common.NewURL("consumer://" + LocalHost + "/com.foo.BarService")

	url1, _ := common.NewURL("dubbo://10.20.3.3:20880/com.foo.BarService?region=hangzhou")
	url2, _ := common.NewURL("dubbo://" + LocalHost + ":20880/com.foo.BarService")
	url3, _ := common.NewURL("dubbo://" + LocalHost + ":20880/com.foo.BarService")

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
			rule:            "attachments[foo] = a " + " => " + " host = 1.2.3.4",

			wantVal: 3,
		},
		{
			name:            "Yes attachments and no host",
			attachmentKey:   "foo",
			attachmentValue: "a",
			rule:            "attachments[foo] = a " + " => " + " host = 1.2.3.4",

			wantVal: 0,
		},
		{
			name:            "No attachments and no host",
			attachmentKey:   "foo",
			attachmentValue: "a",
			rule:            "attachments = a " + " => " + " host = 1.2.3.4",

			wantVal: 3,
		},
		{
			name:            "Yes attachments and region",
			attachmentKey:   "foo",
			attachmentValue: "a",
			rule:            "attachments[foo] = a " + " => " + " region = hangzhou",

			wantVal: 1,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {

			rpcInvocation := invocation.NewRPCInvocation("getBar", nil, nil)
			rpcInvocation.SetAttachment(data.attachmentKey, data.attachmentValue)

			url, err := common.NewURL("condition://" + LocalHost + "/com.foo.BarService")
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			url.AddParam(constant.ForceKey, "true")
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)

			filterInvokers := router.Route(invokerList, consumerUrl, rpcInvocation)

			resVal := len(filterInvokers)
			assert.Equal(t, data.wantVal, resVal)
		})
	}
}

func TestRoute_range_pattern(t *testing.T) {

	consumerUrl, _ := common.NewURL("consumer://" + LocalHost + "/com.foo.BarService")

	url1, _ := common.NewURL("dubbo://10.20.3.3:20880/com.foo.BarService?region=hangzhou")
	url2, _ := common.NewURL("dubbo://" + LocalHost + ":20880/com.foo.BarService")
	url3, _ := common.NewURL("dubbo://" + LocalHost + ":20880/com.foo.BarService")

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
			rule:            "attachments[user_id] = 1~100 " + " => " + " region=hangzhou",

			wantVal: 3,
		},
		{
			name:            "In the range",
			attachmentKey:   "user_id",
			attachmentValue: "80",
			rule:            "attachments[user_id] = 1~100 " + " => " + " region=hangzhou",

			wantVal: 1,
		},
		{
			name:            "Out of range",
			attachmentKey:   "user_id",
			attachmentValue: "101",
			rule:            "attachments[user_id] = 1~100 " + " => " + " region=hangzhou",

			wantVal: 3,
		},
		{
			name:            "In the single interval range",
			attachmentKey:   "user_id",
			attachmentValue: "1",
			rule:            "attachments[user_id] = ~100 " + " => " + " region=hangzhou",

			wantVal: 1,
		},
		{
			name:            "Not in the single interval range",
			attachmentKey:   "user_id",
			attachmentValue: "101",
			rule:            "attachments[user_id] = ~100 " + " => " + " region=hangzhou",

			wantVal: 3,
		},
	}

	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {

			rpcInvocation := invocation.NewRPCInvocation("getBar", nil, nil)
			rpcInvocation.SetAttachment(data.attachmentKey, data.attachmentValue)

			url, err := common.NewURL("condition://" + LocalHost + "/com.foo.BarService")
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			url.AddParam(constant.ForceKey, "true")
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)

			filterInvokers := router.Route(invokerList, consumerUrl, rpcInvocation)

			resVal := len(filterInvokers)
			assert.Equal(t, data.wantVal, resVal)
		})
	}
}

func TestRoute_multiple_conditions(t *testing.T) {
	url1, _ := common.NewURL("dubbo://10.20.3.3:20880/com.foo.BarService?region=hangzhou")
	url2, _ := common.NewURL("dubbo://" + LocalHost + ":20880/com.foo.BarService")
	url3, _ := common.NewURL("dubbo://" + LocalHost + ":20880/com.foo.BarService")

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
		consumerUrl string
		rule        string

		wantVal int
	}{
		{
			name:        "All conditions match",
			argument:    "a",
			consumerUrl: "consumer://" + LocalHost + "/com.foo.BarService?application=consumer_app",
			rule:        "application=consumer_app&arguments[0]=a" + " => " + " host = " + LocalHost,

			wantVal: 2,
		},
		{
			name:        "One of the conditions does not match",
			argument:    "a",
			consumerUrl: "consumer://" + LocalHost + "/com.foo.BarService?application=another_consumer_app",
			rule:        "application=consumer_app&arguments[0]=a" + " => " + " host = " + LocalHost,

			wantVal: 3,
		},
	}
	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			consumerUrl, err := common.NewURL(data.consumerUrl)
			assert.Nil(t, err)

			url, err := common.NewURL("condition://" + LocalHost + "/com.foo.BarService")
			assert.Nil(t, err)
			url.AddParam(constant.RuleKey, data.rule)
			url.AddParam(constant.ForceKey, "true")
			router, err := NewConditionStateRouter(url)
			assert.Nil(t, err)

			arguments := make([]interface{}, 0, 1)
			arguments = append(arguments, data.argument)

			rpcInvocation := invocation.NewRPCInvocation("getBar", arguments, nil)

			filterInvokers := router.Route(invokerList, consumerUrl, rpcInvocation)
			resVal := len(filterInvokers)
			assert.Equal(t, data.wantVal, resVal)
		})
	}
}
