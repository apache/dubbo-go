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
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

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
