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

// L1 micro-framework for the streaming write buffering optimization.
//
// Test methodology (AB, "before vs after"):
//   - Protocol: raw os.Pipe (a real fd), so every small Write issues a real
//     write(2) syscall and strace can observe it. net.Pipe must NOT be used
//     here because it is an in-memory channel with no syscall to count.
//   - Scope: a fixed payload-size ladder (benchMsgSizes: 128 B, 256 B, 512 B,
//     1 KiB, 4 KiB) and identical total bytes across the two sides; each size
//     runs as its own sub-benchmark (BenchmarkStreamWritePerMessage/<size> and
//     BenchmarkStreamWriteBuffered/<size>) so benchstat can pair them size by
//     size. Only the write strategy differs.
//     * BenchmarkStreamWritePerMessage (BEFORE): one os.Pipe.Write per message,
//       mirroring the duplexHTTPCall behaviour before write buffering.
//     * BenchmarkStreamWriteBuffered (AFTER): newStreamBufferWriter coalesces
//       messages into the 32 KiB buffer and flushes the batch, mirroring the
//       CloseRequest-time flush that finishes the tail.
//   - Metrics: ns/op, B/op, allocs/op via -benchmem (L1); write(2)/writev(2)
//     syscall counts via strace -c (L2).
//
// Interpretation: buffering does NOT reduce allocations (it may even keep a
// resident 32 KiB buffer). It reduces per-message syscalls and cross-goroutine
// handshakes, so its win shows up as ns/op throughput, not as allocs. Every
// rung stays well below the 32 KiB threshold, so a flush always covers
// floor(32 KiB / size) messages: 256 messages at 128 B down to 8 at 4 KiB,
// i.e. a 256x down to 8x drop in write(2) syscalls is the expected L2 signal.

import (
	"io"
	"os"
	"strconv"
	"testing"
)

// benchMsgSizes is the payload-size ladder both benchmarks are run at, in
// bytes: 128 B, 256 B, 512 B, 1 KiB and 4 KiB. Every rung is far below the
// 32 KiB defaultStreamWriteBufSize threshold, so the buffered side always takes
// the coalescing path instead of the direct-write branch reserved for payloads
// at or over the limit.
var benchMsgSizes = []int{128, 256, 512, 1024, 4096}

// BenchmarkStreamWritePerMessage measures the BEFORE strategy: every message is
// written straight to the pipe, paying a write(2) syscall and a cross-goroutine
// handshake each time.
func BenchmarkStreamWritePerMessage(b *testing.B) {
	for _, size := range benchMsgSizes {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			r, w, err := os.Pipe()
			if err != nil {
				b.Fatal(err)
			}
			go func() { _, _ = io.Copy(io.Discard, r) }() //nolint:errcheck

			msg := make([]byte, size)
			b.ReportAllocs()
			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := w.Write(msg); err != nil {
					b.Fatal(err)
				}
			}
			_ = r.Close()
			_ = w.Close()
		})
	}
}

// BenchmarkStreamWriteBuffered measures the AFTER strategy: newStreamBufferWriter
// accumulates messages of the current payload size and flushes each full
// 32 KiB batch in one write (the tail goes out on Flush).
func BenchmarkStreamWriteBuffered(b *testing.B) {
	for _, size := range benchMsgSizes {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			r, w, err := os.Pipe()
			if err != nil {
				b.Fatal(err)
			}
			go func() { _, _ = io.Copy(io.Discard, r) }() //nolint:errcheck

			buf := newStreamBufferWriter(w)
			msg := make([]byte, size)
			b.ReportAllocs()
			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := buf.Write(msg); err != nil {
					b.Fatal(err)
				}
			}
			if err := buf.Flush(); err != nil {
				b.Fatal(err)
			}
			_ = r.Close()
			_ = w.Close()
		})
	}
}
