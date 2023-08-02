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
package triple_protocol

//
//import (
//	"context"
//	"net/http"
//	"net/http/httptest"
//	"strings"
//	"testing"
//
//	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect/assert"
//	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect/proto/connect/ping/v1"
//)
//
//func TestClientUnaryGetFallback(t *testing.T) {
//	t.Parallel()
//	mux := http.NewServeMux()
//	mux.Handle("/connect.ping.v1.PingService/Ping", NewUnaryHandler(
//		"/connect.ping.v1.PingService/Ping",
//		func(ctx context.Context, r *Request[pingv1.PingRequest]) (*Response[pingv1.PingResponse], error) {
//			return NewResponse(&pingv1.PingResponse{
//				Number: r.Msg.Number,
//				Text:   r.Msg.Text,
//			}), nil
//		},
//		WithIdempotency(IdempotencyNoSideEffects),
//	))
//	server := httptest.NewUnstartedServer(mux)
//	server.EnableHTTP2 = true
//	server.StartTLS()
//	t.Cleanup(server.Close)
//
//	client := NewClient[pingv1.PingRequest, pingv1.PingResponse](
//		server.Client(),
//		server.URL+"/connect.ping.v1.PingService/Ping",
//		WithHTTPGet(),
//		withHTTPGetMaxURLSize(1, true),
//		WithSendGzip(),
//	)
//	ctx := context.Background()
//
//	_, err := client.CallUnary(ctx, NewRequest[pingv1.PingRequest](nil))
//	assert.Nil(t, err)
//
//	text := strings.Repeat(".", 256)
//	r, err := client.CallUnary(ctx, NewRequest(&pingv1.PingRequest{Text: text}))
//	assert.Nil(t, err)
//	assert.Equal(t, r.Msg.Text, text)
//}
