// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package connect_test

import (
	"context"
	connect "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect"
	"log"
	"os"

	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect/proto/connect/ping/v1"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect/proto/connect/ping/v1/pingv1connect"
)

func ExampleUnaryInterceptorFunc() {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	loggingInterceptor := connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return connect.UnaryFunc(func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
				logger.Println("calling:", request.Spec().Procedure)
				logger.Println("request:", request.Any())
				response, err := next(ctx, request)
				if err != nil {
					logger.Println("error:", err)
				} else {
					logger.Println("response:", response.Any())
				}
				return response, err
			})
		},
	)
	client := pingv1connect.NewPingServiceClient(
		examplePingServer.Client(),
		examplePingServer.URL(),
		connect.WithInterceptors(loggingInterceptor),
	)
	if _, err := client.Ping(context.Background(), connect.NewRequest(&pingv1.PingRequest{Number: 42})); err != nil {
		logger.Println("error:", err)
		return
	}

	// Output:
	// calling: /connect.ping.v1.PingService/Ping
	// request: number:42
	// response: number:42
}

func ExampleWithInterceptors() {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	outer := connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				logger.Println("outer interceptor: before call")
				res, err := next(ctx, req)
				logger.Println("outer interceptor: after call")
				return res, err
			})
		},
	)
	inner := connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				logger.Println("inner interceptor: before call")
				res, err := next(ctx, req)
				logger.Println("inner interceptor: after call")
				return res, err
			})
		},
	)
	client := pingv1connect.NewPingServiceClient(
		examplePingServer.Client(),
		examplePingServer.URL(),
		connect.WithInterceptors(outer, inner),
	)
	if _, err := client.Ping(context.Background(), connect.NewRequest(&pingv1.PingRequest{})); err != nil {
		logger.Println("error:", err)
		return
	}

	// Output:
	// outer interceptor: before call
	// inner interceptor: before call
	// inner interceptor: after call
	// outer interceptor: after call
}
