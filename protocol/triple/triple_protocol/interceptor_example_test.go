// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package triple_protocol_test

import (
	"context"
	triple "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"log"
	"os"

	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1/pingv1connect"
)

func ExampleUnaryInterceptorFunc() {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	loggingInterceptor := triple.UnaryInterceptorFunc(
		func(next triple.UnaryFunc) triple.UnaryFunc {
			return triple.UnaryFunc(func(ctx context.Context, request triple.AnyRequest, response triple.AnyResponse) error {
				logger.Println("calling:", request.Spec().Procedure)
				logger.Println("request:", request.Any())
				err := next(ctx, request, response)
				if err != nil {
					logger.Println("error:", err)
				} else {
					logger.Println("response:", response.Any())
				}
				return err
			})
		},
	)
	client := pingv1connect.NewPingServiceClient(
		examplePingServer.Client(),
		examplePingServer.URL(),
		triple.WithInterceptors(loggingInterceptor),
	)
	if err := client.Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{Number: 42}), triple.NewResponse(&pingv1.PingResponse{})); err != nil {
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
	outer := triple.UnaryInterceptorFunc(
		func(next triple.UnaryFunc) triple.UnaryFunc {
			return triple.UnaryFunc(func(ctx context.Context, req triple.AnyRequest, res triple.AnyResponse) error {
				logger.Println("outer interceptor: before call")
				err := next(ctx, req, res)
				logger.Println("outer interceptor: after call")
				return err
			})
		},
	)
	inner := triple.UnaryInterceptorFunc(
		func(next triple.UnaryFunc) triple.UnaryFunc {
			return triple.UnaryFunc(func(ctx context.Context, req triple.AnyRequest, res triple.AnyResponse) error {
				logger.Println("inner interceptor: before call")
				err := next(ctx, req, res)
				logger.Println("inner interceptor: after call")
				return err
			})
		},
	)
	client := pingv1connect.NewPingServiceClient(
		examplePingServer.Client(),
		examplePingServer.URL(),
		triple.WithInterceptors(outer, inner),
	)
	if err := client.Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{}), triple.NewResponse(&pingv1.PingResponse{})); err != nil {
		logger.Println("error:", err)
		return
	}

	// Output:
	// outer interceptor: before call
	// inner interceptor: before call
	// inner interceptor: after call
	// outer interceptor: after call
}
