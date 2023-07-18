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

package triple_test

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/triple"
	"log"
	"net/http"
	"os"

	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect/proto/connect/ping/v1"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect/proto/connect/ping/v1/pingv1connect"
)

func Example_client() {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	// Unfortunately, pkg.go.dev can't run examples that actually use the
	// network. To keep this example runnable, we'll use an HTTP server and
	// client that communicate over in-memory pipes. The client is still a plain
	// *http.Client!
	var httpClient *http.Client = examplePingServer.Client()

	// By default, clients use the Connect protocol. Add connect.WithGRPC() or
	// connect.WithGRPCWeb() to switch protocols.
	client := pingv1connect.NewPingServiceClient(
		httpClient,
		examplePingServer.URL(),
	)
	response, err := client.Ping(
		context.Background(),
		triple.NewRequest(&pingv1.PingRequest{Number: 42}),
	)
	if err != nil {
		logger.Println("error:", err)
		return
	}
	logger.Println("response content-type:", response.Header().Get("Content-Type"))
	logger.Println("response message:", response.Msg)

	// Output:
	// response content-type: application/proto
	// response message: number:42
}
