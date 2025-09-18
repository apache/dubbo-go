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

package extension

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// Mock implementations for testing
type MockProtocol struct {
	base.BaseProtocol
}

type MockFilter struct{}

func (m *MockFilter) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	return &result.RPCResult{}
}

func (m *MockFilter) OnResponse(ctx context.Context, res result.Result, invoker base.Invoker, invocation base.Invocation) result.Result {
	return res
}

type MockLoadBalance struct{}

func (m *MockLoadBalance) Select(invokers []base.Invoker, invocation base.Invocation) base.Invoker {
	return nil
}

// TestProtocolMemoryManagement tests protocol registration and unregistration
func TestProtocolMemoryManagement(t *testing.T) {
	testName := "test-protocol-memory"

	// Get initial count
	initialCount := len(GetAllProtocolNames())

	// Register a protocol
	SetProtocol(testName, func() base.Protocol {
		return &MockProtocol{}
	})

	// Verify registration
	afterRegisterCount := len(GetAllProtocolNames())
	assert.Equal(t, initialCount+1, afterRegisterCount, "Protocol should be registered")

	// Verify protocol can be retrieved
	assert.NotPanics(t, func() {
		GetProtocol(testName)
	}, "Should be able to get registered protocol")

	// Unregister the protocol
	UnregisterProtocol(testName)

	// Verify unregistration
	afterUnregisterCount := len(GetAllProtocolNames())
	assert.Equal(t, initialCount, afterUnregisterCount, "Protocol should be unregistered")

	// Verify protocol cannot be retrieved
	assert.Panics(t, func() {
		GetProtocol(testName)
	}, "Should panic when trying to get unregistered protocol")
}

// TestFilterMemoryManagement tests filter registration and unregistration
func TestFilterMemoryManagement(t *testing.T) {
	testName := "test-filter-memory"

	// Get initial count
	initialCount := len(GetAllFilterNames())

	// Register a filter
	SetFilter(testName, func() filter.Filter {
		return &MockFilter{}
	})

	// Verify registration
	afterRegisterCount := len(GetAllFilterNames())
	assert.Equal(t, initialCount+1, afterRegisterCount, "Filter should be registered")

	// Verify filter can be retrieved
	f, exists := GetFilter(testName)
	assert.True(t, exists, "Should be able to get registered filter")
	assert.NotNil(t, f, "Retrieved filter should not be nil")

	// Unregister the filter
	UnregisterFilter(testName)

	// Verify unregistration
	afterUnregisterCount := len(GetAllFilterNames())
	assert.Equal(t, initialCount, afterUnregisterCount, "Filter should be unregistered")

	// Verify filter cannot be retrieved
	f, exists = GetFilter(testName)
	assert.False(t, exists, "Should not be able to get unregistered filter")
	assert.Nil(t, f, "Retrieved filter should be nil")
}

// TestLoadbalanceMemoryManagement tests loadbalance registration and unregistration
func TestLoadbalanceMemoryManagement(t *testing.T) {
	testName := "test-loadbalance-memory"

	// Get initial count
	initialCount := len(GetAllLoadbalanceNames())

	// Register a loadbalance
	SetLoadbalance(testName, func() loadbalance.LoadBalance {
		return &MockLoadBalance{}
	})

	// Verify registration
	afterRegisterCount := len(GetAllLoadbalanceNames())
	assert.Equal(t, initialCount+1, afterRegisterCount, "LoadBalance should be registered")

	// Verify loadbalance can be retrieved
	assert.NotPanics(t, func() {
		GetLoadbalance(testName)
	}, "Should be able to get registered loadbalance")

	// Unregister the loadbalance
	UnregisterLoadbalance(testName)

	// Verify unregistration
	afterUnregisterCount := len(GetAllLoadbalanceNames())
	assert.Equal(t, initialCount, afterUnregisterCount, "LoadBalance should be unregistered")

	// Verify loadbalance cannot be retrieved
	assert.Panics(t, func() {
		GetLoadbalance(testName)
	}, "Should panic when trying to get unregistered loadbalance")
}
