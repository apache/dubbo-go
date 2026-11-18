// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This server is intentionally pinned to dubbo-go v3.3.2 through the parent
// module. It lets the current checkout's Go client test the reverse direction.
package main

import (
	"context"
	"fmt"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
)

const serviceName = "org.apache.dubbo.tri.hessian2.api.GreetingsService"

type GreetRequest struct {
	Name string
}

func (*GreetRequest) JavaClassName() string {
	return "org.apache.dubbo.tri.hessian2.api.GreetRequest"
}

type GreetResponse struct {
	Greeting string
}

func (*GreetResponse) JavaClassName() string {
	return "org.apache.dubbo.tri.hessian2.api.GreetResponse"
}

type GreetService struct{}

type greetings interface {
	Greet(context.Context, *GreetRequest) (*GreetResponse, error)
}

func (*GreetService) Greet(_ context.Context, request *GreetRequest) (*GreetResponse, error) {
	return &GreetResponse{Greeting: fmt.Sprintf("The name of the request is:\t %s", request.Name)}, nil
}

func main() {
	hessian.RegisterPOJO(new(GreetRequest))
	hessian.RegisterPOJO(new(GreetResponse))

	srv, err := server.NewServer(
		server.WithServerProtocol(protocol.WithPort(20000), protocol.WithTriple()),
		server.WithServerSerialization(constant.Hessian2Serialization),
	)
	if err != nil {
		panic(err)
	}
	info := &common.ServiceInfo{
		InterfaceName: serviceName,
		ServiceType:   (*greetings)(nil),
		Methods: []common.MethodInfo{{
			Name:        "Greet",
			Type:        constant.CallUnary,
			ReqInitFunc: func() any { return new(GreetRequest) },
			MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
				response, callErr := handler.(greetings).Greet(ctx, args[0].(*GreetRequest))
				if callErr != nil {
					return nil, callErr
				}
				return triple_protocol.NewResponse(response), nil
			},
		}},
	}
	if err := srv.Register(&GreetService{}, info); err != nil {
		panic(err)
	}
	if err := srv.Serve(); err != nil {
		panic(err)
	}
}
