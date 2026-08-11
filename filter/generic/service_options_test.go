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

package generic_test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/filter/generic"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/proxy"
)

func TestGenericServiceInvokeWithTypeCopiesCallOption(t *testing.T) {
	service := generic.NewGenericService("TestService")
	var gotTrailer *http.Header
	service.InvokeWithOptions = func(_ context.Context, _ string, _ []string, _ []hessian.Object, opts ...client.CallOption) (any, error) {
		options := &client.CallOptions{}
		for _, opt := range opts {
			opt(options)
		}
		gotTrailer = options.ResponseTrailer
		return map[string]any{
			"name": "testUser",
			"age":  25,
		}, nil
	}

	var trailers http.Header
	var user struct {
		Name string
		Age  int
	}
	err := service.InvokeWithType(
		context.Background(),
		"getUser",
		[]string{"java.lang.String"},
		[]hessian.Object{"123"},
		&user,
		client.WithResponseTrailer(&trailers),
	)

	require.NoError(t, err)
	require.Same(t, &trailers, gotTrailer)
	require.Equal(t, "testUser", user.Name)
	require.Equal(t, 25, user.Age)
}

type genericOptionsInvoker struct {
	base.BaseInvoker
	invocation base.Invocation
}

func (i *genericOptionsInvoker) Invoke(_ context.Context, inv base.Invocation) result.Result {
	i.invocation = inv
	if reply, ok := inv.Reply().(*any); ok {
		*reply = map[string]any{
			"name": "testUser",
			"age":  25,
		}
	}
	return &result.RPCResult{}
}

func TestGenericServiceInvokeWithTypeOptionsThroughProxy(t *testing.T) {
	invoker := &genericOptionsInvoker{BaseInvoker: *base.NewBaseInvoker(&common.URL{})}
	service := generic.NewGenericService("TestService")
	proxy.NewProxy(invoker, nil, nil).Implement(service)

	var responseHeader http.Header
	var responseTrailer http.Header
	var user struct {
		Name string
		Age  int
	}
	err := service.InvokeWithTypeOptions(
		context.Background(),
		"getUser",
		[]string{"java.lang.String"},
		[]hessian.Object{"123"},
		&user,
		client.WithCallRequestTimeout(time.Second),
		client.WithResponseHeader(&responseHeader),
		client.WithResponseTrailer(&responseTrailer),
	)

	require.NoError(t, err)
	require.Equal(t, "testUser", user.Name)
	require.Equal(t, 25, user.Age)
	require.Equal(t, []any{
		"getUser",
		[]string{"java.lang.String"},
		[]hessian.Object{"123"},
	}, invoker.invocation.Arguments())

	timeout, ok := invoker.invocation.GetAttachment(constant.TimeoutKey)
	require.True(t, ok)
	require.Equal(t, "1s", timeout)
	header, ok := invoker.invocation.GetAttribute(constant.ResponseHeaderKey)
	require.True(t, ok)
	require.Same(t, &responseHeader, header)
	trailer, ok := invoker.invocation.GetAttribute(constant.ResponseTrailerKey)
	require.True(t, ok)
	require.Same(t, &responseTrailer, trailer)
}
