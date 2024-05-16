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

package triple_protocol

import (
	"errors"
	"io"
	"net/http"
)

// ClientStreamForClient is the client's view of a client streaming RPC.
//
// It's returned from [Client].CallClientStream, but doesn't currently have an
// exported constructor function.
type ClientStreamForClient struct {
	conn StreamingClientConn
	// Error from client construction. If non-nil, return for all calls.
	err error
}

// Spec returns the specification for the RPC.
func (c *ClientStreamForClient) Spec() Spec {
	return c.conn.Spec()
}

// Peer describes the server for the RPC.
func (c *ClientStreamForClient) Peer() Peer {
	return c.conn.Peer()
}

// RequestHeader returns the request headers. Headers are sent to the server with the
// first call to Send.
//
// Headers beginning with "Triple-" and "Grpc-" are reserved for use by the
// Triple and gRPC protocols. Applications shouldn't write them.
func (c *ClientStreamForClient) RequestHeader() http.Header {
	if c.err != nil {
		// todo(DMwangnima): since there is error in ClientStreamForClient, maybe we should tell user other than
		// returning a empty Header
		return http.Header{}
	}
	return c.conn.RequestHeader()
}

// Send a message to the server. The first call to Send also sends the request
// headers.
//
// If the server returns an error, Send returns an error that wraps [io.EOF].
// Clients should check for case using the standard library's [errors.Is] or
// [IsEnded] and unmarshal the error using CloseAndReceive.
func (c *ClientStreamForClient) Send(request interface{}) error {
	if c.err != nil {
		return c.err
	}
	// todo(DMwangnima): remove this redundant statement
	if request == nil {
		return c.conn.Send(nil)
	}
	return c.conn.Send(request)
}

// CloseAndReceive closes the send side of the stream and waits for the
// response.
func (c *ClientStreamForClient) CloseAndReceive(response *Response) error {
	if c.err != nil {
		return c.err
	}
	if err := c.conn.CloseRequest(); err != nil {
		_ = c.conn.CloseResponse()
		return err
	}
	if err := receiveUnaryResponse(c.conn, response); err != nil {
		_ = c.conn.CloseResponse()
		return err
	}
	return c.conn.CloseResponse()
}

// Conn exposes the underlying StreamingClientConn. This may be useful if
// you'd prefer to wrap the connection in a different high-level API.
func (c *ClientStreamForClient) Conn() (StreamingClientConn, error) {
	return c.conn, c.err
}

// ServerStreamForClient is the client's view of a server streaming RPC.
//
// It's returned from [Client].CallServerStream, but doesn't currently have an
// exported constructor function.
type ServerStreamForClient struct {
	conn StreamingClientConn
	msg  interface{}
	// Error from client construction. If non-nil, return for all calls.
	constructErr error
	// Error from conn.Receive().
	receiveErr error
}

// Receive advances the stream to the next message, which will then be
// available through the Msg method. It returns false when the stream stops,
// either by reaching the end or by encountering an unexpected error. After
// Receive returns false, the Err method will return any unexpected error
// encountered.
// todo(DMwangnima): add classic usage
func (s *ServerStreamForClient) Receive(msg interface{}) bool {
	if s.constructErr != nil || s.receiveErr != nil {
		return false
	}
	s.msg = msg
	s.receiveErr = s.conn.Receive(s.msg)
	return s.receiveErr == nil
}

// Msg returns the most recent message unmarshaled by a call to Receive.
func (s *ServerStreamForClient) Msg() interface{} {
	// todo(DMwangnima): processing nil pointer
	//if s.msg == nil {
	//	s.msg = new(Res)
	//}
	return s.msg
}

// Err returns the first non-EOF error that was encountered by Receive.
func (s *ServerStreamForClient) Err() error {
	if s.constructErr != nil {
		return s.constructErr
	}
	if s.receiveErr != nil && !errors.Is(s.receiveErr, io.EOF) {
		return s.receiveErr
	}
	return nil
}

// ResponseHeader returns the headers received from the server. It blocks until
// the first call to Receive returns.
func (s *ServerStreamForClient) ResponseHeader() http.Header {
	if s.constructErr != nil {
		// todo(DMwangnima): since there is error in ServerStreamForClient, maybe we should tell user other than
		// returning an empty Header
		return http.Header{}
	}
	return s.conn.ResponseHeader()
}

// ResponseTrailer returns the trailers received from the server. Trailers
// aren't fully populated until Receive() returns an error wrapping io.EOF.
func (s *ServerStreamForClient) ResponseTrailer() http.Header {
	if s.constructErr != nil {
		// todo(DMwangnima): since there is error in ServerStreamForClient, maybe we should tell user other than
		// returning an empty Header
		return http.Header{}
	}
	return s.conn.ResponseTrailer()
}

// Close the receive side of the stream.
func (s *ServerStreamForClient) Close() error {
	if s.constructErr != nil {
		return s.constructErr
	}
	return s.conn.CloseResponse()
}

// Conn exposes the underlying StreamingClientConn. This may be useful if
// you'd prefer to wrap the connection in a different high-level API.
func (s *ServerStreamForClient) Conn() (StreamingClientConn, error) {
	return s.conn, s.constructErr
}

// BidiStreamForClient is the client's view of a bidirectional streaming RPC.
//
// It's returned from [Client].CallBidiStream, but doesn't currently have an
// exported constructor function.
type BidiStreamForClient struct {
	conn StreamingClientConn
	// Error from client construction. If non-nil, return for all calls.
	err error
}

// Spec returns the specification for the RPC.
func (b *BidiStreamForClient) Spec() Spec {
	return b.conn.Spec()
}

// Peer describes the server for the RPC.
func (b *BidiStreamForClient) Peer() Peer {
	return b.conn.Peer()
}

// RequestHeader returns the request headers. Headers are sent with the first
// call to Send.
//
// Headers beginning with "Triple-" and "Grpc-" are reserved for use by the
// Triple and gRPC protocols. Applications shouldn't write them.
func (b *BidiStreamForClient) RequestHeader() http.Header {
	if b.err != nil {
		// todo(DMwangnima): since there is error in BidiStreamForClient, maybe we should tell user other than
		// returning an empty Header
		return http.Header{}
	}
	return b.conn.RequestHeader()
}

// Send a message to the server. The first call to Send also sends the request
// headers. To send just the request headers without a body, call Send with a
// nil pointer.
//
// If the server returns an error, Send returns an error that wraps [io.EOF].
// Clients should check for EOF using the standard library's [errors.Is] and
// call Receive to retrieve the error.
func (b *BidiStreamForClient) Send(msg interface{}) error {
	if b.err != nil {
		return b.err
	}
	// todo(DMwangnima): remove this redundant statement
	if msg == nil {
		return b.conn.Send(nil)
	}
	return b.conn.Send(msg)
}

// CloseRequest closes the send side of the stream.
func (b *BidiStreamForClient) CloseRequest() error {
	if b.err != nil {
		return b.err
	}
	return b.conn.CloseRequest()
}

// Receive a message. When the server is done sending messages and no other
// errors have occurred, Receive will return an error that wraps [io.EOF].
func (b *BidiStreamForClient) Receive(msg interface{}) error {
	if b.err != nil {
		return b.err
	}
	if err := b.conn.Receive(msg); err != nil {
		return err
	}
	return nil
}

// CloseResponse closes the receive side of the stream.
func (b *BidiStreamForClient) CloseResponse() error {
	if b.err != nil {
		return b.err
	}
	return b.conn.CloseResponse()
}

// ResponseHeader returns the headers received from the server. It blocks until
// the first call to Receive returns.
func (b *BidiStreamForClient) ResponseHeader() http.Header {
	if b.err != nil {
		// todo(DMwangnima): since there is error in BidiStreamForClient, maybe we should tell user other than
		// returning an empty Header
		return http.Header{}
	}
	return b.conn.ResponseHeader()
}

// ResponseTrailer returns the trailers received from the server. Trailers
// aren't fully populated until Receive() returns an error wrapping [io.EOF].
func (b *BidiStreamForClient) ResponseTrailer() http.Header {
	if b.err != nil {
		// todo(DMwangnima): since there is error in BidiStreamForClient, maybe we should tell user other than
		// returning an empty Header
		return http.Header{}
	}
	return b.conn.ResponseTrailer()
}

// Conn exposes the underlying StreamingClientConn. This may be useful if
// you'd prefer to wrap the connection in a different high-level API.
func (b *BidiStreamForClient) Conn() (StreamingClientConn, error) {
	return b.conn, b.err
}
