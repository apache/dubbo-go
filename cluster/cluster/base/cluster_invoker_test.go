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

package base

import (
	"fmt"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	clusterpkg "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/random"
	"dubbo.apache.org/dubbo-go/v3/common"
	protocolbase "dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

const (
	baseClusterInvokerMethodName = "getUser"
	baseClusterInvokerFormat     = "dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider"
)

func TestStickyNormal(t *testing.T) {
	var invokers []protocolbase.Invoker
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf(baseClusterInvokerFormat, i))
		url.SetParam("sticky", "true")
		invokers = append(invokers, clusterpkg.NewMockInvoker(url, 1))
	}
	base := &BaseClusterInvoker{}
	base.AvailableCheck = true
	var invoked []protocolbase.Invoker

	tmpRandomBalance := random.NewRandomLoadBalance()
	tmpInvocation := invocation.NewRPCInvocation(baseClusterInvokerMethodName, nil, nil)
	result := base.DoSelect(tmpRandomBalance, tmpInvocation, invokers, invoked)
	result1 := base.DoSelect(tmpRandomBalance, tmpInvocation, invokers, invoked)
	assert.Equal(t, result, result1)
}

func TestStickyNormalWhenError(t *testing.T) {
	invokers := []protocolbase.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf(baseClusterInvokerFormat, i))
		url.SetParam("sticky", "true")
		invokers = append(invokers, clusterpkg.NewMockInvoker(url, 1))
	}
	base := &BaseClusterInvoker{}
	base.AvailableCheck = true

	var invoked []protocolbase.Invoker
	result := base.DoSelect(random.NewRandomLoadBalance(), invocation.NewRPCInvocation(baseClusterInvokerMethodName, nil, nil), invokers, invoked)
	invoked = append(invoked, result)
	result1 := base.DoSelect(random.NewRandomLoadBalance(), invocation.NewRPCInvocation(baseClusterInvokerMethodName, nil, nil), invokers, invoked)
	assert.NotEqual(t, result, result1)
}

// TestStickyConcurrentDoSelect verifies that concurrent calls to DoSelect
// with sticky enabled do not cause a data race on StickyInvoker.
func TestStickyConcurrentDoSelect(t *testing.T) {
	var invokers []protocolbase.Invoker
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf(baseClusterInvokerFormat, i))
		url.SetParam("sticky", "true")
		invokers = append(invokers, clusterpkg.NewMockInvoker(url, 1))
	}
	base := &BaseClusterInvoker{}
	base.AvailableCheck = true

	lb := random.NewRandomLoadBalance()
	invocation1 := invocation.NewRPCInvocation(baseClusterInvokerMethodName, nil, nil)

	const concurrency = 100
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			invoked := make([]protocolbase.Invoker, 0)
			result := base.DoSelect(lb, invocation1, invokers, invoked)
			assert.NotNil(t, result)
		}()
	}
	wg.Wait()
}

// TestStickyConcurrentIsAvailableAndDoSelect verifies that concurrent
// IsAvailable and DoSelect calls do not cause a data race on StickyInvoker.
func TestStickyConcurrentIsAvailableAndDoSelect(t *testing.T) {
	var invokers []protocolbase.Invoker
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf(baseClusterInvokerFormat, i))
		url.SetParam("sticky", "true")
		invokers = append(invokers, clusterpkg.NewMockInvoker(url, 1))
	}

	// Use NewBaseClusterInvoker so that Directory is initialized,
	// allowing IsAvailable() to work without panicking.
	dir := newMockDirectory(invokers)
	base := NewBaseClusterInvoker(dir)
	base.AvailableCheck = true

	lb := random.NewRandomLoadBalance()
	invocation1 := invocation.NewRPCInvocation(baseClusterInvokerMethodName, nil, nil)

	// First DoSelect to set the sticky invoker so IsAvailable uses the sticky path
	invoked := make([]protocolbase.Invoker, 0)
	base.DoSelect(lb, invocation1, invokers, invoked)

	const concurrency = 100
	var wg sync.WaitGroup
	wg.Add(concurrency * 2)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			base.IsAvailable()
		}()
		go func() {
			defer wg.Done()
			base.DoSelect(lb, invocation1, invokers, invoked)
		}()
	}
	wg.Wait()
}

// mockDirectory is a minimal directory.Directory implementation for testing.
type mockDirectory struct {
	invokers []protocolbase.Invoker
	url      *common.URL
}

func newMockDirectory(invokers []protocolbase.Invoker) *mockDirectory {
	url, _ := common.NewURL(baseClusterInvokerFormat)
	url.SetParam("sticky", "true")
	return &mockDirectory{invokers: invokers, url: url}
}

func (d *mockDirectory) GetURL() *common.URL     { return d.url }
func (d *mockDirectory) IsAvailable() bool       { return true }
func (d *mockDirectory) Destroy()                {}
func (d *mockDirectory) List(protocolbase.Invocation) []protocolbase.Invoker { return d.invokers }
func (d *mockDirectory) Subscribe(*common.URL) error { return nil }
