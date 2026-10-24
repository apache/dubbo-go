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
	"context"
	"fmt"
	"net/http"
	"testing"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

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
