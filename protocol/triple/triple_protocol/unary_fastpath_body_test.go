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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

import (
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
)

// TestUnaryFastPathBodyReturnOnAsyncClose verifies that the pooled request
// body is returned to the pool exactly once and left clean when the transport
// closes it after http.Client.Do has returned: Do may return while the
// transport's background writer still reads the body, so the buffer must be
// recycled from the Close callback rather than the Do call site. Repeated
// Close (redirect / retry paths) and a concurrent Read must stay safe.
func TestUnaryFastPathBodyReturnOnAsyncClose(t *testing.T) {
	pool := newBufferPool()
	buf := pool.Get()
	buf.WriteString("payload")
	body := &unaryRequestBody{buf: buf, pool: pool}

	// The transport reads the whole body, then closes it after http.Client.Do
	// has already returned (abort path).
	if _, err := io.Copy(io.Discard, body); err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := body.Close(); err != nil {
		t.Fatalf("close body: %v", err)
	}

	// The buffer must have been returned to the pool.
	body.mu.Lock()
	returned := body.buf == nil
	body.mu.Unlock()
	if !returned {
		t.Fatal("request body buffer was not returned to the pool")
	}
	// Idempotent Close: a second or third Close from redirect / retry /
	// context-cancel paths must not double-return the buffer.
	if err := body.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if err := body.Close(); err != nil {
		t.Fatalf("third close: %v", err)
	}
	// After return, Read must surface EOF instead of the pooled buffer's
	// stale bytes (use-after-return guard).
	if _, err := body.Read(make([]byte, 4)); err != io.EOF {
		t.Fatalf("read after return = %v, want io.EOF", err)
	}
	// Reuse must be clean: whatever buffer the pool hands out next has been
	// Reset, so appending new payload must not carry this request's data.
	reuse := pool.Get()
	reuse.WriteString("new-payload")
	if got := reuse.String(); got != "new-payload" {
		t.Fatalf("reused buffer not clean after return: got %q", got)
	}
	reuse.Reset()
	pool.Put(reuse)
}

// TestUnaryFastPathBodyAbortServer verifies end to end that an early non-2xx
// response aborts the body write, returns the pooled buffer, and leaves the
// pool clean for a later request reusing it.
func TestUnaryFastPathBodyAbortServer(t *testing.T) {
	var first atomic.Int32
	mux := http.NewServeMux()
	mux.Handle("/connect.ping.v1.PingService/Ping", NewUnaryHandler(
		"/connect.ping.v1.PingService/Ping",
		func() any { return &pingv1.PingRequest{} },
		func(ctx context.Context, req *Request) (*Response, error) {
			// First request aborts early with a non-2xx error without reading
			// the request body, so the transport aborts the body write. Later
			// requests decode and echo the payload back, so a poisoned pool is
			// detectable via the echoed text.
			if first.Add(1) == 1 {
				err := NewError(CodePermissionDenied, errors.New("early response"))
				err.meta = make(http.Header)
				err.meta.Set("X-Triple-Error", "meta")
				return nil, err
			}
			pingReq, ok := req.Any().(*pingv1.PingRequest)
			if !ok {
				return nil, fmt.Errorf("unexpected request type %T", req.Any())
			}
			return NewResponse(&pingv1.PingResponse{Text: pingReq.Text}), nil
		},
	))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	httpClient := server.Client()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	header := make(http.Header)
	header.Set(headerContentType, "application/proto")
	header.Set(tripleUnaryHeaderAcceptCompression, "gzip")
	// One shared pool across requests, matching production where the conn
	// reuses the protocol-level BufferPool.
	pool := newBufferPool()

	// First call hits the early-response abort path.
	conn1 := newTripleClientConn(true, &protoBinaryCodec{}, httpClient, serverURL, header, pool)
	if err := conn1.Send(&pingv1.PingRequest{Text: "first-payload"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := conn1.CloseRequest(); err != nil {
		t.Fatalf("close request: %v", err)
	}
	resp1 := NewResponse(&pingv1.PingResponse{})
	if err := receiveUnaryResponse(conn1, resp1); err == nil {
		t.Fatal("expected error from early-response abort, got nil")
	} else {
		var connErr *Error
		if !errors.As(err, &connErr) || connErr.Code() != CodePermissionDenied {
			t.Fatalf("abort error = %v, want CodePermissionDenied", err)
		}
		if got := connErr.meta.Get("X-Triple-Error"); got != "meta" {
			t.Fatalf("abort error meta = %q, want %q", got, "meta")
		}
	}
	if err := conn1.CloseResponse(); err != nil {
		t.Fatalf("close response: %v", err)
	}

	// Second call reuses the same pool: the body must be clean.
	conn2 := newTripleClientConn(true, &protoBinaryCodec{}, httpClient, serverURL, header, pool)
	if err := conn2.Send(&pingv1.PingRequest{Text: "second-payload"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := conn2.CloseRequest(); err != nil {
		t.Fatalf("close request: %v", err)
	}
	resp2 := NewResponse(&pingv1.PingResponse{})
	if err := receiveUnaryResponse(conn2, resp2); err != nil {
		t.Fatalf("second call after abort: %v", err)
	}
	pingResp, ok := resp2.Any().(*pingv1.PingResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", resp2.Any())
	}
	if pingResp.Text != "second-payload" {
		t.Fatalf("second call body contaminated: got text %q", pingResp.Text)
	}
	if err := conn2.CloseResponse(); err != nil {
		t.Fatalf("close response: %v", err)
	}
}
