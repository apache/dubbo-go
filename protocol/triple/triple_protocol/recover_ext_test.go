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

package triple_protocol_test

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
	pingv1triple "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1/pingv1connect"
)

type panicPingServer struct {
	pingv1triple.UnimplementedPingServiceHandler
	panicWith any
}

func (s *panicPingServer) Ping(
	context.Context,
	*triple_protocol.Request,
) (*triple_protocol.Response, error) {
	panic(s.panicWith) //nolint:forbidigo
}

func (s *panicPingServer) CountUp(
	_ context.Context,
	_ *triple_protocol.Request,
	stream *triple_protocol.ServerStream,
) error {
	if err := stream.Send(&pingv1.CountUpResponse{}); err != nil {
		return err
	}
	panic(s.panicWith) //nolint:forbidigo
}

//func TestWithRecover(t *testing.T) {
//	t.Parallel()
//	handle := func(_ context.Context, _ triple_protocol.Spec, _ http.Header, r any) error {
//		return triple_protocol.NewError(triple_protocol.CodeFailedPrecondition, fmt.Errorf("panic: %v", r))
//	}
//	assertHandled := func(err error) {
//		t.Helper()
//		assert.NotNil(t, err)
//		assert.Equal(t, triple_protocol.CodeOf(err), triple_protocol.CodeFailedPrecondition)
//	}
//	assertNotHandled := func(err error) {
//		t.Helper()
//		// When HTTP/2 handlers panic, net/http sends an RST_STREAM frame with code
//		// INTERNAL_ERROR. We should be mapping this back to CodeInternal.
//		assert.Equal(t, triple_protocol.CodeOf(err), triple_protocol.CodeInternal)
//	}
//	drainStream := func(stream *triple_protocol.ServerStreamForClient) error {
//		t.Helper()
//		defer stream.Close()
//		assert.True(t, stream.Receive(&pingv1.CountUpResponse{}))  // expect one response msg
//		assert.False(t, stream.Receive(&pingv1.CountUpResponse{})) // expect panic before second response msg
//		return stream.Err()
//	}
//	pinger := &panicPingServer{}
//	mux := http.NewServeMux()
//	mux.Handle(pingv1triple.NewPingServiceHandler(pinger, triple_protocol.WithRecover(handle)))
//	server := httptest.NewUnstartedServer(mux)
//	server.EnableHTTP2 = true
//	server.StartTLS()
//	defer server.Close()
//	//client := pingv1triple.NewPingServiceClient(
//	//	server.Client(),
//	//	server.URL,
//	//)
//
//	//for _, panicWith := range []any{42, nil} {
//	//	pinger.panicWith = panicWith
//	//
//	//	err := client.Ping(context.Background(), triple_protocol.NewRequest(&pingv1.PingRequest{}), triple_protocol.NewResponse(&pingv1.PingResponse{}))
//	//	assertHandled(err)
//	//
//	//	stream, err := client.CountUp(context.Background(), triple_protocol.NewRequest(&pingv1.CountUpRequest{}))
//	//	assert.Nil(t, err)
//	//	assertHandled(drainStream(stream))
//	//}
//	//
//	//pinger.panicWith = http.ErrAbortHandler
//	//
//	//err := client.Ping(context.Background(), triple_protocol.NewRequest(&pingv1.PingRequest{}), triple_protocol.NewResponse(&pingv1.PingResponse{}))
//	//assertNotHandled(err)
//	//
//	//stream, err := client.CountUp(context.Background(), triple_protocol.NewRequest(&pingv1.CountUpRequest{}))
//	//assert.Nil(t, err)
//	//assertNotHandled(drainStream(stream))
//}
