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
	"log"
	"os"
)

import (
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1/pingv1connect"
)

func Example_client() {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	// Unfortunately, pkg.go.dev can't run examples that actually use the
	// network. To keep this example runnable, we'll use an HTTP server and
	// client that communicate over in-memory pipes. The client is still a plain
	// *http.Client!
	httpClient := examplePingServer.Client()

	// By default, clients use the GRPC protocol. Add triple_protocol.WithTriple() or
	// triple_protocol.WithGRPCWeb() to switch protocols.
	client := pingv1connect.NewPingServiceClient(
		httpClient,
		examplePingServer.URL(),
	)
	response := tri.NewResponse(&pingv1.PingResponse{})
	if err := client.Ping(
		context.Background(),
		tri.NewRequest(&pingv1.PingRequest{Number: 42}),
		response,
	); err != nil {
		logger.Println("error:", err)
		return
	}
	logger.Println("response content-type:", response.Header().Get("Content-Type"))
	logger.Println("response message:", response.Msg)

	// Output:
	// response content-type: application/grpc+proto
	// response message: number:42
}

// This example demonstrates how to use the Triple protocol for unary calls.
func Example_clientWithTripleProtocol() {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	httpClient := examplePingServer.Client()

	// Use WithTriple() to switch to the Triple protocol
	client := pingv1connect.NewPingServiceClient(
		httpClient,
		examplePingServer.URL(),
		tri.WithTriple(),
	)
	resp := &pingv1.PingResponse{}
	response := tri.NewResponse(resp)
	if err := client.Ping(
		context.Background(),
		tri.NewRequest(&pingv1.PingRequest{Number: 100}),
		response,
	); err != nil {
		logger.Println("error:", err)
		return
	}
	logger.Println("triple response:", resp.Number)

	// Output:
	// triple response: 100
}

// This example demonstrates how to set custom headers on requests.
func Example_clientWithHeaders() {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	httpClient := examplePingServer.Client()

	client := pingv1connect.NewPingServiceClient(
		httpClient,
		examplePingServer.URL(),
	)

	// Create a request and add custom headers
	request := tri.NewRequest(&pingv1.PingRequest{Number: 42})
	request.Header().Set("X-Custom-Header", "custom-value")

	resp := &pingv1.PingResponse{}
	response := tri.NewResponse(resp)
	if err := client.Ping(context.Background(), request, response); err != nil {
		logger.Println("error:", err)
		return
	}
	logger.Println("response:", resp.Number)

	// Output:
	// response: 42
}

// This example demonstrates how to handle errors from the server.
func Example_clientErrorHandling() {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	httpClient := examplePingServer.Client()

	client := pingv1connect.NewPingServiceClient(
		httpClient,
		examplePingServer.URL(),
	)

	// Use the Fail method which always returns an error
	resp := &pingv1.FailResponse{}
	response := tri.NewResponse(resp)
	err := client.Fail(
		context.Background(),
		tri.NewRequest(&pingv1.FailRequest{Code: int32(tri.CodeInvalidArgument)}),
		response,
	)
	if err != nil {
		logger.Println("got expected error")
		return
	}
	logger.Println("unexpected success")

	// Output:
	// got expected error
}

// This example demonstrates using client streaming.
func Example_clientStreaming() {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	httpClient := examplePingServer.Client()

	client := pingv1connect.NewPingServiceClient(
		httpClient,
		examplePingServer.URL(),
	)

	// Start a client streaming call
	stream, err := client.Sum(context.Background())
	if err != nil {
		logger.Println("error creating stream:", err)
		return
	}

	// Send multiple requests
	for i := int64(1); i <= 5; i++ {
		if err := stream.Send(&pingv1.SumRequest{Number: i}); err != nil {
			logger.Println("error sending:", err)
			return
		}
	}

	// Close and receive the response
	resp := &pingv1.SumResponse{}
	response := tri.NewResponse(resp)
	if err := stream.CloseAndReceive(response); err != nil {
		logger.Println("error receiving:", err)
		return
	}
	logger.Println("sum:", resp.Sum)

	// Output:
	// sum: 15
}

// This example demonstrates using server streaming.
func Example_serverStreaming() {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	httpClient := examplePingServer.Client()

	client := pingv1connect.NewPingServiceClient(
		httpClient,
		examplePingServer.URL(),
	)

	// Start a server streaming call
	request := tri.NewRequest(&pingv1.CountUpRequest{Number: 3})
	stream, err := client.CountUp(context.Background(), request)
	if err != nil {
		logger.Println("error:", err)
		return
	}
	defer stream.Close()

	// Receive all responses
	var numbers []int64
	for stream.Receive(&pingv1.CountUpResponse{}) {
		if msg, ok := stream.Msg().(*pingv1.CountUpResponse); ok {
			numbers = append(numbers, msg.Number)
		}
	}
	if err := stream.Err(); err != nil {
		logger.Println("stream error:", err)
		return
	}
	logger.Println("received numbers:", numbers)

	// Output:
	// received numbers: [1 2 3]
}

// This example demonstrates using bidirectional streaming.
func Example_bidiStreaming() {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	httpClient := examplePingServer.Client()

	client := pingv1connect.NewPingServiceClient(
		httpClient,
		examplePingServer.URL(),
	)

	// Start a bidirectional streaming call
	stream, err := client.CumSum(context.Background())
	if err != nil {
		logger.Println("error:", err)
		return
	}

	// Send and receive concurrently
	var results []int64
	for i := int64(1); i <= 3; i++ {
		if err := stream.Send(&pingv1.CumSumRequest{Number: i}); err != nil {
			logger.Println("send error:", err)
			return
		}
		response := &pingv1.CumSumResponse{}
		if err := stream.Receive(response); err != nil {
			logger.Println("receive error:", err)
			return
		}
		results = append(results, response.Sum)
	}

	if err := stream.CloseRequest(); err != nil {
		logger.Println("close request error:", err)
	}
	if err := stream.CloseResponse(); err != nil {
		logger.Println("close response error:", err)
	}

	logger.Println("cumulative sums:", results)

	// Output:
	// cumulative sums: [1 3 6]
}
