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
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// writeCounter records the size of every individual Write call, so the probe
// can see how many times the marshaling path hits the underlying writer and
// whether the payloads are handed over in one piece or split.
type writeCounter struct {
	inner io.Writer
	sizes []int
}

func (w *writeCounter) Write(p []byte) (int, error) {
	w.sizes = append(w.sizes, len(p))
	if w.inner != nil {
		return w.inner.Write(p)
	}
	return len(p), nil
}

// probeHTTPClient stands in for the transport and records the shape of the
// request body it is handed: the advertised Content-Length and the number of
// bytes actually readable.
type probeHTTPClient struct {
	contentLength int64
	bodyLen       int
}

func (c *probeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.contentLength = req.ContentLength
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 64)
	for {
		n, err := req.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	c.bodyLen = len(buf)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

// TestEnvelopeWritePathProbe is a lightweight, non-benchmark probe that reports
// how many Write calls each wire makes per message, and whether the default
// unary fast path collapses the 5-byte prefix and the payload into a single
// request body. It asserts nothing about performance; it only pins down the
// write-call structure so optimization decisions rest on facts.
func TestEnvelopeWritePathProbe(t *testing.T) {
	const payloadSize = 128
	msg := newMarshalPerfMessage(payloadSize)

	// A) gRPC wire framing: the 5-byte prefix and the payload are two
	// independent Write calls on the envelope writer.
	grpcOut := &writeCounter{}
	grpcEnv := &envelopeWriter{
		writer:     grpcOut,
		codec:      &protoBinaryCodec{},
		bufferPool: newBufferPool(),
	}
	if err := grpcEnv.Marshal(msg); err != nil {
		t.Fatalf("gRPC wire marshal: %v", err)
	}

	// B) Triple wire: the payload is written once, with no envelope prefix.
	tripleOut := &writeCounter{}
	tripleMarshaler := &tripleUnaryMarshaler{
		writer:     tripleOut,
		codec:      &protoBinaryCodec{},
		bufferPool: newBufferPool(),
		header:     make(http.Header),
	}
	if err := tripleMarshaler.Marshal(msg); err != nil {
		t.Fatalf("Triple wire marshal: %v", err)
	}

	// C) Default unary client fast path: the very same two envelope writes land
	// in one pooled buffer, and the transport is handed a single body with a
	// Content-Length covering prefix + payload.
	client := &probeHTTPClient{}
	call := newUnaryFastPathCall(
		context.Background(),
		client,
		&url.URL{Scheme: "https", Host: "example.com"},
		Spec{StreamType: StreamTypeUnary, Procedure: "/probe/Ping"},
		make(http.Header),
		newBufferPool(),
	)
	call.SetValidateResponse(func(*http.Response) *Error { return nil })
	fastOut := &writeCounter{inner: call}
	fastEnv := &envelopeWriter{
		writer:     fastOut,
		codec:      &protoBinaryCodec{},
		bufferPool: newBufferPool(),
	}
	if err := fastEnv.Marshal(msg); err != nil {
		t.Fatalf("fast path marshal: %v", err)
	}
	if err := call.CloseWrite(); err != nil {
		t.Fatalf("fast path close write: %v", err)
	}

	t.Logf("A) gRPC   wire: writes=%v -> %d write call(s)", grpcOut.sizes, len(grpcOut.sizes))
	t.Logf("B) Triple wire: writes=%v -> %d write call(s)", tripleOut.sizes, len(tripleOut.sizes))
	t.Logf("C) fast path  : envelope writes=%v -> transport saw bodyLen=%d, contentLength=%d",
		fastOut.sizes, client.bodyLen, client.contentLength)

	if len(grpcOut.sizes) != 2 || grpcOut.sizes[0] != 5 {
		t.Fatalf("expected gRPC wire to emit prefix(5)+body as 2 writes, got %v", grpcOut.sizes)
	}
	if len(tripleOut.sizes) != 1 {
		t.Fatalf("expected Triple wire to emit a single write, got %v", tripleOut.sizes)
	}
	want := int64(grpcOut.sizes[0] + grpcOut.sizes[1])
	if client.contentLength != want || int64(client.bodyLen) != want {
		t.Fatalf("expected fast path to absorb the split into one %d-byte body, got bodyLen=%d contentLength=%d",
			want, client.bodyLen, client.contentLength)
	}
}
