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
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"testing/quick"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

func TestBinaryEncodingQuick(t *testing.T) {
	t.Parallel()
	roundtrip := func(binary []byte) bool {
		encoded := EncodeBinaryHeader(binary)
		decoded, err := DecodeBinaryHeader(encoded)
		if err != nil {
			// We want to abort immediately. Don't use our assert package.
			t.Fatalf("decode error: %v", err)
		}
		return bytes.Equal(decoded, binary)
	}
	if err := quick.Check(roundtrip, nil /* config */); err != nil {
		t.Error(err)
	}
}

func TestHeaderMerge(t *testing.T) {
	t.Parallel()
	header := http.Header{
		"Foo": []string{"one"},
	}
	mergeHeaders(header, http.Header{
		"Foo": []string{"two"},
		"Bar": []string{"one"},
		"Baz": nil,
	})
	expect := http.Header{
		"Foo": []string{"one", "two"},
		"Bar": []string{"one"},
		"Baz": nil,
	}
	assert.Equal(t, header, expect)
}

func TestNewIncomingContextClonesHeaders(t *testing.T) {
	baseCtx := NewOutgoingContext(context.Background(), http.Header{
		"Request-Id": []string{"outgoing"},
	})
	inputValues := []string{"incoming"}
	input := http.Header{
		"request-id": inputValues,
	}

	ctx := newIncomingContext(baseCtx, input)
	incoming, ok := FromIncomingContext(ctx)
	assert.True(t, ok)
	incoming.Values("Request-Id")[0] = "changed"
	incoming.Add("Another", "value")

	assert.Equal(t, []string{"incoming"}, inputValues)
	outgoing := ExtractFromOutgoingContext(baseCtx)
	assert.Equal(t, []string{"outgoing"}, outgoing.Values("Request-Id"))
}

func ExampleNewOutgoingContext() {
	ctx := NewOutgoingContext(context.Background(), http.Header{
		"hello": []string{"triple"},
	})
	ctx = AppendToOutgoingContext(ctx, "hello", "dubbo", "hey", "hessian")

	headers := ExtractFromOutgoingContext(ctx)
	fmt.Println(headers.Values("hello"))
	fmt.Println(headers.Get("hey"))

	// Output:
	// [triple dubbo]
	// hessian
}

// TestDecodeBinaryHeader verifies that DecodeBinaryHeader handles empty,
// unpadded, padded and naturally-aligned base64 inputs.
func TestDecodeBinaryHeader(t *testing.T) {
	t.Parallel()
	// Empty input
	got, err := DecodeBinaryHeader("")
	assert.Nil(t, err)
	assert.Equal(t, got, []byte{})
	// Unpadded (len % 4 != 0)
	decoded, err := DecodeBinaryHeader(EncodeBinaryHeader([]byte("hello")))
	assert.Nil(t, err)
	assert.Equal(t, decoded, []byte("hello"))
	// Padded (len % 4 == 0, has '=' padding)
	decoded, err = DecodeBinaryHeader("aGVsbG8=")
	assert.Nil(t, err)
	assert.Equal(t, decoded, []byte("hello"))
	// Naturally aligned (len % 4 == 0, no padding needed)
	decoded, err = DecodeBinaryHeader("YWJj")
	assert.Nil(t, err)
	assert.Equal(t, decoded, []byte("abc"))
}

// TestExtractFromOutgoingContext verifies that ExtractFromOutgoingContext
// returns nil when no outgoing headers are set and returns the headers
// when they have been set via NewOutgoingContext.
func TestExtractFromOutgoingContext(t *testing.T) {
	t.Parallel()
	// No outgoing headers set
	assert.Nil(t, ExtractFromOutgoingContext(context.Background()))
	// Headers set via NewOutgoingContext
	ctx := NewOutgoingContext(context.Background(), http.Header{
		"Foo": []string{"bar"},
	})
	extracted := ExtractFromOutgoingContext(ctx)
	assert.NotNil(t, extracted)
	assert.Equal(t, extracted.Get("Foo"), "bar")
}

// TestNewOutgoingContextReplacesExisting verifies that a second call to
// NewOutgoingContext replaces the existing outgoing headers instead of
// merging them.
func TestNewOutgoingContextReplacesExisting(t *testing.T) {
	t.Parallel()
	ctx := NewOutgoingContext(context.Background(), http.Header{
		"Foo": []string{"bar"},
	})
	ctx = NewOutgoingContext(ctx, http.Header{
		"Baz": []string{"qux"},
	})
	extracted := ExtractFromOutgoingContext(ctx)
	assert.Equal(t, extracted.Get("Foo"), "")
	assert.Equal(t, extracted.Get("Baz"), "qux")
}

// TestAppendToOutgoingContextPanicsOnOddKV verifies that
// AppendToOutgoingContext panics when given an odd number of key-value
// arguments.
func TestAppendToOutgoingContextPanicsOnOddKV(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		// Deliberately pass an odd number of kv arguments to trigger the
		// panic. staticcheck SA5012 flags this as a bug.
		AppendToOutgoingContext(context.Background(), "foo") //nolint:staticcheck
	})
}

// mockHandlerConn is a minimal StreamingHandlerConn for testing SetHeader,
// SendHeader and SetTrailer.
type mockHandlerConn struct {
	responseHdr  http.Header
	responseTrlr http.Header
	requestHdr   http.Header
	sendCalls    int
}

func newMockHandlerConn() *mockHandlerConn {
	return &mockHandlerConn{
		responseHdr:  make(http.Header),
		responseTrlr: make(http.Header),
		requestHdr:   make(http.Header),
	}
}

func (m *mockHandlerConn) Spec() Spec                    { return Spec{} }
func (m *mockHandlerConn) Peer() Peer                    { return Peer{} }
func (m *mockHandlerConn) Receive(any) error             { return nil }
func (m *mockHandlerConn) RequestHeader() http.Header    { return m.requestHdr }
func (m *mockHandlerConn) ExportableHeader() http.Header { return nil }
func (m *mockHandlerConn) Send(any) error                { m.sendCalls++; return nil }
func (m *mockHandlerConn) ResponseHeader() http.Header   { return m.responseHdr }
func (m *mockHandlerConn) ResponseTrailer() http.Header  { return m.responseTrlr }

// TestSetHeader verifies that SetHeader merges headers into the response
// header buffer when called within a handler context.
func TestSetHeader(t *testing.T) {
	t.Parallel()
	conn := newMockHandlerConn()
	ctx := context.WithValue(context.Background(), handlerOutgoingKey{}, conn)
	err := SetHeader(ctx, http.Header{"X-Custom": []string{"value"}})
	assert.Nil(t, err)
	assert.Equal(t, conn.responseHdr.Get("X-Custom"), "value")
}

// TestSetTrailer verifies that SetTrailer merges headers into the response
// trailer buffer when called within a handler context.
func TestSetTrailer(t *testing.T) {
	t.Parallel()
	conn := newMockHandlerConn()
	ctx := context.WithValue(context.Background(), handlerOutgoingKey{}, conn)
	err := SetTrailer(ctx, http.Header{"X-Trailer": []string{"end"}})
	assert.Nil(t, err)
	assert.Equal(t, conn.responseTrlr.Get("X-Trailer"), "end")
}

// TestSendHeader verifies that SendHeader merges headers into the response
// header buffer (not the request header) and triggers a flush via Send.
func TestSendHeader(t *testing.T) {
	t.Parallel()
	conn := newMockHandlerConn()
	ctx := context.WithValue(context.Background(), handlerOutgoingKey{}, conn)
	err := SendHeader(ctx, http.Header{"X-Custom": []string{"value"}})
	assert.Nil(t, err)
	// Headers must land in the response header buffer.
	assert.Equal(t, conn.responseHdr.Get("X-Custom"), "value")
	// The request header must not be polluted.
	assert.Equal(t, len(conn.requestHdr), 0)
	// Send must be called exactly once to flush the headers.
	assert.Equal(t, conn.sendCalls, 1)
}

// TestSendHeaderOutsideHandler verifies that SendHeader returns a CodeInternal
// error when called outside a Triple handler context.
func TestSendHeaderOutsideHandler(t *testing.T) {
	t.Parallel()
	err := SendHeader(context.Background(), http.Header{"X-Custom": []string{"value"}})
	assert.NotNil(t, err)
	tripleErr, ok := asError(err)
	assert.True(t, ok)
	assert.Equal(t, tripleErr.Code(), CodeInternal)
}

// TestSetHeaderOutsideHandler verifies that SetHeader returns a CodeInternal
// error when called outside a Triple handler context.
func TestSetHeaderOutsideHandler(t *testing.T) {
	t.Parallel()
	err := SetHeader(context.Background(), http.Header{"X-Custom": []string{"value"}})
	assert.NotNil(t, err)
	tripleErr, ok := asError(err)
	assert.True(t, ok)
	assert.Equal(t, tripleErr.Code(), CodeInternal)
}

// TestSetTrailerOutsideHandler verifies that SetTrailer returns a
// CodeInternal error when called outside a Triple handler context.
func TestSetTrailerOutsideHandler(t *testing.T) {
	t.Parallel()
	err := SetTrailer(context.Background(), http.Header{"X-Trailer": []string{"end"}})
	assert.NotNil(t, err)
	tripleErr, ok := asError(err)
	assert.True(t, ok)
	assert.Equal(t, tripleErr.Code(), CodeInternal)
}
