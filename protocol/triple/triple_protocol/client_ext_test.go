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
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	triple "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1/pingv1connect"
)

func TestNewClient_InitFailure(t *testing.T) {
	t.Parallel()
	client := pingv1connect.NewPingServiceClient(
		http.DefaultClient,
		"http://127.0.0.1:8080",
		// This triggers an error during initialization, so each call will short circuit returning an error.
		triple.WithSendCompression("invalid"),
	)
	validateExpectedError := func(t *testing.T, err error) {
		t.Helper()
		assert.NotNil(t, err)
		var tripleErr *triple.Error
		assert.True(t, errors.As(err, &tripleErr))
		assert.Equal(t, tripleErr.Message(), `unknown compression "invalid"`)
	}

	t.Run("unary", func(t *testing.T) {
		t.Parallel()
		err := client.Ping(context.Background(), triple.NewRequest(&pingv1.PingRequest{}), triple.NewResponse(&pingv1.PingResponse{}))
		validateExpectedError(t, err)
	})

	t.Run("bidi", func(t *testing.T) {
		t.Parallel()
		bidiStream, err := client.CumSum(context.Background())
		validateExpectedError(t, err)
		err = bidiStream.Send(&pingv1.CumSumRequest{})
		validateExpectedError(t, err)
	})

	t.Run("client_stream", func(t *testing.T) {
		t.Parallel()
		clientStream, err := client.Sum(context.Background())
		validateExpectedError(t, err)
		err = clientStream.Send(&pingv1.SumRequest{})
		validateExpectedError(t, err)
	})

	t.Run("server_stream", func(t *testing.T) {
		t.Parallel()
		_, err := client.CountUp(context.Background(), triple.NewRequest(&pingv1.CountUpRequest{Number: 3}))
		validateExpectedError(t, err)
	})
}

func TestClientPeer(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	run := func(t *testing.T, streamFlag bool, opts ...triple.ClientOption) {
		t.Helper()
		client := pingv1connect.NewPingServiceClient(
			server.Client(),
			server.URL,
			triple.WithClientOptions(opts...),
			triple.WithInterceptors(&assertPeerInterceptor{t}),
		)
		ctx := context.Background()
		// unary
		err := client.Ping(ctx, triple.NewRequest(&pingv1.PingRequest{}), triple.NewResponse(&pingv1.PingResponse{}))
		assert.Nil(t, err)
		text := strings.Repeat(".", 256)
		resp := &pingv1.PingResponse{}
		err = client.Ping(ctx, triple.NewRequest(&pingv1.PingRequest{Text: text}), triple.NewResponse(resp))
		assert.Nil(t, err)
		assert.Equal(t, resp.Text, text)
		// protocol does not support stream just need to test unary
		if !streamFlag {
			return
		}

		//client streaming
		clientStream, err := client.Sum(ctx)
		assert.Nil(t, err)
		t.Cleanup(func() {
			closeErr := clientStream.CloseAndReceive(triple.NewResponse(&pingv1.SumResponse{}))
			assert.Nil(t, closeErr)
		})
		assert.NotZero(t, clientStream.Peer().Addr)
		assert.NotZero(t, clientStream.Peer().Protocol)
		err = clientStream.Send(&pingv1.SumRequest{})
		assert.Nil(t, err)
		//server streaming
		serverStream, err := client.CountUp(ctx, triple.NewRequest(&pingv1.CountUpRequest{}))
		t.Cleanup(func() {
			assert.Nil(t, serverStream.Close())
		})
		assert.Nil(t, err)
		// bidi streaming
		bidiStream, err := client.CumSum(ctx)
		assert.Nil(t, err)
		t.Cleanup(func() {
			assert.Nil(t, bidiStream.CloseRequest())
			assert.Nil(t, bidiStream.CloseResponse())
		})
		assert.NotZero(t, bidiStream.Peer().Addr)
		assert.NotZero(t, bidiStream.Peer().Protocol)
		err = bidiStream.Send(&pingv1.CumSumRequest{})
		assert.Nil(t, err)
	}

	t.Run("triple", func(t *testing.T) {
		t.Parallel()
		run(t /*streamFlag*/, false, triple.WithTriple())
	})
	t.Run("grpc", func(t *testing.T) {
		t.Parallel()
		run(t, true)
	})
}

type assertPeerInterceptor struct {
	tb testing.TB
}

func (a *assertPeerInterceptor) WrapUnaryHandler(next triple.UnaryHandlerFunc) triple.UnaryHandlerFunc {
	return func(ctx context.Context, req triple.AnyRequest) (triple.AnyResponse, error) {
		assert.NotZero(a.tb, req.Peer().Addr)
		assert.NotZero(a.tb, req.Peer().Protocol)
		return next(ctx, req)
	}
}

func (a *assertPeerInterceptor) WrapUnary(next triple.UnaryFunc) triple.UnaryFunc {
	return func(ctx context.Context, req triple.AnyRequest, res triple.AnyResponse) error {
		assert.NotZero(a.tb, req.Peer().Addr)
		assert.NotZero(a.tb, req.Peer().Protocol)
		return next(ctx, req, res)
	}
}

func (a *assertPeerInterceptor) WrapStreamingClient(next triple.StreamingClientFunc) triple.StreamingClientFunc {
	return func(ctx context.Context, spec triple.Spec) triple.StreamingClientConn {
		conn := next(ctx, spec)
		assert.NotZero(a.tb, conn.Peer().Addr)
		assert.NotZero(a.tb, conn.Peer().Protocol)
		assert.NotZero(a.tb, conn.Spec())
		return conn
	}
}

func (a *assertPeerInterceptor) WrapStreamingHandler(next triple.StreamingHandlerFunc) triple.StreamingHandlerFunc {
	return func(ctx context.Context, conn triple.StreamingHandlerConn) error {
		assert.NotZero(a.tb, conn.Peer().Addr)
		assert.NotZero(a.tb, conn.Peer().Protocol)
		assert.NotZero(a.tb, conn.Spec())
		return next(ctx, conn)
	}
}
