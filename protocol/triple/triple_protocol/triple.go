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

// Package triple is a slim RPC framework built on Protocol Buffers and
// [net/http]. In addition to supporting its own protocol, Triple handlers and
// clients are wire-compatible with gRPC, including streaming.
//
// This documentation is intended to explain each type and function in
// isolation. Walkthroughs, FAQs, and other narrative docs are available on the
// [dubbo-go website], and there's a working [demonstration service] on Github.
//
// [dubbo-go website]: https://cn.dubbo.apache.org/zh-cn/overview/mannual/golang-sdk/
// [demonstration service]: https://github.com/apache/dubbo-go-samples
package triple_protocol

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/dubbogo/gost/log/logger"
)

// Version is the semantic version of the triple module.
const Version = "0.1.0"

// These constants are used in compile-time handshakes with triple's generated
// code.
const (
	IsAtLeastVersion0_0_1 = true
	IsAtLeastVersion0_1_0 = true
	IsAtLeastVersion1_6_0 = true
)

// StreamType describes whether the client, server, neither, or both is
// streaming.
type StreamType uint8

const (
	StreamTypeUnary  StreamType = 0b00
	StreamTypeClient StreamType = 0b01
	StreamTypeServer StreamType = 0b10
	StreamTypeBidi              = StreamTypeClient | StreamTypeServer
)

const (
	tripleServiceGroup   = "tri-service-group"
	tripleServiceVersion = "tri-service-version"
)

// StreamingHandlerConn is the server's view of a bidirectional message
// exchange. Interceptors for streaming RPCs may wrap StreamingHandlerConns.
//
// Like the standard library's [http.ResponseWriter], StreamingHandlerConns write
// response headers to the network with the first call to Send. Any subsequent
// mutations are effectively no-ops. Handlers may mutate response trailers at
// any time before returning. When the client has finished sending data,
// Receive returns an error wrapping [io.EOF]. Handlers should check for this
// using the standard library's [errors.Is].
//
// Headers and trailers beginning with "Triple-" and "Grpc-" are reserved for
// use by the gRPC and Triple protocols: applications may read them but
// shouldn't write them.
//
// StreamingHandlerConn implementations provided by this module guarantee that
// all returned errors can be cast to [*Error] using the standard library's
// [errors.As].
//
// StreamingHandlerConn implementations do not need to be safe for concurrent use.
type StreamingHandlerConn interface {
	Spec() Spec
	Peer() Peer

	Receive(any) error
	RequestHeader() http.Header
	ExportableHeader() http.Header

	Send(any) error
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
}

// StreamingClientConn is the client's view of a bidirectional message exchange.
// Interceptors for streaming RPCs may wrap StreamingClientConn.
//
// StreamingClientConn write request headers to the network with the first
// call to Send. Any subsequent mutations are effectively no-ops. When the
// server is done sending data, the StreamingClientConn's Receive method
// returns an error wrapping [io.EOF]. Clients should check for this using the
// standard library's [errors.Is] or [IsEnded]. If the server encounters an error
// during processing, subsequent calls to the StreamingClientConn's Send method
// will return an error wrapping [io.EOF]; clients may then call Receive to
// unmarshal the error.
//
// Headers and trailers beginning with "Triple-" and "Grpc-" are reserved for
// use by the gRPC and Triple protocols: applications may read them but
// shouldn't write them.
//
// StreamingClientConn implementations provided by this module guarantee that
// all returned errors can be cast to [*Error] using the standard library's
// [errors.As].
//
// In order to support bidirectional streaming RPCs, all StreamingClientConn
// implementations must support limited concurrent use. See the comments on
// each group of methods for details.
type StreamingClientConn interface {
	// Spec and Peer must be safe to call concurrently with all other methods.
	Spec() Spec
	Peer() Peer

	// Send, RequestHeader, and CloseRequest may race with each other, but must
	// be safe to call concurrently with all other methods.
	Send(any) error
	RequestHeader() http.Header
	CloseRequest() error

	// Receive, ResponseHeader, ResponseTrailer, and CloseResponse may race with
	// each other, but must be safe to call concurrently with all other methods.
	Receive(any) error
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	CloseResponse() error
}

// Request is a wrapper around a generated request message. It provides
// access to metadata like headers and the RPC specification, as well as
// strongly-typed access to the message itself.
type Request struct {
	Msg any

	spec   Spec
	peer   Peer
	header http.Header
}

// NewRequest wraps a generated request message.
func NewRequest(message any) *Request {
	return &Request{
		Msg: message,
		// Initialized lazily so we don't allocate unnecessarily.
		header: nil,
	}
}

// Any returns the concrete request message as an empty interface, so that
// *Request implements the [AnyRequest] interface.
func (r *Request) Any() any {
	return r.Msg
}

// Spec returns a description of this RPC.
func (r *Request) Spec() Spec {
	return r.spec
}

// Peer describes the other party for this RPC.
func (r *Request) Peer() Peer {
	return r.peer
}

// Header returns the HTTP headers for this request. Headers beginning with
// "Triple-" and "Grpc-" are reserved for use by the Triple and gRPC
// protocols: applications may read them but shouldn't write them.
func (r *Request) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}
	return r.header
}

// internalOnly implements AnyRequest.
func (r *Request) internalOnly() {}

// AnyRequest is the common method set of every [Request], regardless of type
// parameter. It's used in unary interceptors.
//
// Headers and trailers beginning with "Triple-" and "Grpc-" are reserved for
// use by the gRPC and Triple protocols: applications may read them but
// shouldn't write them.
//
// To preserve our ability to add methods to this interface without breaking
// backward compatibility, only types defined in this package can implement
// AnyRequest.
type AnyRequest interface {
	Any() any
	Spec() Spec
	Peer() Peer
	Header() http.Header

	internalOnly()
}

// Response is a wrapper around a generated response message. It provides
// access to metadata like headers and trailers, as well as strongly-typed
// access to the message itself.
type Response struct {
	Msg any

	header  http.Header
	trailer http.Header
}

// NewResponse wraps a generated response message.
func NewResponse(message any) *Response {
	return &Response{
		Msg: message,
		// Initialized lazily so we don't allocate unnecessarily.
		header:  nil,
		trailer: nil,
	}
}

// Any returns the concrete response message as an empty interface, so that
// *Response implements the [AnyResponse] interface.
func (r *Response) Any() any {
	return r.Msg
}

// Header returns the HTTP headers for this response. Headers beginning with
// "Triple-" and "Grpc-" are reserved for use by the Triple and gRPC
// protocols: applications may read them but shouldn't write them.
func (r *Response) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}
	return r.header
}

// Trailer returns the trailers for this response. Depending on the underlying
// RPC protocol, trailers may be sent as HTTP trailers or a protocol-specific
// block of in-body metadata.
//
// Trailers beginning with "Triple-" and "Grpc-" are reserved for use by the
// Triple and gRPC protocols: applications may read them but shouldn't write
// them.
func (r *Response) Trailer() http.Header {
	if r.trailer == nil {
		r.trailer = make(http.Header)
	}
	return r.trailer
}

// internalOnly implements AnyResponse.
func (r *Response) internalOnly() {}

// AnyResponse is the common method set of every [Response], regardless of type
// parameter. It's used in unary interceptors.
//
// Headers and trailers beginning with "Triple-" and "Grpc-" are reserved for
// use by the Triple and Grpc protocols: applications may read them but
// shouldn't write them.
//
// To preserve our ability to add methods to this interface without breaking
// backward compatibility, only types defined in this package can implement
// AnyResponse.
type AnyResponse interface {
	Any() any
	Header() http.Header
	Trailer() http.Header

	internalOnly()
}

// HTTPClient is the interface triple expects HTTP clients to implement. The
// standard library's *http.Client implements HTTPClient.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Spec is a description of a client call or a handler invocation.
//
// If you're using Protobuf, protoc-gen-triple-go generates a constant for the
// fully-qualified Procedure corresponding to each RPC in your schema.
type Spec struct {
	StreamType       StreamType
	Procedure        string // for example, "/acme.foo.v1.FooService/Bar"
	IsClient         bool   // otherwise we're in a handler
	IdempotencyLevel IdempotencyLevel
}

// Peer describes the other party to an RPC.
//
// When accessed client-side, Addr contains the host or host:port from the
// server's URL. When accessed server-side, Addr contains the client's address
// in IP:port format.
//
// On both the client and the server, Protocol is the RPC protocol in use.
// Currently, it's either [ProtocolTriple], [ProtocolGRPC], or
// todo: Should we support ProtocolGRPCWeb?
// [ProtocolGRPCWeb], but additional protocols may be added in the future.
//
// Query contains the query parameters for the request. For the server, this
// will reflect the actual query parameters sent. For the client, it is unset.
type Peer struct {
	Addr     string
	Protocol string
	Query    url.Values // server-only
}

func newPeerFromURL(url *url.URL, protocol string) Peer {
	return Peer{
		Addr:     url.Host,
		Protocol: protocol,
	}
}

// handlerConnCloser extends HandlerConn with a method for handlers to
// terminate the message exchange (and optionally send an error to the client).
type handlerConnCloser interface {
	StreamingHandlerConn

	Close(error) error
}

// receiveUnaryResponse unmarshals a message from a StreamingClientConn, then
// envelopes the message and attaches headers and trailers. It attempts to
// consume the response stream and is not appropriate when receiving multiple
// messages.
func receiveUnaryResponse(conn StreamingClientConn, response AnyResponse) error {
	logger.Warnf("here*******************************")
	resp, ok := response.(*Response)
	if !ok {
		panic(fmt.Sprintf("response %T is not of Response type", response))
	}
	if err := conn.Receive(resp.Msg); err != nil {
		return err
	}
	// In a well-formed stream, the response message may be followed by a block
	// of in-stream trailers or HTTP trailers. To ensure that we receive the
	// trailers, try to read another message from the stream.
	if err := conn.Receive(resp.Msg); err == nil {
		return NewError(CodeUnknown, errors.New("unary stream has multiple messages"))
	} else if err != nil && !errors.Is(err, io.EOF) {
		return NewError(CodeUnknown, err)
	}
	resp.header = conn.ResponseHeader()
	resp.trailer = conn.ResponseTrailer()
	return nil
}

// IsEnded is a convenient function indicating the end of stream. It is introduced to not expose
// io.EOF to beginners. Please refer to https://github.com/apache/dubbo-go/pull/2416#discussion_r1318558801
func IsEnded(err error) bool {
	return errors.Is(err, io.EOF)
}
