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

package triple_protocol

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
)

func TestClientStreamForClient_NoPanics(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	clientStream := &ClientStreamForClient{err: initErr}
	assert.ErrorIs(t, clientStream.Send(&pingv1.PingRequest{}), initErr)
	verifyHeaders(t, clientStream.RequestHeader())
	res := NewResponse(&pingv1.PingResponse{})
	err := clientStream.CloseAndReceive(res)
	assert.ErrorIs(t, err, initErr)
	conn, err := clientStream.Conn()
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}

func TestServerStreamForClient_NoPanics(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	serverStream := &ServerStreamForClient{constructErr: initErr}
	assert.ErrorIs(t, serverStream.Err(), initErr)
	assert.ErrorIs(t, serverStream.Close(), initErr)
	assert.Nil(t, serverStream.Msg())
	assert.False(t, serverStream.Receive(NewResponse(&pingv1.PingResponse{})))
	verifyHeaders(t, serverStream.ResponseHeader())
	verifyHeaders(t, serverStream.ResponseTrailer())
	conn, err := serverStream.Conn()
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}

func TestServerStreamForClient(t *testing.T) {
	t.Parallel()
	stream := &ServerStreamForClient{conn: &nopStreamingClientConn{}}
	res := NewResponse(&pingv1.PingResponse{})
	assert.True(t, stream.Receive(res))
	first := fmt.Sprintf("%p", stream.Msg())
	res1 := NewResponse(&pingv1.PingResponse{})
	assert.True(t, stream.Receive(res1))
	second := fmt.Sprintf("%p", stream.Msg())
	assert.NotEqual(t, first, second)
	conn, err := stream.Conn()
	assert.Nil(t, err)
	assert.NotNil(t, conn)
}

func TestBidiStreamForClient_NoPanics(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	bidiStream := &BidiStreamForClient{err: initErr}
	res := NewResponse(&pingv1.CumSumResponse{})
	err := bidiStream.Receive(res)
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

func (c *nopStreamingClientConn) Receive(msg interface{}) error {
	return nil
}
