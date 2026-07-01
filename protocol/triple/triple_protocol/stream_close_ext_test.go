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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

import (
	triple "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1/pingv1connect"
)

const closeTestTimeout = time.Second

// TestServerStreamCloseDoesNotDrainResponse covers the public
// ServerStreamForClient.Close path for connect-go#791-style lifecycle
// behavior. The server sends one response and then intentionally keeps the
// stream open; client-side Close should still return promptly because it is a
// cleanup operation, not a request to read the stream to EOF.
func TestServerStreamCloseDoesNotDrainResponse(t *testing.T) {
	release, unblock := newCloseRelease()
	t.Cleanup(unblock)
	sent := make(chan struct{})
	client := newCloseLifecyclePingClient(t, &pluggablePingServer{
		countUp: func(ctx context.Context, req *triple.Request, stream *triple.ServerStream) error {
			if err := stream.Send(&pingv1.CountUpResponse{Number: 1}); err != nil {
				return err
			}
			close(sent)
			<-release
			return nil
		},
	})

	stream, err := client.CountUp(context.Background(), triple.NewRequest(&pingv1.CountUpRequest{}))
	assert.Nil(t, err)

	select {
	case <-sent:
	case <-time.After(closeTestTimeout):
		t.Fatal("server did not send the first response before close verification")
	}

	if !stream.Receive(&pingv1.CountUpResponse{}) {
		t.Fatalf("failed to receive first response before close verification: %v", stream.Err())
	}
	msg := stream.Msg().(*pingv1.CountUpResponse)
	assert.Equal(t, msg.Number, int64(1))

	done := make(chan error, 1)
	go func() {
		done <- stream.Close()
	}()
	assert.Nil(t, assertCloseReturnsPromptly(t, done, unblock, "ServerStreamForClient.Close blocked while draining the response"))
}

// TestBidiStreamCloseResponseDoesNotDrainResponse covers the public
// BidiStreamForClient.CloseResponse path. After one successful receive, the
// server keeps the response side open; closing the receive side should close
// the HTTP response body instead of draining it.
func TestBidiStreamCloseResponseDoesNotDrainResponse(t *testing.T) {
	release, unblock := newCloseRelease()
	t.Cleanup(unblock)
	received := make(chan struct{})
	sent := make(chan struct{})
	client := newCloseLifecyclePingClient(t, &pluggablePingServer{
		cumSum: func(ctx context.Context, stream *triple.BidiStream) error {
			req := &pingv1.CumSumRequest{}
			if err := stream.Receive(req); err != nil {
				return err
			}
			close(received)
			if err := stream.Send(&pingv1.CumSumResponse{Sum: req.Number}); err != nil {
				return err
			}
			close(sent)
			<-release
			return nil
		},
	})

	stream, err := client.CumSum(context.Background())
	assert.Nil(t, err)
	if err := stream.Send(&pingv1.CumSumRequest{Number: 2}); err != nil {
		t.Fatalf("failed to send setup request before close verification: %v", err)
	}

	select {
	case <-received:
	case <-time.After(closeTestTimeout):
		t.Fatal("server did not receive the setup request before close verification")
	}
	select {
	case <-sent:
	case <-time.After(closeTestTimeout):
		t.Fatal("server did not send the setup response before close verification")
	}

	res := &pingv1.CumSumResponse{}
	if err := stream.Receive(res); err != nil {
		t.Fatalf("failed to receive setup response before close verification: %v", err)
	}
	assert.Equal(t, res.Sum, int64(2))

	done := make(chan error, 1)
	go func() {
		done <- stream.CloseResponse()
	}()
	assert.Nil(t, assertCloseReturnsPromptly(t, done, unblock, "BidiStreamForClient.CloseResponse blocked while draining the response"))
	assert.Nil(t, stream.CloseRequest())
}

// TestBidiStreamCloseResponseAfterServerStopsReading covers the case where the
// server returns before consuming the rest of the request stream. Once the
// client has received the server-side error, closing the receive side should be
// a local cleanup step and must not wait for additional response bytes.
func TestBidiStreamCloseResponseAfterServerStopsReading(t *testing.T) {
	serverReceived := make(chan struct{})
	serverReturn := make(chan struct{})
	client := newCloseLifecyclePingClient(t, &pluggablePingServer{
		cumSum: func(ctx context.Context, stream *triple.BidiStream) error {
			req := &pingv1.CumSumRequest{}
			if err := stream.Receive(req); err != nil {
				return err
			}
			close(serverReceived)
			<-serverReturn
			return triple.NewError(triple.CodeUnavailable, errors.New("server stopped reading"))
		},
	})

	stream, err := client.CumSum(context.Background())
	assert.Nil(t, err)
	if err := stream.Send(&pingv1.CumSumRequest{Number: 1}); err != nil {
		t.Fatalf("failed to send setup request before close verification: %v", err)
	}

	select {
	case <-serverReceived:
	case <-time.After(closeTestTimeout):
		t.Fatal("server did not receive the setup request before close verification")
	}
	close(serverReturn)

	err = stream.Receive(&pingv1.CumSumResponse{})
	assert.NotNil(t, err)
	assert.Equal(t, triple.CodeOf(err), triple.CodeUnavailable)

	done := make(chan error, 1)
	go func() {
		done <- stream.CloseResponse()
	}()
	assert.Nil(t, assertCloseReturnsPromptly(t, done, nil, "BidiStreamForClient.CloseResponse blocked after server stopped reading"))
	assert.Nil(t, stream.CloseRequest())
}

// TestClientStreamCloseAndReceiveAfterServerReturnsError covers a client-stream
// RPC where the server returns an error before the client finishes sending all
// messages. CloseAndReceive should surface the server error and finish cleanup
// without hanging in CloseResponse.
func TestClientStreamCloseAndReceiveAfterServerReturnsError(t *testing.T) {
	client := newCloseLifecyclePingClient(t, &pluggablePingServer{
		sum: func(ctx context.Context, stream *triple.ClientStream) (*triple.Response, error) {
			return nil, triple.NewError(triple.CodeUnavailable, errors.New("server returned early"))
		},
	})

	stream, err := client.Sum(context.Background())
	assert.Nil(t, err)
	if err = stream.Send(&pingv1.SumRequest{Number: 1}); err != nil {
		assert.ErrorIs(t, err, io.EOF)
	}

	done := make(chan error, 1)
	go func() {
		done <- stream.CloseAndReceive(triple.NewResponse(&pingv1.SumResponse{}))
	}()
	err = assertCloseReturnsPromptly(t, done, nil, "ClientStreamForClient.CloseAndReceive blocked after server returned early")
	assert.NotNil(t, err)
	assert.Equal(t, triple.CodeOf(err), triple.CodeUnavailable)
}

func newCloseLifecyclePingClient(t *testing.T, pingServer pingv1connect.PingServiceHandler) pingv1connect.PingServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	return pingv1connect.NewPingServiceClient(server.Client(), server.URL)
}

// newCloseRelease returns a channel that lets tests keep the server handler
// alive until the client-side close call has completed. The unblock function is
// idempotent so cleanup can safely release the handler after failures.
func newCloseRelease() (<-chan struct{}, func()) {
	release := make(chan struct{})
	var once sync.Once
	return release, func() {
		once.Do(func() {
			close(release)
		})
	}
}

// assertCloseReturnsPromptly treats a timeout as evidence that close attempted
// to drain the response stream. On failure it releases the server first so the
// goroutine running the close call can finish before the test exits.
func assertCloseReturnsPromptly(t *testing.T, done <-chan error, unblock func(), failure string) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(closeTestTimeout):
		if unblock != nil {
			unblock()
			err := <-done
			assert.Nil(t, err)
		}
		t.Fatal(failure)
		return nil
	}
}
