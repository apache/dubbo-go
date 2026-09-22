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
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

// writeCountRecorder records how many Write calls and total bytes reached the
// underlying writer, so tests can verify how many handshakes were amortized.
type writeCountRecorder struct {
	writes int
	total  int
}

func (w *writeCountRecorder) Write(p []byte) (int, error) {
	w.writes++
	w.total += len(p)
	return len(p), nil
}

// TestStreamBufferWriterCoalesces verifies that several small messages arriving
// before Flush are aggregated into a single underlying Write, i.e. the
// per-message io.Pipe handshake is collapsed by the buffer.
func TestStreamBufferWriterCoalesces(t *testing.T) {
	under := &writeCountRecorder{}
	w := newStreamBufferWriter(under)
	msg := make([]byte, 100)

	for range 3 {
		if _, err := w.Write(msg); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	if under.writes != 0 {
		t.Fatalf("expected nothing flushed yet, got %d underlying write(s)", under.writes)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if under.writes != 1 {
		t.Fatalf("expected a single aggregated write, got %d", under.writes)
	}
	if under.total != 300 {
		t.Fatalf("expected 300 bytes delivered, got %d", under.total)
	}
}

// TestStreamBufferWriterFlushesAtLimit verifies that the buffer auto-flushes
// once accumulated bytes reach the capacity, keeping memory bounded without an
// explicit Flush call.
func TestStreamBufferWriterFlushesAtLimit(t *testing.T) {
	under := &writeCountRecorder{}
	w := newStreamBufferWriter(under)
	msg := make([]byte, 100)

	for i := range (defaultStreamWriteBufSize / 100) + 1 {
		if _, err := w.Write(msg); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}
	// 327 writes of 100 B leave the buffer just under the limit; the final
	// write is the single threshold crossing, so exactly one flush carries the
	// whole batch and nothing is left buffered.
	if under.writes != 1 {
		t.Fatalf("expected exactly one threshold-triggered flush, got %d", under.writes)
	}
	if under.total != len(msg)*((defaultStreamWriteBufSize/100)+1) {
		t.Fatalf("expected %d bytes to reach the writer, got %d",
			len(msg)*((defaultStreamWriteBufSize/100)+1), under.total)
	}
}

// TestStreamBufferWriterLargeMessageWritesDirectly verifies that a single
// message at or above the buffer capacity bypasses buffering entirely, so a
// large payload is not staged in memory twice.
func TestStreamBufferWriterLargeMessageWritesDirectly(t *testing.T) {
	under := &writeCountRecorder{}
	w := newStreamBufferWriter(under)
	large := make([]byte, defaultStreamWriteBufSize) // >= limit

	if _, err := w.Write(large); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if under.writes != 1 {
		t.Fatalf("expected a direct write for an over-limit message, got %d", under.writes)
	}
	if under.total != len(large) {
		t.Fatalf("expected %d bytes, got %d", len(large), under.total)
	}
}

// TestStreamBufferWriterFlushEmptyIsNoOp verifies that flushing an empty buffer
// performs no underlying Write.
func TestStreamBufferWriterFlushEmptyIsNoOp(t *testing.T) {
	under := &writeCountRecorder{}
	w := newStreamBufferWriter(under)
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if under.writes != 0 {
		t.Fatalf("expected no write for an empty buffer, got %d", under.writes)
	}
}

// TestStreamBufferWriterWriteAfterError verifies that once a flush has failed,
// the error sticks: subsequent Writes report the same underlying error.
func TestStreamBufferWriterWriteAfterError(t *testing.T) {
	boom := fmt.Errorf("boom")
	under := errorWriter{err: boom}
	w := newStreamBufferWriter(under)

	msg := make([]byte, 4096)
	if _, err := w.Write(msg); err != nil {
		t.Fatalf("small write should buffer, got error: %v", err)
	}
	// The failure surfaces when the accumulated buffer is flushed.
	if err := w.Flush(); err == nil {
		t.Fatal("expected the underlying write failure to surface on Flush")
	}
	// Once a flush fails the buffer stays failed: subsequent writes report
	// the same error.
	if _, err := w.Write(msg); err == nil || err != boom {
		t.Fatalf("expected the sticky error %v, got %v", boom, err)
	}
}

// errorWriter always fails with a fixed error.
type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

// TestStreamBufferWriterConcurrent verifies that concurrent Send (Write) and
// Flush (as issued by CloseRequest) calls are race-free and do not drop or
// corrupt data.
func TestStreamBufferWriterConcurrent(t *testing.T) {
	under := &atomicWriteCountRecorder{}
	w := newStreamBufferWriter(under)
	msg := make([]byte, 128)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				_, _ = w.Write(msg)
			}
		})
	}
	// A flusher running concurrently with writers exercises the lock on Flush.
	wg.Go(func() {
		for range 50 {
			_ = w.Flush()
		}
	})
	wg.Wait()
	if err := w.Flush(); err != nil {
		t.Fatalf("final Flush: %v", err)
	}
	if got := under.total.Load(); got != int64(len(msg)*8*100) {
		t.Fatalf("expected %d bytes delivered, got %d", int64(len(msg)*8*100), got)
	}
}

// TestStreamBufferWriterWriteAfterClose verifies that a Send arriving after the
// buffer has been sealed fails with io.EOF, the same error an un-buffered
// duplexHTTPCall reports once CloseWrite has run, and that the pending tail
// was still delivered by Close.
func TestStreamBufferWriterWriteAfterClose(t *testing.T) {
	under := &writeCountRecorder{}
	w := newStreamBufferWriter(under)
	msg := make([]byte, 100)

	if _, err := w.Write(msg); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if under.total != len(msg) {
		t.Fatalf("expected the pending tail to flush on Close, got %d bytes", under.total)
	}
	if _, err := w.Write(msg); !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF after Close, got %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close must be idempotent, got %v", err)
	}
	if under.total != len(msg) {
		t.Fatalf("second Close must not re-send, got %d bytes", under.total)
	}
}

// TestStreamBufferWriterCloseNeverDropsRacingWrite verifies the
// StreamingClientConn concurrency contract, which allows Send to race with
// CloseRequest: a Write racing with Close must either land in Close's final
// flush or fail with io.EOF, but must never be accepted and then
// silently dropped. The invariant checked is that the bytes reaching the
// underlying writer equal the bytes the caller was told were accepted.
func TestStreamBufferWriterCloseNeverDropsRacingWrite(t *testing.T) {
	msg := make([]byte, 128)
	for attempt := range 200 {
		under := &atomicWriteCountRecorder{}
		w := newStreamBufferWriter(under)
		accepted := &atomic.Int64{}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, err := w.Write(msg); err == nil {
				accepted.Add(1)
			}
		}()
		go func() {
			defer wg.Done()
			_ = w.Close()
		}()
		wg.Wait()

		if got, want := under.total.Load(), accepted.Load()*int64(len(msg)); got != want {
			t.Fatalf("attempt %d: %d accepted byte(s) but %d reached the writer (message dropped)",
				attempt, want, got)
		}
	}
}

// atomicWriteCountRecorder is concurrency-safe for use under -race.
type atomicWriteCountRecorder struct {
	total atomic.Int64
}

func (w *atomicWriteCountRecorder) Write(p []byte) (int, error) {
	w.total.Add(int64(len(p)))
	return len(p), nil
}

// countWriterFunc adapts a counting func to the io.Writer interface.
type countWriterFunc func([]byte) (int, error)

func (f countWriterFunc) Write(p []byte) (int, error) { return f(p) }

// TestWriteBufferingStreamingFlushOnClose verifies the end-to-end assembly:
// a streamBufferWriter wrapping a real duplexHTTPCall must deliver every Send to
// the server, including the un-flushed tail that CloseRequest flushes before the
// write side closes. It exercises the exact Send / CloseRequest sequence the
// Triple streaming client performs with WithWriteBuffering enabled.
func TestWriteBufferingStreamingFlushOnClose(t *testing.T) {
	t.Parallel()

	received := &atomic.Int64{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(countWriterFunc(func(p []byte) (int, error) {
			received.Add(int64(len(p)))
			return len(p), nil
		}), r.Body)
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	const (
		msgCount = 1000
		msgSize  = 100
	)
	// 100 KiB total, far above the 32 KiB threshold, so the server must observe
	// a flushing mid-stream AND the remainder on CloseRequest.
	msg := make([]byte, msgSize)

	call := newTestDuplexClientCall(t, server.Client(), serverURL)
	bw := newStreamBufferWriter(call)
	for i := range msgCount {
		if _, err := bw.Write(msg); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	// CloseRequest: seal with a final flush, then close the write side.
	if err := bw.Close(); err != nil {
		t.Fatalf("close before CloseWrite: %v", err)
	}
	_ = call.CloseWrite()
	_ = call.CloseRead()

	if want := int64(msgCount * msgSize); received.Load() != want {
		t.Fatalf("server received %d bytes, want %d (tail must not be dropped)",
			received.Load(), want)
	}
}

// newBufferingParams builds the minimal protocolClientParams NewConn needs to
// assemble a client conn. CompressionPools must be non-nil because NewConn
// consults it when building the marshaler even when compression is disabled.
// Callers choose whether write buffering and the unary fast path are active.
func newBufferingParams(buffering, unaryFastPath bool) protocolClientParams {
	return protocolClientParams{
		HTTPClient:       &http.Client{},
		URL:              &url.URL{Scheme: "http", Host: "example.com"},
		BufferPool:       newBufferPool(),
		Codec:            &protoBinaryCodec{},
		CompressionPools: newReadOnlyCompressionPools(map[string]*compressionPool{}, nil),
		UnaryFastPath:    unaryFastPath,
		WriteBuffering:   buffering,
	}
}

// connWriteBuffer returns the write buffer backing conn's envelope writer, or
// nil when the conn writes straight to the call.
func connWriteBuffer(t *testing.T, conn StreamingClientConn) *streamBufferWriter {
	t.Helper()
	translated, ok := conn.(*errorTranslatingClientConn)
	assert.True(t, ok, assert.Sprintf("unexpected conn wrapper %T", conn))
	switch typed := translated.StreamingClientConn.(type) {
	case *grpcClientConn:
		if typed.writeBuffer != nil {
			assert.True(t, typed.marshaler.writer == io.Writer(typed.writeBuffer),
				assert.Sprintf("gRPC conn envelope writer is not routed through its write buffer"))
		} else {
			assert.True(t, typed.marshaler.writer == io.Writer(typed.call),
				assert.Sprintf("unbuffered gRPC conn envelope writer is not the call itself"))
		}
		return typed.writeBuffer
	case *tripleUnaryClientConn:
		if typed.writeBuffer != nil {
			assert.True(t, typed.marshaler.writer == io.Writer(typed.writeBuffer),
				assert.Sprintf("triple conn envelope writer is not routed through its write buffer"))
		} else {
			assert.True(t, typed.marshaler.writer == io.Writer(typed.call),
				assert.Sprintf("unbuffered triple conn envelope writer is not the call itself"))
		}
		return typed.writeBuffer
	default:
		t.Fatalf("unexpected conn %T", translated.StreamingClientConn)
		return nil
	}
}

// TestWriteBufferingIsWiredIntoStreamingConns verifies that WithWriteBuffering
// reaches the streaming write path of both protocol clients: the conn must carry
// a non-nil writeBuffer and its envelope writer must be routed through it. It
// also verifies that unary fast-path calls stay unbuffered, keeping the option
// effective on the default (gRPC wire) client.
func TestWriteBufferingIsWiredIntoStreamingConns(t *testing.T) {
	t.Parallel()

	unarySpec := Spec{StreamType: StreamTypeUnary, Procedure: "/connect.ping.v1.PingService/Ping"}
	streamSpec := Spec{StreamType: StreamTypeBidi, Procedure: "/connect.ping.v1.PingService/Ping"}

	// Default gRPC wire: streaming calls are buffered only when opted in.
	grpcBuffered := &grpcClient{protocolClientParams: newBufferingParams(true, true)}
	assert.True(t, connWriteBuffer(t, grpcBuffered.NewConn(context.Background(), streamSpec, make(http.Header))) != nil,
		assert.Sprintf("gRPC streaming conn dropped WithWriteBuffering"))
	grpcPlain := &grpcClient{protocolClientParams: newBufferingParams(false, true)}
	assert.Nil(t, connWriteBuffer(t, grpcPlain.NewConn(context.Background(), streamSpec, make(http.Header))))

	// Triple wire keeps the same contract on its streaming path.
	tripleBuffered := &tripleClient{protocolClientParams: newBufferingParams(true, true)}
	assert.True(t, connWriteBuffer(t, tripleBuffered.NewConn(context.Background(), streamSpec, make(http.Header))) != nil,
		assert.Sprintf("triple streaming conn dropped WithWriteBuffering"))
	triplePlain := &tripleClient{protocolClientParams: newBufferingParams(false, true)}
	assert.Nil(t, connWriteBuffer(t, triplePlain.NewConn(context.Background(), streamSpec, make(http.Header))))

	// Unary fast-path conns must never pay for the extra copy.
	assert.Nil(t, connWriteBuffer(t, grpcBuffered.NewConn(context.Background(), unarySpec, make(http.Header))))
	assert.Nil(t, connWriteBuffer(t, tripleBuffered.NewConn(context.Background(), unarySpec, make(http.Header))))
}

// TestWriteBufferingCoversUnaryNonFastPath verifies that unary calls which skip
// the fast path are buffered too: they still run over duplexHTTPCall and
// io.Pipe, so WithWriteBuffering must reach them; only the fast path stays
// unbuffered.
func TestWriteBufferingCoversUnaryNonFastPath(t *testing.T) {
	t.Parallel()

	unarySpec := Spec{StreamType: StreamTypeUnary, Procedure: "/connect.ping.v1.PingService/Ping"}

	grpcSlow := &grpcClient{protocolClientParams: newBufferingParams(true, false)}
	assert.True(t, connWriteBuffer(t, grpcSlow.NewConn(context.Background(), unarySpec, make(http.Header))) != nil,
		assert.Sprintf("gRPC unary conn off the fast path dropped WithWriteBuffering"))

	tripleSlow := &tripleClient{protocolClientParams: newBufferingParams(true, false)}
	assert.True(t, connWriteBuffer(t, tripleSlow.NewConn(context.Background(), unarySpec, make(http.Header))) != nil,
		assert.Sprintf("triple unary conn off the fast path dropped WithWriteBuffering"))
}

// newRequestSpyServer starts a server that reports on reached the first time it
// is handed a request, so a test can tell whether CloseRequest actually issued
// one. It drains the body and replies 200 so the client can finish cleanly.
func newRequestSpyServer(t *testing.T) (*httptest.Server, <-chan struct{}, *url.URL) {
	t.Helper()
	reached := make(chan struct{}, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case reached <- struct{}{}:
		default:
		}
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	return server, reached, serverURL
}

// TestWriteBufferingEmptyStreamStillSendsRequest verifies that a stream which
// never sent a message still issues the HTTP request when the request side
// closes: sealing an empty buffer is a no-op, so CloseWrite's fallback to
// ensureRequestMade is the only thing standing between the server and a request
// that never arrives.
func TestWriteBufferingEmptyStreamStillSendsRequest(t *testing.T) {
	t.Parallel()

	server, reached, serverURL := newRequestSpyServer(t)
	call := newTestDuplexClientCall(t, server.Client(), serverURL)
	conn := &grpcClientConn{call: call, writeBuffer: newStreamBufferWriter(call)}

	if err := conn.CloseRequest(); err != nil {
		t.Fatalf("CloseRequest on an empty stream: %v", err)
	}
	select {
	case <-reached:
	case <-time.After(closeTestTimeout):
		t.Fatal("an empty buffered stream must still send the request on CloseRequest")
	}
}

// TestCloseRequestClosesWriteSideAfterFlushFailure verifies that CloseRequest
// closes the write side even when the final flush fails: the flush error is
// reported to the caller, but the peer must not be left waiting on a request
// body that never closes.
func TestCloseRequestClosesWriteSideAfterFlushFailure(t *testing.T) {
	t.Parallel()

	boom := errors.New("flush boom")
	server, reached, serverURL := newRequestSpyServer(t)
	call := newTestDuplexClientCall(t, server.Client(), serverURL)
	conn := &grpcClientConn{call: call, writeBuffer: newStreamBufferWriter(errorWriter{err: boom})}
	if _, err := conn.writeBuffer.Write(make([]byte, 64)); err != nil {
		t.Fatalf("small write should buffer, got %v", err)
	}

	if err := conn.CloseRequest(); !errors.Is(err, boom) {
		t.Fatalf("CloseRequest must report the flush failure %v, got %v", boom, err)
	}
	select {
	case <-reached:
	case <-time.After(closeTestTimeout):
		t.Fatal("CloseRequest must close the write side even after a flush failure")
	}
}

// TestStreamBufferWriterStickyErrorOnEveryExit verifies that once an underlying
// write fails, every entry point keeps reporting that same error.
func TestStreamBufferWriterStickyErrorOnEveryExit(t *testing.T) {
	boom := errors.New("boom")
	w := newStreamBufferWriter(errorWriter{err: boom})
	msg := make([]byte, 64)

	if _, err := w.Write(msg); err != nil {
		t.Fatalf("small write should buffer, got %v", err)
	}
	if err := w.Flush(); !errors.Is(err, boom) {
		t.Fatalf("Flush: expected %v, got %v", boom, err)
	}
	if _, err := w.Write(msg); !errors.Is(err, boom) {
		t.Fatalf("Write after failure: expected %v, got %v", boom, err)
	}
	if err := w.Flush(); !errors.Is(err, boom) {
		t.Fatalf("second Flush: expected %v, got %v", boom, err)
	}
	if err := w.Close(); !errors.Is(err, boom) {
		t.Fatalf("Close after failure: expected %v, got %v", boom, err)
	}
}

// TestStreamBufferWriterWriteReportsFlushFailure verifies that the Send which
// trips the watermark is the one that hears about a failing flush, rather than
// the failure being deferred to a later call the caller never makes.
func TestStreamBufferWriterWriteReportsFlushFailure(t *testing.T) {
	boom := errors.New("boom")
	seen := 0
	w := newStreamBufferWriter(countWriterFunc(func(p []byte) (int, error) {
		seen += len(p)
		return 0, boom
	}))
	msg := make([]byte, defaultStreamWriteBufSize/2)

	if _, err := w.Write(msg); err != nil {
		t.Fatalf("write below the watermark should buffer, got %v", err)
	}
	// The second half fills the buffer, so this Send triggers the flush. The
	// triggering Write reports (len(p), err): the payload was accepted into the
	// buffer even though flushing it failed.
	n, err := w.Write(msg)
	if !errors.Is(err, boom) {
		t.Fatalf("expected the triggering Write to report %v, got %v", boom, err)
	}
	if n != len(msg) {
		t.Fatalf("expected the triggering Write to report %d accepted bytes, got %d", len(msg), n)
	}
	if seen == 0 {
		t.Fatal("expected the batch to reach the underlying writer")
	}
}

// TestStreamBufferWriterLargeMessageErrorPropagates verifies that the direct
// write path still flushes pending messages first and surfaces a failure of
// that flush, instead of overwriting the pending tail with the large message.
func TestStreamBufferWriterLargeMessageErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	w := newStreamBufferWriter(errorWriter{err: boom})

	if _, err := w.Write(make([]byte, 64)); err != nil {
		t.Fatalf("small write should buffer, got %v", err)
	}
	large := make([]byte, defaultStreamWriteBufSize)
	if _, err := w.Write(large); !errors.Is(err, boom) {
		t.Fatalf("expected the pending-flush failure to surface, got %v", err)
	}
}

// TestStreamBufferWriterCloseFlushFailureReturnsError verifies that Close
// reports the failure of its final flush instead of returning nil.
func TestStreamBufferWriterCloseFlushFailureReturnsError(t *testing.T) {
	boom := errors.New("boom")
	w := newStreamBufferWriter(errorWriter{err: boom})

	if _, err := w.Write(make([]byte, 64)); err != nil {
		t.Fatalf("small write should buffer, got %v", err)
	}
	if err := w.Close(); !errors.Is(err, boom) {
		t.Fatalf("expected Close to report the flush failure %v, got %v", boom, err)
	}
}

// TestStreamBufferWriterConcurrentCloseAndFlush verifies that Close racing with
// Send and Flush is race-free and never drops an accepted message: Flush after
// Close reports nil, Close stays idempotent, and the bytes reaching the
// underlying writer match the bytes the caller was told were accepted.
func TestStreamBufferWriterConcurrentCloseAndFlush(t *testing.T) {
	under := &atomicWriteCountRecorder{}
	w := newStreamBufferWriter(under)
	msg := make([]byte, 128)
	accepted := &atomic.Int64{}

	var wg sync.WaitGroup
	wg.Go(func() {
		for range 200 {
			if _, err := w.Write(msg); err == nil {
				accepted.Add(1)
			}
		}
	})
	wg.Go(func() {
		for range 200 {
			_ = w.Flush()
		}
		_ = w.Close()
	})
	wg.Wait()

	if got, want := under.total.Load(), accepted.Load()*int64(len(msg)); got != want {
		t.Fatalf("%d accepted byte(s) but %d reached the writer", want, got)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close must be idempotent, got %v", err)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush after Close must report nil, got %v", err)
	}
}

// TestStreamBufferWriterConcurrentBufferBound verifies that aggregation stays
// bounded under concurrent Send: no batch handed to the underlying writer
// exceeds the watermark plus the single message that triggered the flush, so a
// burst of tiny messages cannot grow the buffer without limit.
func TestStreamBufferWriterConcurrentBufferBound(t *testing.T) {
	const msgSize = 100
	maxBatch := &atomic.Int64{}
	w := newStreamBufferWriter(countWriterFunc(func(p []byte) (int, error) {
		for {
			observed := maxBatch.Load()
			if int64(len(p)) <= observed || maxBatch.CompareAndSwap(observed, int64(len(p))) {
				break
			}
		}
		return len(p), nil
	}))
	msg := make([]byte, msgSize)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 500 {
				_, _ = w.Write(msg)
			}
		})
	}
	wg.Wait()
	if err := w.Flush(); err != nil {
		t.Fatalf("final Flush: %v", err)
	}
	if got := maxBatch.Load(); got > int64(defaultStreamWriteBufSize+msgSize) {
		t.Fatalf("batch of %d bytes exceeds the %d-byte bound", got, defaultStreamWriteBufSize+msgSize)
	}
}

// TestStreamBufferWriterLargeMessageFlushesPendingFirst verifies write ordering:
// the direct write for an over-limit message must not overtake messages already
// sitting in the buffer, or the receiver would see frames out of order.
func TestStreamBufferWriterLargeMessageFlushesPendingFirst(t *testing.T) {
	var batches [][]byte
	w := newStreamBufferWriter(countWriterFunc(func(p []byte) (int, error) {
		batches = append(batches, append([]byte(nil), p...))
		return len(p), nil
	}))
	small := make([]byte, 64)
	large := make([]byte, defaultStreamWriteBufSize)

	if _, err := w.Write(small); err != nil {
		t.Fatalf("small write: %v", err)
	}
	if _, err := w.Write(large); err != nil {
		t.Fatalf("large write: %v", err)
	}

	if len(batches) != 2 {
		t.Fatalf("expected two underlying writes, got %d", len(batches))
	}
	if len(batches[0]) != len(small) {
		t.Fatalf("pending batch must go first: got %d bytes, want %d", len(batches[0]), len(small))
	}
	if len(batches[1]) != len(large) {
		t.Fatalf("large message must follow the pending batch: got %d bytes, want %d", len(batches[1]), len(large))
	}
}

// TestStreamBufferWriterShortWriteIsFailure verifies that a downstream writer
// which accepts fewer bytes than offered without reporting an error is treated
// as a failure rather than discarding the undelivered tail.
func TestStreamBufferWriterShortWriteIsFailure(t *testing.T) {
	short := countWriterFunc(func(p []byte) (int, error) {
		return len(p) / 2, nil
	})
	w := newStreamBufferWriter(short)

	if _, err := w.Write(make([]byte, 64)); err != nil {
		t.Fatalf("small write should buffer, got %v", err)
	}
	if err := w.Flush(); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("expected io.ErrShortWrite, got %v", err)
	}
	if _, err := w.Write(make([]byte, 64)); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("expected the sticky short-write error, got %v", err)
	}
}

// TestStreamBufferWriterCloseAfterCloseFailure verifies that a Close whose
// final flush failed is not later reported as a clean close: the failure stays
// sticky across repeated Close calls.
func TestStreamBufferWriterCloseAfterCloseFailure(t *testing.T) {
	boom := errors.New("boom")
	w := newStreamBufferWriter(errorWriter{err: boom})

	if _, err := w.Write(make([]byte, 64)); err != nil {
		t.Fatalf("small write should buffer, got %v", err)
	}
	if err := w.Close(); !errors.Is(err, boom) {
		t.Fatalf("first Close: expected %v, got %v", boom, err)
	}
	if err := w.Close(); !errors.Is(err, boom) {
		t.Fatalf("second Close must keep reporting %v, got %v", boom, err)
	}
}

// TestStreamBufferWriterFlushAfterClose verifies that Flush after a successful
// Close is a no-op reporting nil: it must neither resurrect the sealed buffer
// nor report io.EOF, and it must not write again.
func TestStreamBufferWriterFlushAfterClose(t *testing.T) {
	under := &writeCountRecorder{}
	w := newStreamBufferWriter(under)

	if _, err := w.Write(make([]byte, 64)); err != nil {
		t.Fatalf("small write should buffer, got %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush after Close must report nil, got %v", err)
	}
	if under.writes != 1 {
		t.Fatalf("Flush after Close must not write again, got %d write(s)", under.writes)
	}
}
