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
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/chain"
	"github.com/apache/dubbo-go/cluster/router/utils"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

const (
	factory1111Ip               = "1.1.1.1"
	factoryUrlFormat            = "condition://%s/com.foo.BarService"
	factoryDubboFormat          = "dubbo://%s:20880/com.foo.BarService"
	factoryConsumerMethodFormat = "consumer://%s/com.foo.BarService?methods=getFoo"
	factory333URL               = "dubbo://10.20.3.3:20880/com.foo.BarService"
	factoryConsumerFormat       = "consumer://%s/com.foo.BarService"
	factoryHostIp1234Format     = "host = %s =>  host = 1.2.3.4"
)

type MockInvoker struct {
	url          *common.URL
	available    bool
	destroyed    bool
	successCount int
}

func NewMockInvoker(url *common.URL, successCount int) *MockInvoker {
	return &MockInvoker{
		url:          url,
		available:    true,
		destroyed:    false,
		successCount: successCount,
	}
}

func (bi *MockInvoker) GetUrl() *common.URL {
	return bi.url
}

func getRouteUrl(rule string) *common.URL {
	url, _ := common.NewURL(fmt.Sprintf(factoryUrlFormat, constant.ANYHOST_VALUE))
	url.AddParam("rule", rule)
	url.AddParam("force", "true")
	return url
}

func getRouteUrlWithForce(rule, force string) *common.URL {
	url, _ := common.NewURL(fmt.Sprintf(factoryUrlFormat, constant.ANYHOST_VALUE))
	url.AddParam("rule", rule)
	url.AddParam("force", force)
	return url
}

func getRouteUrlWithNoForce(rule string) *common.URL {
	url, _ := common.NewURL(fmt.Sprintf(factoryUrlFormat, constant.ANYHOST_VALUE))
	url.AddParam("rule", rule)
	return url
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

func (bi *MockInvoker) Invoke(_ context.Context, _ protocol.Invocation) protocol.Result {
	count++

	var (
		success bool
		err     error
	)
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
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	router, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule), notify)
	cUrl, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, factory1111Ip))
	matchWhen := router.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, true, matchWhen)
	rule1 := base64.URLEncoding.EncodeToString([]byte("host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4"))
	router1, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule1), notify)
	matchWhen1 := router1.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, true, matchWhen1)
	rule2 := base64.URLEncoding.EncodeToString([]byte("host = 2.2.2.2,1.1.1.1,3.3.3.3 & host !=1.1.1.1 => host = 1.2.3.4"))
	router2, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule2), notify)
	matchWhen2 := router2.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, false, matchWhen2)
	rule3 := base64.URLEncoding.EncodeToString([]byte("host !=4.4.4.4 & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4"))
	router3, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule3), notify)
	matchWhen3 := router3.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, true, matchWhen3)
	rule4 := base64.URLEncoding.EncodeToString([]byte("host !=4.4.4.* & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4"))
	router4, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule4), notify)
	matchWhen4 := router4.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, true, matchWhen4)
	rule5 := base64.URLEncoding.EncodeToString([]byte("host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.1 => host = 1.2.3.4"))
	router5, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule5), notify)
	matchWhen5 := router5.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, false, matchWhen5)
	rule6 := base64.URLEncoding.EncodeToString([]byte("host = 2.2.2.2,1.1.1.*,3.3.3.3 & host != 1.1.1.2 => host = 1.2.3.4"))
	router6, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule6), notify)
	matchWhen6 := router6.(*ConditionRouter).MatchWhen(cUrl, inv)
	assert.Equal(t, true, matchWhen6)
}

func TestRoute_matchFilter(t *testing.T) {
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	localIP := common.GetLocalIp()
	t.Logf("The local ip is %s", localIP)
	url1, _ := common.NewURL("dubbo://10.20.3.3:20880/com.foo.BarService?default.serialization=fastjson")
	url2, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	url3, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	invokers := []protocol.Invoker{NewMockInvoker(url1, 1), NewMockInvoker(url2, 2), NewMockInvoker(url3, 3)}
	rule1 := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = 10.20.3.3"))
	rule2 := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = 10.20.3.* & host != 10.20.3.3"))
	rule3 := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = 10.20.3.3  & host != 10.20.3.3"))
	rule4 := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = 10.20.3.2,10.20.3.3,10.20.3.4"))
	rule5 := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host != 10.20.3.3"))
	rule6 := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " serialization = fastjson"))
	router1, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule1), notify)
	router2, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule2), notify)
	router3, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule3), notify)
	router4, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule4), notify)
	router5, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule5), notify)
	router6, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule6), notify)
	cUrl, _ := common.NewURL(fmt.Sprintf(factoryConsumerFormat, localIP))
	ret1 := router1.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), cUrl, &invocation.RPCInvocation{})
	ret2 := router2.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), cUrl, &invocation.RPCInvocation{})
	ret3 := router3.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), cUrl, &invocation.RPCInvocation{})
	ret4 := router4.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), cUrl, &invocation.RPCInvocation{})
	ret5 := router5.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), cUrl, &invocation.RPCInvocation{})
	ret6 := router6.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), cUrl, &invocation.RPCInvocation{})
	assert.Equal(t, 1, len(ret1.ToArray()))
	assert.Equal(t, 0, len(ret2.ToArray()))
	assert.Equal(t, 0, len(ret3.ToArray()))
	assert.Equal(t, 1, len(ret4.ToArray()))
	assert.Equal(t, 2, len(ret5.ToArray()))
	assert.Equal(t, 1, len(ret6.ToArray()))

}

func TestRoute_methodRoute(t *testing.T) {
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("getFoo"), invocation.WithParameterTypes([]reflect.Type{}), invocation.WithArguments([]interface{}{}))
	rule := base64.URLEncoding.EncodeToString([]byte("host !=4.4.4.* & host = 2.2.2.2,1.1.1.1,3.3.3.3 => host = 1.2.3.4"))
	r, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule), notify)
	url, _ := common.NewURL("consumer://1.1.1.1/com.foo.BarService?methods=setFoo,getFoo,findFoo")
	matchWhen := r.(*ConditionRouter).MatchWhen(url, inv)
	assert.Equal(t, true, matchWhen)
	url1, _ := common.NewURL(fmt.Sprintf(factoryConsumerMethodFormat, factory1111Ip))
	matchWhen = r.(*ConditionRouter).MatchWhen(url1, inv)
	assert.Equal(t, true, matchWhen)
	url2, _ := common.NewURL(fmt.Sprintf(factoryConsumerMethodFormat, factory1111Ip))
	rule2 := base64.URLEncoding.EncodeToString([]byte("methods=getFoo & host!=1.1.1.1 => host = 1.2.3.4"))
	router2, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule2), notify)
	matchWhen = router2.(*ConditionRouter).MatchWhen(url2, inv)
	assert.Equal(t, false, matchWhen)
	url3, _ := common.NewURL(fmt.Sprintf(factoryConsumerMethodFormat, factory1111Ip))
	rule3 := base64.URLEncoding.EncodeToString([]byte("methods=getFoo & host=1.1.1.1 => host = 1.2.3.4"))
	router3, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule3), notify)
	matchWhen = router3.(*ConditionRouter).MatchWhen(url3, inv)
	assert.Equal(t, true, matchWhen)

}

func TestRoute_ReturnFalse(t *testing.T) {
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	url, _ := common.NewURL("")
	localIP := common.GetLocalIp()
	invokers := []protocol.Invoker{NewMockInvoker(url, 1), NewMockInvoker(url, 2), NewMockInvoker(url, 3)}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => false"))
	curl, _ := common.NewURL(fmt.Sprintf(factoryConsumerFormat, localIP))
	r, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule), notify)
	ret := r.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), curl, inv)
	assert.Equal(t, 0, len(ret.ToArray()))
}

func TestRoute_ReturnEmpty(t *testing.T) {
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	localIP := common.GetLocalIp()
	url, _ := common.NewURL("")
	invokers := []protocol.Invoker{NewMockInvoker(url, 1), NewMockInvoker(url, 2), NewMockInvoker(url, 3)}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => "))
	curl, _ := common.NewURL(fmt.Sprintf(factoryConsumerFormat, localIP))
	r, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule), notify)
	ret := r.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), curl, inv)
	assert.Equal(t, 0, len(ret.ToArray()))
}

func TestRoute_ReturnAll(t *testing.T) {
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	localIP := common.GetLocalIp()
	urlString := "dubbo://" + localIP + "/com.foo.BarService"
	dubboURL, _ := common.NewURL(urlString)
	mockInvoker1 := NewMockInvoker(dubboURL, 1)
	mockInvoker2 := NewMockInvoker(dubboURL, 1)
	mockInvoker3 := NewMockInvoker(dubboURL, 1)
	invokers := []protocol.Invoker{mockInvoker1, mockInvoker2, mockInvoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = " + localIP))
	curl, _ := common.NewURL(fmt.Sprintf(factoryConsumerFormat, localIP))
	r, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule), notify)
	ret := r.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), curl, inv)
	assert.Equal(t, len(invokers), len(ret.ToArray()))
}

func TestRoute_HostFilter(t *testing.T) {
	localIP := common.GetLocalIp()
	url1, _ := common.NewURL(factory333URL)
	url2, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	url3, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	invoker1 := NewMockInvoker(url1, 1)
	invoker2 := NewMockInvoker(url2, 2)
	invoker3 := NewMockInvoker(url3, 3)
	invokers := []protocol.Invoker{invoker1, invoker2, invoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = " + localIP))
	curl, _ := common.NewURL(fmt.Sprintf(factoryConsumerFormat, localIP))
	r, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule), notify)
	ret := r.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), curl, inv)
	assert.Equal(t, 2, len(ret.ToArray()))
	assert.Equal(t, invoker2, invokers[ret.ToArray()[0]])
	assert.Equal(t, invoker3, invokers[ret.ToArray()[1]])
}

func TestRoute_Empty_HostFilter(t *testing.T) {
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	localIP := common.GetLocalIp()
	url1, _ := common.NewURL(factory333URL)
	url2, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	url3, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	invoker1 := NewMockInvoker(url1, 1)
	invoker2 := NewMockInvoker(url2, 2)
	invoker3 := NewMockInvoker(url3, 3)
	invokers := []protocol.Invoker{invoker1, invoker2, invoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte(" => " + " host = " + localIP))
	curl, _ := common.NewURL(fmt.Sprintf(factoryConsumerFormat, localIP))
	r, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule), notify)
	ret := r.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), curl, inv)
	assert.Equal(t, 2, len(ret.ToArray()))
	assert.Equal(t, invoker2, invokers[ret.ToArray()[0]])
	assert.Equal(t, invoker3, invokers[ret.ToArray()[1]])
}

func TestRoute_False_HostFilter(t *testing.T) {
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	localIP := common.GetLocalIp()
	url1, _ := common.NewURL(factory333URL)
	url2, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	url3, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	invoker1 := NewMockInvoker(url1, 1)
	invoker2 := NewMockInvoker(url2, 2)
	invoker3 := NewMockInvoker(url3, 3)
	invokers := []protocol.Invoker{invoker1, invoker2, invoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("true => " + " host = " + localIP))
	curl, _ := common.NewURL(fmt.Sprintf(factoryConsumerFormat, localIP))
	r, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule), notify)
	ret := r.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), curl, inv)
	assert.Equal(t, 2, len(ret.ToArray()))
	assert.Equal(t, invoker2, invokers[ret.ToArray()[0]])
	assert.Equal(t, invoker3, invokers[ret.ToArray()[1]])
}

func TestRoute_Placeholder(t *testing.T) {
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	localIP := common.GetLocalIp()
	url1, _ := common.NewURL(factory333URL)
	url2, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	url3, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	invoker1 := NewMockInvoker(url1, 1)
	invoker2 := NewMockInvoker(url2, 2)
	invoker3 := NewMockInvoker(url3, 3)
	invokers := []protocol.Invoker{invoker1, invoker2, invoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte("host = " + localIP + " => " + " host = $host"))
	curl, _ := common.NewURL(fmt.Sprintf(factoryConsumerFormat, localIP))
	r, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrl(rule), notify)
	ret := r.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), curl, inv)
	assert.Equal(t, 2, len(ret.ToArray()))
	assert.Equal(t, invoker2, invokers[ret.ToArray()[0]])
	assert.Equal(t, invoker3, invokers[ret.ToArray()[1]])
}

func TestRoute_NoForce(t *testing.T) {
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	localIP := common.GetLocalIp()
	url1, _ := common.NewURL(factory333URL)
	url2, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	url3, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	invoker1 := NewMockInvoker(url1, 1)
	invoker2 := NewMockInvoker(url2, 2)
	invoker3 := NewMockInvoker(url3, 3)
	invokers := []protocol.Invoker{invoker1, invoker2, invoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf(factoryHostIp1234Format, localIP)))
	curl, _ := common.NewURL(fmt.Sprintf(factoryConsumerFormat, localIP))
	r, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrlWithNoForce(rule), notify)
	ret := r.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), curl, inv)
	assert.Equal(t, len(invokers), len(ret.ToArray()))
}

func TestRoute_Force(t *testing.T) {
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	localIP := common.GetLocalIp()
	url1, _ := common.NewURL(factory333URL)
	url2, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	url3, _ := common.NewURL(fmt.Sprintf(factoryDubboFormat, localIP))
	invoker1 := NewMockInvoker(url1, 1)
	invoker2 := NewMockInvoker(url2, 2)
	invoker3 := NewMockInvoker(url3, 3)
	invokers := []protocol.Invoker{invoker1, invoker2, invoker3}
	inv := &invocation.RPCInvocation{}
	rule := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf(factoryHostIp1234Format, localIP)))
	curl, _ := common.NewURL(fmt.Sprintf(factoryConsumerFormat, localIP))
	r, _ := newConditionRouterFactory().NewPriorityRouter(getRouteUrlWithForce(rule, "true"), notify)
	fileredInvokers := r.Route(utils.ToBitmap(invokers), setUpAddrCache(invokers), curl, inv)
	assert.Equal(t, 0, len(fileredInvokers.ToArray()))
}

func TestNewConditionRouterFactory(t *testing.T) {
	factory := newConditionRouterFactory()
	assert.NotNil(t, factory)
}

func TestNewAppRouterFactory(t *testing.T) {
	factory := newAppRouterFactory()
	assert.NotNil(t, factory)
}

func setUpAddrCache(addrs []protocol.Invoker) router.Cache {
	return chain.BuildCache(addrs)
}
