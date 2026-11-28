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

package exec_limit

import (
	"context"
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

func TestFilterInvokeIgnored(t *testing.T) {
	methodName := "hello"
	invoc := invocation.NewRPCInvocation(methodName, []any{"OK"}, make(map[string]any))

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.InterfaceKey, methodName))

	limitFilter := newFilter()

	result := limitFilter.Invoke(context.Background(), base.NewBaseInvoker(invokeUrl), invoc)
	assert.NotNil(t, result)
	assert.NoError(t, result.Error())
}

// countingInvoker counts the calls that reach the service.
type countingInvoker struct {
	*base.BaseInvoker
	calls int
}

func (c *countingInvoker) Invoke(context.Context, base.Invocation) result.Result {
	c.calls++
	return &result.RPCResult{Rest: "OK"}
}

// TestFilterInvokeConfigureError checks that an invalid execute.limit fails the
// call with an error naming the configuration, without invoking the service.
func TestFilterInvokeConfigureError(t *testing.T) {
	for _, tc := range []struct {
		name   string
		key    string
		target string
	}{
		{name: "service level", key: constant.ExecuteLimitKey, target: "greet.GreetService"},
		{name: "method level", key: "methods.Greet." + constant.ExecuteLimitKey, target: "greet.GreetService#Greet"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invoc := invocation.NewRPCInvocation("Greet", []any{"OK"}, make(map[string]any))
			invokeURL := common.NewURLWithOptions(
				common.WithParams(url.Values{}),
				common.WithParamsValue(constant.InterfaceKey, "greet.GreetService"),
				common.WithParamsValue(tc.key, "13a"),
			)
			invoker := &countingInvoker{BaseInvoker: base.NewBaseInvoker(invokeURL)}

			res := newFilter().Invoke(context.Background(), invoker, invoc)

			require.Error(t, res.Error())
			assert.Equal(t, "the configuration of execute.limit is invalid, target="+tc.target, res.Error().Error())
			assert.NotContains(t, res.Error().Error(), "13a", "the configured value stays out of the consumer error")
			assert.Zero(t, invoker.calls)
		})
	}
}

func TestFilterInvoke(t *testing.T) {
	methodName := "hello1"
	invoc := invocation.NewRPCInvocation(methodName, []any{"OK"}, make(map[string]any))

	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.InterfaceKey, methodName),
		common.WithParamsValue(constant.ExecuteLimitKey, "20"),
	)

	limitFilter := newFilter()

	result := limitFilter.Invoke(context.Background(), base.NewBaseInvoker(invokeUrl), invoc)
	assert.NotNil(t, result)
	assert.NoError(t, result.Error())
}
