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
	"io"
	"net/http"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
)

// stubHandlerConn is a configurable StreamingHandlerConn used to exercise
// receive stops, error propagation and empty metadata without involving a
// real transport.
type stubHandlerConn struct {
	receiveErr error
	sendErr    error
	exportable http.Header

	receiveCalls int
	sent         []any
}

func (c *stubHandlerConn) Spec() Spec                    { return Spec{} }
func (c *stubHandlerConn) Peer() Peer                    { return Peer{} }
func (c *stubHandlerConn) Receive(any) error             { c.receiveCalls++; return c.receiveErr }
func (c *stubHandlerConn) RequestHeader() http.Header    { return nil }
func (c *stubHandlerConn) ExportableHeader() http.Header { return c.exportable }
func (c *stubHandlerConn) Send(msg any) error {
	c.sent = append(c.sent, msg)
	return c.sendErr
}
func (c *stubHandlerConn) ResponseHeader() http.Header  { return http.Header{} }
func (c *stubHandlerConn) ResponseTrailer() http.Header { return http.Header{} }

// TestClientStreamReceiveStopsAfterError verifies that once Receive has
// recorded an error, subsequent calls return false without advancing the
// underlying connection.
func TestClientStreamReceiveStopsAfterError(t *testing.T) {
	t.Parallel()
	conn := &stubHandlerConn{receiveErr: errors.New("receive failure")}
	stream := &ClientStream{conn: conn}

	assert.False(t, stream.Receive(&pingv1.PingRequest{}))
	assert.False(t, stream.Receive(&pingv1.PingRequest{}))
	assert.Equal(t, conn.receiveCalls, 1)
}

// TestClientStreamErrPropagation verifies that Err returns the first non-EOF
// error encountered by Receive and returns nil for a normal end of stream.
func TestClientStreamErrPropagation(t *testing.T) {
	t.Parallel()
	receiveErr := errors.New("receive failure")
	conn := &stubHandlerConn{receiveErr: receiveErr}
	stream := &ClientStream{conn: conn}

	// A non-EOF error is returned by Err.
	assert.False(t, stream.Receive(&pingv1.PingRequest{}))
	assert.ErrorIs(t, stream.Err(), receiveErr)

	// EOF is treated as a normal end of stream and Err returns nil.
	connEOF := &stubHandlerConn{receiveErr: io.EOF}
	streamEOF := &ClientStream{conn: connEOF}
	assert.False(t, streamEOF.Receive(&pingv1.PingRequest{}))
	assert.Nil(t, streamEOF.Err())
}

// TestClientStreamMsgNil verifies that Msg returns nil when Receive has not
// been called.
func TestClientStreamMsgNil(t *testing.T) {
	t.Parallel()
	stream := &ClientStream{conn: &stubHandlerConn{}}
	assert.Nil(t, stream.Msg())
}

// TestClientStreamRequestHeaderEmpty verifies that RequestHeader returns an
// empty header.
func TestClientStreamRequestHeaderEmpty(t *testing.T) {
	t.Parallel()
	stream := &ClientStream{conn: &stubHandlerConn{}}
	assert.Equal(t, len(stream.RequestHeader()), 0)
}

// TestServerStreamSendNil verifies that Send delegates a nil msg to the
// underlying connection unchanged.
func TestServerStreamSendNil(t *testing.T) {
	t.Parallel()
	conn := &stubHandlerConn{}
	stream := &ServerStream{conn: conn}

	assert.Nil(t, stream.Send(nil))
	assert.Equal(t, len(conn.sent), 1)
	assert.Nil(t, conn.sent[0])
}

// TestServerStreamResponseHeaderEmpty verifies that ResponseHeader and
// ResponseTrailer return empty headers.
func TestServerStreamResponseHeaderEmpty(t *testing.T) {
	t.Parallel()
	stream := &ServerStream{conn: &stubHandlerConn{}}
	assert.Equal(t, len(stream.ResponseHeader()), 0)
	assert.Equal(t, len(stream.ResponseTrailer()), 0)
}

// TestBidiStreamReceiveErrorPropagation verifies that Receive returns the
// underlying connection error unchanged.
func TestBidiStreamReceiveErrorPropagation(t *testing.T) {
	t.Parallel()
	receiveErr := errors.New("receive failure")
	conn := &stubHandlerConn{receiveErr: receiveErr}
	stream := &BidiStream{conn: conn}

	err := stream.Receive(&pingv1.CumSumRequest{})
	assert.ErrorIs(t, err, receiveErr)
}

// TestBidiStreamSendNil verifies that Send delegates a nil msg to the
// underlying connection unchanged.
func TestBidiStreamSendNil(t *testing.T) {
	t.Parallel()
	conn := &stubHandlerConn{}
	stream := &BidiStream{conn: conn}

	assert.Nil(t, stream.Send(nil))
	assert.Equal(t, len(conn.sent), 1)
	assert.Nil(t, conn.sent[0])
}

// TestBidiStreamExportableHeader verifies that ExportableHeader forwards the
// underlying connection's metadata unchanged.
func TestBidiStreamExportableHeader(t *testing.T) {
	t.Parallel()
	hdr := http.Header{"X-Custom": []string{"value"}}
	conn := &stubHandlerConn{exportable: hdr}
	stream := &BidiStream{conn: conn}

	assert.Equal(t, stream.ExportableHeader().Get("X-Custom"), "value")
}
