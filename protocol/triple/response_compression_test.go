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

package triple

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestResolveHandlerCompressMinBytesValidation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		configured int
		override   string
		wantError  bool
	}{
		{"zero rollback", 0, "", false},
		{"configured threshold", 1024, "", false},
		{"explicit service zero", 1024, "0", false},
		{"negative global", -1, "", true},
		{"negative service", 1024, "-1", true},
		{"malformed service", 1024, "1KiB", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := NewOptions(WithServerCompressMinBytes(tc.configured)).Triple
			url := common.NewURLWithOptions(common.WithAttribute(constant.TripleConfigKey, config))
			if tc.override != "" {
				url.SetParam(ServerCompressMinBytesParam, tc.override)
			}
			_, err := resolveHandlerOptions(url)
			if tc.wantError {
				require.ErrorContains(t, err, "expected a non-negative integer")
			} else {
				require.NoError(t, err)
			}
			clone := config.Clone()
			require.Equal(t, strconv.Itoa(tc.configured), strconv.Itoa(*clone.ServerCompressMinBytes))
			*clone.ServerCompressMinBytes = 256
			require.Equal(t, tc.configured, *config.ServerCompressMinBytes, "clone must not alias threshold pointer")
		})
	}
}

func safetyPayload(t *testing.T, size int) []byte {
	t.Helper()
	for n := size - 6; n < size; n++ {
		if n >= 0 {
			payload := bytes.Repeat([]byte("x"), n)
			if proto.Size(&wrapperspb.BytesValue{Value: payload}) == size {
				return payload
			}
		}
	}
	t.Fatalf("cannot make serialized size %d", size)
	return nil
}

func safetyHandler(t *testing.T, path string, config *global.TripleConfig, override string) http.Handler {
	t.Helper()
	url := common.NewURLWithOptions(common.WithAttribute(constant.TripleConfigKey, config))
	if override != "" {
		url.SetParam(ServerCompressMinBytesParam, override)
	}
	options, err := resolveHandlerOptions(url)
	require.NoError(t, err)
	return tri.NewUnaryHandler(path, func() any { return &wrapperspb.BytesValue{} },
		func(_ context.Context, req *tri.Request) (*tri.Response, error) {
			return tri.NewResponse(req.Msg), nil
		}, options...)
}

func safetyServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(handler)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

func safetyWireCall(t *testing.T, server *httptest.Server, path, wire string, payload []byte, requestGzip, acceptGzip bool) (*http.Response, []byte) {
	t.Helper()
	encoded, err := proto.Marshal(&wrapperspb.BytesValue{Value: payload})
	require.NoError(t, err)
	flag := byte(0)
	if requestGzip {
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		_, err = writer.Write(encoded)
		require.NoError(t, err)
		require.NoError(t, writer.Close())
		encoded, flag = compressed.Bytes(), 1
	}
	contentType, accept, encoding := "application/proto", "Accept-Encoding", "Content-Encoding"
	if wire == "grpc" {
		body := make([]byte, 5, 5+len(encoded))
		body[0] = flag
		binary.BigEndian.PutUint32(body[1:], uint32(len(encoded)))
		encoded = append(body, encoded...)
		contentType, accept, encoding = "application/grpc+proto", "Grpc-Accept-Encoding", "Grpc-Encoding"
	}
	request, err := http.NewRequest(http.MethodPost, server.URL+path, bytes.NewReader(encoded))
	require.NoError(t, err)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Te", "trailers")
	// Explicit identity also prevents net/http's automatic Accept-Encoding.
	request.Header.Set(accept, "identity")
	if acceptGzip {
		request.Header.Set(accept, "gzip")
	}
	if requestGzip {
		request.Header.Set(encoding, "gzip")
	}
	response, err := server.Client().Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, 2, response.ProtoMajor)
	return response, body
}

func safetyAssertEcho(t *testing.T, response *http.Response, body []byte, wire string, payload []byte, compressed bool) {
	t.Helper()
	require.Equal(t, http.StatusOK, response.StatusCode, "%s", body)
	if wire == "grpc" {
		require.Equal(t, "0", response.Trailer.Get("Grpc-Status"))
		require.GreaterOrEqual(t, len(body), 5)
		require.Equal(t, compressed, body[0] == 1)
		require.Equal(t, len(body)-5, int(binary.BigEndian.Uint32(body[1:5])))
		body = body[5:]
		require.Contains(t, response.Header.Get("Grpc-Accept-Encoding"), "gzip")
	} else {
		require.Equal(t, compressed, response.Header.Get("Content-Encoding") == "gzip")
		require.Contains(t, response.Header.Get("Accept-Encoding"), "gzip")
	}
	if compressed {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		require.NoError(t, err)
		body, err = io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
	}
	var message wrapperspb.BytesValue
	require.NoError(t, proto.Unmarshal(body, &message))
	require.Equal(t, payload, message.Value)
}

type safetyThreshold struct {
	name    string
	options []Option
	minimum int
}

func safetyThresholds(explicit ...int) []safetyThreshold {
	thresholds := []safetyThreshold{{name: "default", minimum: 512}}
	for _, min := range explicit {
		thresholds = append(thresholds, safetyThreshold{fmt.Sprint(min), []Option{WithServerCompressMinBytes(min)}, min})
	}
	return thresholds
}

// TestServerDefaultCompressionWire verifies that responses below the configured
// threshold are sent uncompressed and responses at or above it stay compressed,
// across both wires, gzip request variants and configuration sources.
func TestServerDefaultCompressionWire(t *testing.T) {
	const path = "/test.DefaultCompression/Echo"
	for _, tc := range []struct {
		name     string
		config   *global.TripleConfig
		attach   bool
		override string
		minimum  int
	}{
		{"no configuration", nil, false, "", 512},
		{"nil configuration", nil, true, "", 512},
		{"empty configuration", &global.TripleConfig{}, true, "", 512},
		{"default instance", global.DefaultTripleConfig(), true, "", 512},
		{"protocol zero", NewOptions(WithServerCompressMinBytes(0)).Triple, true, "", 0},
		{"protocol 1024", NewOptions(WithServerCompressMinBytes(1024)).Triple, true, "", 1024},
		{"service zero", NewOptions(WithServerCompressMinBytes(1024)).Triple, true, "0", 0},
		{"service 256", NewOptions(WithServerCompressMinBytes(1024)).Triple, true, "256", 256},
		{"service 512", NewOptions(WithServerCompressMinBytes(0)).Triple, true, "512", 512},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url := common.NewURLWithOptions()
			if tc.attach {
				url.SetAttribute(constant.TripleConfigKey, tc.config)
			}
			if tc.override != "" {
				url.SetParam(ServerCompressMinBytesParam, tc.override)
			}
			options, err := resolveHandlerOptions(url)
			require.NoError(t, err)
			handler := tri.NewUnaryHandler(path, func() any { return &wrapperspb.BytesValue{} },
				func(_ context.Context, req *tri.Request) (*tri.Response, error) { return tri.NewResponse(req.Msg), nil }, options...)
			server := safetyServer(t, handler)
			for _, wire := range []string{"grpc", "triple"} {
				for _, requestGzip := range []bool{false, true} {
					for _, size := range []int{0, 255, 256, 511, 512, 513, 1023, 1024} {
						t.Run(fmt.Sprintf("%s/gzip%t/%d", wire, requestGzip, size), func(t *testing.T) {
							var payload []byte
							if size != 0 {
								payload = safetyPayload(t, size)
							}
							response, body := safetyWireCall(t, server, path, wire, payload, requestGzip, true)
							safetyAssertEcho(t, response, body, wire, payload, size >= tc.minimum)
						})
					}
				}
			}
		})
	}
}

// TestServerDefaultCompressionServiceIsolation verifies that a per-service
// override to zero leaves sibling services unaffected and that resolving
// handler options does not modify the shared configuration.
func TestServerDefaultCompressionServiceIsolation(t *testing.T) {
	config := global.DefaultTripleConfig()
	mux := http.NewServeMux()
	mux.Handle("/test.Default/Echo", safetyHandler(t, "/test.Default/Echo", config, ""))
	mux.Handle("/test.Rollback/Echo", safetyHandler(t, "/test.Rollback/Echo", config, "0"))
	server := safetyServer(t, mux)
	for _, wire := range []string{"grpc", "triple"} {
		for _, path := range []string{"/test.Default/Echo", "/test.Rollback/Echo"} {
			for _, size := range []int{511, 512, 513, 511} {
				payload := safetyPayload(t, size)
				response, body := safetyWireCall(t, server, path, wire, payload, false, true)
				safetyAssertEcho(t, response, body, wire, payload, size >= 512 || path == "/test.Rollback/Echo")
			}
		}
	}
	require.Nil(t, config.ServerCompressMinBytes, "resolving services must not mutate shared configuration")
}

func TestServerCompressMinBytes(t *testing.T) {
	for _, tc := range []struct{ threshold, limit, want int }{
		{0, 64, 0}, {256, 64, 65}, {2048, 64, 65},
		{64, 64, 64}, {65, 64, 65}, {256, 4096, 256},
		{256, 0, 256}, {256, -1, 256},
		{math.MaxInt, math.MaxInt, math.MaxInt},
		{math.MaxInt, math.MaxInt - 1, math.MaxInt},
	} {
		require.Equal(t, tc.want, serverCompressMinBytes(tc.threshold, tc.limit), "%+v", tc)
	}
}

func TestResolveServerSendMaxBytes(t *testing.T) {
	for _, tc := range []struct {
		name, legacy, configured string
		want                     int
	}{
		{"defaults", "", "", constant.DefaultMaxServerSendMsgSize},
		{"legacy", "64B", "", 64},
		{"config overrides legacy", "64B", "128B", 128},
		{"invalid config keeps legacy", "64B", "invalid", 64},
		{"zero config keeps legacy", "64B", "0", 64},
		{"invalid legacy uses default", "invalid", "", constant.DefaultMaxServerSendMsgSize},
		{"zero legacy uses default", "0", "", constant.DefaultMaxServerSendMsgSize},
		{"config without legacy", "", "1KiB", 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url := common.NewURLWithOptions()
			url.SetParam(constant.MaxServerSendMsgSize, tc.legacy)
			require.Equal(t, tc.want, resolveServerSendMaxBytes(url, &global.TripleConfig{MaxServerSendMsgSize: tc.configured}))
		})
	}
	require.Equal(t, constant.DefaultMaxServerSendMsgSize, resolveServerSendMaxBytes(common.NewURLWithOptions(), nil))
}

func TestServerCompressionSendLimitSafetyWire(t *testing.T) {
	const path = "/test.CompressionSafety/Echo"
	for _, threshold := range safetyThresholds(0, 256, 2048) {
		t.Run(threshold.name, func(t *testing.T) {
			config := NewOptions(append(threshold.options, WithMaxServerSendMsgSize("64B"))...).Triple
			server := safetyServer(t, safetyHandler(t, path, config, ""))
			for _, wire := range []string{"grpc", "triple"} {
				for _, requestGzip := range []bool{false, true} {
					for _, size := range []int{63, 64, 65, 128} {
						t.Run(fmt.Sprintf("%s/gzip%t/%d", wire, requestGzip, size), func(t *testing.T) {
							payload := safetyPayload(t, size)
							response, body := safetyWireCall(t, server, path, wire, payload, requestGzip, true)
							safetyAssertEcho(t, response, body, wire, payload, threshold.minimum == 0 || size > 64)
						})
					}
				}
			}
		})
	}
}

func TestServerCompressionSendLimitStillEnforced(t *testing.T) {
	const path = "/test.CompressionSafety/Echo"
	for _, wire := range []string{"grpc", "triple"} {
		for _, mode := range []string{"random", "no-accept"} {
			for _, threshold := range safetyThresholds(0, 256, 2048) {
				t.Run(fmt.Sprintf("%s/%s/%s", wire, mode, threshold.name), func(t *testing.T) {
					config := NewOptions(append(threshold.options, WithMaxServerSendMsgSize("64B"))...).Triple
					server := safetyServer(t, safetyHandler(t, path, config, ""))
					payload := safetyPayload(t, 128)
					if mode == "random" {
						_, err := rand.New(rand.NewSource(20261015)).Read(payload)
						require.NoError(t, err)
					}
					response, body := safetyWireCall(t, server, path, wire, payload, false, mode != "no-accept")
					if wire == "grpc" {
						status := response.Trailer.Get("Grpc-Status")
						if status == "" {
							status = response.Header.Get("Grpc-Status")
						}
						require.Equal(t, "8", status)
					} else {
						var failure struct {
							Code string `json:"code"`
						}
						require.NoError(t, json.Unmarshal(body, &failure))
						require.Equal(t, "resource_exhausted", failure.Code)
					}
				})
			}
		}
	}
}

func TestServerCompressionThresholdServiceIsolation(t *testing.T) {
	config := NewOptions(WithServerCompressMinBytes(256), WithMaxServerSendMsgSize("4KiB")).Triple
	mux := http.NewServeMux()
	mux.Handle("/test.Safety/Threshold", safetyHandler(t, "/test.Safety/Threshold", config, ""))
	mux.Handle("/test.Safety/Rollback", safetyHandler(t, "/test.Safety/Rollback", config, "0"))
	server := safetyServer(t, mux)
	for _, tc := range []struct {
		path string
		size int
		want bool
	}{
		{"/test.Safety/Threshold", 128, false},
		{"/test.Safety/Rollback", 128, true},
		{"/test.Safety/Threshold", 255, false},
		{"/test.Safety/Threshold", 256, true},
		{"/test.Safety/Threshold", 257, true},
	} {
		for _, wire := range []string{"grpc", "triple"} {
			payload := safetyPayload(t, tc.size)
			response, body := safetyWireCall(t, server, tc.path, wire, payload, false, true)
			safetyAssertEcho(t, response, body, wire, payload, tc.want)
		}
	}
}

func TestServerCompressionSendLimitSafetyStream(t *testing.T) {
	for _, threshold := range safetyThresholds(2048) {
		t.Run(threshold.name, func(t *testing.T) {
			testServerCompressionSendLimitSafetyStream(t, threshold.options)
		})
	}
}

func testServerCompressionSendLimitSafetyStream(t *testing.T, thresholdOptions []Option) {
	const path = "/test.CompressionSafety/Stream"
	sizes := []int{63, 64, 65, 128}
	payloads := make(map[int][]byte, len(sizes))
	for _, size := range sizes {
		payloads[size] = safetyPayload(t, size)
	}
	config := NewOptions(append(thresholdOptions, WithMaxServerSendMsgSize("64B"))...).Triple
	url := common.NewURLWithOptions(common.WithAttribute(constant.TripleConfigKey, config))
	options, err := resolveHandlerOptions(url)
	require.NoError(t, err)
	handler := tri.NewServerStreamHandler(path, func() any { return &wrapperspb.BytesValue{} },
		func(_ context.Context, _ *tri.Request, stream *tri.ServerStream) error {
			for _, size := range sizes {
				if err := stream.Send(&wrapperspb.BytesValue{Value: payloads[size]}); err != nil {
					return err
				}
			}
			return nil
		}, options...)
	server := safetyServer(t, handler)
	for _, requestGzip := range []bool{false, true} {
		response, body := safetyWireCall(t, server, path, "grpc", nil, requestGzip, true)
		require.Equal(t, "0", response.Trailer.Get("Grpc-Status"))
		for _, size := range sizes {
			require.GreaterOrEqual(t, len(body), 5)
			length := int(binary.BigEndian.Uint32(body[1:5])) + 5
			require.GreaterOrEqual(t, len(body), length)
			safetyAssertEcho(t, response, body[:length], "grpc", payloads[size], size > 64)
			body = body[length:]
		}
		require.Empty(t, body)
	}
}
