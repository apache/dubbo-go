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
	"net/url"
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

type ProxyInvokerService struct{}

func (s *ProxyInvokerService) Hello(_ context.Context, name string) (string, error) {
	return "hello:" + name, nil
}

type PassThroughService struct{}

func (s *PassThroughService) Service(method string, argTypes []string, args [][]byte, attachments map[string]any) (any, error) {
	body := ""
	if len(args) > 0 {
		body = string(args[0])
	}
	return fmt.Sprintf("%s|%s|%s", method, strings.Join(argTypes, ","), body), nil
}

func registerService(t *testing.T, protocol, interfaceName string, svc common.RPCService) {
	t.Helper()
	_, err := common.ServiceMap.Register(interfaceName, protocol, "", "", svc)
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = common.ServiceMap.UnRegister(interfaceName, protocol, common.ServiceKey(interfaceName, "", ""))
	})
}

func newURL(protocol, interfaceName string) *common.URL {
	return common.NewURLWithOptions(
		common.WithProtocol(protocol),
		common.WithPath(interfaceName),
		common.WithInterface(interfaceName),
		common.WithParams(url.Values{constant.InterfaceKey: {interfaceName}}),
	)
}

func TestProxyInvoker_Invoke(t *testing.T) {
	const (
		protocol      = "test-protocol"
		interfaceName = "ProxyInvokerService"
	)
	registerService(t, protocol, interfaceName, &ProxyInvokerService{})
	u := newURL(protocol, interfaceName)
	invoker := &ProxyInvoker{BaseInvoker: *base.NewBaseInvoker(u)}

	t.Run("invoke success", func(t *testing.T) {
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName("Hello"),
			invocation.WithArguments([]any{"world"}),
			invocation.WithAttachments(map[string]any{"trace": "t1"}),
		)
		result := invoker.Invoke(context.Background(), inv)
		assert.NoError(t, result.Error())
		assert.Equal(t, "hello:world", result.Result())
		assert.Equal(t, "t1", result.Attachments()["trace"])
	})

	t.Run("method not found", func(t *testing.T) {
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName("Missing"),
			invocation.WithArguments([]any{}),
		)
		result := invoker.Invoke(context.Background(), inv)
		assert.Error(t, result.Error())
	})

	t.Run("service not found", func(t *testing.T) {
		absentURL := newURL(protocol, "UnknownService")
		absentInvoker := &ProxyInvoker{BaseInvoker: *base.NewBaseInvoker(absentURL)}
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName("Hello"),
			invocation.WithArguments([]any{}),
		)
		result := absentInvoker.Invoke(context.Background(), inv)
		assert.Error(t, result.Error())
	})
}

func TestPassThroughProxyInvoker_Invoke(t *testing.T) {
	const (
		protocol      = "pass-protocol"
		interfaceName = "PassThroughService"
	)
	registerService(t, protocol, interfaceName, &PassThroughService{})
	u := newURL(protocol, interfaceName)
	invoker := &PassThroughProxyInvoker{
		ProxyInvoker: &ProxyInvoker{BaseInvoker: *base.NewBaseInvoker(u)},
	}

	t.Run("pass through success", func(t *testing.T) {
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName("RawMethod"),
			invocation.WithArguments([]any{[]byte("payload")}),
			invocation.WithAttachments(map[string]any{
				constant.ParamsTypeKey: []string{"bytes"},
				"trace":                "abc",
			}),
		)
		result := invoker.Invoke(context.Background(), inv)
		assert.NoError(t, result.Error())
		assert.Equal(t, "RawMethod|bytes|payload", result.Result())
		assert.Equal(t, "abc", result.Attachments()["trace"])
	})

	t.Run("argument type mismatch", func(t *testing.T) {
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName("RawMethod"),
			invocation.WithArguments([]any{"not-bytes"}),
		)
		result := invoker.Invoke(context.Background(), inv)
		assert.EqualError(t, result.Error(), "the param type is not []byte")
	})
}
