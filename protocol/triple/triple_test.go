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
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

import (
	grpc_go "github.com/dubbogo/grpc-go"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
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
	triplePort                = "21000"
	dubbo3Port                = "21001"
	listenAddr                = "0.0.0.0"
	localAddr                 = "127.0.0.1"
	name                      = "triple"
	group                     = "g1"
	version                   = "v1"
	customTripleInterfaceName = "apache.dubbo.org.triple"
	customDubbo3InterfaceName = "apache.dubbo.org.dubbo3"
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

func runTripleServer(interfaceName string, group string, version string, addr string, info *server.ServiceInfo, handler interface{}) {
	url := common.NewURLWithOptions(
		common.WithPath(interfaceName),
		common.WithLocation(addr),
		common.WithPort(triplePort),
		common.WithProtocol(TRIPLE),
		common.WithInterface(interfaceName),
	)
	url.SetParam(constant.GroupKey, group)
	url.SetParam(constant.VersionKey, version)
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

func runOldTripleServer(interfaceName string, group string, version string, addr string, desc *grpc_go.ServiceDesc, svc common.RPCService) {
	url := common.NewURLWithOptions(
		// todo(DMwangnima): figure this out
		common.WithPath(interfaceName),
		common.WithLocation(addr),
		common.WithPort(dubbo3Port),
		common.WithProtocol(TRIPLE),
		common.WithInterface(interfaceName),
	)
	url.SetParam(constant.GroupKey, group)
	url.SetParam(constant.VersionKey, version)
	// todo(DMwangnima): add protocol config
	config.SetRootConfig(
		*config.NewRootConfigBuilder().
			SetProvider(
				config.NewProviderConfigBuilder().
					AddService(common.GetReference(svc), config.NewServiceConfigBuilder().
						SetInterface(interfaceName).
						SetGroup(group).
						SetVersion(version).
						Build()).
					SetProxyFactory("default").
					Build()).
			Build())
	config.SetProviderService(svc)
	common.ServiceMap.Register(interfaceName, TRIPLE, group, version, svc)
	invoker := extension.GetProxyFactory("default").GetInvoker(url)
	GetProtocol().(*TripleProtocol).exportForTest(invoker, nil)
}

func TestMain(m *testing.M) {
	runTripleServer(
		customTripleInterfaceName,
		"",
		"",
		listenAddr,
		&greettriple.GreetService_ServiceInfo,
		new(api.GreetTripleServer),
	)
	runTripleServer(
		customTripleInterfaceName,
		group,
		version,
		listenAddr,
		&greettriple.GreetService_ServiceInfo,
		new(api.GreetTripleServerGroup1Version1),
	)
	runOldTripleServer(
		customDubbo3InterfaceName,
		"",
		"",
		listenAddr,
		&dubbo3_greet.GreetService_ServiceDesc,
		new(dubbo3_api.GreetDubbo3Server),
	)
	runOldTripleServer(
		customDubbo3InterfaceName,
		group,
		version,
		listenAddr,
		&dubbo3_greet.GreetService_ServiceDesc,
		new(dubbo3_api.GreetDubbo3ServerGroup1Version1),
	)
	time.Sleep(3 * time.Second)
	m.Run()
	os.Exit(0)
}

func TestInvoke(t *testing.T) {
	tripleInvokerInit := func(location string, port string, interfaceName string, group string, version string, methods []string, info *client.ClientInfo) (protocol.Invoker, error) {
		newURL := common.NewURLWithOptions(
			common.WithInterface(interfaceName),
			common.WithLocation(location),
			common.WithPort(port),
			common.WithMethods(methods),
			common.WithAttribute(constant.ClientInfoKey, info),
		)
		newURL.SetParam(constant.GroupKey, group)
		newURL.SetParam(constant.VersionKey, version)
		return NewTripleInvoker(newURL)
	}
	dubbo3InvokerInit := func(location string, port string, interfaceName string, group string, version string, svc common.RPCService) (protocol.Invoker, error) {
		newURL := common.NewURLWithOptions(
			common.WithInterface(interfaceName),
			common.WithLocation(location),
			common.WithPort(port),
		)
		newURL.SetParam(constant.GroupKey, group)
		newURL.SetParam(constant.VersionKey, version)
		// dubbo3 needs to retrieve ConsumerService directly
		config.SetConsumerServiceByInterfaceName(interfaceName, svc)
		return NewDubbo3Invoker(newURL)
	}
	tripleInvocationInit := func(methodName string, rawParams []interface{}, callType string) protocol.Invocation {
		newInv := invocation_impl.NewRPCInvocationWithOptions(
			invocation_impl.WithMethodName(methodName),
			invocation_impl.WithParameterRawValues(rawParams),
		)
		newInv.SetAttribute(constant.CallTypeKey, callType)
		return newInv
	}
	dubbo3InvocationInit := func(methodName string, params []reflect.Value, reply interface{}) protocol.Invocation {
		newInv := invocation_impl.NewRPCInvocationWithOptions(
			invocation_impl.WithMethodName(methodName),
			invocation_impl.WithParameterValues(params),
		)
		newInv.SetReply(reply)
		return newInv
	}
	dubbo3ReplyInit := func(fieldType reflect.Type) interface{} {
		var reply reflect.Value
		replyType := fieldType.Out(0)
		if replyType.Kind() == reflect.Ptr {
			reply = reflect.New(replyType.Elem())
		} else {
			reply = reflect.New(replyType)
		}
		return reply.Interface()
	}

	invokeTripleCodeFunc := func(t *testing.T, invoker protocol.Invoker, identifier string) {
		tests := []struct {
			methodName string
			callType   string
			rawParams  []interface{}
			validate   func(t *testing.T, rawParams []interface{}, res protocol.Result)
		}{
			{
				methodName: "Greet",
				callType:   constant.CallUnary,
				rawParams: []interface{}{
					&greet.GreetRequest{
						Name: name,
					},
					&greet.GreetResponse{},
				},
				validate: func(t *testing.T, params []interface{}, res protocol.Result) {
					assert.Nil(t, res.Result())
					assert.Nil(t, res.Error())
					req := params[0].(*greet.GreetRequest)
					resp := params[1].(*greet.GreetResponse)
					assert.Equal(t, identifier+req.Name, resp.Greeting)
				},
			},
			{
				methodName: "GreetClientStream",
				callType:   constant.CallClientStream,
				validate: func(t *testing.T, params []interface{}, res protocol.Result) {
					assert.Nil(t, res.Error())
					streamRaw, ok := res.Result().(*triple_protocol.ClientStreamForClient)
					assert.True(t, ok)
					stream := &greettriple.GreetServiceGreetClientStreamClient{ClientStreamForClient: streamRaw}

					var expectRes []string
					times := 5
					for i := 1; i <= times; i++ {
						expectRes = append(expectRes, identifier+name)
						err := stream.Send(&greet.GreetClientStreamRequest{Name: name})
						assert.Nil(t, err)
					}
					expectStr := strings.Join(expectRes, ",")
					resp, err := stream.CloseAndRecv()
					assert.Nil(t, err)
					assert.Equal(t, expectStr, resp.Greeting)
				},
			},
			{
				methodName: "GreetServerStream",
				callType:   constant.CallServerStream,
				rawParams: []interface{}{
					&greet.GreetServerStreamRequest{
						Name: "dubbo",
					},
				},
				validate: func(t *testing.T, params []interface{}, res protocol.Result) {
					assert.Nil(t, res.Error())
					req := params[0].(*greet.GreetServerStreamRequest)
					streamRaw, ok := res.Result().(*triple_protocol.ServerStreamForClient)
					stream := &greettriple.GreetServiceGreetServerStreamClient{ServerStreamForClient: streamRaw}
					assert.True(t, ok)
					times := 5
					for i := 1; i <= times; i++ {
						for stream.Recv() {
							assert.Nil(t, stream.Err())
							assert.Equal(t, identifier+req.Name, stream.Msg().Greeting)
						}
						assert.True(t, true, errors.Is(stream.Err(), io.EOF))
					}
				},
			},
			{
				methodName: "GreetStream",
				callType:   constant.CallBidiStream,
				validate: func(t *testing.T, params []interface{}, res protocol.Result) {
					assert.Nil(t, res.Error())
					streamRaw, ok := res.Result().(*triple_protocol.BidiStreamForClient)
					assert.True(t, ok)
					stream := &greettriple.GreetServiceGreetStreamClient{BidiStreamForClient: streamRaw}
					for i := 1; i <= 5; i++ {
						err := stream.Send(&greet.GreetStreamRequest{Name: name})
						assert.Nil(t, err)
						resp, err := stream.Recv()
						assert.Nil(t, err)
						assert.Equal(t, identifier+name, resp.Greeting)
					}
					assert.Nil(t, stream.CloseRequest())
					assert.Nil(t, stream.CloseResponse())
				},
			},
		}

		for _, test := range tests {
			t.Run(test.methodName, func(t *testing.T) {
				inv := tripleInvocationInit(test.methodName, test.rawParams, test.callType)
				res := invoker.Invoke(context.Background(), inv)
				test.validate(t, test.rawParams, res)
			})
		}
	}
	invokeDubbo3CodeFunc := func(t *testing.T, invoker protocol.Invoker, svc common.RPCService, identifier string) {
		tests := []struct {
			methodName string
			params     []reflect.Value
			validate   func(t *testing.T, params []reflect.Value, res protocol.Result)
		}{
			{
				methodName: "Greet",
				params: []reflect.Value{
					reflect.ValueOf(&greet.GreetRequest{
						Name: name,
					}),
				},
				validate: func(t *testing.T, Params []reflect.Value, res protocol.Result) {
					assert.Nil(t, res.Error())
					req := Params[0].Interface().(*greet.GreetRequest)
					resp := res.Result().(*greet.GreetResponse)
					assert.Equal(t, identifier+req.Name, resp.Greeting)
				},
			},
			{
				methodName: "GreetClientStream",
				validate: func(t *testing.T, reflectParams []reflect.Value, res protocol.Result) {
					assert.Nil(t, res.Error())
					stream, ok := res.Result().(*dubbo3_greet.GreetService_GreetClientStreamClient)
					assert.True(t, ok)

					var expectRes []string
					times := 5
					for i := 1; i <= times; i++ {
						expectRes = append(expectRes, identifier+name)
						err := (*stream).Send(&greet.GreetClientStreamRequest{Name: name})
						assert.Nil(t, err)
					}
					expectStr := strings.Join(expectRes, ",")
					resp, err := (*stream).CloseAndRecv()
					assert.Nil(t, err)
					assert.Equal(t, expectStr, resp.Greeting)
				},
			},
			{
				methodName: "GreetServerStream",
				params: []reflect.Value{
					reflect.ValueOf(&greet.GreetServerStreamRequest{
						Name: "dubbo",
					}),
				},
				validate: func(t *testing.T, params []reflect.Value, res protocol.Result) {
					assert.Nil(t, res.Error())
					req := params[0].Interface().(*greet.GreetServerStreamRequest)
					stream, ok := res.Result().(*dubbo3_greet.GreetService_GreetServerStreamClient)
					assert.True(t, ok)
					times := 5
					for i := 1; i <= times; i++ {
						msg, err := (*stream).Recv()
						assert.Nil(t, err)
						assert.Equal(t, identifier+req.Name, msg.Greeting)
					}
				},
			},
			{
				methodName: "GreetStream",
				validate: func(t *testing.T, params []reflect.Value, res protocol.Result) {
					assert.Nil(t, res.Error())
					stream, ok := res.Result().(*dubbo3_greet.GreetService_GreetStreamClient)
					assert.True(t, ok)
					for i := 1; i <= 5; i++ {
						err := (*stream).Send(&greet.GreetStreamRequest{Name: name})
						assert.Nil(t, err)
						resp, err := (*stream).Recv()
						assert.Nil(t, err)
						assert.Equal(t, identifier+name, resp.Greeting)
					}
					assert.Nil(t, (*stream).CloseSend())
				},
			},
		}

		svcPtrVal := reflect.ValueOf(svc)
		svcVal := svcPtrVal.Elem()
		svcType := svcVal.Type()
		for _, test := range tests {
			t.Run(test.methodName, func(t *testing.T) {
				funcField, ok := svcType.FieldByName(test.methodName)
				assert.True(t, ok)
				reply := dubbo3ReplyInit(funcField.Type)
				inv := dubbo3InvocationInit(test.methodName, test.params, reply)
				res := invoker.Invoke(context.Background(), inv)
				test.validate(t, test.params, res)
			})
		}
	}

	t.Run("triple2triple", func(t *testing.T) {
		invoker, err := tripleInvokerInit(localAddr, triplePort, customTripleInterfaceName, "", "", greettriple.GreetService_ClientInfo.MethodNames, &greettriple.GreetService_ClientInfo)
		assert.Nil(t, err)
		invokeTripleCodeFunc(t, invoker, "")
	})
	t.Run("triple2triple_Group1Version1", func(t *testing.T) {
		invoker, err := tripleInvokerInit(localAddr, triplePort, customTripleInterfaceName, group, version, greettriple.GreetService_ClientInfo.MethodNames, &greettriple.GreetService_ClientInfo)
		assert.Nil(t, err)
		invokeTripleCodeFunc(t, invoker, api.GroupVersionIdentifier)
	})
	t.Run("triple2dubbo3", func(t *testing.T) {
		invoker, err := tripleInvokerInit(localAddr, dubbo3Port, customDubbo3InterfaceName, "", "", greettriple.GreetService_ClientInfo.MethodNames, &greettriple.GreetService_ClientInfo)
		assert.Nil(t, err)
		invokeTripleCodeFunc(t, invoker, "")
	})
	t.Run("triple2dubbo3_Group1Version1", func(t *testing.T) {
		invoker, err := tripleInvokerInit(localAddr, dubbo3Port, customDubbo3InterfaceName, group, version, greettriple.GreetService_ClientInfo.MethodNames, &greettriple.GreetService_ClientInfo)
		assert.Nil(t, err)
		invokeTripleCodeFunc(t, invoker, dubbo3_api.GroupVersionIdentifier)
	})
	t.Run("dubbo32triple", func(t *testing.T) {
		svc := new(dubbo3_greet.GreetServiceClientImpl)
		invoker, err := dubbo3InvokerInit(localAddr, triplePort, customTripleInterfaceName, "", "", svc)
		assert.Nil(t, err)
		invokeDubbo3CodeFunc(t, invoker, svc, "")
	})
	t.Run("dubbo32triple_Group1Version1", func(t *testing.T) {
		svc := new(dubbo3_greet.GreetServiceClientImpl)
		invoker, err := dubbo3InvokerInit(localAddr, triplePort, customTripleInterfaceName, group, version, svc)
		assert.Nil(t, err)
		invokeDubbo3CodeFunc(t, invoker, svc, api.GroupVersionIdentifier)
	})
	t.Run("dubbo32dubbo3", func(t *testing.T) {
		svc := new(dubbo3_greet.GreetServiceClientImpl)
		invoker, err := dubbo3InvokerInit(localAddr, dubbo3Port, customDubbo3InterfaceName, "", "", svc)
		assert.Nil(t, err)
		invokeDubbo3CodeFunc(t, invoker, svc, "")
	})
	t.Run("dubbo32dubbo3_Group1Version1", func(t *testing.T) {
		svc := new(dubbo3_greet.GreetServiceClientImpl)
		invoker, err := dubbo3InvokerInit(localAddr, dubbo3Port, customDubbo3InterfaceName, group, version, svc)
		assert.Nil(t, err)
		invokeDubbo3CodeFunc(t, invoker, svc, dubbo3_api.GroupVersionIdentifier)
	})
}
