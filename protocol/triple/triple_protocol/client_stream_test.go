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
	"io"
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

func (c *nopStreamingClientConn) Receive(msg any) error {
	return nil
}

// stubStreamingClientConn is a configurable StreamingClientConn used to
// exercise close paths, error propagation and empty metadata without
// involving a real transport.
type stubStreamingClientConn struct {
	StreamingClientConn

	sendErr      error
	receiveErr   error
	closeReqErr  error
	closeRespErr error
	requestHdr   http.Header
	responseHdr  http.Header
	responseTrlr http.Header

	sendCalls      int
	receiveCalls   int
	closeReqCalls  int
	closeRespCalls int
}

func (c *stubStreamingClientConn) Send(any) error {
	c.sendCalls++
	return c.sendErr
}

func (c *stubStreamingClientConn) RequestHeader() http.Header {
	return c.requestHdr
}

func (c *stubStreamingClientConn) CloseRequest() error {
	c.closeReqCalls++
	return c.closeReqErr
}

func (c *stubStreamingClientConn) Receive(any) error {
	c.receiveCalls++
	return c.receiveErr
}

func (c *stubStreamingClientConn) ResponseHeader() http.Header {
	return c.responseHdr
}

func (c *stubStreamingClientConn) ResponseTrailer() http.Header {
	return c.responseTrlr
}

func (c *stubStreamingClientConn) CloseResponse() error {
	c.closeRespCalls++
	return c.closeRespErr
}

// TestServerStreamForClientClose verifies that Close delegates to the
// underlying connection's CloseResponse and returns its result.
func TestServerStreamForClientClose(t *testing.T) {
	t.Parallel()
	conn := &stubStreamingClientConn{}
	stream := &ServerStreamForClient{conn: conn}

	assert.Nil(t, stream.Close())
	assert.Equal(t, conn.closeRespCalls, 1)
}

// TestServerStreamForClientCloseRepeated verifies that calling Close more
// than once keeps delegating to the underlying connection.
func TestServerStreamForClientCloseRepeated(t *testing.T) {
	t.Parallel()
	conn := &stubStreamingClientConn{}
	stream := &ServerStreamForClient{conn: conn}

	assert.Nil(t, stream.Close())
	assert.Nil(t, stream.Close())
	assert.Equal(t, conn.closeRespCalls, 2)
}

// TestServerStreamForClientErrPropagation verifies that Err returns the
// first non-EOF error encountered by Receive and returns nil for a normal
// end of stream.
func TestServerStreamForClientErrPropagation(t *testing.T) {
	t.Parallel()
	receiveErr := errors.New("receive failure")
	conn := &stubStreamingClientConn{receiveErr: receiveErr}
	stream := &ServerStreamForClient{conn: conn}

	// A non-EOF error is returned by Err.
	assert.False(t, stream.Receive(NewResponse(&pingv1.PingResponse{})))
	assert.ErrorIs(t, stream.Err(), receiveErr)

	// EOF is treated as a normal end of stream and Err returns nil.
	connEOF := &stubStreamingClientConn{receiveErr: io.EOF}
	streamEOF := &ServerStreamForClient{conn: connEOF}
	assert.False(t, streamEOF.Receive(NewResponse(&pingv1.PingResponse{})))
	assert.Nil(t, streamEOF.Err())
}

// TestServerStreamForClientReceiveStopsAfterError verifies that once Receive
// has recorded an error, subsequent calls return false without advancing the
// underlying connection.
func TestServerStreamForClientReceiveStopsAfterError(t *testing.T) {
	t.Parallel()
	conn := &stubStreamingClientConn{receiveErr: errors.New("receive failure")}
	stream := &ServerStreamForClient{conn: conn}

	assert.False(t, stream.Receive(NewResponse(&pingv1.PingResponse{})))
	assert.False(t, stream.Receive(NewResponse(&pingv1.PingResponse{})))
	assert.Equal(t, conn.receiveCalls, 1)
}

// TestServerStreamForClientEmptyMetadata verifies that ResponseHeader and
// ResponseTrailer return empty headers.
func TestServerStreamForClientEmptyMetadata(t *testing.T) {
	t.Parallel()
	conn := &stubStreamingClientConn{responseHdr: http.Header{}, responseTrlr: http.Header{}}
	stream := &ServerStreamForClient{conn: conn}

	assert.Equal(t, len(stream.ResponseHeader()), 0)
	assert.Equal(t, len(stream.ResponseTrailer()), 0)
}

// TestBidiStreamForClientReceiveErrorPropagation verifies that Receive
// returns the underlying connection error unchanged.
func TestBidiStreamForClientReceiveErrorPropagation(t *testing.T) {
	t.Parallel()
	receiveErr := errors.New("receive failure")
	conn := &stubStreamingClientConn{receiveErr: receiveErr}
	stream := &BidiStreamForClient{conn: conn}

	err := stream.Receive(NewResponse(&pingv1.PingResponse{}))
	assert.ErrorIs(t, err, receiveErr)
}

// TestBidiStreamForClientRequestHeaderEmpty verifies that RequestHeader
// returns an empty header.
func TestBidiStreamForClientRequestHeaderEmpty(t *testing.T) {
	t.Parallel()
	conn := &stubStreamingClientConn{requestHdr: http.Header{}}
	stream := &BidiStreamForClient{conn: conn}

	assert.Equal(t, len(stream.RequestHeader()), 0)
}

// TestBidiStreamForClientCloseRequestRepeated verifies that calling
// CloseRequest more than once keeps delegating to the underlying connection.
func TestBidiStreamForClientCloseRequestRepeated(t *testing.T) {
	t.Parallel()
	conn := &stubStreamingClientConn{}
	stream := &BidiStreamForClient{conn: conn}

	assert.Nil(t, stream.CloseRequest())
	assert.Nil(t, stream.CloseRequest())
	assert.Equal(t, conn.closeReqCalls, 2)
}

// TestBidiStreamForClientCloseResponseRepeated verifies that calling
// CloseResponse more than once keeps delegating to the underlying connection.
func TestBidiStreamForClientCloseResponseRepeated(t *testing.T) {
	t.Parallel()
	conn := &stubStreamingClientConn{}
	stream := &BidiStreamForClient{conn: conn}

	assert.Nil(t, stream.CloseResponse())
	assert.Nil(t, stream.CloseResponse())
	assert.Equal(t, conn.closeRespCalls, 2)
}

// TestClientStreamForClientCloseAndReceiveClosesResponseOnCloseRequestError
// verifies that CloseAndReceive closes the receive side before returning
// the error when closing the send side fails.
func TestClientStreamForClientCloseAndReceiveClosesResponseOnCloseRequestError(t *testing.T) {
	t.Parallel()
	closeReqErr := errors.New("close request failure")
	conn := &stubStreamingClientConn{closeReqErr: closeReqErr}
	stream := &ClientStreamForClient{conn: conn}

	err := stream.CloseAndReceive(NewResponse(&pingv1.PingResponse{}))
	assert.ErrorIs(t, err, closeReqErr)
	assert.Equal(t, conn.closeRespCalls, 1)
}

// TestClientStreamForClientCloseAndReceiveClosesResponseOnReceiveError
// verifies that CloseAndReceive closes the receive side before returning
// the error when receiving the response fails.
func TestClientStreamForClientCloseAndReceiveClosesResponseOnReceiveError(t *testing.T) {
	t.Parallel()
	receiveErr := errors.New("receive failure")
	conn := &stubStreamingClientConn{receiveErr: receiveErr}
	stream := &ClientStreamForClient{conn: conn}

	err := stream.CloseAndReceive(NewResponse(&pingv1.PingResponse{}))
	assert.ErrorIs(t, err, receiveErr)
	assert.Equal(t, conn.closeReqCalls, 1)
	assert.Equal(t, conn.closeRespCalls, 1)
}
