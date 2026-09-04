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
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

import (
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
)

// syncBuffer is an io.Writer that is safe for concurrent use. It lets
// concurrency tests share one envelopeWriter without corrupting output.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.buf.Bytes()...)
}

func (s *syncBuffer) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Len()
}

// regressionCodecList returns the two codecs under test: the fast path
// (implements marshalAppender) and the slow path (does not).
func regressionCodecList() []struct {
	name  string
	codec Codec
} {
	return []struct {
		name  string
		codec Codec
	}{
		{name: "fast", codec: &protoBinaryCodec{}},
		{name: "slow", codec: &noAppenderCodec{&protoBinaryCodec{}}},
	}
}

func newTestGzipCompressionPool() *compressionPool {
	return newCompressionPool(
		func() Decompressor { return &gzip.Reader{} },
		func() Compressor { return gzip.NewWriter(io.Discard) },
	)
}

// newEnvelopeWriterForTest builds an envelopeWriter (gRPC/Triple wire) with
// optional gzip compression.
func newEnvelopeWriterForTest(t *testing.T, codec Codec, compress bool, compressMinBytes, sendMaxBytes int) (*envelopeWriter, *syncBuffer) {
	t.Helper()
	out := &syncBuffer{}
	w := &envelopeWriter{
		writer:           out,
		codec:            codec,
		compressMinBytes: compressMinBytes,
		bufferPool:       newBufferPool(),
		sendMaxBytes:     sendMaxBytes,
	}
	if compress {
		w.compressionPool = newTestGzipCompressionPool()
	}
	return w, out
}

// newTripleMarshalerForTest builds a tripleUnaryMarshaler (Triple HTTP body
// wire) with optional gzip compression.
func newTripleMarshalerForTest(
	t *testing.T,
	codec Codec,
	compress bool,
	compressMinBytes, sendMaxBytes int,
) (*tripleUnaryMarshaler, *syncBuffer, http.Header) {
	t.Helper()
	out := &syncBuffer{}
	header := make(http.Header)
	m := &tripleUnaryMarshaler{
		writer:           out,
		codec:            codec,
		compressMinBytes: compressMinBytes,
		bufferPool:       newBufferPool(),
		header:           header,
		sendMaxBytes:     sendMaxBytes,
	}
	if compress {
		m.compressionPool = newTestGzipCompressionPool()
		m.compressionName = compressionGzip
	}
	return m, out, header
}

func regressionPayloadSizes() []int {
	mi := 1024 * 1024
	return []int{0, 1, 511, 512, 513, 1024, 8*mi - 1, 8 * mi, 8*mi + 1}
}

func regressionSizeLabel(size int) string {
	switch size {
	case 0:
		return "empty"
	case 1024:
		return "1KiB"
	case 8*1024*1024 - 1:
		return "8MiB-1"
	case 8 * 1024 * 1024:
		return "8MiB"
	case 8*1024*1024 + 1:
		return "8MiB+1"
	default:
		return fmt.Sprintf("%dB", size)
	}
}

// envelopePrefix parses the 5-byte gRPC/Triple envelope prefix written by
// envelopeWriter: byte 0 is flags, bytes 1..4 (wire[1:5]) are the big-endian
// payload length.
func envelopePrefix(t *testing.T, wire []byte) (flags byte, length int) {
	t.Helper()
	if len(wire) < 5 {
		t.Fatalf("wire output shorter than envelope prefix: got %d bytes", len(wire))
	}
	return wire[0], int(binary.BigEndian.Uint32(wire[1:5]))
}

func wantCode(t *testing.T, err *Error, want Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected *Error with code %v, got nil", want)
	}
	if got := err.Code(); got != want {
		t.Fatalf("expected code %v, got %v (message: %v)", want, got, err)
	}
}

// TestMarshalPerfWireParity verifies that the MarshalAppend fast path and the
// codec.Marshal slow path emit byte-identical wire output on both wire types.
func TestMarshalPerfWireParity(t *testing.T) {
	for _, compress := range []bool{false, true} {
		t.Run(fmt.Sprintf("compressed=%v", compress), func(t *testing.T) {
			for _, size := range regressionPayloadSizes() {
				t.Run(regressionSizeLabel(size), func(t *testing.T) {
					t.Parallel()
					msg := newMarshalPerfMessage(size)

					// triple HTTP body wire
					t.Run("triple", func(t *testing.T) {
						var fastWire, slowWire []byte
						var fastHeader, slowHeader http.Header
						for _, c := range regressionCodecList() {
							m, out, header := newTripleMarshalerForTest(t, c.codec, compress, 0, 0)
							if err := m.Marshal(msg); err != nil {
								t.Fatalf("%s path triple marshal: %v", c.name, err)
							}
							switch c.name {
							case "fast":
								fastWire, fastHeader = out.Bytes(), header
							case "slow":
								slowWire, slowHeader = out.Bytes(), header
							}
						}
						if !bytes.Equal(fastWire, slowWire) {
							t.Fatalf("triple wire mismatch: fast %d bytes vs slow %d bytes", len(fastWire), len(slowWire))
						}
						if compress {
							if fastHeader.Get(tripleUnaryHeaderCompression) != compressionGzip ||
								slowHeader.Get(tripleUnaryHeaderCompression) != compressionGzip {
								t.Fatalf("compression header not set identically: fast=%q slow=%q",
									fastHeader.Get(tripleUnaryHeaderCompression), slowHeader.Get(tripleUnaryHeaderCompression))
							}
						}
					})

					// gRPC/Triple envelope wire (5-byte prefix framing)
					t.Run("envelope", func(t *testing.T) {
						var fastWire, slowWire []byte
						for _, c := range regressionCodecList() {
							w, out := newEnvelopeWriterForTest(t, c.codec, compress, 0, 0)
							if err := w.Marshal(msg); err != nil {
								t.Fatalf("%s path envelope marshal: %v", c.name, err)
							}
							switch c.name {
							case "fast":
								fastWire = out.Bytes()
							case "slow":
								slowWire = out.Bytes()
							}
						}
						if !bytes.Equal(fastWire, slowWire) {
							t.Fatalf("envelope wire mismatch: fast %d bytes vs slow %d bytes", len(fastWire), len(slowWire))
						}
					})
				})
			}
		})
	}
}

// TestMarshalPerfPoolInvariants verifies buffer pool reuse invariants and
// envelope framing for empty and nil messages.
func TestMarshalPerfPoolInvariants(t *testing.T) {
	t.Parallel()

	t.Run("get returns empty buffer", func(t *testing.T) {
		pool := newBufferPool()
		buf := pool.Get()
		defer pool.Put(buf)
		if buf.Len() != 0 {
			t.Fatalf("bufferPool.Get() returned buffer with Len()==%d, want 0", buf.Len())
		}
	})

	t.Run("put resets length", func(t *testing.T) {
		pool := newBufferPool()
		buf := pool.Get()
		if _, err := buf.WriteString("junk to reset"); err != nil {
			t.Fatal(err)
		}
		pool.Put(buf)
		again := pool.Get()
		defer pool.Put(again)
		if again.Len() != 0 {
			t.Fatalf("reused buffer not reset: Len()==%d, want 0", again.Len())
		}
	})

	t.Run("nil message writes nothing", func(t *testing.T) {
		for _, c := range regressionCodecList() {
			w, out := newEnvelopeWriterForTest(t, c.codec, false, 0, 0)
			if err := w.Marshal(nil); err != nil {
				t.Fatalf("%s path nil message: %v", c.name, err)
			}
			if out.Len() != 0 {
				t.Fatalf("%s path nil message wrote %d bytes, want 0", c.name, out.Len())
			}
		}
	})

	t.Run("empty proto yields zero-length envelope", func(t *testing.T) {
		// compressMinBytes=1: an empty payload (0 < 1) must not be compressed,
		// so the envelope is exactly a 5-byte prefix announcing length 0.
		for _, c := range regressionCodecList() {
			w, out := newEnvelopeWriterForTest(t, c.codec, true, 1, 0)
			msg := &pingv1.PingRequest{Text: ""}
			if err := w.Marshal(msg); err != nil {
				t.Fatalf("%s path: %v", c.name, err)
			}
			flags, length := envelopePrefix(t, out.Bytes())
			if flags != 0 {
				t.Fatalf("%s path: empty payload compressed (flags=%#x), want 0", c.name, flags)
			}
			if length != 0 {
				t.Fatalf("%s path: envelope length %d, want 0", c.name, length)
			}
			if out.Len() != 5 {
				t.Fatalf("%s path: wire length %d, want exactly 5 (prefix only)", c.name, out.Len())
			}
		}
	})

	t.Run("compressMinBytes boundary does not panic", func(t *testing.T) {
		// The compression decision uses the codec's encoded output length: a
		// threshold equal to the encoded length compresses, one byte more does not.
		codec := &protoBinaryCodec{}
		for _, size := range []int{511, 512, 513} {
			msg := newMarshalPerfMessage(size)
			raw, err := codec.Marshal(msg)
			if err != nil {
				t.Fatal(err)
			}
			encodedLen := len(raw)
			for _, minBytes := range []int{encodedLen, encodedLen + 1} {
				wantCompressed := minBytes <= encodedLen
				for _, c := range regressionCodecList() {
					w, out := newEnvelopeWriterForTest(t, c.codec, true, minBytes, 0)
					if err := w.Marshal(msg); err != nil {
						t.Fatalf("%s path size %d minBytes %d: %v", c.name, size, minBytes, err)
					}
					flags, _ := envelopePrefix(t, out.Bytes())
					gotCompressed := flags&flagEnvelopeCompressed != 0
					if gotCompressed != wantCompressed {
						t.Fatalf("%s path size %d (encoded %d) minBytes %d: compressed=%v, want %v",
							c.name, size, encodedLen, minBytes, gotCompressed, wantCompressed)
					}
				}
			}
		}
	})
}

// errorCodec implements Codec and marshalAppender, always failing. It forces
// the envelopeWriter fast path (MarshalAppend) and slow path (Marshal) into
// their backup-codec fallback branches.
type errorCodec struct{}

var _ Codec = errorCodec{}
var _ marshalAppender = errorCodec{}

func (errorCodec) Name() string { return "always-error" }

func (errorCodec) Marshal(any) ([]byte, error) {
	return nil, errors.New("primary codec marshal failed")
}

func (errorCodec) MarshalAppend(dst []byte, _ any) ([]byte, error) {
	return nil, errors.New("primary codec marshal failed")
}

func (errorCodec) Unmarshal([]byte, any) error {
	return errors.New("primary codec unmarshal failed")
}

// TestMarshalPerfBackupCodecFallback verifies the backup-codec fallback on the
// fast (MarshalAppend) and slow (Marshal) paths.
func TestMarshalPerfBackupCodecFallback(t *testing.T) {
	msg := &pingv1.PingRequest{Text: "fallback"}
	// Reference wire bytes produced by the healthy codec.
	var refBytes []byte
	{
		ref, out := newEnvelopeWriterForTest(t, &protoBinaryCodec{}, false, 0, 0)
		if err := ref.Marshal(msg); err != nil {
			t.Fatal(err)
		}
		refBytes = out.Bytes()
	}

	t.Run("primary fails fast path falls back", func(t *testing.T) {
		// errorCodec implements marshalAppender, so the fast path is taken;
		// on failure it must fall back to the backup codec and produce the
		// same wire bytes as the healthy codec.
		w, out := newEnvelopeWriterForTest(t, errorCodec{}, false, 0, 0)
		w.backupCodec = &protoBinaryCodec{}
		if err := w.Marshal(msg); err != nil {
			t.Fatalf("fast fallback: %v", err)
		}
		if got := out.Bytes(); !bytes.Equal(got, refBytes) {
			t.Fatalf("fast fallback wire mismatch: got %d bytes, want %d", len(got), len(refBytes))
		}
	})

	t.Run("primary fails slow path falls back", func(t *testing.T) {
		w, out := newEnvelopeWriterForTest(t, &noAppenderCodec{errorCodec{}}, false, 0, 0)
		w.backupCodec = &protoBinaryCodec{}
		if err := w.Marshal(msg); err != nil {
			t.Fatalf("slow fallback: %v", err)
		}
		if got := out.Bytes(); !bytes.Equal(got, refBytes) {
			t.Fatalf("slow fallback wire mismatch: got %d bytes, want %d", len(got), len(refBytes))
		}
	})

	t.Run("write failure with healthy appender does not fall back", func(t *testing.T) {
		// The fast path falls back only on a marshal failure: a write error
		// must surface as-is, without consulting the backup codec or retrying.
		backup := &countingNamedCodec{Codec: &protoBinaryCodec{}, name: "hessian2"}
		out := &failingWriter{}
		w := &envelopeWriter{
			writer:      out,
			codec:       &protoBinaryCodec{}, // implements marshalAppender
			backupCodec: backup,
			bufferPool:  newBufferPool(),
		}
		wantCode(t, w.Marshal(msg), CodeUnknown)
		if out.writes != 1 {
			t.Fatalf("writer called %d times, want 1 (write failure must not retry)", out.writes)
		}
		if backup.marshalCalls != 0 {
			t.Fatalf("backup codec Marshal called %d times on a write failure, want 0", backup.marshalCalls)
		}
	})

	t.Run("same name does not double fallback", func(t *testing.T) {
		// A backup sharing the codec's name must not be consulted, so the error
		// surfaces on both the fast and slow legs.
		legs := []struct {
			name  string
			codec Codec
		}{
			{name: "fast", codec: errorCodec{}},
			{name: "slow", codec: &noAppenderCodec{errorCodec{}}},
		}
		for _, leg := range legs {
			w, _ := newEnvelopeWriterForTest(t, leg.codec, false, 0, 0)
			w.backupCodec = &failingNamedCodec{name: "always-error"}
			if err := w.Marshal(msg); err == nil {
				t.Fatalf("%s path: expected error when backup shares codec name, got nil", leg.name)
			} else if asErr, ok := asError(err); ok && asErr.Code() != CodeInternal {
				t.Fatalf("%s path: expected CodeInternal, got %v", leg.name, asErr.Code())
			}
		}
	})

	t.Run("nil backup returns internal error", func(t *testing.T) {
		for _, c := range regressionCodecList() {
			w, _ := newEnvelopeWriterForTest(t, errorCodec{}, false, 0, 0)
			if c.name == "slow" {
				w.codec = &noAppenderCodec{errorCodec{}}
			}
			w.backupCodec = nil
			err := w.Marshal(msg)
			if err == nil {
				t.Fatalf("%s path: expected error with nil backup, got nil", c.name)
			}
			wantCode(t, err, CodeInternal)
		}
	})
}

// failingNamedCodec lets tests control the Name() seen by the fallback check.
type failingNamedCodec struct {
	errorCodec
	name string
}

func (c *failingNamedCodec) Name() string { return c.name }

// failingWriter is an io.Writer that records how many times it was called and
// always fails, letting tests observe spurious write retries.
type failingWriter struct {
	writes int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	w.writes++
	return 0, errors.New("write failed")
}

// countingNamedCodec wraps a codec under a distinct Name and counts Marshal
// calls, letting tests observe whether the backup-codec fallback ran.
type countingNamedCodec struct {
	Codec
	name         string
	marshalCalls int
}

func (c *countingNamedCodec) Name() string { return c.name }

func (c *countingNamedCodec) Marshal(message any) ([]byte, error) {
	c.marshalCalls++
	return c.Codec.Marshal(message)
}

// TestMarshalPerfCompressionAndMaxBytes verifies the sendMaxBytes limit and
// compression headers on both paths.
func TestMarshalPerfCompressionAndMaxBytes(t *testing.T) {
	t.Parallel()

	t.Run("sendMaxBytes exceeded returns ResourceExhausted on both paths", func(t *testing.T) {
		msg := newMarshalPerfMessage(1024)
		for _, wire := range []string{"envelope", "triple"} {
			for _, c := range regressionCodecList() {
				switch wire {
				case "envelope":
					w, _ := newEnvelopeWriterForTest(t, c.codec, false, 0, 100)
					wantCode(t, w.Marshal(msg), CodeResourceExhausted)
				case "triple":
					m, _, _ := newTripleMarshalerForTest(t, c.codec, false, 0, 100)
					wantCode(t, m.Marshal(msg), CodeResourceExhausted)
				}
			}
		}
	})

	t.Run("compressed over-limit also returns ResourceExhausted", func(t *testing.T) {
		// sendMaxBytes=1 with gzip enabled: even the compressed form of any
		// non-empty message exceeds 1 byte, so both paths must reject it.
		msg := newMarshalPerfMessage(64)
		for _, wire := range []string{"envelope", "triple"} {
			for _, c := range regressionCodecList() {
				switch wire {
				case "envelope":
					w, _ := newEnvelopeWriterForTest(t, c.codec, true, 0, 1)
					wantCode(t, w.Marshal(msg), CodeResourceExhausted)
				case "triple":
					m, _, _ := newTripleMarshalerForTest(t, c.codec, true, 0, 1)
					wantCode(t, m.Marshal(msg), CodeResourceExhausted)
				}
			}
		}
	})

	t.Run("compression header set identically on triple wire", func(t *testing.T) {
		msg := newMarshalPerfMessage(1024)
		for _, c := range regressionCodecList() {
			m, _, header := newTripleMarshalerForTest(t, c.codec, true, 512, 0)
			if err := m.Marshal(msg); err != nil {
				t.Fatalf("%s path: %v", c.name, err)
			}
			if got := header.Get(tripleUnaryHeaderCompression); got != compressionGzip {
				t.Fatalf("%s path: compression header %q, want %q", c.name, got, compressionGzip)
			}
		}
	})
}

// TestMarshalPerfLargeBufferDropped verifies that buffers larger than
// maxRecycleBufferSize are dropped rather than recycled.
func TestMarshalPerfLargeBufferDropped(t *testing.T) {
	t.Parallel()

	t.Run("pool refuses to recycle oversized buffer", func(t *testing.T) {
		pool := newBufferPool()
		big := pool.Get()
		data := make([]byte, 8*1024*1024+1)
		if _, err := big.Write(data); err != nil {
			t.Fatal(err)
		}
		if big.Cap() <= maxRecycleBufferSize {
			t.Fatalf("test setup: buffer cap %d not larger than %d", big.Cap(), maxRecycleBufferSize)
		}
		pool.Put(big)
		next := pool.Get()
		defer pool.Put(next)
		if next.Cap() > maxRecycleBufferSize {
			t.Fatalf("pool recycled an oversized buffer: cap %d > max %d", next.Cap(), maxRecycleBufferSize)
		}
		if next.Len() != 0 {
			t.Fatalf("pool returned non-empty buffer: Len()==%d", next.Len())
		}
	})

	t.Run("large message does not poison subsequent small message", func(t *testing.T) {
		large := newMarshalPerfMessage(8*1024*1024 + 1)
		small := newMarshalPerfMessage(1)
		for _, wire := range []string{"envelope", "triple"} {
			for _, c := range regressionCodecList() {
				// newMarshalTo returns a marshal function whose marshaler is
				// bound to the given buffer pool, so callers control sharing.
				newMarshalTo := func(pool *bufferPool) func(*pingv1.PingRequest) ([]byte, error) {
					out := &bytes.Buffer{}
					switch wire {
					case "envelope":
						w := &envelopeWriter{writer: out, codec: c.codec, bufferPool: pool}
						return func(msg *pingv1.PingRequest) ([]byte, error) {
							out.Reset()
							if err := w.Marshal(msg); err != nil {
								return nil, err
							}
							return out.Bytes(), nil
						}
					default: // triple
						m := &tripleUnaryMarshaler{writer: out, codec: c.codec, bufferPool: pool}
						return func(msg *pingv1.PingRequest) ([]byte, error) {
							out.Reset()
							if err := m.Marshal(msg); err != nil {
								return nil, err
							}
							return out.Bytes(), nil
						}
					}
				}
				// Marshal the huge message first, then the tiny one through the
				// same pool; recycled residue would leak into the small output.
				shared := newMarshalTo(newBufferPool())
				if _, err := shared(large); err != nil {
					t.Fatalf("%s/%s large marshal: %v", wire, c.name, err)
				}
				afterLarge, err := shared(small)
				if err != nil {
					t.Fatalf("%s/%s small marshal after large: %v", wire, c.name, err)
				}
				// The tiny message's wire bytes must be exactly the same as if
				// it had never been preceded by an 8MiB+1 message. The reference
				// is marshaled on an independent, uncontaminated pool.
				expected, err := newMarshalTo(newBufferPool())(small)
				if err != nil {
					t.Fatalf("%s/%s reference small marshal: %v", wire, c.name, err)
				}
				if !bytes.Equal(afterLarge, expected) {
					t.Fatalf("%s/%s small message output changed after 8MiB+1 message: got %d bytes, want %d",
						wire, c.name, len(afterLarge), len(expected))
				}
				if len(afterLarge) == 0 {
					t.Fatalf("%s/%s small message produced empty output", wire, c.name)
				}
			}
		}
	})
}

// TestMarshalPerfConcurrentSend verifies bufferPool safety under concurrent use.
func TestMarshalPerfConcurrentSend(t *testing.T) {
	// Drive concurrent envelopeWriters sharing one bufferPool to verify
	// sync.Pool race safety and that no output is lost or corrupted.
	msg := newMarshalPerfMessage(128)
	sharedPool := newBufferPool()
	// Per-iteration output length on a single reference call.
	ref := &envelopeWriter{writer: &syncBuffer{}, codec: &protoBinaryCodec{}, bufferPool: sharedPool}
	refOut := &syncBuffer{}
	ref.writer = refOut
	if err := ref.Marshal(msg); err != nil {
		t.Fatal(err)
	}
	perIter := refOut.Len()
	if perIter <= 5 {
		t.Fatalf("unexpected reference envelope length %d", perIter)
	}

	const goroutines = 32
	const iters = 100
	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)
	for range goroutines {
		wg.Go(func() {
			out := &syncBuffer{}
			w := &envelopeWriter{
				writer:     out,
				codec:      &protoBinaryCodec{},
				bufferPool: sharedPool,
			}
			for range iters {
				if err := w.Marshal(msg); err != nil {
					errCh <- fmt.Errorf("concurrent marshal: %w", err)
					return
				}
			}
			if out.Len() != iters*perIter {
				errCh <- fmt.Errorf("goroutine output length %d, want %d", out.Len(), iters*perIter)
			}
		})
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}

// Type guard: only protoBinaryCodec and the tripleServerCodecSession wrapper
// may expose the marshalAppender fast path; all other codecs must stay on
// codec.Marshal.
func TestMarshalPerfTypeGuard(t *testing.T) {
	t.Parallel()

	implements := func(c Codec) bool {
		_, ok := c.(marshalAppender)
		return ok
	}

	if !implements(&protoBinaryCodec{}) {
		t.Fatal("protoBinaryCodec must implement marshalAppender")
	}
	// tripleServerCodecSession wraps the proto codec on the server response
	// path; it must reach the fast path too, or the optimization is skipped
	// for every native triple server response.
	if !implements(&tripleServerCodecSession{delegate: &protoBinaryCodec{}}) {
		t.Fatal("tripleServerCodecSession must implement marshalAppender (server response fast path)")
	}

	never := []struct {
		name  string
		codec Codec
	}{
		{name: "noAppender wrapper", codec: &noAppenderCodec{&protoBinaryCodec{}}},
		{name: "proto wrapper (hessian inner)", codec: newProtoWrapperCodec(&hessian2Codec{})},
		{name: "hessian2", codec: &hessian2Codec{}},
		{name: "msgpack", codec: &msgpackCodec{}},
		{name: "json", codec: &protoJSONCodec{name: codecNameJSON}},
	}
	for _, tc := range never {
		if implements(tc.codec) {
			t.Fatalf("%s must NOT implement marshalAppender (fast path scope leak)", tc.name)
		}
	}
}

// TestMarshalPerfErrorGuard verifies that non-proto messages never panic:
// MarshalAppend returns errNotProto, and the envelope and triple marshalers
// surface CodeInternal on both the fast and slow paths.
func TestMarshalPerfErrorGuard(t *testing.T) {
	t.Parallel()

	nonProto := any("definitely not a proto message")

	t.Run("MarshalAppend rejects non-proto", func(t *testing.T) {
		c := &protoBinaryCodec{}
		_, err := c.MarshalAppend(make([]byte, 0, 16), nonProto)
		if err == nil {
			t.Fatal("expected errNotProto from MarshalAppend, got nil")
		}
	})

	t.Run("marshalers return CodeInternal without panicking", func(t *testing.T) {
		for _, wire := range []string{"envelope", "triple"} {
			for _, c := range regressionCodecList() {
				var err *Error
				switch wire {
				case "envelope":
					w, _ := newEnvelopeWriterForTest(t, c.codec, false, 0, 0)
					err = w.Marshal(nonProto)
				case "triple":
					m, _, _ := newTripleMarshalerForTest(t, c.codec, false, 0, 0)
					err = m.Marshal(nonProto)
				}
				wantCode(t, err, CodeInternal)
			}
		}
	})

	t.Run("primary error with nil backup is CodeInternal", func(t *testing.T) {
		// Already exercised in the fallback test; assert the plain error text
		// does not swallow the underlying errNotProto cause.
		c := &protoBinaryCodec{}
		_, err := c.MarshalAppend(nil, nonProto)
		if err == nil || !strings.Contains(err.Error(), "doesn't implement proto.Message") {
			t.Fatalf("expected errNotProto naming proto.Message, got: %v", err)
		}
	})
}

// fastPathProbeCodec distinguishes the fast and slow paths: Marshal always
// fails while MarshalAppend counts calls, so a slow-path leak fails the test.
type fastPathProbeCodec struct {
	appendCalls int
}

var _ Codec = (*fastPathProbeCodec)(nil)
var _ marshalAppender = (*fastPathProbeCodec)(nil)

func (c *fastPathProbeCodec) Name() string { return codecNameProto }
func (c *fastPathProbeCodec) Marshal(any) ([]byte, error) {
	return nil, errors.New("slow path taken: codec.Marshal must not run on the marshalAppender fast path")
}
func (c *fastPathProbeCodec) Unmarshal([]byte, any) error { return nil }
func (c *fastPathProbeCodec) MarshalAppend(dst []byte, _ any) ([]byte, error) {
	c.appendCalls++
	return append(dst, "probe-payload"...), nil
}

// TestMarshalPerfServerSessionFastPath verifies that tripleServerCodecSession
// responses reach the MarshalAppend fast path: IDL proto output stays
// byte-identical to the naked codec, non-IDL MarshalAppend equals Marshal,
// and delegates without the appender extension fall back gracefully.
func TestMarshalPerfServerSessionFastPath(t *testing.T) {
	t.Parallel()

	msg := &pingv1.PingRequest{Text: "server-session"}

	t.Run("IDL proto response matches naked codec wire", func(t *testing.T) {
		// codecSession implements marshalAppender, so tripleUnaryMarshaler
		// takes the MarshalAppend branch on the server response path.
		fast, outFast, _ := newTripleMarshalerForTest(t, &tripleServerCodecSession{delegate: &protoBinaryCodec{}}, false, 0, 0)
		if err := fast.Marshal(msg); err != nil {
			t.Fatalf("session marshal: %v", err)
		}
		ref, outRef, _ := newTripleMarshalerForTest(t, &protoBinaryCodec{}, false, 0, 0)
		if err := ref.Marshal(msg); err != nil {
			t.Fatalf("reference marshal: %v", err)
		}
		if got, want := outFast.Bytes(), outRef.Bytes(); !bytes.Equal(got, want) {
			t.Fatalf("session fast path wire mismatch: got %d bytes, want %d", len(got), len(want))
		}
	})

	t.Run("Non-IDL wrapped MarshalAppend equals Marshal", func(t *testing.T) {
		messages := []struct {
			name string
			msg  any
		}{
			{name: "scalar payload", msg: "payload"},
			{name: "one-result container", msg: []any{"payload"}},
			{name: "void container", msg: []any{}},
		}
		for _, serializeType := range []string{codecNameHessian2, codecNameMsgPack} {
			for _, mc := range messages {
				session := &tripleServerCodecSession{delegate: &protoBinaryCodec{}, serializeType: serializeType}
				got, err := session.MarshalAppend(nil, mc.msg)
				if err != nil {
					t.Fatalf("%s/%s MarshalAppend: %v", serializeType, mc.name, err)
				}
				want, err := session.Marshal(mc.msg)
				if err != nil {
					t.Fatalf("%s/%s Marshal: %v", serializeType, mc.name, err)
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("%s/%s MarshalAppend != Marshal (%d vs %d bytes)", serializeType, mc.name, len(got), len(want))
				}
			}
		}
	})

	t.Run("custom proto delegate without appender falls back", func(t *testing.T) {
		// A session over a proto codec lacking the appender extension must not
		// break: MarshalAppend degrades to marshal-plus-append, byte-equal to
		// the naked codec's output.
		session := &tripleServerCodecSession{delegate: &noAppenderCodec{&protoBinaryCodec{}}}
		got, err := session.MarshalAppend(nil, msg)
		if err != nil {
			t.Fatalf("fallback MarshalAppend: %v", err)
		}
		want, err := (&protoBinaryCodec{}).Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("fallback MarshalAppend mismatch: got %d bytes, want %d", len(got), len(want))
		}
	})

	t.Run("server response must hit MarshalAppend, not the slow path", func(t *testing.T) {
		// The probe's Marshal always fails, so a successful Marshal proves the
		// MarshalAppend fast path was taken.
		for _, compress := range []bool{false, true} {
			probe := &fastPathProbeCodec{}
			session := &tripleServerCodecSession{delegate: probe}
			m, _, _ := newTripleMarshalerForTest(t, session, compress, 0, 0)
			if err := m.Marshal(msg); err != nil {
				t.Fatalf("compressed=%v Marshal: %v (slow path leaked into server responses?)", compress, err)
			}
			if probe.appendCalls != 1 {
				t.Fatalf("compressed=%v MarshalAppend calls = %d, want 1 (fast path not taken)", compress, probe.appendCalls)
			}
		}
	})

	t.Run("envelope writer session must hit MarshalAppend", func(t *testing.T) {
		// The session may also sit behind an envelopeWriter (gRPC/Triple
		// envelope wire); the same no-slow-path guarantee must hold there.
		probe := &fastPathProbeCodec{}
		session := &tripleServerCodecSession{delegate: probe}
		w, _ := newEnvelopeWriterForTest(t, session, false, 0, 0)
		if err := w.Marshal(msg); err != nil {
			t.Fatalf("envelope Marshal: %v (slow path leaked?)", err)
		}
		if probe.appendCalls != 1 {
			t.Fatalf("envelope MarshalAppend calls = %d, want 1 (fast path not taken)", probe.appendCalls)
		}
	})
}
