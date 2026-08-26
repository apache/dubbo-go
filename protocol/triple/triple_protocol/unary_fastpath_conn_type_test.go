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
	"net/http"
	"net/url"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

// TestUnaryFastPathNewConnType verifies that tripleClient.NewConn routes unary
// calls through unaryFastPathCall when enabled and duplexHTTPCall otherwise;
// streaming calls always use duplexHTTPCall.
func TestUnaryFastPathNewConnType(t *testing.T) {
	newClient := func(fast bool) *tripleClient {
		// CompressionPools must be non-nil; NewConn consults it when
		// building the marshaler even if compression is disabled.
		return &tripleClient{
			protocolClientParams: protocolClientParams{
				HTTPClient:       &http.Client{},
				URL:              &url.URL{Scheme: "http", Host: "example.com"},
				BufferPool:       newBufferPool(),
				Codec:            &protoBinaryCodec{},
				CompressionPools: newReadOnlyCompressionPools(map[string]*compressionPool{}, nil),
				UnaryFastPath:    fast,
			},
			peer: Peer{},
		}
	}
	assertCallType := func(t *testing.T, conn StreamingClientConn, want string) {
		t.Helper()
		translated, ok := conn.(*errorTranslatingClientConn)
		assert.True(t, ok, assert.Sprintf("unexpected conn wrapper %T", conn))
		unaryConn, ok := translated.StreamingClientConn.(*tripleUnaryClientConn)
		assert.True(t, ok, assert.Sprintf("unexpected unary conn %T", translated.StreamingClientConn))
		switch want {
		case "fastpath":
			_, ok := unaryConn.call.(*unaryFastPathCall)
			assert.True(t, ok, assert.Sprintf("unary call type = %T, want *unaryFastPathCall", unaryConn.call))
		case "duplex":
			_, ok := unaryConn.call.(*duplexHTTPCall)
			assert.True(t, ok, assert.Sprintf("unary call type = %T, want *duplexHTTPCall", unaryConn.call))
		default:
			// Guard against a misspelled want string silently passing:
			// the switch would otherwise skip every case and report success.
			t.Fatalf("assertCallType: unknown want %q", want)
		}
	}

	unarySpec := Spec{StreamType: StreamTypeUnary, Procedure: "/connect.ping.v1.PingService/Ping"}
	// WithUnaryFastPath enabled -> unary calls take the fast path.
	assertCallType(t, newClient(true).NewConn(context.Background(), unarySpec, make(http.Header)), "fastpath")
	// Default (option disabled) -> unary calls keep using duplexHTTPCall.
	assertCallType(t, newClient(false).NewConn(context.Background(), unarySpec, make(http.Header)), "duplex")
	// Streaming calls always use duplexHTTPCall, regardless of the switch.
	streamSpec := Spec{StreamType: StreamTypeBidi, Procedure: "/connect.ping.v1.PingService/Ping"}
	assertCallType(t, newClient(true).NewConn(context.Background(), streamSpec, make(http.Header)), "duplex")
}
