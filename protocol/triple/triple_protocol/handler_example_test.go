// // Copyright 2021-2023 Buf Technologies, Inc.
// //
// // Licensed under the Apache License, Version 2.0 (the "License");
// // you may not use this file except in compliance with the License.
// // You may obtain a copy of the License at
// //
// //      http://www.apache.org/licenses/LICENSE-2.0
// //
// // Unless required by applicable law or agreed to in writing, software
// // distributed under the License is distributed on an "AS IS" BASIS,
// // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// // See the License for the specific language governing permissions and
// // limitations under the License.
package triple_protocol_test

//
//import (
//	"context"
//	"net/http"
//
//	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/proto/connect/ping/v1"
//	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/proto/connect/ping/v1/pingv1connect"
//)
//
//// ExamplePingServer implements some trivial business logic. The Protobuf
//// definition for this API is in proto/connect/ping/v1/ping.proto.
//type ExamplePingServer struct {
//	pingv1connect.UnimplementedPingServiceHandler
//}
//
//// Ping implements pingv1connect.PingServiceHandler.
//func (*ExamplePingServer) Ping(
//	_ context.Context,
//	request *pingv1.PingRequest,
//) (*pingv1.PingResponse, error) {
//	return &pingv1.PingResponse{
//		Number: request.Number,
//		Text:   request.Text,
//	}, nil
//}
//
//func Example_handler() {
//	// protoc-gen-connect-go generates constructors that return plain net/http
//	// Handlers, so they're compatible with most Go HTTP routers and middleware
//	// (for example, net/http's StripPrefix). Each handler automatically supports
//	// the Connect, gRPC, and gRPC-Web protocols.
//	mux := http.NewServeMux()
//	mux.Handle(
//		pingv1connect.NewPingServiceHandler(
//			&ExamplePingServer{}, // our business logic
//		),
//	)
//	// You can serve gRPC's health and server reflection APIs using
//	// github.com/bufbuild/connect-grpchealth-go and
//	// github.com/bufbuild/connect-grpcreflect-go.
//	_ = http.ListenAndServeTLS(
//		"localhost:8080",
//		"internal/testdata/server.crt",
//		"internal/testdata/server.key",
//		mux,
//	)
//	// To serve HTTP/2 requests without TLS (as many gRPC clients expect), import
//	// golang.org/x/net/http2/h2c and golang.org/x/net/http2 and change to:
//	// _ = http.ListenAndServe(
//	// 	"localhost:8080",
//	// 	h2c.NewHandler(mux, &http2.Server{}),
//	// )
//}
