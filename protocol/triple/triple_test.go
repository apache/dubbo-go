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
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

import (
	grpc_go "github.com/dubbogo/grpc-go"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	dubbo3_api "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/dubbo3_server/api"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	dubbo3_greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/dubbo3_gen"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/triple_gen/greettriple"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/server/api"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	"dubbo.apache.org/dubbo-go/v3/server"
)

const (
	triplePort = "20000"
	dubbo3Port = "20001"
	listenAddr = "0.0.0.0"
	name       = "triple"
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
		common.WithPort(triplePort),
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
	GetProtocol().(*TripleProtocol).exportForTest(invoker, info)
}

func runOldTripleServer(addr string, desc *grpc_go.ServiceDesc) {
	url := common.NewURLWithOptions(
		common.WithPath(desc.ServiceName),
		common.WithLocation(addr),
		common.WithPort(dubbo3Port),
		common.WithProtocol(TRIPLE),
	)
	srv := new(dubbo3_api.GreetDubbo3Server)
	// todo(DMwangnima): add protocol config
	config.SetRootConfig(
		*config.NewRootConfigBuilder().
			SetProvider(
				config.NewProviderConfigBuilder().
					AddService(common.GetReference(srv), config.NewServiceConfigBuilder().
						SetInterface(desc.ServiceName).
						Build()).
					SetProxyFactory("default").
					Build()).
			Build())
	config.SetProviderService(srv)
	common.ServiceMap.Register(desc.ServiceName, TRIPLE, "", "", srv)
	invoker := extension.GetProxyFactory("default").GetInvoker(url)
	GetProtocol().(*TripleProtocol).exportForTest(invoker, nil)
}

func TestMain(m *testing.M) {
	runTripleServer(
		greettriple.GreetServiceName,
		listenAddr,
		&greettriple.GreetService_ServiceInfo,
		new(api.GreetTripleServer),
	)
	runOldTripleServer(
		listenAddr,
		&dubbo3_greet.GreetService_ServiceDesc,
	)
	time.Sleep(3 * time.Second)
	m.Run()
}

func TestInvoke(t *testing.T) {
	invokeFunc := func(t *testing.T, port string, interfaceName string, methods []string) {
		url := common.NewURLWithOptions(
			common.WithInterface(interfaceName),
			common.WithLocation("127.0.0.1"),
			common.WithPort(port),
			common.WithMethods(methods),
			common.WithProtocol(TRIPLE),
		)
		invoker, err := NewTripleInvoker(url)
		if err != nil {
			t.Fatal(err)
		}
		tests := []struct {
			desc       string
			methodName string
			params     []interface{}
			invoke     func(t *testing.T, params []interface{}, res protocol.Result)
			callType   string
		}{
			{
				desc:       "Unary",
				methodName: "Greet",
				params: []interface{}{
					&greet.GreetRequest{
						Name: name,
					},
					&greet.GreetResponse{},
				},
				invoke: func(t *testing.T, params []interface{}, res protocol.Result) {
					assert.Nil(t, res.Result())
					assert.Nil(t, res.Error())
					req := params[0].(*greet.GreetRequest)
					resp := params[1].(*greet.GreetResponse)
					assert.Equal(t, req.Name, resp.Greeting)
				},
				callType: constant.CallUnary,
			},
			{
				desc:       "ClientStream",
				methodName: "GreetClientStream",
				invoke: func(t *testing.T, params []interface{}, res protocol.Result) {
					assert.Nil(t, res.Error())
					streamRaw, ok := res.Result().(*triple_protocol.ClientStreamForClient)
					assert.True(t, ok)
					stream := &greettriple.GreetServiceGreetClientStreamClient{ClientStreamForClient: streamRaw}

					var expectRes []string
					times := 5
					for i := 1; i <= times; i++ {
						expectRes = append(expectRes, name)
						err := stream.Send(&greet.GreetClientStreamRequest{Name: name})
						assert.Nil(t, err)
					}
					expectStr := strings.Join(expectRes, ",")
					resp, err := stream.CloseAndRecv()
					assert.Nil(t, err)
					assert.Equal(t, expectStr, resp.Greeting)
				},
				callType: constant.CallClientStream,
			},
			{
				desc:       "ServerStream",
				methodName: "GreetServerStream",
				params: []interface{}{
					&greet.GreetServerStreamRequest{
						Name: "dubbo",
					},
				},
				invoke: func(t *testing.T, params []interface{}, res protocol.Result) {
					assert.Nil(t, res.Error())
					req := params[0].(*greet.GreetServerStreamRequest)
					streamRaw, ok := res.Result().(*triple_protocol.ServerStreamForClient)
					stream := &greettriple.GreetServiceGreetServerStreamClient{ServerStreamForClient: streamRaw}
					assert.True(t, ok)
					times := 5
					for i := 1; i <= times; i++ {
						for stream.Recv() {
							assert.Nil(t, stream.Err())
							assert.Equal(t, req.Name, stream.Msg().Greeting)
						}
						assert.True(t, true, errors.Is(stream.Err(), io.EOF))
					}
				},
				callType: constant.CallServerStream,
			},
			{
				desc:       "BidiStream",
				methodName: "GreetStream",
				invoke: func(t *testing.T, params []interface{}, res protocol.Result) {
					assert.Nil(t, res.Error())
					streamRaw, ok := res.Result().(*triple_protocol.BidiStreamForClient)
					assert.True(t, ok)
					stream := &greettriple.GreetServiceGreetStreamClient{BidiStreamForClient: streamRaw}
					for i := 1; i <= 5; i++ {
						err := stream.Send(&greet.GreetStreamRequest{Name: name})
						assert.Nil(t, err)
						resp, err := stream.Recv()
						assert.Nil(t, err)
						assert.Equal(t, name, resp.Greeting)
					}
					assert.Nil(t, stream.CloseRequest())
					assert.Nil(t, stream.CloseResponse())
				},
				callType: constant.CallBidiStream,
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
				test.invoke(t, test.params, res)
			})
		}
	}
	t.Run("invoke server code generated by triple", func(t *testing.T) {
		invokeFunc(t, triplePort, greettriple.GreetService_ClientInfo.InterfaceName, greettriple.GreetService_ClientInfo.MethodNames)
	})
	t.Run("invoke server code generated by dubbo3", func(t *testing.T) {
		desc := dubbo3_greet.GreetService_ServiceDesc
		var methods []string
		for _, method := range desc.Methods {
			methods = append(methods, method.MethodName)
		}
		for _, stream := range desc.Streams {
			methods = append(methods, stream.StreamName)
		}

		invokeFunc(t, dubbo3Port, desc.ServiceName, methods)
	})

}
