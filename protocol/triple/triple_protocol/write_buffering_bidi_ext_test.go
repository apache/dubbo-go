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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

import (
	triple "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1/pingv1connect"
)

// bidiReplyTimeout bounds how long one ping-pong step may wait for its reply.
// Receive has no context escape hatch: it parks on the call's responseReady
// channel, which is only closed once the HTTP request has been issued, so a
// ctx deadline cannot break the wait. The watchdog therefore has to be the
// test's own deadline, generous enough to absorb a TLS and HTTP/2 handshake on
// a loaded machine.
const bidiReplyTimeout = 5 * time.Second

// newBufferedPingClient starts an HTTP/2 server and returns a client with write
// buffering enabled. Buffering is on by default, so no option is needed here.
// Bidi streams need full-duplex HTTP/2: the handler rejects HTTP/1.x outright
// rather than letting the client hang on a half-closed connection.
func newBufferedPingClient(t *testing.T, opts ...triple.ClientOption) pingv1connect.PingServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(pingv1connect.NewPingServiceHandler(pingServer{}))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	return pingv1connect.NewPingServiceClient(
		server.Client(),
		server.URL,
		triple.WithClientOptions(opts...),
	)
}

// TestWriteBufferingBidiPingPong verifies the send-reply-send-reply pattern on a
// bidi stream whose messages all stay below the write buffer watermark.
//
// This is a regression guard for a deadlock that write buffering introduced: a
// buffered Send returns nil without touching the wire, so nothing issued the
// HTTP request, while Receive waited on the response of a request that had
// never been sent. The request only reaches the server once something flushes
// the buffer, and here that something must be Receive itself, since CloseRequest
// legally comes after the last reply in this pattern.
//
// Only the gRPC wire is exercised: the Triple wire's server side does not
// register a handler for streaming procedures, so a triple-wire bidi call is
// rejected regardless of buffering.
func TestWriteBufferingBidiPingPong(t *testing.T) {
	t.Parallel()
	client := newBufferedPingClient(t)

	ctx := t.Context()
	stream, err := client.CumSum(ctx)
	assert.Nil(t, err)
	defer func() {
		// Releasing the request side also unblocks a Receive left waiting by a
		// failed step, so no goroutine outlives the test.
		_ = stream.CloseRequest()
		_ = stream.CloseResponse()
	}()

	var sum int64
	for i, number := range []int64{3, 5, 1} {
		assert.Nil(t, stream.Send(&pingv1.CumSumRequest{Number: number}))
		sum += number
		reply := &pingv1.CumSumResponse{}
		received := make(chan error, 1)
		go func() {
			received <- stream.Receive(reply)
		}()
		select {
		case err := <-received:
			assert.Nil(t, err)
			assert.Equal(t, reply.Sum, sum)
		case <-time.After(bidiReplyTimeout):
			t.Fatalf("Receive #%d blocked: every Send stayed below the write buffer watermark, so the buffered request was never flushed and the server never saw it", i)
		}
	}
}

// TestWriteBufferingBidiWaitEntries verifies that every entry point that awaits
// the response side of a buffered bidi stream flushes the write buffer first.
//
// All four entries share one failure mode: a buffered Send stages bytes locally
// and returns nil without issuing the HTTP request, so an entry that blocks on
// the response before anything flushed the buffer waits on a request that was
// never sent. Each entry runs on its own stream, so a hang in one cannot mask a
// missing flush in another. TestWriteBufferingBidiPingPong covers the
// interleaved send-reply pattern on top of this.
//
// Only the gRPC wire is exercised: the Triple wire's server side does not
// register a handler for streaming procedures, so a triple-wire bidi call is
// rejected regardless of buffering.
func TestWriteBufferingBidiWaitEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		wait func(*triple.BidiStreamForClient) error
	}{
		{
			name: "Receive",
			wait: func(stream *triple.BidiStreamForClient) error {
				return stream.Receive(&pingv1.CumSumResponse{})
			},
		},
		{
			name: "ResponseHeader",
			wait: func(stream *triple.BidiStreamForClient) error {
				_ = stream.ResponseHeader()
				return nil
			},
		},
		{
			name: "ResponseTrailer",
			wait: func(stream *triple.BidiStreamForClient) error {
				_ = stream.ResponseTrailer()
				return nil
			},
		},
		{
			name: "CloseResponse",
			wait: func(stream *triple.BidiStreamForClient) error {
				return stream.CloseResponse()
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client := newBufferedPingClient(t)
			ctx := t.Context()
			stream, err := client.CumSum(ctx)
			assert.Nil(t, err)
			defer func() {
				// Releasing the request side also unblocks an entry left waiting
				// by a failed step, so no goroutine outlives the test.
				_ = stream.CloseRequest()
				_ = stream.CloseResponse()
			}()

			// The message stays far below the write buffer watermark, so the entry
			// point under test is the only thing that can put it on the wire.
			assert.Nil(t, stream.Send(&pingv1.CumSumRequest{Number: 3}))

			waited := make(chan error, 1)
			go func() {
				waited <- test.wait(stream)
			}()
			select {
			case err := <-waited:
				assert.Nil(t, err)
			case <-time.After(bidiReplyTimeout):
				t.Fatalf("%s blocked: the buffered Send stayed below the watermark and nothing flushed it before the response was awaited", test.name)
			}
		})
	}
}

// TestWriteBufferingTripleWireUnary verifies that a buffered call completes end
// to end over the Triple wire, where unary is the only supported procedure type
// and the unary fast path must be switched off for the buffered duplex path to
// be reached. The request body is far below the watermark, so the whole request
// rides in the buffer until the request side closes.
func TestWriteBufferingTripleWireUnary(t *testing.T) {
	t.Parallel()
	client := newBufferedPingClient(t, triple.WithTriple(), triple.WithoutUnaryFastPath())

	const text = "buffered over the triple wire"
	ctx, cancel := context.WithTimeout(context.Background(), bidiReplyTimeout)
	defer cancel()
	resp := &pingv1.PingResponse{}
	err := client.Ping(
		ctx,
		triple.NewRequest(&pingv1.PingRequest{Number: 42, Text: text}),
		triple.NewResponse(resp),
	)
	assert.Nil(t, err)
	assert.Equal(t, resp.Number, int64(42))
	assert.Equal(t, resp.Text, text)
}
