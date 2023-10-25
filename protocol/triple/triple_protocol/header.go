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

type headerIncomingKey struct{}
type headerOutgoingKey struct{}
type handlerOutgoingKey struct{}

func newIncomingContext(ctx context.Context, header http.Header) context.Context {
	return context.WithValue(ctx, headerIncomingKey{}, header)
}

// NewOutgoingContext sets headers entirely. If there are existing headers, they would be replaced.
// It is used for passing headers to server-side.
// It is like grpc.NewOutgoingContext.
// Please refer to https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#sending-metadata.
func NewOutgoingContext(ctx context.Context, header http.Header) context.Context {
	return context.WithValue(ctx, headerOutgoingKey{}, header)
}

// AppendToOutgoingContext merges kv pairs from user and existing headers.
// It is used for passing headers to server-side.
// It is like grpc.AppendToOutgoingContext.
// Please refer to https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#sending-metadata.
func AppendToOutgoingContext(ctx context.Context, kv ...string) context.Context {
	if len(kv)%2 == 1 {
		panic(fmt.Sprintf("AppendToOutgoingContext got an odd number of input pairs for header: %d", len(kv)))
	}
	var header http.Header
	headerRaw := ctx.Value(headerOutgoingKey{})
	if headerRaw == nil {
		header = make(http.Header)
	} else {
		header = headerRaw.(http.Header)
	}
	for i := 0; i < len(kv); i += 2 {
		// todo(DMwangnima): think about lowering
		header.Add(kv[i], kv[i+1])
	}
	return context.WithValue(ctx, headerOutgoingKey{}, header)
}

// FromIncomingContext retrieves headers passed by client-side. It is like grpc.FromIncomingContext.
// Please refer to https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#receiving-metadata-1.
func FromIncomingContext(ctx context.Context) (http.Header, bool) {
	header, ok := ctx.Value(headerIncomingKey{}).(http.Header)
	if !ok {
		return nil, false
	}
	return header, true
}

// SetHeader is used for setting response header in server-side. It is like grpc.SendHeader(ctx, header) but
// not send header.
// Please refer to https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#unary-call-2.
func SetHeader(ctx context.Context, header http.Header) error {
	conn, ok := ctx.Value(handlerOutgoingKey{}).(StreamingHandlerConn)
	if !ok {
		// todo(DMwangnima): return standard error
		return fmt.Errorf("triple: failed to fetch the connection from the context %v", ctx)
	}
	mergeHeaders(conn.ResponseHeader(), header)
	return nil
}

// SetTrailer is used for setting response trailers in server-side. It is like grpc.SetTrailer(ctx, header).
// Please refer to https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#unary-call-2.
func SetTrailer(ctx context.Context, trailer http.Header) error {
	conn, ok := ctx.Value(handlerOutgoingKey{}).(StreamingHandlerConn)
	if !ok {
		// todo(DMwangnima): return standard error
		return fmt.Errorf("triple: failed to fetch the connection from the context %v", ctx)
	}
	mergeHeaders(conn.ResponseTrailer(), trailer)
	return nil
}

// SendHeader is used for setting response headers in server-side and send them directly. It is like grpc.SendHeader(ctx, header).
// Please refer to https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md#unary-call-2.
func SendHeader(ctx context.Context, header http.Header) error {
	conn, ok := ctx.Value(handlerOutgoingKey{}).(StreamingHandlerConn)
	if !ok {
		// todo(DMwangnima): return standard error
		return fmt.Errorf("triple: failed to fetch the connection from the context %v", ctx)
	}
	mergeHeaders(conn.RequestHeader(), header)
	return conn.Send(nil)
}
