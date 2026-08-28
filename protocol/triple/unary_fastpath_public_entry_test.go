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
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

// TestCallUnaryFastPathWireSignature verifies that a unary call issued through
// the public client entry (clientManager.callUnary, the path reached by
// TripleInvoker.Invoke) reaches the server with a pre-declared Content-Length
// when the fast path is on, and streams without one when it is explicitly
// disabled. The fast path buffers the whole request body and sets
// Content-Length; duplexHTTPCall streams through an io.Pipe and cannot, so the
// server sees Content-Length == -1. This guards the dispatch wiring between
// the RPC layer and the gRPC protocol client.
func TestCallUnaryFastPathWireSignature(t *testing.T) {
	for _, tc := range []struct {
		name         string
		tripleConf   *global.TripleConfig
		wantPositive bool
	}{
		{"default-on", nil, true},
		{"explicitly-off", &global.TripleConfig{UnaryFastPath: false}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				mu         sync.Mutex
				contentLen int64
			)
			pingHandler := tri.NewUnaryHandler(
				"/connect.ping.v1.PingService/Ping",
				func() any { return &wrapperspb.StringValue{} },
				func(_ context.Context, req *tri.Request) (*tri.Response, error) {
					sv := req.Any().(*wrapperspb.StringValue)
					return tri.NewResponse(&wrapperspb.StringValue{Value: sv.Value}), nil
				},
			)
			wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				contentLen = r.ContentLength
				mu.Unlock()
				pingHandler.ServeHTTP(w, r)
			})
			server := &http.Server{Handler: h2c.NewHandler(wrapped, &http2.Server{})}
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer server.Close()
			go func() {
				_ = server.Serve(ln)
			}()

			url, err := common.NewURL(
				"tri://"+ln.Addr().String()+"/connect.ping.v1.PingService",
				common.WithMethods([]string{"Ping"}),
				common.WithProtocol(TRIPLE),
			)
			require.NoError(t, err)
			if tc.tripleConf != nil {
				url.SetAttribute(constant.TripleConfigKey, tc.tripleConf)
			}

			cm, err := newClientManager(url)
			require.NoError(t, err)
			defer cm.close()

			resp := &wrapperspb.StringValue{}
			err = cm.callUnary(
				context.Background(),
				"Ping",
				&wrapperspb.StringValue{Value: "hello"},
				resp,
				nil,
				nil,
			)
			require.NoError(t, err)
			assert.Equal(t, "hello", resp.Value)

			mu.Lock()
			defer mu.Unlock()
			if tc.wantPositive {
				assert.Positive(t, contentLen,
					"server saw Content-Length %d, want > 0 (fast path signature)", contentLen)
			} else {
				assert.Equal(t, int64(-1), contentLen,
					"server saw Content-Length %d, want -1 (duplex streams the body)", contentLen)
			}
		})
	}
}

// TestCallUnaryFastPathErrorPaths verifies that error propagation through the
// public client entry (clientManager.callUnary) behaves identically whether
// the fast path is on (default) or disabled (duplex): a canceled context fails
// fast, a handler error surfaces with its code and message, and a gRPC
// trailers-only response (grpc-status with no message body) is decoded
// correctly. This guards the gRPC wire error path of the fast path.
func TestCallUnaryFastPathErrorPaths(t *testing.T) {
	scenarios := []struct {
		name    string
		serve   func(context.Context, *tri.Request) (*tri.Response, error)
		wantErr func(*testing.T, error)
	}{
		{
			name: "handler-error",
			serve: func(_ context.Context, _ *tri.Request) (*tri.Response, error) {
				return nil, tri.NewError(tri.CodeUnavailable, errors.New("boom"))
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Equal(t, tri.CodeUnavailable, tri.CodeOf(err))
				assert.Contains(t, err.Error(), "boom")
			},
		},
		{
			name: "trailers-only",
			serve: func(_ context.Context, _ *tri.Request) (*tri.Response, error) {
				// A gRPC error response carries grpc-status/grpc-message in
				// trailers only, with no message body.
				return nil, tri.NewError(tri.CodeCanceled, nil)
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Equal(t, tri.CodeCanceled, tri.CodeOf(err))
			},
		},
		{
			name: "context-canceled",
			serve: func(_ context.Context, _ *tri.Request) (*tri.Response, error) {
				return tri.NewResponse(&wrapperspb.StringValue{Value: "echo"}), nil
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			for _, cfg := range []struct {
				name       string
				tripleConf *global.TripleConfig
			}{
				{"fastpath", nil},
				{"duplex", &global.TripleConfig{UnaryFastPath: false}},
			} {
				t.Run(cfg.name, func(t *testing.T) {
					pingHandler := tri.NewUnaryHandler(
						"/connect.ping.v1.PingService/Ping",
						func() any { return &wrapperspb.StringValue{} },
						sc.serve,
					)
					server := &http.Server{Handler: h2c.NewHandler(pingHandler, &http2.Server{})}
					ln, err := net.Listen("tcp", "127.0.0.1:0")
					require.NoError(t, err)
					defer server.Close()
					go func() {
						_ = server.Serve(ln)
					}()

					url, err := common.NewURL(
						"tri://"+ln.Addr().String()+"/connect.ping.v1.PingService",
						common.WithMethods([]string{"Ping"}),
						common.WithProtocol(TRIPLE),
					)
					require.NoError(t, err)
					if cfg.tripleConf != nil {
						url.SetAttribute(constant.TripleConfigKey, cfg.tripleConf)
					}

					cm, err := newClientManager(url)
					require.NoError(t, err)
					defer cm.close()

					ctx := context.Background()
					if sc.name == "context-canceled" {
						var cancel context.CancelFunc
						ctx, cancel = context.WithCancel(ctx)
						cancel()
					}
					resp := &wrapperspb.StringValue{}
					err = cm.callUnary(
						ctx,
						"Ping",
						&wrapperspb.StringValue{Value: "hello"},
						resp,
						nil,
						nil,
					)
					sc.wantErr(t, err)
				})
			}
		})
	}
}
