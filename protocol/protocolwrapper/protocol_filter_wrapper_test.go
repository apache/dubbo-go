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
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const mockFilterKey = "mockEcho"

func TestProtocolFilterWrapperExport(t *testing.T) {
	filtProto := extension.GetProtocol(FILTER)
	filtProto.(*ProtocolFilterWrapper).protocol = &protocol.BaseProtocol{}

	u := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.ServiceFilterKey, mockFilterKey))
	exporter := filtProto.Export(protocol.NewBaseInvoker(u))
	_, ok := exporter.GetInvoker().(*FilterInvoker)
	assert.True(t, ok)
}

func TestProtocolFilterWrapperRefer(t *testing.T) {
	filtProto := extension.GetProtocol(FILTER)
	filtProto.(*ProtocolFilterWrapper).protocol = &protocol.BaseProtocol{}

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

func (ef *mockEchoFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("invoking echo filter.")
	logger.Debugf("%v,%v", invocation.MethodName(), len(invocation.Arguments()))
	if invocation.MethodName() == constant.Echo && len(invocation.Arguments()) == 1 {
		return &protocol.RPCResult{
			Rest: invocation.Arguments()[0],
		}
	}

	return invoker.Invoke(ctx, invocation)
}

func (ef *mockEchoFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func newFilter() filter.Filter {
	return &mockEchoFilter{}
}
