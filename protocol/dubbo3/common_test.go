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

package dubbo3

import (
	"context"
	"fmt"
)

import (
	native_grpc "github.com/dubbogo/grpc-go"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo3/internal"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// userd dubbo3 biz service
func addService() {
	config.SetProviderService(newGreeterProvider())
}

type greeterProvider struct {
	internal.UnimplementedGreeterServer
}

func newGreeterProvider() *greeterProvider {
	return &greeterProvider{}
}

func (g *greeterProvider) SayHello(ctx context.Context, req *internal.HelloRequest) (reply *internal.HelloReply, err error) {
	fmt.Printf("req: %v", req)
	return &internal.HelloReply{Message: "this is message from reply"}, nil
}

func dubboGreeterSayHelloHandler(srv interface{}, ctx context.Context,
	dec func(interface{}) error, interceptor native_grpc.UnaryServerInterceptor) (interface{}, error) {

	in := new(internal.HelloRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	base := srv.(Dubbo3GrpcService)

	args := []interface{}{}
	args = append(args, in)
	invo := invocation.NewRPCInvocation("SayHello", args, nil)

	if interceptor == nil {
		result := base.XXX_GetProxyImpl().Invoke(context.Background(), invo)
		return result.Result(), result.Error()
	}
	info := &native_grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/helloworld.Greeter/SayHello",
	}
	handler := func(context.Context, interface{}) (interface{}, error) {
		result := base.XXX_GetProxyImpl().Invoke(context.Background(), invo)
		return result.Result(), result.Error()
	}
	return interceptor(ctx, in, info, handler)
}
