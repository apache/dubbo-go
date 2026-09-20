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

package triple_protocol_test

// This is a lightweight, deterministic probe (not a benchmark). It quantifies
// the per-message cost of the Triple streaming client write path: without any
// buffering, every Send lands as one synchronous io.Pipe handshake, so the
// number of segments reaching the peer reader grows 1:1 with the message
// count. A buffered (aggregated) write collapses those segments. The probe
// reports the segment-count ratio so the buffering decision rests on numbers,
// not on an assumed clock.
//
// io.Pipe transfers each Write as a complete, individually-readable chunk (as
// long as it fits the reader buffer), which is exactly the cross-goroutine
// handshake the streaming client pays for on every Send.

import (
	"io"
	"testing"
)

// countSegments reads r to EOF and returns how many distinct chunks arrived,
// mirroring how many io.Pipe handshakes a writer-side aggregation would amortize.
func countSegments(r io.Reader, buf int) int {
	rb := make([]byte, buf)
	segments := 0
	for {
		n, err := r.Read(rb)
		if n > 0 {
			segments++
		}
		if err != nil {
			break
		}
	}
	return segments
}

// TestWritePathAggregationProbe sends a fixed number of small messages through
// io.Pipe two ways: written one-by-one (the current duplexHTTPCall baseline)
// and written after aggregating into a single buffer (the proposed buffering).
// It reports how many read-side segments each path produces, quantifying the
// handshake amortization without measuring wall-clock time.
func TestWritePathAggregationProbe(t *testing.T) {
	const (
		msgCount = 1000
		msgSize  = 128
	)
	msg := make([]byte, msgSize)

	// Baseline: one Write per message, the current streaming client behaviour.
	prA, pwA := io.Pipe()
	doneA := make(chan int)
	go func() {
		doneA <- countSegments(prA, 1<<20)
	}()
	for i := 0; i < msgCount; i++ {
		if _, err := pwA.Write(msg); err != nil {
			t.Fatalf("baseline write: %v", err)
		}
	}
	pwA.Close() //nolint:errcheck
	segA := <-doneA

	// Buffered: aggregate all messages into one buffer, then a single Write.
	prB, pwB := io.Pipe()
	doneB := make(chan int)
	go func() {
		doneB <- countSegments(prB, 1<<20)
	}()
	agg := make([]byte, 0, msgCount*msgSize)
	for i := 0; i < msgCount; i++ {
		agg = append(agg, msg...)
	}
	if _, err := pwB.Write(agg); err != nil { //nolint:errcheck
		t.Fatalf("buffered write: %v", err)
	}
	pwB.Close() //nolint:errcheck
	segB := <-doneB

	t.Logf("baseline(bufSize=0):  %d msgs -> %d read-side segment(s)  (1:1 per message)",
		msgCount, segA)
	t.Logf("buffered(agg+1 write): %d msgs -> %d read-side segment(s)  (collapsed)",
		msgCount, segB)
	t.Logf("handshake amortization: %dx fewer segments", segA/segB)

	if segA != msgCount {
		t.Fatalf("expected baseline to emit one segment per message, got %d", segA)
	}
	if segB >= segA {
		t.Fatalf("expected buffering to collapse segments, got %d vs baseline %d", segB, segA)
	}
}
