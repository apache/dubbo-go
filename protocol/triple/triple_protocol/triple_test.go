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

package triple_protocol_test

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
)

import (
	triple "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestStreamType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		st       triple.StreamType
		expected triple.StreamType
	}{
		{
			name:     "unary",
			st:       triple.StreamTypeUnary,
			expected: 0b00,
		},
		{
			name:     "client",
			st:       triple.StreamTypeClient,
			expected: 0b01,
		},
		{
			name:     "server",
			st:       triple.StreamTypeServer,
			expected: 0b10,
		},
		{
			name:     "bidi",
			st:       triple.StreamTypeBidi,
			expected: triple.StreamTypeClient | triple.StreamTypeServer,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.st != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, tt.st)
			}
		})
	}
}

func TestNewRequest(t *testing.T) {
	t.Parallel()

	type TestMessage struct {
		Field string
	}

	msg := &TestMessage{Field: "test"}
	req := triple.NewRequest(msg)

	if req.Msg != msg {
		t.Errorf("expected message %v, got %v", msg, req.Msg)
	}

	// Header should be lazily initialized
	reqHeader := req.Header()
	if reqHeader == nil {
		t.Error("expected header to be initialized")
	}
}

func TestRequest_Any(t *testing.T) {
	t.Parallel()

	type TestMessage struct {
		Field string
	}

	msg := &TestMessage{Field: "test"}
	req := triple.NewRequest(msg)

	any := req.Any()
	if any != msg {
		t.Errorf("expected Any() to return %v, got %v", msg, any)
	}
}

func TestRequest_Spec(t *testing.T) {
	t.Parallel()

	req := triple.NewRequest("test")
	spec := req.Spec()

	// Default spec should have some values
	_ = spec
}

func TestRequest_Peer(t *testing.T) {
	t.Parallel()

	req := triple.NewRequest("test")
	peer := req.Peer()

	// Default peer should exist
	_ = peer
}

func TestRequest_Header(t *testing.T) {
	t.Parallel()

	req := triple.NewRequest("test")

	header := req.Header()
	if header == nil {
		t.Errorf("expected non-nil header after first access")
	}

	// Set a value
	header.Set("Test-Header", "test-value")

	// Second call should return the same header
	header2 := req.Header()
	if header2.Get("Test-Header") != "test-value" {
		t.Errorf("expected header to be persistent")
	}
}

func TestNewResponse(t *testing.T) {
	t.Parallel()

	type TestMessage struct {
		Field string
	}

	msg := &TestMessage{Field: "test"}
	resp := triple.NewResponse(msg)

	if resp.Msg != msg {
		t.Errorf("expected message %v, got %v", msg, resp.Msg)
	}
}

func TestResponse_Any(t *testing.T) {
	t.Parallel()

	type TestMessage struct {
		Field string
	}

	msg := &TestMessage{Field: "test"}
	resp := triple.NewResponse(msg)

	any := resp.Any()
	if any != msg {
		t.Errorf("expected Any() to return %v, got %v", msg, any)
	}
}

func TestResponse_Header(t *testing.T) {
	t.Parallel()

	resp := triple.NewResponse("test")

	header := resp.Header()
	if header == nil {
		t.Errorf("expected non-nil header after first access")
	}

	// Set a value
	header.Set("Test-Header", "test-value")

	// Second call should return the same header
	header2 := resp.Header()
	if header2.Get("Test-Header") != "test-value" {
		t.Errorf("expected header to be persistent")
	}
}

func TestResponse_Trailer(t *testing.T) {
	t.Parallel()

	resp := triple.NewResponse("test")

	trailer := resp.Trailer()
	if trailer == nil {
		t.Errorf("expected non-nil trailer after first access")
	}

	// Set a value
	trailer.Set("Test-Trailer", "test-value")

	// Second call should return the same trailer
	trailer2 := resp.Trailer()
	if trailer2.Get("Test-Trailer") != "test-value" {
		t.Errorf("expected trailer to be persistent")
	}
}

func TestSpec(t *testing.T) {
	t.Parallel()

	spec := triple.Spec{
		StreamType:       triple.StreamTypeBidi,
		Procedure:        "/test.Service/Method",
		IsClient:         true,
		IdempotencyLevel: triple.IdempotencyNoSideEffects,
	}

	if spec.StreamType != triple.StreamTypeBidi {
		t.Errorf("expected StreamType %v, got %v", triple.StreamTypeBidi, spec.StreamType)
	}

	if spec.Procedure != "/test.Service/Method" {
		t.Errorf("expected Procedure %s, got %s", "/test.Service/Method", spec.Procedure)
	}

	if !spec.IsClient {
		t.Errorf("expected IsClient to be true")
	}

	if spec.IdempotencyLevel != triple.IdempotencyNoSideEffects {
		t.Errorf("expected IdempotencyLevel %v, got %v", triple.IdempotencyNoSideEffects, spec.IdempotencyLevel)
	}
}

func TestPeer(t *testing.T) {
	t.Parallel()

	peer := triple.Peer{
		Addr:     "localhost:8080",
		Protocol: triple.ProtocolTriple,
		Query:    url.Values{"key": []string{"value"}},
	}

	if peer.Addr != "localhost:8080" {
		t.Errorf("expected Addr %s, got %s", "localhost:8080", peer.Addr)
	}

	if peer.Protocol != triple.ProtocolTriple {
		t.Errorf("expected Protocol %s, got %s", triple.ProtocolTriple, peer.Protocol)
	}

	if peer.Query.Get("key") != "value" {
		t.Errorf("expected Query key=value, got %s", peer.Query.Get("key"))
	}
}

func TestIsEnded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil_error",
			err:      nil,
			expected: false,
		},
		{
			name:     "eof_error",
			err:      io.EOF,
			expected: true,
		},
		{
			name:     "wrapped_eof_error",
			err:      errors.New("wrapped: " + io.EOF.Error()),
			expected: false,
		},
		{
			name:     "other_error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := triple.IsEnded(tt.err)
			if result != tt.expected {
				t.Errorf("expected IsEnded(%v) to be %v, got %v", tt.err, tt.expected, result)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	if triple.Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestVersionConstants(t *testing.T) {
	t.Parallel()

	if !triple.IsAtLeastVersion0_0_1 {
		t.Error("IsAtLeastVersion0_0_1 should be true")
	}

	if !triple.IsAtLeastVersion0_1_0 {
		t.Error("IsAtLeastVersion0_1_0 should be true")
	}

	if !triple.IsAtLeastVersion1_6_0 {
		t.Error("IsAtLeastVersion1_6_0 should be true")
	}
}

// Mock implementations for testing
type mockStreamingClientConn struct {
	receiveCount    int
	receiveErr      error
	responseHeader  http.Header
	responseTrailer http.Header
	msgReceived     bool
}

func (m *mockStreamingClientConn) Spec() triple.Spec {
	return triple.Spec{}
}

func (m *mockStreamingClientConn) Peer() triple.Peer {
	return triple.Peer{}
}

func (m *mockStreamingClientConn) Send(any) error {
	return nil
}

func (m *mockStreamingClientConn) RequestHeader() http.Header {
	return http.Header{}
}

func (m *mockStreamingClientConn) CloseRequest() error {
	return nil
}

func (m *mockStreamingClientConn) Receive(msg any) error {
	m.receiveCount++
	if m.receiveCount == 1 {
		m.msgReceived = true
		return m.receiveErr
	}
	return io.EOF
}

func (m *mockStreamingClientConn) ResponseHeader() http.Header {
	if m.responseHeader == nil {
		m.responseHeader = make(http.Header)
	}
	return m.responseHeader
}

func (m *mockStreamingClientConn) ResponseTrailer() http.Header {
	if m.responseTrailer == nil {
		m.responseTrailer = make(http.Header)
	}
	return m.responseTrailer
}

func (m *mockStreamingClientConn) CloseResponse() error {
	return nil
}

func TestHTTPClientInterface(t *testing.T) {
	t.Parallel()

	// Verify that *http.Client implements HTTPClient
	var _ triple.HTTPClient = (*http.Client)(nil)
}

func TestAnyRequestInterface(t *testing.T) {
	t.Parallel()

	// Verify that *Request implements AnyRequest
	req := triple.NewRequest("test")
	var _ triple.AnyRequest = req

	// Test that all AnyRequest methods can be called
	_ = req.Any()
	_ = req.Spec()
	_ = req.Peer()
	_ = req.Header()
}

func TestAnyResponseInterface(t *testing.T) {
	t.Parallel()

	// Verify that *Response implements AnyResponse
	resp := triple.NewResponse("test")
	var _ triple.AnyResponse = resp

	// Test that all AnyResponse methods can be called
	_ = resp.Any()
	_ = resp.Header()
	_ = resp.Trailer()
}

func TestStreamTypeBitOperations(t *testing.T) {
	t.Parallel()

	// Test that bidi is the combination of client and server
	bidi := triple.StreamTypeClient | triple.StreamTypeServer
	if bidi != triple.StreamTypeBidi {
		t.Errorf("expected bidi to be %v, got %v", triple.StreamTypeBidi, bidi)
	}

	// Test that we can check for client streaming
	if triple.StreamTypeBidi&triple.StreamTypeClient == 0 {
		t.Error("expected bidi to include client streaming")
	}

	// Test that we can check for server streaming
	if triple.StreamTypeBidi&triple.StreamTypeServer == 0 {
		t.Error("expected bidi to include server streaming")
	}

	// Test that unary has neither
	if triple.StreamTypeUnary&triple.StreamTypeClient != 0 {
		t.Error("expected unary not to include client streaming")
	}
	if triple.StreamTypeUnary&triple.StreamTypeServer != 0 {
		t.Error("expected unary not to include server streaming")
	}
}

func TestRequestResponseHeadersLazy(t *testing.T) {
	t.Parallel()

	t.Run("request_header_lazy_init", func(t *testing.T) {
		req := triple.NewRequest("test")
		header1 := req.Header()
		header2 := req.Header()

		// Should be the same instance
		header1.Set("Test", "value")
		if header2.Get("Test") != "value" {
			t.Error("headers should be the same instance")
		}
	})

	t.Run("response_header_lazy_init", func(t *testing.T) {
		resp := triple.NewResponse("test")
		header1 := resp.Header()
		header2 := resp.Header()

		// Should be the same instance
		header1.Set("Test", "value")
		if header2.Get("Test") != "value" {
			t.Error("headers should be the same instance")
		}
	})

	t.Run("response_trailer_lazy_init", func(t *testing.T) {
		resp := triple.NewResponse("test")
		trailer1 := resp.Trailer()
		trailer2 := resp.Trailer()

		// Should be the same instance
		trailer1.Set("Test", "value")
		if trailer2.Get("Test") != "value" {
			t.Error("trailers should be the same instance")
		}
	})
}

func TestIsEndedWithWrappedErrors(t *testing.T) {
	t.Parallel()

	// Test with properly wrapped error
	wrappedEOF := errors.Join(errors.New("context"), io.EOF)
	if !triple.IsEnded(wrappedEOF) {
		t.Error("expected IsEnded to return true for wrapped EOF")
	}

	// Test with non-EOF error
	otherErr := errors.New("other error")
	if triple.IsEnded(otherErr) {
		t.Error("expected IsEnded to return false for non-EOF error")
	}
}
