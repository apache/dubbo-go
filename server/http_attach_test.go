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
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/triple"
)

type httpMountTestHandler struct{}

func (h *httpMountTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func TestServerAttachHTTPHandler(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	handler := &httpMountTestHandler{}
	err = srv.AttachHTTPHandler(handler)
	require.NoError(t, err)

	assert.Same(t, handler, srv.attachedHTTPHandler)
}

func TestServerAttachHTTPHandlerRejectsNil(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	err = srv.AttachHTTPHandler(nil)
	require.ErrorContains(t, err, "must not be nil")
}

func TestServerAttachHTTPHandlerRejectsDuplicate(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	first := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	second := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	require.NoError(t, srv.AttachHTTPHandler(first))

	err = srv.AttachHTTPHandler(second)
	require.ErrorContains(t, err, "already been attached")
}

func TestServerAttachHTTPHandlerRejectsAfterServe(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	srv.serve = true
	err = srv.AttachHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	require.ErrorContains(t, err, "before Serve")
}

func TestHostAttachedHTTPHandlerRequiresTripleProtocol(t *testing.T) {
	srv, err := NewServer(
		WithServerProtocol(
			protocol.WithDubbo(),
		),
	)
	require.NoError(t, err)

	require.NoError(t, srv.AttachHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))

	err = srv.hostAttachedHTTPHandler()
	require.ErrorContains(t, err, "requires at least one triple protocol")
}

func TestHostAttachedHTTPHandlerRequiresExplicitPort(t *testing.T) {
	srv, err := NewServer(
		WithServerProtocol(
			protocol.WithTriple(),
		),
	)
	require.NoError(t, err)

	srv.cfg.Protocols[constant.TriProtocol].Port = ""
	require.NoError(t, srv.AttachHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))

	err = srv.hostAttachedHTTPHandler()
	require.ErrorContains(t, err, "requires an explicit triple port")
}

func TestServeDoesNotHostAttachedHTTPHandlerBeforeLaterStartupFailures(t *testing.T) {
	port := testFreePort(t)
	srv, err := NewServer(
		WithServerProtocol(
			protocol.WithTriple(),
			protocol.WithIp("127.0.0.1"),
			protocol.WithPort(port),
		),
	)
	require.NoError(t, err)

	require.NoError(t, srv.AttachHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			_, _ = w.Write([]byte("healthy"))
		default:
			http.NotFound(w, r)
		}
	})))

	internalProLock.Lock()
	originalInternalServices := internalProServices
	internalProServices = []*InternalService{
		{Name: "broken-internal-service"},
	}
	internalProLock.Unlock()
	t.Cleanup(func() {
		internalProLock.Lock()
		internalProServices = originalInternalServices
		internalProLock.Unlock()
	})

	t.Cleanup(func() {
		extension.GetProtocol(constant.TriProtocol).Destroy()
	})

	err = srv.Serve()
	require.ErrorContains(t, err, "internal service init func is empty")

	client := &http.Client{Timeout: 100 * time.Millisecond}
	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		status, body, reqErr := simpleHTTPRequest(client, http.MethodGet, baseURL+"/healthz")
		if reqErr == nil {
			t.Fatalf("attached HTTP handler started after Serve failure: status=%d body=%q", status, body)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
