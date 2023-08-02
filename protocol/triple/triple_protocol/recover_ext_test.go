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
//	triple "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/triple"
//	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/triple/assert"
//	"fmt"
//	"net/http"
//	"net/http/httptest"
//	"testing"
//
//	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/triple/proto/triple/ping/v1"
//	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/triple/proto/triple/ping/v1/pingv1triple"
//)
//
//type panicPingServer struct {
//	pingv1triple.UnimplementedPingServiceHandler
//
//	panicWith any
//}
//
//func (s *panicPingServer) Ping(
//	context.Context,
//	*triple.Request[pingv1.PingRequest],
//) (*triple.Response[pingv1.PingResponse], error) {
//	panic(s.panicWith) //nolint:forbidigo
//}
//
//func (s *panicPingServer) CountUp(
//	_ context.Context,
//	_ *triple.Request[pingv1.CountUpRequest],
//	stream *triple.ServerStream[pingv1.CountUpResponse],
//) error {
//	if err := stream.Send(&pingv1.CountUpResponse{}); err != nil {
//		return err
//	}
//	panic(s.panicWith) //nolint:forbidigo
//}
//
//func TestWithRecover(t *testing.T) {
//	t.Parallel()
//	handle := func(_ context.Context, _ triple.Spec, _ http.Header, r any) error {
//		return triple.NewError(triple.CodeFailedPrecondition, fmt.Errorf("panic: %v", r))
//	}
//	assertHandled := func(err error) {
//		t.Helper()
//		assert.NotNil(t, err)
//		assert.Equal(t, triple.CodeOf(err), triple.CodeFailedPrecondition)
//	}
//	assertNotHandled := func(err error) {
//		t.Helper()
//		// When HTTP/2 handlers panic, net/http sends an RST_STREAM frame with code
//		// INTERNAL_ERROR. We should be mapping this back to CodeInternal.
//		assert.Equal(t, triple.CodeOf(err), triple.CodeInternal)
//	}
//	drainStream := func(stream *triple.ServerStreamForClient[pingv1.CountUpResponse]) error {
//		t.Helper()
//		defer stream.Close()
//		assert.True(t, stream.Receive())  // expect one response msg
//		assert.False(t, stream.Receive()) // expect panic before second response msg
//		return stream.Err()
//	}
//	pinger := &panicPingServer{}
//	mux := http.NewServeMux()
//	mux.Handle(pingv1triple.NewPingServiceHandler(pinger, triple.WithRecover(handle)))
//	server := httptest.NewUnstartedServer(mux)
//	server.EnableHTTP2 = true
//	server.StartTLS()
//	defer server.Close()
//	client := pingv1triple.NewPingServiceClient(
//		server.Client(),
//		server.URL,
//	)
//
//	for _, panicWith := range []any{42, nil} {
//		pinger.panicWith = panicWith
//
//		_, err := client.Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{}))
//		assertHandled(err)
//
//		stream, err := client.CountUp(context.Background(), triple.NewRequest(&pingv1.CountUpRequest{}))
//		assert.Nil(t, err)
//		assertHandled(drainStream(stream))
//	}
//
//	pinger.panicWith = http.ErrAbortHandler
//
//	_, err := client.Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{}))
//	assertNotHandled(err)
//
//	stream, err := client.CountUp(context.Background(), triple.NewRequest(&pingv1.CountUpRequest{}))
//	assert.Nil(t, err)
//	assertNotHandled(drainStream(stream))
//}
