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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

const closeTestTimeout = time.Second

// newTestDuplexClientCall builds a duplexHTTPCall for a client-streaming RPC
// backed by a server that simply drains the request body and replies 200. It is
// the minimal setup needed to exercise the Write/CloseWrite/CloseRead lifecycle
// without pulling in the full codec/stream stack.
func newTestDuplexClientCall(t *testing.T, client HTTPClient, serverURL *url.URL) *duplexHTTPCall {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	call := newDuplexHTTPCall(
		ctx,
		client,
		serverURL,
		Spec{StreamType: StreamTypeClient, Procedure: "/triple.test.v1.RaceService/Sum"},
		http.Header{},
	)
	// makeRequest dereferences validateResponse, so it must be set.
	call.SetValidateResponse(func(*http.Response) *Error { return nil })
	return call
}

// TestDuplexHTTPCallConcurrentWriteAndCloseWrite is a regression GUARD -- not a
// reproducer -- for the concurrent Send + CloseAndReceive nil-pointer
// dereference fixed upstream in connect-go v1.19.2 (connectrpc/connect-go#919).
//
// Upstream created the request-body io.Pipe lazily, inside the CompareAndSwap
// winner branch of the first Send. A racing CloseWrite (reached via
// CloseAndReceive -> CloseRequest -> CloseWrite) could win that CAS first and
// return without creating the pipe, so a later Send dereferenced a nil
// requestBodyWriter and crashed at io.(*PipeWriter).Write.
//
// Dubbo-go's newDuplexHTTPCall instead allocates the pipe at construction and
// never reassigns requestBodyWriter, so that ordering is structurally
// impossible here. Two consequences worth stating plainly:
//
//   - On a correct tree this test ALWAYS passes and it cannot reproduce the
//     panic: with requestBodyWriter set once at construction, concurrent Write
//     and CloseWrite are safe in either order, so the interleaving below is not
//     load-bearing.
//   - What actually pins the invariant is assert.NotNil(call.requestBodyWriter)
//     plus the source comment on that field. If pipe creation is moved back into
//     a lazy/first-send path, the assertion fails and the concurrent Write
//     begins dereferencing nil again -- the regression we want to catch.
//
// At this layer Write/CloseWrite/CloseRead are what the Triple client-stream
// APIs call (grpcClientConn.Send -> marshaler -> Write, CloseRequest ->
// CloseWrite, CloseResponse -> CloseRead), so it covers the client-stream
// send + close lifecycle the issue is concerned with.
func TestDuplexHTTPCallConcurrentWriteAndCloseWrite(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	assert.Nil(t, err)

	const iterations = 10
	for range iterations {
		call := newTestDuplexClientCall(t, server.Client(), serverURL)

		// The invariant that makes the upstream nil-deref impossible here: the
		// write side of the request-body pipe is usable immediately after
		// construction, before any Write or CloseWrite call.
		assert.NotNil(t, call.requestBodyWriter)

		var (
			wg       sync.WaitGroup
			panicVal any
		)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicVal = r
				}
			}()
			_, _ = call.Write([]byte{0, 0, 0, 0, 0})
		}()

		// Race CloseWrite against the concurrent Write above. The order is
		// intentionally unspecified: both interleavings are safe once
		// requestBodyWriter is set at construction.
		_ = call.CloseWrite()
		wg.Wait()

		if panicVal != nil {
			t.Fatalf("concurrent Write during CloseWrite panicked: %v", panicVal)
		}
		_ = call.CloseRead()
	}
}

// TestDuplexHTTPCallConcurrentWriteCloseRace exercises concurrent Write,
// CloseWrite, CloseRead and the response accessors to guard the concurrent
// send + close/receive lifecycle (the scenario behind connect-go#919). Like the
// test above this is a guard, not a reproducer: on a correct tree it always
// passes. It is most valuable under `go test -race`, where it would flag a data
// race on requestBodyWriter or response if the construction-time pipe setup were
// ever broken.
func TestDuplexHTTPCallConcurrentWriteCloseRace(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		w.Header().Set("Test-Header", "value")
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	assert.Nil(t, err)

	const (
		iterations = 30
		writers    = 4
	)
	for range iterations {
		call := newTestDuplexClientCall(t, server.Client(), serverURL)

		// Same construction-time invariant as the guard test above; assert it
		// here too so a lazy-init regression fails with a clear assertion
		// instead of panicking on a nil writer inside the goroutines below.
		assert.NotNil(t, call.requestBodyWriter)

		var wg sync.WaitGroup
		for range writers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// After the send side is closed Write returns io.EOF rather
				// than panicking; any panic here is a real regression and is
				// allowed to fail the test.
				_, _ = call.Write([]byte{0, 0, 0, 0, 0})
			}()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = call.CloseWrite()
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			// These block until the response is ready and then read the
			// response that makeRequest stored; exercises the response
			// happens-before edge established by closing responseReady.
			_ = call.ResponseHeader()
			_ = call.ResponseTrailer()
		}()
		wg.Wait()
		_ = call.CloseRead()
	}
}

// TestDuplexHTTPCallCloseReadDoesNotDrainResponseBody verifies that CloseRead
// closes the response body directly instead of reading until EOF. This guards
// the stream-close hang fixed in connect-go#791.
//
// A streaming peer may keep the response open after it stops sending data, so
// draining the body during close can block forever. Callers that need final
// trailers should read to EOF before closing the read side.
func TestDuplexHTTPCallCloseReadDoesNotDrainResponseBody(t *testing.T) {
	t.Parallel()
	body := newBlockingReadCloser()
	call := &duplexHTTPCall{
		ctx:           context.Background(),
		responseReady: make(chan struct{}),
		response: &http.Response{
			Body:    body,
			Trailer: make(http.Header),
		},
	}
	close(call.responseReady)

	done := make(chan error, 1)
	go func() {
		done <- call.CloseRead()
	}()

	select {
	case err := <-done:
		assert.Nil(t, err)
	case <-body.readStarted:
		body.unblock()
		assert.Nil(t, <-done)
		t.Fatal("CloseRead attempted to drain the response body")
	case <-time.After(closeTestTimeout):
		body.unblock()
		assert.Nil(t, <-done)
		t.Fatal("CloseRead blocked while closing the response body")
	}

	select {
	case <-body.closed:
	default:
		t.Fatal("CloseRead did not close the response body")
	}
}

// blockingReadCloser records attempted reads and then blocks. It lets the test
// catch drain-before-close behavior without depending on a real stalled stream.
type blockingReadCloser struct {
	readStarted chan struct{}
	readUnblock chan struct{}
	closed      chan struct{}

	startReadOnce sync.Once
	unblockOnce   sync.Once
	closeOnce     sync.Once
}

func newBlockingReadCloser() *blockingReadCloser {
	return &blockingReadCloser{
		readStarted: make(chan struct{}),
		readUnblock: make(chan struct{}),
		closed:      make(chan struct{}),
	}
}

func (b *blockingReadCloser) Read([]byte) (int, error) {
	b.startReadOnce.Do(func() {
		close(b.readStarted)
	})
	<-b.readUnblock
	return 0, io.EOF
}

func (b *blockingReadCloser) Close() error {
	b.closeOnce.Do(func() {
		close(b.closed)
	})
	return nil
}

func (b *blockingReadCloser) unblock() {
	b.unblockOnce.Do(func() {
		close(b.readUnblock)
	})
}
