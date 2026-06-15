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
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
)

// EncodeBinaryHeader base64-encodes the data. It always emits unpadded values.
//
// In the Triple, gRPC, and gRPC-Web protocols, binary headers must have keys
// ending in "-Bin".
func EncodeBinaryHeader(data []byte) string {
	// gRPC specification says that implementations should emit unpadded values.
	return base64.RawStdEncoding.EncodeToString(data)
}

// DecodeBinaryHeader base64-decodes the data. It can decode padded or unpadded
// values. Following usual HTTP semantics, multiple base64-encoded values may
// be joined with a comma. When receiving such comma-separated values, split
// them with [strings.Split] before calling DecodeBinaryHeader.
//
// Binary headers sent using the Triple, gRPC, and gRPC-Web protocols have
// keys ending in "-Bin".
func DecodeBinaryHeader(data string) ([]byte, error) {
	if len(data)%4 != 0 {
		// Data definitely isn't padded.
		return base64.RawStdEncoding.DecodeString(data)
	}
	// Either the data was padded, or padding wasn't necessary. In both cases,
	// the padding-aware decoder works.
	return base64.StdEncoding.DecodeString(data)
}

func mergeHeaders(into, from http.Header) {
	for k, vals := range from {
		into[k] = append(into[k], vals...)
	}
}

// getCanonicalHeader is a shortcut for Header.Get() which
// bypasses the CanonicalMIMEHeaderKey operation when we
// know the key is already in canonical form.
func getHeaderCanonical(h http.Header, key string) string {
	if h == nil {
		return ""
	}
	v := h[key]
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// setHeaderCanonical is a shortcut for Header.Set() which
// bypasses the CanonicalMIMEHeaderKey operation when we
// know the key is already in canonical form.
func setHeaderCanonical(h http.Header, key, value string) {
	h[key] = []string{value}
}

// delHeaderCanonical is a shortcut for Header.Del() which
// bypasses the CanonicalMIMEHeaderKey operation when we
// know the key is already in canonical form.
func delHeaderCanonical(h http.Header, key string) {
	delete(h, key)
}

// addHeaderCanonical is a shortcut for Header.Add() which
// bypasses the CanonicalMIMEHeaderKey operation when we
// know the key is already in canonical form.
func addHeaderCanonical(h http.Header, key, value string) {
	h[key] = append(h[key], value)
}

type extraDataKey struct{}

const headerIncomingKey string = "headerIncomingKey"
const headerOutgoingKey string = "headerOutgoingKey"

type handlerOutgoingKey struct{}

func newIncomingContext(ctx context.Context, data http.Header) context.Context {
	var header = http.Header{}
	extraData, ok := ctx.Value(extraDataKey{}).(map[string]http.Header)
	if !ok {
		extraData = map[string]http.Header{}
	}

	for key, vals := range data {
		header[http.CanonicalHeaderKey(key)] = vals
	}

	extraData[headerIncomingKey] = header
	return context.WithValue(ctx, extraDataKey{}, extraData)
}

// NewOutgoingContext sets outgoing request headers on ctx. If ctx already has
// outgoing headers, they are replaced.
//
// For example:
//
//	ctx := NewOutgoingContext(context.Background(), http.Header{
//		"hello": []string{"triple"},
//	})
//	resp, err := client.Greet(ctx, &greet.GreetRequest{})
//
// Use AppendToOutgoingContext to add more values without replacing existing
// outgoing headers. These APIs follow the same pattern as grpc metadata:
// https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#sending-metadata.
func NewOutgoingContext(ctx context.Context, data http.Header) context.Context {
	var header = http.Header{}

	for key, vals := range data {
		header[http.CanonicalHeaderKey(key)] = append([]string(nil), vals...)
	}

	extraData, ok := ctx.Value(extraDataKey{}).(map[string]http.Header)
	if !ok {
		extraData = map[string]http.Header{}
	} else {
		extraData = cloneExtraData(extraData)
	}
	extraData[headerOutgoingKey] = header
	return context.WithValue(ctx, extraDataKey{}, extraData)
}

func cloneExtraData(data map[string]http.Header) map[string]http.Header {
	cloned := make(map[string]http.Header, len(data))
	for extraKey, header := range data {
		headerClone := make(http.Header, len(header))
		for headerKey, vals := range header {
			headerClone[headerKey] = append([]string(nil), vals...)
		}
		cloned[extraKey] = headerClone
	}
	return cloned
}

// AppendToOutgoingContext appends key-value pairs to outgoing request headers.
// It panics when kv contains an odd number of strings.
//
// For example:
//
//	ctx := NewOutgoingContext(context.Background(), http.Header{
//		"hello": []string{"triple"},
//	})
//	ctx = AppendToOutgoingContext(ctx, "hello", "dubbo", "hey", "hessian")
//	resp, err := client.Greet(ctx, &greet.GreetRequest{})
//
// The example sends "hello: triple", "hello: dubbo", and "hey: hessian" to the
// server. This API follows the same pattern as grpc metadata:
// https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#sending-metadata.
func AppendToOutgoingContext(ctx context.Context, kv ...string) context.Context {
	if len(kv)%2 == 1 {
		panic(fmt.Sprintf("AppendToOutgoingContext got an odd number of input pairs for header: %d", len(kv)))
	}
	extraData, ok := ctx.Value(extraDataKey{}).(map[string]http.Header)
	if !ok {
		extraData = map[string]http.Header{}
		ctx = context.WithValue(ctx, extraDataKey{}, extraData)
	}
	header, ok := extraData[headerOutgoingKey]
	if !ok {
		header = make(http.Header)
		extraData[headerOutgoingKey] = header
	}
	for i := 0; i < len(kv); i += 2 {
		key := http.CanonicalHeaderKey(kv[i])
		header[key] = append(header[key], kv[i+1])
	}
	return ctx
}

func ExtractFromOutgoingContext(ctx context.Context) http.Header {
	extraData, ok := ctx.Value(extraDataKey{}).(map[string]http.Header)
	if !ok {
		return nil
	}
	if outGoingDataHeader, ok := extraData[headerOutgoingKey]; !ok {
		return nil
	} else {
		return outGoingDataHeader
	}
}

// FromIncomingContext retrieves request headers received by a server handler.
//
// For example:
//
//	func (srv *Server) Greet(ctx context.Context, req *greet.GreetRequest) (*greet.GreetResponse, error) {
//		headers, ok := FromIncomingContext(ctx)
//		if ok {
//			values := headers.Values("hello")
//			// Use values.
//		}
//		return &greet.GreetResponse{}, nil
//	}
//
// This API follows the same pattern as grpc metadata:
// https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#receiving-metadata-1.
func FromIncomingContext(ctx context.Context) (http.Header, bool) {
	data, ok := ctx.Value(extraDataKey{}).(map[string]http.Header)
	if !ok {
		return nil, false
	} else if incomingDataHeader, ok := data[headerIncomingKey]; !ok {
		return nil, false
	} else {
		return incomingDataHeader, true
	}
}

// SetHeader appends response headers from a server handler. The headers are
// buffered and sent with the response instead of being sent immediately.
//
// For example:
//
//	func (srv *Server) Greet(ctx context.Context, req *greet.GreetRequest) (*greet.GreetResponse, error) {
//		if err := SetHeader(ctx, http.Header{"hi": []string{"triple"}}); err != nil {
//			return nil, err
//		}
//		return &greet.GreetResponse{}, nil
//	}
//
// This API follows the same pattern as grpc metadata:
// https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#unary-call-2.
func SetHeader(ctx context.Context, header http.Header) error {
	conn, ok := ctx.Value(handlerOutgoingKey{}).(StreamingHandlerConn)
	if !ok {
		// todo(DMwangnima): return standard error
		return fmt.Errorf("triple: failed to fetch the connection from the context %v", ctx)
	}
	mergeHeaders(conn.ResponseHeader(), header)
	return nil
}

// SetTrailer appends response trailers from a server handler.
//
// For example:
//
//	func (srv *Server) Greet(ctx context.Context, req *greet.GreetRequest) (*greet.GreetResponse, error) {
//		if err := SetTrailer(ctx, http.Header{"end": []string{"end"}}); err != nil {
//			return nil, err
//		}
//		return &greet.GreetResponse{}, nil
//	}
//
// This API follows the same pattern as grpc metadata:
// https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#unary-call-2.
func SetTrailer(ctx context.Context, trailer http.Header) error {
	conn, ok := ctx.Value(handlerOutgoingKey{}).(StreamingHandlerConn)
	if !ok {
		// todo(DMwangnima): return standard error
		return fmt.Errorf("triple: failed to fetch the connection from the context %v", ctx)
	}
	mergeHeaders(conn.ResponseTrailer(), trailer)
	return nil
}

// SendHeader appends response headers from a server handler and sends them
// immediately. This is useful for streaming handlers that need to flush headers
// before the first message.
//
// For example:
//
//	func (srv *Server) GreetStream(ctx context.Context, stream greet.GreetService_GreetStreamServer) error {
//		if err := SendHeader(ctx, http.Header{"hi": []string{"triple"}}); err != nil {
//			return err
//		}
//		return stream.Send(&greet.GreetResponse{})
//	}
//
// This API follows the same pattern as grpc metadata:
// https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#unary-call-2.
func SendHeader(ctx context.Context, header http.Header) error {
	conn, ok := ctx.Value(handlerOutgoingKey{}).(StreamingHandlerConn)
	if !ok {
		// todo(DMwangnima): return standard error
		return fmt.Errorf("triple: failed to fetch the connection from the context %v", ctx)
	}
	mergeHeaders(conn.RequestHeader(), header)
	return conn.Send(nil)
}
