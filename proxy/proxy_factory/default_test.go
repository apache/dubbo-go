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

package proxy_factory

import (
	"context"
	"fmt"
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
)

func TestGetProxy(t *testing.T) {
	proxyFactory := NewDefaultProxyFactory()
	url := common.NewURLWithOptions()
	proxy := proxyFactory.GetProxy(base.NewBaseInvoker(url), url)
	assert.NotNil(t, proxy)
}

type TestAsync struct{}

func (u *TestAsync) CallBack(res common.CallbackResponse) {
	fmt.Println("CallBack res:", res)
}

func TestGetAsyncProxy(t *testing.T) {
	proxyFactory := NewDefaultProxyFactory()
	url := common.NewURLWithOptions()
	async := &TestAsync{}
	proxy := proxyFactory.GetAsyncProxy(base.NewBaseInvoker(url), async.CallBack, url)
	assert.NotNil(t, proxy)
}

func TestGetInvoker(t *testing.T) {
	proxyFactory := NewDefaultProxyFactory()
	url := common.NewURLWithOptions()
	invoker := proxyFactory.GetInvoker(url)
	assert.True(t, invoker.IsAvailable())
}

func TestInfoProxyInvoker_InvokePropagatesGenericVariadicMarker(t *testing.T) {
	info := &common.ServiceInfo{
		Methods: []common.MethodInfo{
			{
				Name: "HelloVariadic",
				MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
					marked, ok := ctx.Value(constant.DubboCtxKey(constant.GenericVariadicCallSliceKey)).(bool)
					require.True(t, ok)
					assert.True(t, marked)
					assert.Equal(t, []any{"hello", []string{"alice", "bob"}}, args)
					return "ok", nil
				},
			},
		},
	}

	invoker := newInfoInvoker(common.NewURLWithOptions(), info, struct{}{})
	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("HelloVariadic"),
		invocation.WithArguments([]any{"hello", []string{"alice", "bob"}}),
	)
	inv.SetAttribute(constant.GenericVariadicCallSliceKey, true)

	res := invoker.Invoke(context.Background(), inv)
	require.NoError(t, res.Error())
	assert.Equal(t, "ok", res.Result())
}
