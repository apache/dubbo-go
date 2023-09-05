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

package triple

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/greettriple"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/server/api"
	"dubbo.apache.org/dubbo-go/v3/server"
	"fmt"
	"testing"
	"time"
)

type tripleInvoker struct {
	url     *common.URL
	info    *server.ServiceInfo
	base    *protocol.BaseInvoker
	handler interface{}
}

func (t *tripleInvoker) GetURL() *common.URL {
	return t.url
}

func (t *tripleInvoker) IsAvailable() bool {
	return t.base.IsAvailable()
}

func (t *tripleInvoker) Destroy() {
	t.base.Destroy()
}

func (t *tripleInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	name := invocation.MethodName()
	args := invocation.Arguments()
	// todo(DMwangnima): user map to represent Methods
	for _, method := range t.info.Methods {
		if method.Name == name {
			res, err := method.MethodFunc(ctx, args, t.handler)
			result := new(protocol.RPCResult)
			result.SetResult(res)
			result.SetError(err)
			return result
		}
	}
	panic(fmt.Sprintf("no match method for %s", name))
}

func runTripleServer(interfaceName string, addr string, info *server.ServiceInfo, handler interface{}) {
	url := common.NewURLWithOptions(
		common.WithPath(interfaceName),
		common.WithLocation(addr),
		common.WithPort("20000"),
	)
	var invoker protocol.Invoker
	if info != nil {
		invoker = &tripleInvoker{
			url:     url,
			info:    info,
			base:    protocol.NewBaseInvoker(url),
			handler: handler,
		}
	}
	triple := NewTripleProtocol()
	triple.exportForTest(invoker, info)
}

func TestMain(m *testing.M) {
	runTripleServer(
		greettriple.GreetServiceName,
		"0.0.0.0",
		&greettriple.GreetService_ServiceInfo,
		new(api.GreetConnectServer),
	)
	time.Sleep(3 * time.Second)
	m.Run()
}

func TestInvoke(t *testing.T) {
	info := greettriple.GreetService_ClientInfo
	url := common.NewURLWithOptions(
		common.WithInterface(info.InterfaceName),
		common.WithLocation("127.0.0.1"),
		common.WithPort("20000"),
		common.WithMethods(info.MethodNames),
	)
	invoker, err := NewTripleInvoker(url)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		desc       string
		methodName string
		params     []interface{}
		callType   string
	}{
		{
			desc:       "Unary",
			methodName: "Greet",
			params: []interface{}{
				&greet.GreetRequest{
					Name: "dubbo",
				},
				&greet.GreetResponse{},
			},
			callType: constant.CallUnary,
		},
		{
			desc:       "ClientStream",
			methodName: "GreetClientStream",
			callType:   constant.CallClientStream,
		},
		{
			desc:       "ServerStream",
			methodName: "GreetServerStream",
			params: []interface{}{
				&greet.GreetServerStreamRequest{
					Name: "dubbo",
				},
			},
			callType: constant.CallServerStream,
		},
		{
			desc:       "BidiStream",
			methodName: "GreetStream",
			callType:   constant.CallBidiStream,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			inv := invocation_impl.NewRPCInvocationWithOptions(
				invocation_impl.WithMethodName(test.methodName),
				// todo: process opts
				invocation_impl.WithParameterRawValues(test.params),
			)
			inv.SetAttribute(constant.CallTypeKey, test.callType)
			res := invoker.Invoke(context.Background(), inv)
			t.Logf("%+v", res)
		})
	}
}
