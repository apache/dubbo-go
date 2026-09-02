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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

// concurrentEchoServer starts an HTTP/2 server that records every request body
// it receives, so tests can assert on what actually reached the wire.
func concurrentEchoServer(t *testing.T) (HTTPClient, *url.URL, *[]string, *sync.Mutex) {
	t.Helper()
	var bodies []string
	var mu sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("/connect.ping.v1.PingService/Ping", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(b))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return server.Client(), serverURL, &bodies, &mu
}

func newConcurrentCall(httpClient HTTPClient, serverURL *url.URL, pool *bufferPool) *unaryFastPathCall {
	call := newUnaryFastPathCall(
		context.Background(),
		httpClient,
		serverURL,
		Spec{
			StreamType: StreamTypeUnary,
			Procedure:  "/connect.ping.v1.PingService/Ping",
		},
		make(http.Header),
		pool,
	)
	// The conn layer injects validateResponse via SetValidateResponse; mirror
	// that here.
	call.SetValidateResponse(func(*http.Response) *Error { return nil })
	return call
}

// TestUnaryFastPathWriteCloseConcurrent verifies the write-side concurrency
// contract: a Write racing a CloseWrite must fully succeed or fail with
// io.EOF, and the server must never see a torn body.
func TestUnaryFastPathWriteCloseConcurrent(t *testing.T) {
	httpClient, serverURL, bodies, bodiesMu := concurrentEchoServer(t)
	pool := newBufferPool()
	payload := []byte("payload-concurrent")

	const iterations = 30
	for i := range iterations {
		call := newConcurrentCall(httpClient, serverURL, pool)
		var (
			writeN   int
			writeErr error
			wg       sync.WaitGroup
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			writeN, writeErr = call.Write(payload)
		}()
		go func() {
			defer wg.Done()
			// CloseWrite always returns nil; transport failures surface from
			// Read below, as on the duplex path.
			_ = call.CloseWrite()
		}()
		wg.Wait()

		// Drain the response so the request body Close callback runs and the
		// pooled buffer returns to the pool.
		if _, err := io.Copy(io.Discard, call); err != nil {
			t.Fatalf("iter %d: read response: %v", i, err)
		}
		if err := call.CloseRead(); err != nil {
			t.Fatalf("iter %d: close response: %v", i, err)
		}

		// Write must have fully succeeded, or been rejected with io.EOF
		// because the racing CloseWrite already sent the body.
		if writeErr != nil && !errors.Is(writeErr, io.EOF) {
			t.Fatalf("iter %d: write error = %v, want nil or io.EOF", i, writeErr)
		}
		if writeErr == nil && writeN != len(payload) {
			t.Fatalf("iter %d: write n = %d, want %d", i, writeN, len(payload))
		}
	}

	bodiesMu.Lock()
	defer bodiesMu.Unlock()
	if len(*bodies) != iterations {
		t.Fatalf("server received %d requests, want %d", len(*bodies), iterations)
	}
	for _, body := range *bodies {
		if body != "" && body != string(payload) {
			t.Fatalf("server received torn body %q, want empty or the full payload", body)
		}
	}
}

// TestUnaryFastPathWriteAfterClose verifies that Write after CloseWrite has
// dispatched the body fails with io.EOF instead of racing the transport.
func TestUnaryFastPathWriteAfterClose(t *testing.T) {
	httpClient, serverURL, _, _ := concurrentEchoServer(t)
	pool := newBufferPool()

	call := newConcurrentCall(httpClient, serverURL, pool)
	if err := call.CloseWrite(); err != nil {
		t.Fatalf("close request: %v", err)
	}
	n, err := call.Write([]byte("late-write"))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("write after close = (n=%d, err=%v), want (0, io.EOF)", n, err)
	}
	if n != 0 {
		t.Fatalf("write after close n = %d, want 0", n)
	}

	if _, err := io.Copy(io.Discard, call); err != nil {
		t.Fatalf("read response: %v", err)
	}
	if err := call.CloseRead(); err != nil {
		t.Fatalf("close response: %v", err)
	}
}
