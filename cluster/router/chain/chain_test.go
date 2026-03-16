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

package chain

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

type mockStaticRouter struct {
	configs []*global.RouterConfig
}

func (m *mockStaticRouter) Route(invokers []base.Invoker, _ *common.URL, _ base.Invocation) []base.Invoker {
	return invokers
}

func (m *mockStaticRouter) URL() *common.URL {
	return nil
}

func (m *mockStaticRouter) Priority() int64 {
	return 0
}

func (m *mockStaticRouter) Notify(_ []base.Invoker) {
}

func (m *mockStaticRouter) SetStaticConfig(cfg *global.RouterConfig) {
	m.configs = append(m.configs, cfg)
}

type mockRouter struct{}

func (m *mockRouter) Route(invokers []base.Invoker, _ *common.URL, _ base.Invocation) []base.Invoker {
	return invokers
}

func (m *mockRouter) URL() *common.URL {
	return nil
}

func (m *mockRouter) Priority() int64 {
	return 0
}

func (m *mockRouter) Notify(_ []base.Invoker) {
}

// TestIsRouterMatch_ServiceMatch tests service router matching
func TestIsRouterMatch_ServiceMatch(t *testing.T) {
	routerCfg := &global.RouterConfig{
		Scope: "service",
		Key:   "test.service",
	}

	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)
	url.SubURL = common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithPath("test.service"),
	)

	assert.True(t, isRouterMatch(routerCfg, url, ""))
}

// TestIsRouterMatch_ServiceMismatch tests service router mismatch
func TestIsRouterMatch_ServiceMismatch(t *testing.T) {
	routerCfg := &global.RouterConfig{
		Scope: "service",
		Key:   "different.service",
	}

	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)
	url.SubURL = common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithPath("test.service"),
	)

	assert.False(t, isRouterMatch(routerCfg, url, ""))
}

// TestIsRouterMatch_ApplicationMatch tests application router matching
func TestIsRouterMatch_ApplicationMatch(t *testing.T) {
	routerCfg := &global.RouterConfig{
		Scope: "application",
		Key:   "test-app",
	}

	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)
	url.SetParam(constant.ApplicationKey, "test-app")

	assert.True(t, isRouterMatch(routerCfg, url, "test-app"))
}

// TestIsRouterMatch_ApplicationMismatch tests application router mismatch
func TestIsRouterMatch_ApplicationMismatch(t *testing.T) {
	routerCfg := &global.RouterConfig{
		Scope: "application",
		Key:   "different-app",
	}

	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)
	url.SetParam(constant.ApplicationKey, "test-app")

	assert.False(t, isRouterMatch(routerCfg, url, "test-app"))
}

// TestGetApplicationName_FromURL tests getting application name from URL
func TestGetApplicationName_FromURL(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)
	url.SetParam(constant.ApplicationKey, "my-app")

	assert.Equal(t, "my-app", getApplicationName(url))
}

// TestGetApplicationName_FromSubURL tests getting application name from SubURL
func TestGetApplicationName_FromSubURL(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)
	url.SubURL = common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithPath("test.service"),
	)
	url.SubURL.SetParam(constant.ApplicationKey, "my-app")

	assert.Equal(t, "my-app", getApplicationName(url))
}

func TestIsRouterMatch_ServiceMatchWithoutSubURL(t *testing.T) {
	routerCfg := &global.RouterConfig{
		Scope: constant.RouterScopeService,
		Key:   "test.service",
	}

	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)

	assert.True(t, isRouterMatch(routerCfg, url, ""))
}

func TestIsRouterMatch_UnknownScope(t *testing.T) {
	routerCfg := &global.RouterConfig{
		Scope: "unknown",
		Key:   "test.service",
	}

	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)

	assert.False(t, isRouterMatch(routerCfg, url, "test-app"))
}

func TestInjectStaticRouters(t *testing.T) {
	trueValue := true
	falseValue := false

	staticRouter := &mockStaticRouter{}
	chain := &RouterChain{
		routers: []router.PriorityRouter{
			staticRouter,
			&mockRouter{},
		},
	}

	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("consumer.path"),
	)
	url.SubURL = common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithPath("svc.test"),
	)
	url.SubURL.SetParam(constant.ApplicationKey, "app.test")
	url.SubURL.SetAttribute(constant.RoutersConfigKey, []*global.RouterConfig{
		nil,
		{
			Scope:   constant.RouterScopeService,
			Key:     "svc.test",
			Enabled: &trueValue,
			Valid:   &trueValue,
		},
		{
			Scope: constant.RouterScopeApplication,
			Key:   "app.test",
		},
		{
			Scope:   constant.RouterScopeApplication,
			Key:     "disabled",
			Enabled: &falseValue,
		},
		{
			Scope: constant.RouterScopeApplication,
			Key:   "invalid",
			Valid: &falseValue,
		},
		{
			Scope: constant.RouterScopeApplication,
			Key:   "other-app",
		},
	})

	chain.injectStaticRouters(url)

	if assert.Len(t, staticRouter.configs, 2) {
		assert.Equal(t, "svc.test", staticRouter.configs[0].Key)
		assert.Equal(t, "app.test", staticRouter.configs[1].Key)
	}
}

func TestInjectStaticRouters_InvalidAttributeType(t *testing.T) {
	staticRouter := &mockStaticRouter{}
	chain := &RouterChain{
		routers: []router.PriorityRouter{staticRouter},
	}

	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("svc.test"),
	)
	url.SetAttribute(constant.RoutersConfigKey, "invalid")

	chain.injectStaticRouters(url)

	assert.Empty(t, staticRouter.configs)
}

const testConsumerServiceURL = "consumer://127.0.0.1/com.demo.Service"

type testPriorityRouter struct {
	priority int64
	called   int
	lastSize int

	notifyFn func([]base.Invoker)
	routeFn  func([]base.Invoker, *common.URL, base.Invocation) []base.Invoker
}

func (r *testPriorityRouter) Route(invokers []base.Invoker, url *common.URL, inv base.Invocation) []base.Invoker {
	r.called++
	r.lastSize = len(invokers)
	if r.routeFn != nil {
		return r.routeFn(invokers, url, inv)
	}
	return invokers
}

func (r *testPriorityRouter) URL() *common.URL {
	return nil
}

func (r *testPriorityRouter) Priority() int64 {
	return r.priority
}

func (r *testPriorityRouter) Notify(invokers []base.Invoker) {
	if r.notifyFn != nil {
		r.notifyFn(invokers)
	}
}

func buildInvoker(t *testing.T, rawURL string) base.Invoker {
	u, err := common.NewURL(rawURL)
	require.NoError(t, err)
	return base.NewBaseInvoker(u)
}

func TestRouteUsesServiceKeyMatchWhenAvailable(t *testing.T) {
	consumerURL, err := common.NewURL(testConsumerServiceURL)
	require.NoError(t, err)

	match := buildInvoker(t, "dubbo://127.0.0.1:20000/com.demo.Service")
	nonMatch := buildInvoker(t, "dubbo://127.0.0.1:20001/com.other.Service")

	r := &testPriorityRouter{priority: 1}
	chain := &RouterChain{
		invokers: []base.Invoker{match, nonMatch},
		routers:  []router.PriorityRouter{r},
	}

	result := chain.Route(consumerURL, invocation.NewRPCInvocation("Say", nil, nil))
	assert.Len(t, result, 1)
	assert.Equal(t, match.GetURL().String(), result[0].GetURL().String())
	assert.Equal(t, 1, r.called)
	assert.Equal(t, 1, r.lastSize)
}

func TestRouteFallsBackToAllInvokersWhenNoMatch(t *testing.T) {
	consumerURL, err := common.NewURL(testConsumerServiceURL)
	require.NoError(t, err)

	invokerA := buildInvoker(t, "dubbo://127.0.0.1:20000/com.foo.Service")
	invokerB := buildInvoker(t, "dubbo://127.0.0.1:20001/com.bar.Service")

	r := &testPriorityRouter{priority: 1}
	chain := &RouterChain{
		invokers: []base.Invoker{invokerA, invokerB},
		routers:  []router.PriorityRouter{r},
	}

	result := chain.Route(consumerURL, invocation.NewRPCInvocation("Say", nil, nil))
	assert.Len(t, result, 2)
	assert.Equal(t, 1, r.called)
	assert.Equal(t, 2, r.lastSize)
}

func TestRouteAppliesRoutersOnSnapshot(t *testing.T) {
	consumerURL, err := common.NewURL(testConsumerServiceURL)
	require.NoError(t, err)

	invokerA := buildInvoker(t, "dubbo://127.0.0.1:20000/com.demo.Service")
	invokerB := buildInvoker(t, "dubbo://127.0.0.1:20001/com.demo.Service")

	r1 := &testPriorityRouter{priority: 1, routeFn: func(invokers []base.Invoker, _ *common.URL, _ base.Invocation) []base.Invoker {
		return invokers[:1]
	}}
	r2 := &testPriorityRouter{priority: 2}

	chain := &RouterChain{
		invokers: []base.Invoker{invokerA, invokerB},
		routers:  []router.PriorityRouter{r1, r2},
	}

	result := chain.Route(consumerURL, invocation.NewRPCInvocation("Say", nil, nil))
	assert.Len(t, result, 1)
	assert.Equal(t, invokerA.GetURL().String(), result[0].GetURL().String())
	assert.Equal(t, 1, r1.called)
	assert.Equal(t, 1, r2.called)
	assert.Equal(t, 1, r2.lastSize)
}
