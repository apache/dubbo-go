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

package triple_protocol_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1/pingv1connect"
)

// newPingBenchmarkServer starts an HTTP/2 test server serving the PingService
// handler, and returns the server and its HTTP client.
func newPingBenchmarkServer(b *testing.B) (*httptest.Server, *http.Client) {
	b.Helper()
	mux := http.NewServeMux()
	mux.Handle(
		pingv1connect.NewPingServiceHandler(
			&ExamplePingServer{},
		),
	)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	b.Cleanup(server.Close)

	httpClient := server.Client()
	httpTransport, ok := httpClient.Transport.(*http.Transport)
	assert.True(b, ok)
	httpTransport.DisableCompression = true
	return server, httpClient
}

// unaryBenchPayloadSizes covers small (fixed-overhead dominated) through large
// (bandwidth dominated) message sizes.
var unaryBenchPayloadSizes = []int{
	128, 1024, 16 * 1024, 1024 * 1024,
}

// BenchmarkUnaryDuplex measures the production unary path with the fast path
// disabled (the default). The generated client defaults to the gRPC protocol,
// so WithTriple selects the Triple protocol; without WithUnaryFastPath,
// NewConn keeps using duplexHTTPCall for unary RPCs. It is the control group
// for BenchmarkUnaryFastPathProduction.
func BenchmarkUnaryDuplex(b *testing.B) {
	server, httpClient := newPingBenchmarkServer(b)
	client := pingv1connect.NewPingServiceClient(httpClient, server.URL, triple_protocol.WithTriple())
	benchmarkUnaryPing(b, client)
}

// BenchmarkUnaryFastPathProduction measures the production unary fast path via
// the generated client, differing from BenchmarkUnaryDuplex only by the
// WithUnaryFastPath option.
func BenchmarkUnaryFastPathProduction(b *testing.B) {
	server, httpClient := newPingBenchmarkServer(b)
	client := pingv1connect.NewPingServiceClient(
		httpClient,
		server.URL,
		triple_protocol.WithTriple(),
		triple_protocol.WithUnaryFastPath(),
	)
	benchmarkUnaryPing(b, client)
}

// benchmarkUnaryPing runs one parallel sub-benchmark per payload size through
// the generated client.
func benchmarkUnaryPing(b *testing.B, client pingv1connect.PingServiceClient) {
	b.Helper()
	for _, size := range unaryBenchPayloadSizes {
		text := strings.Repeat("a", size)
		b.Run(sizeLabel(size), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					unaryPingIteration(b, client, text)
				}
			})
		})
	}
}

// unaryPingIteration performs one unary Ping call through the generated client.
func unaryPingIteration(b *testing.B, client pingv1connect.PingServiceClient, text string) {
	b.Helper()
	req := pingv1.PingRequest{Text: text}
	res := pingv1.PingResponse{}
	if err := client.Ping(
		context.Background(),
		triple_protocol.NewRequest(&req),
		triple_protocol.NewResponse(&res),
	); err != nil {
		b.Fatalf("ping: %v", err)
	}
}

func sizeLabel(size int) string {
	switch size {
	case 1024 * 1024:
		return "1MiB"
	default:
		return fmt.Sprintf("%dB", size)
	}
}
