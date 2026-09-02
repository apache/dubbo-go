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
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

import (
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
)

// behaviorHTTPClient adapts a closure to HTTPClient so tests can capture or
// fake the wire behavior of the fast path.
type behaviorHTTPClient struct {
	do func(*http.Request) (*http.Response, error)
}

func (c *behaviorHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.do(req)
}

// newPingHandler builds a Triple unary handler in-package, avoiding the
// generated pingv1connect package (which would create an import cycle).
func newPingHandler() *Handler {
	return NewUnaryHandler(
		"/connect.ping.v1.PingService/Ping",
		func() any { return &pingv1.PingRequest{} },
		func(ctx context.Context, req *Request) (*Response, error) {
			pingReq, ok := req.Any().(*pingv1.PingRequest)
			if !ok {
				return nil, fmt.Errorf("unexpected request type %T", req.Any())
			}
			return NewResponse(&pingv1.PingResponse{Text: pingReq.Text}), nil
		},
		WithCompression(
			compressionGzip,
			func() Decompressor { return &gzip.Reader{} },
			func() Compressor { return gzip.NewWriter(io.Discard) },
		),
	)
}

// pingCompressionPools registers gzip so response compression is negotiated,
// and is shared by the behavior tests to keep the pools warm.
var pingCompressionPools = newReadOnlyCompressionPools(
	map[string]*compressionPool{
		compressionGzip: newCompressionPool(
			func() Decompressor { return &gzip.Reader{} },
			func() Compressor { return gzip.NewWriter(io.Discard) },
		),
	},
	[]string{compressionGzip},
)

// newBehaviorCall constructs an unaryFastPathCall wired to the given client,
// mirroring tripleClient.NewConn's construction.
func newBehaviorCall(ctx context.Context, httpClient HTTPClient, serverURL *url.URL, pool *bufferPool) *unaryFastPathCall {
	call := newUnaryFastPathCall(
		ctx,
		httpClient,
		serverURL,
		Spec{
			StreamType: StreamTypeUnary,
			Procedure:  "/connect.ping.v1.PingService/Ping",
		},
		make(http.Header),
		pool,
	)
	call.SetValidateResponse(func(*http.Response) *Error { return nil })
	return call
}

// newTripleClientConn constructs a tripleClient and routes the unary call
// through NewConn, exercising the real production entry point.
func newTripleClientConn(useFastPath bool, codec Codec, httpClient HTTPClient, serverURL *url.URL, header http.Header, pool *bufferPool) StreamingClientConn {
	client := &tripleClient{
		protocolClientParams: protocolClientParams{
			HTTPClient:       httpClient,
			URL:              serverURL,
			BufferPool:       pool,
			Codec:            codec,
			CompressionPools: pingCompressionPools,
			UnaryFastPath:    useFastPath,
		},
		peer: Peer{Addr: serverURL.String(), Protocol: ProtocolTriple},
	}
	return client.NewConn(context.Background(), Spec{
		StreamType: StreamTypeUnary,
		Procedure:  "/connect.ping.v1.PingService/Ping",
	}, header)
}

// TestUnaryFastPathWriteAccumulates verifies that Write appends into the
// pooled body without touching the network, and that CloseWrite hands the
// whole payload to the transport once, with an exact Content-Length.
func TestUnaryFastPathWriteAccumulates(t *testing.T) {
	pool := newBufferPool()
	serverURL, err := url.Parse("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	var (
		calls      atomic.Int32
		wireBody   []byte
		contentLen int64
	)
	httpClient := &behaviorHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		data, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		if err := req.Body.Close(); err != nil {
			return nil, err
		}
		wireBody = data
		contentLen = req.ContentLength
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}}

	call := newBehaviorCall(context.Background(), httpClient, serverURL, pool)
	if n, err := call.Write([]byte("hello")); n != 5 || err != nil {
		t.Fatalf("first write = (n=%d, err=%v), want (5, nil)", n, err)
	}
	if n, err := call.Write([]byte("-world")); n != 6 || err != nil {
		t.Fatalf("second write = (n=%d, err=%v), want (6, nil)", n, err)
	}
	if err := call.CloseWrite(); err != nil {
		t.Fatalf("close request: %v", err)
	}
	// A second CloseWrite must not dispatch the request again.
	if err := call.CloseWrite(); err != nil {
		t.Fatalf("second close request: %v", err)
	}
	if err := call.CloseRead(); err != nil {
		t.Fatalf("close response: %v", err)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("transport saw %d requests, want 1", got)
	}
	if got := string(wireBody); got != "hello-world" {
		t.Fatalf("wire body = %q, want %q", got, "hello-world")
	}
	if contentLen != int64(len("hello-world")) {
		t.Fatalf("content length = %d, want %d", contentLen, len("hello-world"))
	}
}

// TestUnaryFastPathEmptyBodyUsesNoBody verifies that a call with no Write uses
// http.NoBody and a zero Content-Length, skipping the pooled buffer.
func TestUnaryFastPathEmptyBodyUsesNoBody(t *testing.T) {
	pool := newBufferPool()
	serverURL, err := url.Parse("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	var (
		calls      atomic.Int32
		sawNoBody  bool
		contentLen int64
	)
	httpClient := &behaviorHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		sawNoBody = req.Body == http.NoBody
		contentLen = req.ContentLength
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}}

	call := newBehaviorCall(context.Background(), httpClient, serverURL, pool)
	if err := call.CloseWrite(); err != nil {
		t.Fatalf("close request: %v", err)
	}
	if err := call.CloseRead(); err != nil {
		t.Fatalf("close response: %v", err)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("transport saw %d requests, want 1", got)
	}
	if !sawNoBody {
		t.Fatal("empty call did not use http.NoBody")
	}
	if contentLen != 0 {
		t.Fatalf("content length = %d, want 0", contentLen)
	}
}

// TestUnaryFastPathConcurrentReadClose verifies that a Read racing a Close
// never panics or reads recycled bytes: both paths are guarded by the body
// mutex and surface a clean EOF.
func TestUnaryFastPathConcurrentReadClose(t *testing.T) {
	pool := newBufferPool()
	buf := pool.Get()
	buf.WriteString("payload")
	body := &unaryRequestBody{buf: buf, pool: pool}

	const (
		readers = 8
		closers = 8
	)
	var wg sync.WaitGroup
	wg.Add(readers + closers)
	for range closers {
		go func() {
			defer wg.Done()
			_ = body.Close()
		}()
	}
	for range readers {
		go func() {
			defer wg.Done()
			for {
				if _, err := body.Read(make([]byte, 4)); err != nil {
					return
				}
			}
		}()
	}
	wg.Wait()
}

// TestUnaryFastPathRequestHeaderConcurrent verifies that RequestHeader is
// safe to race with Write and CloseWrite, per the streaming client conn
// contract's write-side group.
func TestUnaryFastPathRequestHeaderConcurrent(t *testing.T) {
	httpClient, serverURL, _, _ := concurrentEchoServer(t)
	pool := newBufferPool()
	call := newBehaviorCall(context.Background(), httpClient, serverURL, pool)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		for range 50 {
			_ = call.Header()
		}
	}()
	go func() {
		defer wg.Done()
		for range 50 {
			_, _ = call.Write([]byte("payload"))
		}
	}()
	go func() {
		defer wg.Done()
		_ = call.CloseWrite()
	}()
	wg.Wait()

	if call.Header() == nil {
		t.Fatal("header must remain readable after CloseWrite")
	}
	if _, err := io.Copy(io.Discard, call); err != nil {
		t.Fatalf("read response: %v", err)
	}
	if err := call.CloseRead(); err != nil {
		t.Fatalf("close response: %v", err)
	}
}

// TestUnaryFastPathWriteAfterTransportError verifies that a failed request
// rejects later writes with io.EOF and Read surfaces the wrapped
// CodeUnavailable error.
func TestUnaryFastPathWriteAfterTransportError(t *testing.T) {
	pool := newBufferPool()
	serverURL, err := url.Parse("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	transportErr := errors.New("transport failed")
	httpClient := &behaviorHTTPClient{do: func(*http.Request) (*http.Response, error) {
		return nil, transportErr
	}}

	call := newBehaviorCall(context.Background(), httpClient, serverURL, pool)
	if _, writeErr := call.Write([]byte("payload")); writeErr != nil {
		t.Fatalf("write: %v", writeErr)
	}
	if closeErr := call.CloseWrite(); closeErr != nil {
		t.Fatalf("close request: %v", closeErr)
	}
	n, err := call.Write([]byte("late"))
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("write after transport error = (n=%d, err=%v), want (0, io.EOF)", n, err)
	}
	_, err = call.Read(make([]byte, 16))
	var connErr *Error
	if !errors.As(err, &connErr) || connErr.Code() != CodeUnavailable {
		t.Fatalf("read after transport error = %v, want CodeUnavailable", err)
	}
}

// TestUnaryFastPathSetErrorRejectsWrite verifies that a stored error rejects
// writes before the body is sent. Unlike duplexHTTPCall's io.EOF, the fast
// path surfaces the stored error so the underlying failure stays visible.
func TestUnaryFastPathSetErrorRejectsWrite(t *testing.T) {
	pool := newBufferPool()
	serverURL, err := url.Parse("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	httpClient := &behaviorHTTPClient{do: func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}}

	call := newBehaviorCall(context.Background(), httpClient, serverURL, pool)
	storedErr := errors.New("stored")
	call.SetError(storedErr)
	n, err := call.Write([]byte("payload"))
	if n != 0 || !errors.Is(err, storedErr) {
		t.Fatalf("write after SetError = (n=%d, err=%v), want (0, %v)", n, err, storedErr)
	}
}

// TestUnaryFastPathWireConsistent verifies that the fast path sends the same
// request body bytes as the duplex path for the same message.
func TestUnaryFastPathWireConsistent(t *testing.T) {
	httpClient, serverURL, bodies, bodiesMu := concurrentEchoServer(t)
	header := make(http.Header)
	header.Set(headerContentType, "application/proto")
	pool := newBufferPool()

	for _, tc := range []struct {
		name string
		fast bool
	}{
		{"duplex", false},
		{"fastpath", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := newTripleClientConn(tc.fast, &protoBinaryCodec{}, httpClient, serverURL, header, pool)
			if err := conn.Send(&pingv1.PingRequest{Text: "wire-consistency"}); err != nil {
				t.Fatalf("send: %v", err)
			}
			if err := conn.CloseRequest(); err != nil {
				t.Fatalf("close request: %v", err)
			}
			// The echo server returns an empty 200 body, which unmarshals
			// into an empty message; only the request side matters here.
			if err := conn.Receive(&pingv1.PingResponse{}); err != nil {
				t.Fatalf("receive: %v", err)
			}
			if err := conn.CloseResponse(); err != nil {
				t.Fatalf("close response: %v", err)
			}
		})
	}

	bodiesMu.Lock()
	defer bodiesMu.Unlock()
	if len(*bodies) != 2 {
		t.Fatalf("server received %d requests, want 2", len(*bodies))
	}
	if (*bodies)[0] != (*bodies)[1] {
		t.Fatalf("wire mismatch: duplex %q vs fastpath %q", (*bodies)[0], (*bodies)[1])
	}
}

// TestUnaryFastPathContextCancel verifies that canceling the call context
// aborts the in-flight request and Read surfaces a CodeCanceled error.
func TestUnaryFastPathContextCancel(t *testing.T) {
	pool := newBufferPool()
	serverURL, err := url.Parse("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	httpClient := &behaviorHTTPClient{do: func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	}}

	call := newBehaviorCall(ctx, httpClient, serverURL, pool)
	if _, writeErr := call.Write([]byte("payload")); writeErr != nil {
		t.Fatalf("write: %v", writeErr)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = call.CloseWrite()
	}()
	cancel()
	<-done
	_, err = call.Read(make([]byte, 16))
	var connErr *Error
	if !errors.As(err, &connErr) || connErr.Code() != CodeCanceled {
		t.Fatalf("read after cancel = %v, want CodeCanceled", err)
	}
}

// TestUnaryFastPathEndToEnd verifies that the fast path is wire- and
// response-compatible with the duplex path: same message, headers, and
// trailers on a real HTTP/2 server.
func TestUnaryFastPathEndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/connect.ping.v1.PingService/Ping", NewUnaryHandler(
		"/connect.ping.v1.PingService/Ping",
		func() any { return &pingv1.PingRequest{} },
		func(ctx context.Context, req *Request) (*Response, error) {
			pingReq, ok := req.Any().(*pingv1.PingRequest)
			if !ok {
				return nil, fmt.Errorf("unexpected request type %T", req.Any())
			}
			resp := NewResponse(&pingv1.PingResponse{Text: pingReq.Text})
			resp.Header().Set("X-Triple-Echo", "header")
			resp.Trailer().Set("X-Triple-Echo-Trailer", "trailer")
			return resp, nil
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
	pool := newBufferPool()

	for _, tc := range []struct {
		name string
		fast bool
	}{
		{"duplex", false},
		{"fastpath", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := newTripleClientConn(tc.fast, &protoBinaryCodec{}, httpClient, serverURL, header, pool)
			if err := conn.Send(&pingv1.PingRequest{Text: "hello"}); err != nil {
				t.Fatalf("send: %v", err)
			}
			if err := conn.CloseRequest(); err != nil {
				t.Fatalf("close request: %v", err)
			}
			resp := NewResponse(&pingv1.PingResponse{})
			if err := receiveUnaryResponse(conn, resp); err != nil {
				t.Fatalf("receive: %v", err)
			}
			pingResp, ok := resp.Any().(*pingv1.PingResponse)
			if !ok {
				t.Fatalf("unexpected response type %T", resp.Any())
			}
			if pingResp.Text != "hello" {
				t.Fatalf("response text = %q, want %q", pingResp.Text, "hello")
			}
			if got := resp.Header().Get("X-Triple-Echo"); got != "header" {
				t.Fatalf("response header X-Triple-Echo = %q, want %q", got, "header")
			}
			if got := resp.Trailer().Get("X-Triple-Echo-Trailer"); got != "trailer" {
				t.Fatalf("response trailer X-Triple-Echo-Trailer = %q, want %q", got, "trailer")
			}
			if err := conn.CloseResponse(); err != nil {
				t.Fatalf("close response: %v", err)
			}
		})
	}
}

// TestUnaryFastPathProtoJSONCodec verifies that the fast path carries
// JSON-encoded payloads end to end, matching the duplex path's codec support.
func TestUnaryFastPathProtoJSONCodec(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/connect.ping.v1.PingService/Ping", newPingHandler())
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
	header.Set(headerContentType, tripleUnaryContentTypeJSON)
	header.Set(tripleUnaryHeaderAcceptCompression, "gzip")
	pool := newBufferPool()

	for _, tc := range []struct {
		name string
		fast bool
	}{
		{"duplex", false},
		{"fastpath", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := newTripleClientConn(tc.fast, &protoJSONCodec{name: codecNameJSON}, httpClient, serverURL, header, pool)
			if err := conn.Send(&pingv1.PingRequest{Text: "hello"}); err != nil {
				t.Fatalf("send: %v", err)
			}
			if err := conn.CloseRequest(); err != nil {
				t.Fatalf("close request: %v", err)
			}
			resp := NewResponse(&pingv1.PingResponse{})
			if err := receiveUnaryResponse(conn, resp); err != nil {
				t.Fatalf("receive: %v", err)
			}
			pingResp, ok := resp.Any().(*pingv1.PingResponse)
			if !ok {
				t.Fatalf("unexpected response type %T", resp.Any())
			}
			if pingResp.Text != "hello" {
				t.Fatalf("response text = %q, want %q", pingResp.Text, "hello")
			}
			if err := conn.CloseResponse(); err != nil {
				t.Fatalf("close response: %v", err)
			}
		})
	}
}

// TestUnaryFastPathGRPCRoutesUnary verifies that grpcClient.NewConn routes
// unary calls through unaryFastPathCall when the switch is enabled and keeps
// duplexHTTPCall otherwise; streaming calls always use duplexHTTPCall.
func TestUnaryFastPathGRPCRoutesUnary(t *testing.T) {
	newClient := func(fast bool) *grpcClient {
		return &grpcClient{
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
		if !ok {
			t.Fatalf("unexpected conn wrapper %T", conn)
		}
		grpcConn, ok := translated.StreamingClientConn.(*grpcClientConn)
		if !ok {
			t.Fatalf("unexpected grpc conn %T", translated.StreamingClientConn)
		}
		switch want {
		case "fastpath":
			if _, ok := grpcConn.call.(*unaryFastPathCall); !ok {
				t.Fatalf("grpc call type = %T, want *unaryFastPathCall", grpcConn.call)
			}
		case "duplex":
			if _, ok := grpcConn.call.(*duplexHTTPCall); !ok {
				t.Fatalf("grpc call type = %T, want *duplexHTTPCall", grpcConn.call)
			}
		default:
			// Guard against a misspelled want string silently passing.
			t.Fatalf("assertCallType: unknown want %q", want)
		}
	}

	unarySpec := Spec{StreamType: StreamTypeUnary, Procedure: "/connect.ping.v1.PingService/Ping"}
	// Enabled -> unary calls take the fast path on the gRPC protocol too.
	assertCallType(t, newClient(true).NewConn(context.Background(), unarySpec, make(http.Header)), "fastpath")
	// Disabled -> unary calls keep using duplexHTTPCall.
	assertCallType(t, newClient(false).NewConn(context.Background(), unarySpec, make(http.Header)), "duplex")
	// Streaming calls always use duplexHTTPCall, regardless of the switch.
	streamSpec := Spec{StreamType: StreamTypeBidi, Procedure: "/connect.ping.v1.PingService/Ping"}
	assertCallType(t, newClient(true).NewConn(context.Background(), streamSpec, make(http.Header)), "duplex")
}
