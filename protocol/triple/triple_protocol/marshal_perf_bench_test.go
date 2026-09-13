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
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

import (
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
)

// noAppenderCodec embeds a Codec without the marshalAppender extension, so
// the marshaler's appender type assertion fails and marshaling takes the
// codec.Marshal slow path — the baseline that the fast path is measured
// against.
type noAppenderCodec struct{ Codec }

// marshalPerfPayloadSizes covers fixed-overhead-dominated small messages
// through bandwidth-dominated large messages.
var marshalPerfPayloadSizes = []int{128, 1024, 16 * 1024, 1024 * 1024}

func marshalPerfSizeLabel(size int) string {
	if size == 1024*1024 {
		return "1MiB"
	}
	return fmt.Sprintf("%dB", size)
}

func newMarshalPerfMessage(size int) *pingv1.PingRequest {
	return &pingv1.PingRequest{Text: strings.Repeat("a", size)}
}

// benchMarshalConfig selects which marshal branches a benchmark drives: gzip
// compression and the sendMaxBytes limit checks.
type benchMarshalConfig struct {
	compress     bool
	sendMaxBytes int
}

// newBenchGzipPool returns a gzip compression pool for benchmarks. The
// initial io.Discard sink is replaced by the compressionPool with Reset on
// each use.
func newBenchGzipPool() *compressionPool {
	return newCompressionPool(
		func() Decompressor { return &gzip.Reader{} },
		func() Compressor { return gzip.NewWriter(io.Discard) },
	)
}

// benchTripleUnaryMarshaler drives tripleUnaryMarshaler.Marshal with the given
// codec. protoBinaryCodec exercises the MarshalAppend fast path;
// noAppenderCodec exercises the codec.Marshal slow path. The message is typed
// as any so both IDL and non-IDL wrapper payloads can reuse this driver.
func benchTripleUnaryMarshaler(b *testing.B, codec Codec, message any, cfg benchMarshalConfig) {
	b.Helper()
	m := &tripleUnaryMarshaler{
		writer:       io.Discard,
		codec:        codec,
		bufferPool:   newBufferPool(),
		sendMaxBytes: cfg.sendMaxBytes,
	}
	if cfg.compress {
		m.compressionPool = newBenchGzipPool()
		m.compressionName = compressionGzip
		// setHeaderCanonical writes into header on the compressed path.
		m.header = make(http.Header)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := m.Marshal(message); err != nil {
			b.Fatal(err)
		}
	}
}

func runUnaryMarshalerBench(b *testing.B, codec Codec, cfg benchMarshalConfig) {
	for _, size := range marshalPerfPayloadSizes {
		b.Run(marshalPerfSizeLabel(size), func(b *testing.B) {
			benchTripleUnaryMarshaler(b, codec, newMarshalPerfMessage(size), cfg)
		})
	}
}

func BenchmarkUnaryMarshalerFastPath(b *testing.B) {
	runUnaryMarshalerBench(b, &protoBinaryCodec{}, benchMarshalConfig{})
}

func BenchmarkUnaryMarshalerSlowPath(b *testing.B) {
	runUnaryMarshalerBench(b, &noAppenderCodec{&protoBinaryCodec{}}, benchMarshalConfig{})
}

// BenchmarkUnaryMarshaler*Compressed drive the gzip-compressed fast and slow
// paths with a generous sendMaxBytes that is exercised but never tripped.
func BenchmarkUnaryMarshalerFastPathCompressed(b *testing.B) {
	runUnaryMarshalerBench(b, &protoBinaryCodec{}, benchMarshalConfig{compress: true, sendMaxBytes: 1 << 30})
}

func BenchmarkUnaryMarshalerSlowPathCompressed(b *testing.B) {
	runUnaryMarshalerBench(b, &noAppenderCodec{&protoBinaryCodec{}}, benchMarshalConfig{compress: true, sendMaxBytes: 1 << 30})
}

// benchEnvelopeWriter drives envelopeWriter.Marshal with the given codec. The
// message is typed as any so both IDL and non-IDL wrapper payloads can reuse
// this driver.
func benchEnvelopeWriter(b *testing.B, codec Codec, message any, cfg benchMarshalConfig) {
	b.Helper()
	w := &envelopeWriter{
		writer:       io.Discard,
		codec:        codec,
		bufferPool:   newBufferPool(),
		sendMaxBytes: cfg.sendMaxBytes,
	}
	if cfg.compress {
		w.compressionPool = newBenchGzipPool()
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := w.Marshal(message); err != nil {
			b.Fatal(err)
		}
	}
}

func runEnvelopeWriterBench(b *testing.B, codec Codec, cfg benchMarshalConfig) {
	for _, size := range marshalPerfPayloadSizes {
		b.Run(marshalPerfSizeLabel(size), func(b *testing.B) {
			benchEnvelopeWriter(b, codec, newMarshalPerfMessage(size), cfg)
		})
	}
}

func BenchmarkEnvelopeWriterFastPath(b *testing.B) {
	runEnvelopeWriterBench(b, &protoBinaryCodec{}, benchMarshalConfig{})
}

func BenchmarkEnvelopeWriterSlowPath(b *testing.B) {
	runEnvelopeWriterBench(b, &noAppenderCodec{&protoBinaryCodec{}}, benchMarshalConfig{})
}

// BenchmarkEnvelopeWriter*Compressed drive the gzip-compressed branch of
// envelopeWriter.Write (compression into a second pooled buffer plus the
// compressed-size limit check).
func BenchmarkEnvelopeWriterFastPathCompressed(b *testing.B) {
	runEnvelopeWriterBench(b, &protoBinaryCodec{}, benchMarshalConfig{compress: true, sendMaxBytes: 1 << 30})
}

func BenchmarkEnvelopeWriterSlowPathCompressed(b *testing.B) {
	runEnvelopeWriterBench(b, &noAppenderCodec{&protoBinaryCodec{}}, benchMarshalConfig{compress: true, sendMaxBytes: 1 << 30})
}

// newWrapperPerfMessage builds a non-IDL request payload: one string argument
// of the given size, the shape protoWrapperCodec wraps in a
// TripleRequestWrapper.
func newWrapperPerfMessage(size int) []any {
	return []any{strings.Repeat("a", size)}
}

// runWrapperBench drives both send wires with a non-IDL wrapper codec. slow
// hides MarshalAppend behind noAppenderCodec, reproducing the pre-optimization
// client behaviour, so the pair isolates the outer-wrapper allocation.
func runWrapperBench(b *testing.B, slow bool) {
	for _, inner := range []struct {
		name  string
		codec Codec
	}{
		{name: "hessian2", codec: &hessian2Codec{}},
		{name: "msgpack", codec: &msgpackCodec{}},
	} {
		b.Run(inner.name, func(b *testing.B) {
			var codec Codec = newProtoWrapperCodec(inner.codec)
			if slow {
				codec = &noAppenderCodec{codec}
			}
			for _, size := range marshalPerfPayloadSizes {
				b.Run(marshalPerfSizeLabel(size), func(b *testing.B) {
					msg := newWrapperPerfMessage(size)
					b.Run("triple", func(b *testing.B) {
						benchTripleUnaryMarshaler(b, codec, msg, benchMarshalConfig{})
					})
					b.Run("envelope", func(b *testing.B) {
						benchEnvelopeWriter(b, codec, msg, benchMarshalConfig{})
					})
				})
			}
		})
	}
}

// BenchmarkWrapperCodecFastPath / SlowPath measure the non-IDL wrapper send
// path with and without the marshalAppender fast path on both wires.
func BenchmarkWrapperCodecFastPath(b *testing.B) { runWrapperBench(b, false) }
func BenchmarkWrapperCodecSlowPath(b *testing.B) { runWrapperBench(b, true) }
