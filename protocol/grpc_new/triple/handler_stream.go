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
	"io"
	"net/http"
)

// ClientStream is the handler's view of a client streaming RPC.
//
// It's constructed as part of [Handler] invocation, but doesn't currently have
// an exported constructor.
type ClientStream struct {
	conn StreamingHandlerConn
	msg  any
	err  error
}

// Spec returns the specification for the RPC.
func (c *ClientStream) Spec() Spec {
	return c.conn.Spec()
}

// Peer describes the client for this RPC.
func (c *ClientStream) Peer() Peer {
	return c.conn.Peer()
}

// RequestHeader returns the headers received from the client.
func (c *ClientStream) RequestHeader() http.Header {
	return c.conn.RequestHeader()
}

// Receive advances the stream to the next message, which will then be
// available through the Msg method. It returns false when the stream stops,
// either by reaching the end or by encountering an unexpected error. After
// Receive returns false, the Err method will return any unexpected error
// encountered.
func (c *ClientStream) Receive(msg any) bool {
	if c.err != nil {
		return false
	}
	c.msg = msg
	c.err = c.conn.Receive(c.msg)
	return c.err == nil
}

// Msg returns the most recent message unmarshaled by a call to Receive.
func (c *ClientStream) Msg() any {
	// todo:// process nil pointer
	//if c.msg == nil {
	//	c.msg = new(Req)
	//}
	return c.msg
}

// Err returns the first non-EOF error that was encountered by Receive.
func (c *ClientStream) Err() error {
	if c.err == nil || errors.Is(c.err, io.EOF) {
		return nil
	}
	return c.err
}

// Conn exposes the underlying StreamingHandlerConn. This may be useful if
// you'd prefer to wrap the connection in a different high-level API.
func (c *ClientStream) Conn() StreamingHandlerConn {
	return c.conn
}

// ServerStream is the handler's view of a server streaming RPC.
//
// It's constructed as part of [Handler] invocation, but doesn't currently have
// an exported constructor.
type ServerStream struct {
	conn StreamingHandlerConn
}

// ResponseHeader returns the response headers. Headers are sent with the first
// call to Send.
//
// Headers beginning with "Triple-" and "Grpc-" are reserved for use by the
// Triple and gRPC protocols. Applications shouldn't write them.
func (s *ServerStream) ResponseHeader() http.Header {
	return s.conn.ResponseHeader()
}

// ResponseTrailer returns the response trailers. Handlers may write to the
// response trailers at any time before returning.
//
// Trailers beginning with "Triple-" and "Grpc-" are reserved for use by the
// Triple and gRPC protocols. Applications shouldn't write them.
func (s *ServerStream) ResponseTrailer() http.Header {
	return s.conn.ResponseTrailer()
}

// Send a message to the client. The first call to Send also sends the response
// headers.
func (s *ServerStream) Send(msg any) error {
	if msg == nil {
		return s.conn.Send(nil)
	}
	return s.conn.Send(msg)
}

// Conn exposes the underlying StreamingHandlerConn. This may be useful if
// you'd prefer to wrap the connection in a different high-level API.
func (s *ServerStream) Conn() StreamingHandlerConn {
	return s.conn
}

// BidiStream is the handler's view of a bidirectional streaming RPC.
//
// It's constructed as part of [Handler] invocation, but doesn't currently have
// an exported constructor.
type BidiStream struct {
	conn StreamingHandlerConn
}

// Spec returns the specification for the RPC.
func (b *BidiStream) Spec() Spec {
	return b.conn.Spec()
}

// Peer describes the client for this RPC.
func (b *BidiStream) Peer() Peer {
	return b.conn.Peer()
}

// RequestHeader returns the headers received from the client.
func (b *BidiStream) RequestHeader() http.Header {
	return b.conn.RequestHeader()
}

// Receive a message. When the client is done sending messages, Receive will
// return an error that wraps [io.EOF].
func (b *BidiStream) Receive(msg any) error {
	if err := b.conn.Receive(&msg); err != nil {
		return err
	}
	return nil
}

// ResponseHeader returns the response headers. Headers are sent with the first
// call to Send.
//
// Headers beginning with "Triple-" and "Grpc-" are reserved for use by the
// Triple and gRPC protocols. Applications shouldn't write them.
func (b *BidiStream) ResponseHeader() http.Header {
	return b.conn.ResponseHeader()
}

// ResponseTrailer returns the response trailers. Handlers may write to the
// response trailers at any time before returning.
//
// Trailers beginning with "Triple-" and "Grpc-" are reserved for use by the
// Triple and gRPC protocols. Applications shouldn't write them.
func (b *BidiStream) ResponseTrailer() http.Header {
	return b.conn.ResponseTrailer()
}

// Send a message to the client. The first call to Send also sends the response
// headers.
func (b *BidiStream) Send(msg any) error {
	if msg == nil {
		return b.conn.Send(nil)
	}
	return b.conn.Send(msg)
}

// Conn exposes the underlying StreamingHandlerConn. This may be useful if
// you'd prefer to wrap the connection in a different high-level API.
func (b *BidiStream) Conn() StreamingHandlerConn {
	return b.conn
}
