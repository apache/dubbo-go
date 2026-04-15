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

package server

import (
	"net/http"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type httpMountTestHandler struct{}

func (h *httpMountTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func TestServerMountHTTPHandler(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	handler := &httpMountTestHandler{}
	err = srv.MountHTTPHandler(handler)
	require.NoError(t, err)

	assert.Same(t, handler, srv.mountedHTTPHandlerSnapshot())
}

func TestServerMountHTTPHandlerRejectsNil(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	err = srv.MountHTTPHandler(nil)
	require.ErrorContains(t, err, "must not be nil")
}

func TestServerMountHTTPHandlerRejectsDuplicate(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	first := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	second := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	require.NoError(t, srv.MountHTTPHandler(first))

	err = srv.MountHTTPHandler(second)
	require.ErrorContains(t, err, "already been mounted")
}

func TestServerMountHTTPHandlerRejectsAfterServe(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	srv.serve = true
	err = srv.MountHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	require.ErrorContains(t, err, "before Serve")
}

func TestMountHTTPHandlersRequiresTripleProtocol(t *testing.T) {
	srv, err := NewServer(
		WithServerProtocol(
			protocol.WithDubbo(),
		),
	)
	require.NoError(t, err)

	require.NoError(t, srv.MountHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))

	err = srv.mountHTTPHandlers()
	require.ErrorContains(t, err, "requires at least one triple protocol")
}

func TestMountHTTPHandlersRequiresExplicitPort(t *testing.T) {
	srv, err := NewServer(
		WithServerProtocol(
			protocol.WithTriple(),
		),
	)
	require.NoError(t, err)

	srv.cfg.Protocols[constant.TriProtocol].Port = ""
	require.NoError(t, srv.MountHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))

	err = srv.mountHTTPHandlers()
	require.ErrorContains(t, err, "requires an explicit triple port")
}
