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

package router

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"testing"
)

import (
	perrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/common/utils"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

type MockInvoker struct {
	url          common.URL
	available    bool
	destroyed    bool
	successCount int
}

func NewMockInvoker(url common.URL, successCount int) *MockInvoker {
	return &MockInvoker{
		url:          url,
		available:    true,
		destroyed:    false,
		successCount: successCount,
	}
}

func (bi *MockInvoker) GetUrl() common.URL {
	return bi.url
}

func getRouteUrl(rule string) *common.URL {
	url, _ := common.NewURL(context.TODO(), "condition://0.0.0.0/com.foo.BarService")
	url.AddParam("rule", rule)
	url.AddParam("force", "true")
	return &url
}

func getRouteUrlWithForce(rule, force string) *common.URL {
	url, _ := common.NewURL(context.TODO(), "condition://0.0.0.0/com.foo.BarService")
	url.AddParam("rule", rule)
	url.AddParam("force", force)
	return &url
}

func getRouteUrlWithNoForce(rule string) *common.URL {
	url, _ := common.NewURL(context.TODO(), "condition://0.0.0.0/com.foo.BarService")
	url.AddParam("rule", rule)
	return &url
}

func (bi *MockInvoker) IsAvailable() bool {
	return bi.available
}

func (bi *MockInvoker) IsDestroyed() bool {
	return bi.destroyed
}

type rest struct {
	tried   int
	success bool
}

var count int

func (bi *MockInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	count++
	var success bool
	var err error = nil
	if count >= bi.successCount {
		success = true
	} else {
		err = perrors.New("error")
	}
	result := &protocol.RPCResult{Err: err, Rest: rest{tried: count, success: success}}
	return result
}

func (bi *MockInvoker) Destroy() {
	logger.Infof("Destroy invoker: %v", bi.GetUrl().String())
	bi.destroyed = true
	bi.available = false
}

func TestRoute_matchWhen(t *testing.T) {
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("=> host = 1.2.3.4"))
	router, _ := NewConditionRouterFactory().Router(getRouteUrl(rule))
	cUrl, _ := common.NewURL(context.TODO(), "consumer://1.1.1.1/com.foo.BarService")
	matchWhen, _ := router.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, true, matchWhen)
	rule1 := base64.URLEncoding.EncodeToString([]byte("host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4"))
	router1, _ := NewConditionRouterFactory().Router(getRouteUrl(rule1))
	matchWhen1, _ := router1.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, true, matchWhen1)
	rule2 := base64.URLEncoding.EncodeToString([]byte("host = 2.2.2.2,1.1.1.1,3.3.3.3 & host !=1.1.1.1 => host = 1.2.3.4"))
	router2, _ := NewConditionRouterFactory().Router(getRouteUrl(rule2))
	matchWhen2, _ := router2.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, false, matchWhen2)
	rule3 := base64.URLEncoding.EncodeToString([]byte("host !=4.4.4.4 & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4"))
	router3, _ := NewConditionRouterFactory().Router(getRouteUrl(rule3))
	matchWhen3, _ := router3.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, true, matchWhen3)
	rule4 := base64.URLEncoding.EncodeToString([]byte("host !=4.4.4.* & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4"))
	router4, _ := NewConditionRouterFactory().Router(getRouteUrl(rule4))
	matchWhen4, _ := router4.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, true, matchWhen4)
	rule5 := base64.URLEncoding.EncodeToString([]byte("host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.1 => host = 1.2.3.4"))
	router5, _ := NewConditionRouterFactory().Router(getRouteUrl(rule5))
	matchWhen5, _ := router5.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, false, matchWhen5)
	rule6 := base64.URLEncoding.EncodeToString([]byte("host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.2 => host = 1.2.3.4"))
	router6, _ := NewConditionRouterFactory().Router(getRouteUrl(rule6))
	matchWhen6, _ := router6.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, true, matchWhen6)
}

func TestRoute_matchFilter(t *testing.T) {
	localIP, _ := utils.GetLocalIP()
	url1, _ := common.NewURL(context.TODO(), "dubbo://10.20.3.3:20880/com.foo.BarService?default.serialization=fastjson")
	url2, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	url3, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	invokers := []protocol.Invoker{NewMockInvoker(url1, 1), NewMockInvoker(url2, 2), NewMockInvoker(url3, 3)}
	rule1 := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = 10.20.3.3"))
	rule2 := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = 10.20.3.* & host != 10.20.3.3"))
	rule3 := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = 10.20.3.3  & host != 10.20.3.3"))
	rule4 := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = 10.20.3.2,10.20.3.3,10.20.3.4"))
	rule5 := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host != 10.20.3.3"))
	rule6 := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " serialization = fastjson"))
	router1, _ := NewConditionRouterFactory().Router(getRouteUrl(rule1))
	router2, _ := NewConditionRouterFactory().Router(getRouteUrl(rule2))
	router3, _ := NewConditionRouterFactory().Router(getRouteUrl(rule3))
	router4, _ := NewConditionRouterFactory().Router(getRouteUrl(rule4))
	router5, _ := NewConditionRouterFactory().Router(getRouteUrl(rule5))
	router6, _ := NewConditionRouterFactory().Router(getRouteUrl(rule6))
	cUrl, _ := common.NewURL(context.TODO(), "consumer://"+localIP+"/com.foo.BarService")
	fileredInvokers1 := router1.Route(invokers, cUrl, &invocation.RPCInvocation{})
	fileredInvokers2 := router2.Route(invokers, cUrl, &invocation.RPCInvocation{})
	fileredInvokers3 := router3.Route(invokers, cUrl, &invocation.RPCInvocation{})
	fileredInvokers4 := router4.Route(invokers, cUrl, &invocation.RPCInvocation{})
	fileredInvokers5 := router5.Route(invokers, cUrl, &invocation.RPCInvocation{})
	fileredInvokers6 := router6.Route(invokers, cUrl, &invocation.RPCInvocation{})
	assert.Equal(t, 1, len(fileredInvokers1))
	assert.Equal(t, 0, len(fileredInvokers2))
	assert.Equal(t, 0, len(fileredInvokers3))
	assert.Equal(t, 1, len(fileredInvokers4))
	assert.Equal(t, 2, len(fileredInvokers5))
	assert.Equal(t, 1, len(fileredInvokers6))

}

func TestRoute_methodRoute(t *testing.T) {
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("getFoo"), invocation.WithParameterTypes([]reflect.Type{}), invocation.WithArguments([]interface{}{}))
	rule := base64.URLEncoding.EncodeToString([]byte("host !=4.4.4.* & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4"))
	router, _ := NewConditionRouterFactory().Router(getRouteUrl(rule))
	url, _ := common.NewURL(context.TODO(), "consumer://1.1.1.1/com.foo.BarService?methods=setFoo,getFoo,findFoo")
	matchWhen, _ := router.(*ConditionRouter).MatchWhen(url, inv)
	assert.Equal(t, true, matchWhen)
	url1, _ := common.NewURL(context.TODO(), "consumer://1.1.1.1/com.foo.BarService?methods=getFoo")
	matchWhen, _ = router.(*ConditionRouter).MatchWhen(url1, inv)
	assert.Equal(t, true, matchWhen)
	url2, _ := common.NewURL(context.TODO(), "consumer://1.1.1.1/com.foo.BarService?methods=getFoo")
	rule2 := base64.URLEncoding.EncodeToString([]byte("methods=getFoo & host!=1.1.1.1 => host = 1.2.3.4"))
	router2, _ := NewConditionRouterFactory().Router(getRouteUrl(rule2))
	matchWhen, _ = router2.(*ConditionRouter).MatchWhen(url2, inv)
	assert.Equal(t, false, matchWhen)
	url3, _ := common.NewURL(context.TODO(), "consumer://1.1.1.1/com.foo.BarService?methods=getFoo")
	rule3 := base64.URLEncoding.EncodeToString([]byte("methods=getFoo & host=1.1.1.1 => host = 1.2.3.4"))
	router3, _ := NewConditionRouterFactory().Router(getRouteUrl(rule3))
	matchWhen, _ = router3.(*ConditionRouter).MatchWhen(url3, inv)
	assert.Equal(t, true, matchWhen)

}

func TestRoute_ReturnFalse(t *testing.T) {
	url, _ := common.NewURL(context.TODO(), "")
	localIP, _ := utils.GetLocalIP()
	invokers := []protocol.Invoker{NewMockInvoker(url, 1), NewMockInvoker(url, 2), NewMockInvoker(url, 3)}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => false"))
	curl, _ := common.NewURL(context.TODO(), "consumer://"+localIP+"/com.foo.BarService")
	router, _ := NewConditionRouterFactory().Router(getRouteUrl(rule))
	fileredInvokers := router.(*ConditionRouter).Route(invokers, curl, inv)
	assert.Equal(t, 0, len(fileredInvokers))
}

func TestRoute_ReturnEmpty(t *testing.T) {
	localIP, _ := utils.GetLocalIP()
	url, _ := common.NewURL(context.TODO(), "")
	invokers := []protocol.Invoker{NewMockInvoker(url, 1), NewMockInvoker(url, 2), NewMockInvoker(url, 3)}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => "))
	curl, _ := common.NewURL(context.TODO(), "consumer://"+localIP+"/com.foo.BarService")
	router, _ := NewConditionRouterFactory().Router(getRouteUrl(rule))
	fileredInvokers := router.(*ConditionRouter).Route(invokers, curl, inv)
	assert.Equal(t, 0, len(fileredInvokers))
}

func TestRoute_ReturnAll(t *testing.T) {
	localIP, _ := utils.GetLocalIP()
	invokers := []protocol.Invoker{&MockInvoker{}, &MockInvoker{}, &MockInvoker{}}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = " + localIP))
	curl, _ := common.NewURL(context.TODO(), "consumer://"+localIP+"/com.foo.BarService")
	router, _ := NewConditionRouterFactory().Router(getRouteUrl(rule))
	fileredInvokers := router.(*ConditionRouter).Route(invokers, curl, inv)
	assert.Equal(t, invokers, fileredInvokers)
}

func TestRoute_HostFilter(t *testing.T) {
	localIP, _ := utils.GetLocalIP()
	url1, _ := common.NewURL(context.TODO(), "dubbo://10.20.3.3:20880/com.foo.BarService")
	url2, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	url3, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	invoker1 := NewMockInvoker(url1, 1)
	invoker2 := NewMockInvoker(url2, 2)
	invoker3 := NewMockInvoker(url3, 3)
	invokers := []protocol.Invoker{invoker1, invoker2, invoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = " + localIP))
	curl, _ := common.NewURL(context.TODO(), "consumer://"+localIP+"/com.foo.BarService")
	router, _ := NewConditionRouterFactory().Router(getRouteUrl(rule))
	fileredInvokers := router.(*ConditionRouter).Route(invokers, curl, inv)
	assert.Equal(t, 2, len(fileredInvokers))
	assert.Equal(t, invoker2, fileredInvokers[0])
	assert.Equal(t, invoker3, fileredInvokers[1])
}

func TestRoute_Empty_HostFilter(t *testing.T) {
	localIP, _ := utils.GetLocalIP()
	url1, _ := common.NewURL(context.TODO(), "dubbo://10.20.3.3:20880/com.foo.BarService")
	url2, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	url3, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	invoker1 := NewMockInvoker(url1, 1)
	invoker2 := NewMockInvoker(url2, 2)
	invoker3 := NewMockInvoker(url3, 3)
	invokers := []protocol.Invoker{invoker1, invoker2, invoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte(" => " + " host = " + localIP))
	curl, _ := common.NewURL(context.TODO(), "consumer://"+localIP+"/com.foo.BarService")
	router, _ := NewConditionRouterFactory().Router(getRouteUrl(rule))
	fileredInvokers := router.(*ConditionRouter).Route(invokers, curl, inv)
	assert.Equal(t, 2, len(fileredInvokers))
	assert.Equal(t, invoker2, fileredInvokers[0])
	assert.Equal(t, invoker3, fileredInvokers[1])
}

func TestRoute_False_HostFilter(t *testing.T) {
	localIP, _ := utils.GetLocalIP()
	url1, _ := common.NewURL(context.TODO(), "dubbo://10.20.3.3:20880/com.foo.BarService")
	url2, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	url3, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	invoker1 := NewMockInvoker(url1, 1)
	invoker2 := NewMockInvoker(url2, 2)
	invoker3 := NewMockInvoker(url3, 3)
	invokers := []protocol.Invoker{invoker1, invoker2, invoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("true => " + " host = " + localIP))
	curl, _ := common.NewURL(context.TODO(), "consumer://"+localIP+"/com.foo.BarService")
	router, _ := NewConditionRouterFactory().Router(getRouteUrl(rule))
	fileredInvokers := router.(*ConditionRouter).Route(invokers, curl, inv)
	assert.Equal(t, 2, len(fileredInvokers))
	assert.Equal(t, invoker2, fileredInvokers[0])
	assert.Equal(t, invoker3, fileredInvokers[1])
}

func TestRoute_Placeholder(t *testing.T) {
	localIP, _ := utils.GetLocalIP()
	url1, _ := common.NewURL(context.TODO(), "dubbo://10.20.3.3:20880/com.foo.BarService")
	url2, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	url3, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	invoker1 := NewMockInvoker(url1, 1)
	invoker2 := NewMockInvoker(url2, 2)
	invoker3 := NewMockInvoker(url3, 3)
	invokers := []protocol.Invoker{invoker1, invoker2, invoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = $host"))
	curl, _ := common.NewURL(context.TODO(), "consumer://"+localIP+"/com.foo.BarService")
	router, _ := NewConditionRouterFactory().Router(getRouteUrl(rule))
	fileredInvokers := router.(*ConditionRouter).Route(invokers, curl, inv)
	assert.Equal(t, 2, len(fileredInvokers))
	assert.Equal(t, invoker2, fileredInvokers[0])
	assert.Equal(t, invoker3, fileredInvokers[1])
}

func TestRoute_NoForce(t *testing.T) {
	localIP, _ := utils.GetLocalIP()
	url1, _ := common.NewURL(context.TODO(), "dubbo://10.20.3.3:20880/com.foo.BarService")
	url2, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	url3, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	invoker1 := NewMockInvoker(url1, 1)
	invoker2 := NewMockInvoker(url2, 2)
	invoker3 := NewMockInvoker(url3, 3)
	invokers := []protocol.Invoker{invoker1, invoker2, invoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = 1.2.3.4"))
	curl, _ := common.NewURL(context.TODO(), "consumer://"+localIP+"/com.foo.BarService")
	router, _ := NewConditionRouterFactory().Router(getRouteUrlWithNoForce(rule))
	fileredInvokers := router.(*ConditionRouter).Route(invokers, curl, inv)
	assert.Equal(t, invokers, fileredInvokers)
}

func TestRoute_Force(t *testing.T) {
	localIP, _ := utils.GetLocalIP()
	url1, _ := common.NewURL(context.TODO(), "dubbo://10.20.3.3:20880/com.foo.BarService")
	url2, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	url3, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://%s:20880/com.foo.BarService", localIP))
	invoker1 := NewMockInvoker(url1, 1)
	invoker2 := NewMockInvoker(url2, 2)
	invoker3 := NewMockInvoker(url3, 3)
	invokers := []protocol.Invoker{invoker1, invoker2, invoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = 1.2.3.4"))
	curl, _ := common.NewURL(context.TODO(), "consumer://"+localIP+"/com.foo.BarService")
	router, _ := NewConditionRouterFactory().Router(getRouteUrlWithForce(rule, "true"))
	fileredInvokers := router.(*ConditionRouter).Route(invokers, curl, inv)
	assert.Equal(t, 0, len(fileredInvokers))
}
