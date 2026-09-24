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
	"io"
	"sync"
)

// defaultStreamWriteBufSize is the aggregation threshold for the streaming
// write buffer. It matches the gRPC-Go default write buffer size: large
// enough to batch small, high-rate messages together, small enough to bound
// how much a burst of tiny messages spends buffered (and therefore delays)
// before hitting the wire.
const defaultStreamWriteBufSize = 32 << 10 // 32 KiB

// preWriteChecker lets a buffered writer preserve checks normally performed
// by its underlying writer before accepting a write into the buffer.
type preWriteChecker interface {
	checkBeforeWrite() error
}

// streamBufferWriter is an opt-in aggregation layer over a duplexHTTPCall for
// streaming requests. Without it, every streamed message is flushed down the
// io.Pipe alone, paying a synchronous cross-goroutine handshake per message.
// streamBufferWriter amortizes those handshakes by coalescing small messages
// in a capacity-bounded buffer and flushing to the underlying writer only when
// the buffer fills (or when Flush or Close is called). It is safe for
// concurrent use; StreamingClientConn requires Send and CloseRequest to be
// concurrency-safe.
type streamBufferWriter struct {
	mu sync.Mutex

	next    io.Writer // wrapped duplexHTTPCall
	checker preWriteChecker
	limit   int
	buf     *bytes.Buffer

	err    error
	closed bool
}

// newStreamBufferWriter wraps next with an aggregation buffer of default
// capacity.
func newStreamBufferWriter(next io.Writer) *streamBufferWriter {
	checker, _ := next.(preWriteChecker)
	return &streamBufferWriter{
		next:    next,
		checker: checker,
		limit:   defaultStreamWriteBufSize,
		buf:     makeStreamWriteBuf(),
	}
}

// makeStreamWriteBuf preallocates the aggregation capacity.
func makeStreamWriteBuf() *bytes.Buffer {
	return bytes.NewBuffer(make([]byte, 0, defaultStreamWriteBufSize))
}

// Write coalesces a small message into the buffer, flushing the batch once the
// buffer reaches its limit. Payloads at or over the limit bypass the buffer
// and go directly to the underlying writer so a single big message is not
// held in memory twice. Empty writes also pass through to start the HTTP
// request.
func (w *streamBufferWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return 0, w.err
	}
	if w.closed {
		// Mirror duplexHTTPCall.Write: a late Send after the write side closed
		// reports io.EOF.
		return 0, io.EOF
	}
	if w.checker != nil {
		if err := w.checker.checkBeforeWrite(); err != nil {
			w.err = err
			w.closed = true
			return 0, err
		}
	}
	if len(p) == 0 {
		// duplexHTTPCall uses this write to start the HTTP request.
		return w.writeDirectLocked(p)
	}
	if len(p) >= w.limit {
		if err := w.flushLocked(); err != nil {
			return 0, err
		}
		return w.writeDirectLocked(p)
	}
	w.buf.Write(p)
	if w.buf.Len() >= w.limit && w.flushLocked() != nil {
		return len(p), w.err
	}
	return len(p), nil
}

func (w *streamBufferWriter) writeDirectLocked(p []byte) (int, error) {
	n, err := w.next.Write(p)
	if err == nil && n < len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		w.err = err
		w.closed = true
	}
	return n, err
}

// Flush pushes any pending buffered messages to the underlying writer in a
// single write, collapsing the handshake cost of each individual Send.
func (w *streamBufferWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return w.err
	}
	if w.closed {
		return nil
	}
	return w.flushLocked()
}

// Close flushes pending messages and seals the buffer, atomically under the
// same lock. Sealing matters because StreamingClientConn lets Send race with
// CloseRequest: a Send that wins the lock before Close lands its data in this
// final flush, and one that loses it fails with io.EOF. It is idempotent and
// safe to call after a failed flush.
func (w *streamBufferWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return w.err
	}
	if w.closed {
		return nil
	}
	err := w.flushLocked()
	w.closed = true
	return err
}

func (w *streamBufferWriter) flushLocked() error {
	if w.buf.Len() == 0 {
		return nil
	}
	n, err := w.next.Write(w.buf.Bytes())
	if err == nil && n < w.buf.Len() {
		// The downstream writer accepted fewer bytes than offered but reported
		// no error, breaking the io.Writer contract. Treat it as a write
		// failure and freeze the buffer.
		err = io.ErrShortWrite
	}
	if err != nil {
		// A write failure on a stream is terminal: freeze the buffer so
		// subsequent writes report the same error.
		w.err = err
		w.closed = true
		return err
	}
	w.buf.Reset()
	return nil
}
