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

package triple

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect/assert"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect/proto/connect/ping/v1"
)

func TestClientStreamForClient_NoPanics(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	clientStream := &ClientStreamForClient[pingv1.PingRequest, pingv1.PingResponse]{err: initErr}
	assert.ErrorIs(t, clientStream.Send(&pingv1.PingRequest{}), initErr)
	verifyHeaders(t, clientStream.RequestHeader())
	res, err := clientStream.CloseAndReceive()
	assert.Nil(t, res)
	assert.ErrorIs(t, err, initErr)
	conn, err := clientStream.Conn()
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}

func TestServerStreamForClient_NoPanics(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	serverStream := &ServerStreamForClient[pingv1.PingResponse]{constructErr: initErr}
	assert.ErrorIs(t, serverStream.Err(), initErr)
	assert.ErrorIs(t, serverStream.Close(), initErr)
	assert.NotNil(t, serverStream.Msg())
	assert.False(t, serverStream.Receive())
	verifyHeaders(t, serverStream.ResponseHeader())
	verifyHeaders(t, serverStream.ResponseTrailer())
	conn, err := serverStream.Conn()
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}

func TestServerStreamForClient(t *testing.T) {
	t.Parallel()
	stream := &ServerStreamForClient[pingv1.PingResponse]{conn: &nopStreamingClientConn{}}
	// Ensure that each call to Receive allocates a new message. This helps
	// vtprotobuf, which doesn't automatically zero messages before unmarshaling
	// (see https://github.com/bufbuild/connect-go/issues/345), and it's also
	// less error-prone for users.
	assert.True(t, stream.Receive())
	first := fmt.Sprintf("%p", stream.Msg())
	assert.True(t, stream.Receive())
	second := fmt.Sprintf("%p", stream.Msg())
	assert.NotEqual(t, first, second)
	conn, err := stream.Conn()
	assert.Nil(t, err)
	assert.NotNil(t, conn)
}

func TestBidiStreamForClient_NoPanics(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	bidiStream := &BidiStreamForClient[pingv1.CumSumRequest, pingv1.CumSumResponse]{err: initErr}
	res, err := bidiStream.Receive()
	assert.Nil(t, res)
	assert.ErrorIs(t, err, initErr)
	verifyHeaders(t, bidiStream.RequestHeader())
	verifyHeaders(t, bidiStream.ResponseHeader())
	verifyHeaders(t, bidiStream.ResponseTrailer())
	assert.ErrorIs(t, bidiStream.Send(&pingv1.CumSumRequest{}), initErr)
	assert.ErrorIs(t, bidiStream.CloseRequest(), initErr)
	assert.ErrorIs(t, bidiStream.CloseResponse(), initErr)
	conn, err := bidiStream.Conn()
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}

func verifyHeaders(t *testing.T, headers http.Header) {
	t.Helper()
	assert.Equal(t, headers, http.Header{})

	// Verify set/del don't panic
	headers.Set("a", "b")
	headers.Del("a")
}

type nopStreamingClientConn struct {
	StreamingClientConn
}

func (c *nopStreamingClientConn) Receive(msg any) error {
	return nil
}
