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
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/quick"
	"time"
	"unicode/utf8"
)

import (
	"github.com/google/go-cmp/cmp"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

func TestGRPCHandlerSender(t *testing.T) {
	t.Parallel()
	newConn := func(web bool) *grpcHandlerConn {
		responseWriter := httptest.NewRecorder()
		protobufCodec := &protoBinaryCodec{}
		bufferPool := newBufferPool()
		request, err := http.NewRequest(
			http.MethodPost,
			"https://demo.example.com",
			strings.NewReader(""),
		)
		assert.Nil(t, err)
		return &grpcHandlerConn{
			spec:       Spec{},
			web:        web,
			bufferPool: bufferPool,
			protobuf:   protobufCodec,
			marshaler: grpcMarshaler{
				envelopeWriter: envelopeWriter{
					writer:     responseWriter,
					codec:      protobufCodec,
					bufferPool: bufferPool,
				},
			},
			responseWriter:  responseWriter,
			responseHeader:  make(http.Header),
			responseTrailer: make(http.Header),
			request:         request,
			unmarshaler: grpcUnmarshaler{
				envelopeReader: envelopeReader{
					reader:     request.Body,
					codec:      protobufCodec,
					bufferPool: bufferPool,
				},
			},
		}
	}
	t.Run("web", func(t *testing.T) {
		t.Parallel()
		testGRPCHandlerConnMetadata(t, newConn(true))
	})
	t.Run("http2", func(t *testing.T) {
		t.Parallel()
		testGRPCHandlerConnMetadata(t, newConn(false))
	})
}

func testGRPCHandlerConnMetadata(t *testing.T, conn handlerConnCloser) {
	// Closing the sender shouldn't unpredictably mutate user-visible headers or
	// trailers.
	t.Helper()
	expectHeaders := conn.ResponseHeader().Clone()
	expectTrailers := conn.ResponseTrailer().Clone()
	conn.Close(NewError(CodeUnavailable, errors.New("oh no")))
	if diff := cmp.Diff(expectHeaders, conn.ResponseHeader()); diff != "" {
		t.Errorf("headers changed:\n%s", diff)
	}
	gotTrailers := conn.ResponseTrailer()
	if diff := cmp.Diff(expectTrailers, gotTrailers); diff != "" {
		t.Errorf("trailers changed:\n%s", diff)
	}
}

func TestGRPCParseTimeout(t *testing.T) {
	t.Parallel()
	_, err := grpcParseTimeout("")
	assert.True(t, errors.Is(err, errNoTimeout))

	_, err = grpcParseTimeout("foo")
	assert.NotNil(t, err)
	_, err = grpcParseTimeout("12xS")
	assert.NotNil(t, err)
	_, err = grpcParseTimeout("999999999n") // 9 digits
	assert.NotNil(t, err)
	assert.False(t, errors.Is(err, errNoTimeout))
	_, err = grpcParseTimeout("99999999H") // 8 digits but overflows time.Duration
	assert.True(t, errors.Is(err, errNoTimeout))

	duration, err := grpcParseTimeout("45S")
	assert.Nil(t, err)
	assert.Equal(t, duration, 45*time.Second)

	const long = "99999999S"
	duration, err = grpcParseTimeout(long) // 8 digits, shouldn't overflow
	assert.Nil(t, err)
	assert.Equal(t, duration, 99999999*time.Second)
}

func TestGRPCEncodeTimeout(t *testing.T) {
	t.Parallel()
	timeout, err := grpcEncodeTimeout(time.Hour + time.Second)
	assert.Nil(t, err)
	assert.Equal(t, timeout, "3601000m")
	timeout, err = grpcEncodeTimeout(time.Duration(math.MaxInt64))
	assert.Nil(t, err)
	assert.Equal(t, timeout, "2562047H")
	timeout, err = grpcEncodeTimeout(-1 * time.Hour)
	assert.Nil(t, err)
	assert.Equal(t, timeout, "0n")
}

func TestGRPCEncodeTimeoutQuick(t *testing.T) {
	t.Parallel()
	// Ensure that the error case is actually unreachable.
	encode := func(d time.Duration) bool {
		_, err := grpcEncodeTimeout(d)
		return err == nil
	}
	if err := quick.Check(encode, nil); err != nil {
		t.Error(err)
	}
}

func TestGRPCPercentEncodingQuick(t *testing.T) {
	t.Parallel()
	pool := newBufferPool()
	roundtrip := func(input string) bool {
		if !utf8.ValidString(input) {
			return true
		}
		encoded := grpcPercentEncode(pool, input)
		decoded := grpcPercentDecode(pool, encoded)
		return decoded == input
	}
	if err := quick.Check(roundtrip, nil /* config */); err != nil {
		t.Error(err)
	}
}

func TestGRPCPercentEncoding(t *testing.T) {
	t.Parallel()
	pool := newBufferPool()
	roundtrip := func(input string) {
		assert.True(t, utf8.ValidString(input), assert.Sprintf("input invalid UTF-8"))
		encoded := grpcPercentEncode(pool, input)
		t.Logf("%q encoded as %q", input, encoded)
		decoded := grpcPercentDecode(pool, encoded)
		assert.Equal(t, decoded, input)
	}

	roundtrip("foo")
	roundtrip("foo bar")
	roundtrip(`foo%bar`)
	roundtrip("fiancée")
}

func TestGRPCWebTrailerMarshalling(t *testing.T) {
	t.Parallel()
	responseWriter := httptest.NewRecorder()
	marshaler := grpcMarshaler{
		envelopeWriter: envelopeWriter{
			writer:     responseWriter,
			bufferPool: newBufferPool(),
		},
	}
	trailer := http.Header{}
	trailer.Add("grpc-status", "0")
	trailer.Add("Grpc-Message", "Foo")
	trailer.Add("User-Provided", "bar")
	err := marshaler.MarshalWebTrailers(trailer)
	assert.Nil(t, err)
	responseWriter.Body.Next(5) // skip flags and message length
	marshaled := responseWriter.Body.String()
	assert.Equal(t, marshaled, "grpc-message: Foo\r\ngrpc-status: 0\r\nuser-provided: bar\r\n")
}
