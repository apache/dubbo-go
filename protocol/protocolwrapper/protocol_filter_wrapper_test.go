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

package protocolwrapper

import (
	"context"
	"net/url"
	"testing"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

const mockFilterKey = "mockEcho"

func TestProtocolFilterWrapperExport(t *testing.T) {
	filtProto := extension.GetProtocol(FILTER)
	filtProto.(*ProtocolFilterWrapper).protocol = &base.BaseProtocol{}

	u := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.ServiceFilterKey, mockFilterKey))
	exporter := filtProto.Export(base.NewBaseInvoker(u))
	_, ok := exporter.GetInvoker().(*FilterInvoker)
	assert.True(t, ok)
}

func TestProtocolFilterWrapperRefer(t *testing.T) {
	filtProto := extension.GetProtocol(FILTER)
	filtProto.(*ProtocolFilterWrapper).protocol = &base.BaseProtocol{}

	u := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.ReferenceFilterKey, mockFilterKey))
	invoker := filtProto.Refer(u)
	_, ok := invoker.(*FilterInvoker)
	assert.True(t, ok)
}

// The initialization of mockEchoFilter, for test
func init() {
	extension.SetFilter(mockFilterKey, newFilter)
}

type mockEchoFilter struct{}

func (ef *mockEchoFilter) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	logger.Infof("invoking echo filter.")
	logger.Debugf("%v,%v", invocation.MethodName(), len(invocation.Arguments()))
	if invocation.MethodName() == constant.Echo && len(invocation.Arguments()) == 1 {
		return &result.RPCResult{
			Rest: invocation.Arguments()[0],
		}
	}

	return invoker.Invoke(ctx, invocation)
}

func (ef *mockEchoFilter) OnResponse(ctx context.Context, result result.Result, invoker base.Invoker, invocation base.Invocation) result.Result {
	return result
}

func newFilter() filter.Filter {
	return &mockEchoFilter{}
}

// TestBuildInvokerChainSkipsUnknownFilter verifies that an unregistered filter
// name (typo / unimported) is skipped rather than producing a nil-filter
// FilterInvoker that nil-derefs at invoke time. Regression for #3547.
func TestBuildInvokerChainSkipsUnknownFilter(t *testing.T) {
	u := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.ServiceFilterKey, mockFilterKey+",unknownFilterName"))
	invoker := base.NewBaseInvoker(u)
	chain := BuildInvokerChain(invoker, constant.ServiceFilterKey)

	// top of the chain is the registered mockEcho FilterInvoker (unknown skipped)
	_, ok := chain.(*FilterInvoker)
	assert.True(t, ok)

	// A non-echo method forces mockEcho to call next.Invoke; on the old code path
	// the unknown filter produced a nil-filter FilterInvoker that nil-derefs here.
	assert.NotPanics(t, func() {
		_ = chain.Invoke(context.Background(), invocation.NewRPCInvocation("someMethod", nil, nil))
	})
}
