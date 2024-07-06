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
	"net/http"
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
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

	srv := NewServer("127.0.0.1:20000")
	for _, test := range tests {
		err := srv.RegisterUnaryHandler(test.path, nil, nil)
		assert.Nil(t, err)
		_, pattern := srv.mux.Handler(&http.Request{
			URL: &url.URL{
				Path: test.path,
			},
		})
		assert.Equal(t, test.path, pattern)
	}
}
