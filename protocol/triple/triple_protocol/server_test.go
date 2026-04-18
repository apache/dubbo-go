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
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

func TestServer_RegisterMuxHandle(t *testing.T) {
	tests := []struct {
		desc         string
		path         string
		registerFunc func(srv *Server, path string) error
	}{
		{
			desc: "RegisterUnaryHandler_MuxHandle",
			path: "/Unary",
			registerFunc: func(srv *Server, path string) error {
				return srv.RegisterUnaryHandler(path, nil, nil)
			},
		},
		{
			desc: "RegisterClientStreamHandler_MuxHandle",
			path: "/ClientStream",
			registerFunc: func(srv *Server, path string) error {
				return srv.RegisterClientStreamHandler(path, nil)
			},
		},
		{
			desc: "RegisterServerStreamHandler_MuxHandle",
			path: "/ServerStream",
			registerFunc: func(srv *Server, path string) error {
				return srv.RegisterServerStreamHandler(path, nil, nil)
			},
		},
		{
			desc: "RegisterBidiStreamHandler_MuxHandle",
			path: "/BidiStream",
			registerFunc: func(srv *Server, path string) error {
				return srv.RegisterBidiStreamHandler(path, nil)
			},
		},
		{
			desc: "RegisterCompatUnaryHandler_MuxHandle",
			path: "/CompatUnary",
			registerFunc: func(srv *Server, path string) error {
				return srv.RegisterCompatUnaryHandler(path, "", nil, nil)
			},
		},
		{
			desc: "RegisterCompatStreamHandler_MuxHandle",
			path: "/CompatStream",
			registerFunc: func(srv *Server, path string) error {
				return srv.RegisterCompatStreamHandler(path, nil, StreamTypeBidi, nil)
			},
		},
	}

	srv := NewServer("127.0.0.1:20000", &global.TripleConfig{
		Http3: &global.Http3Config{
			Enable:      true,
			Negotiation: true,
		},
	})
	for _, test := range tests {
		err := test.registerFunc(srv, test.path)
		require.NoError(t, err)
		_, pattern := srv.mux.Handler(&http.Request{
			URL: &url.URL{
				Path: test.path,
			},
		})
		assert.Equal(t, test.path, pattern)
	}
}

func TestNewQUICConfig(t *testing.T) {
	t.Run("defaults_preserved_when_unset", func(t *testing.T) {
		quicConfig, err := newQUICConfig(&global.Http3Config{})
		require.NoError(t, err)
		require.NotNil(t, quicConfig)
		assert.Zero(t, quicConfig.KeepAlivePeriod)
		assert.Zero(t, quicConfig.MaxIdleTimeout)
		assert.Zero(t, quicConfig.MaxIncomingStreams)
		assert.Zero(t, quicConfig.MaxIncomingUniStreams)
	})

	t.Run("explicit_fields_are_mapped", func(t *testing.T) {
		quicConfig, err := newQUICConfig(&global.Http3Config{
			KeepAlivePeriod:       "15s",
			MaxIdleTimeout:        "30s",
			MaxIncomingStreams:    128,
			MaxIncomingUniStreams: 64,
		})
		require.NoError(t, err)
		require.NotNil(t, quicConfig)
		assert.Equal(t, 15*time.Second, quicConfig.KeepAlivePeriod)
		assert.Equal(t, 30*time.Second, quicConfig.MaxIdleTimeout)
		assert.Equal(t, int64(128), quicConfig.MaxIncomingStreams)
		assert.Equal(t, int64(64), quicConfig.MaxIncomingUniStreams)
	})

	t.Run("invalid_keep_alive_period_returns_error", func(t *testing.T) {
		quicConfig, err := newQUICConfig(&global.Http3Config{
			KeepAlivePeriod: "invalid",
		})
		require.Error(t, err)
		assert.Nil(t, quicConfig)
		assert.ErrorContains(t, err, "keep-alive-period")
	})

	t.Run("invalid_max_idle_timeout_returns_error", func(t *testing.T) {
		quicConfig, err := newQUICConfig(&global.Http3Config{
			MaxIdleTimeout: "invalid",
		})
		require.Error(t, err)
		assert.Nil(t, quicConfig)
		assert.ErrorContains(t, err, "max-idle-timeout")
	})
}

func TestServer_HTTP3PathsUseQUICConfigHelper(t *testing.T) {
	t.Run("start_http3_returns_parse_error", func(t *testing.T) {
		srv := NewServer("127.0.0.1:0", &global.TripleConfig{
			Http3: &global.Http3Config{
				KeepAlivePeriod: "invalid",
			},
		})

		err := srv.startHttp3(&tls.Config{})
		require.Error(t, err)
		assert.ErrorContains(t, err, "keep-alive-period")
		assert.Nil(t, srv.http3Srv)
	})

	t.Run("start_http2_and_http3_returns_parse_error", func(t *testing.T) {
		srv := NewServer("127.0.0.1:0", &global.TripleConfig{
			Http3: &global.Http3Config{
				MaxIdleTimeout: "invalid",
			},
		})

		err := srv.startHttp2AndHttp3(&tls.Config{})
		require.Error(t, err)
		assert.ErrorContains(t, err, "max-idle-timeout")
		assert.Nil(t, srv.http3Srv)
		assert.Nil(t, srv.httpSrv)
	})
}
