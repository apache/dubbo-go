/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to you under the Apache License, Version 2.0
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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dubbo.apache.org/dubbo-go/v3/global"
)

func TestRegisterHTTPHandlersDispatchesByGroupAndVersion(t *testing.T) {
	server := NewServer("", nil)
	const routePath = "/v1/books/{name}"

	route := func(body string) HTTPRoute {
		return HTTPRoute{
			Method:  http.MethodGet,
			Path:    routePath,
			RPC:     "/library.Library/GetBook",
			Group:   body,
			Version: "v1",
			Handler: func(w http.ResponseWriter, _ *http.Request, pathParams map[string]string) {
				_, _ = w.Write([]byte(body + ":" + pathParams["name"]))
			},
		}
	}

	require.NoError(t, server.RegisterHTTPHandlers([]HTTPRoute{route("g1"), route("g2")}))
	for _, test := range []struct {
		group string
		want  string
	}{
		{group: "g1", want: "g1:alice"},
		{group: "g2", want: "g2:alice"},
	} {
		req := httptest.NewRequest(http.MethodGet, "/v1/books/alice", nil)
		req.Header.Set(tripleServiceGroup, test.group)
		req.Header.Set(tripleServiceVersion, "v1")
		resp := httptest.NewRecorder()
		server.mux.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, test.want, resp.Body.String())
	}

	missing := httptest.NewRequest(http.MethodGet, "/v1/books/alice", nil)
	missingResp := httptest.NewRecorder()
	server.mux.ServeHTTP(missingResp, missing)
	assert.Equal(t, http.StatusNotImplemented, missingResp.Code)
	assert.Contains(t, missingResp.Body.String(), `"code":12`)

	wrongMethod := httptest.NewRequest(http.MethodPost, "/v1/books/alice", nil)
	wrongMethodResp := httptest.NewRecorder()
	server.mux.ServeHTTP(wrongMethodResp, wrongMethod)
	assert.Equal(t, http.StatusMethodNotAllowed, wrongMethodResp.Code)

	notFound := httptest.NewRequest(http.MethodGet, "/v1/missing", nil)
	notFoundResp := httptest.NewRecorder()
	server.mux.ServeHTTP(notFoundResp, notFound)
	assert.Equal(t, http.StatusNotFound, notFoundResp.Code)
}

func TestRegisterHTTPHandlersRejectsConflictsAtomically(t *testing.T) {
	server := NewServer("", nil)
	handler := func(http.ResponseWriter, *http.Request, map[string]string) {}

	err := server.RegisterHTTPHandlers([]HTTPRoute{
		{Method: http.MethodGet, Path: "/v1/books/{name}", RPC: "/library.Library/GetBook", Handler: handler},
		{Method: http.MethodGet, Path: "/v1/books/{name}", RPC: "/library.Library/FindBook", Handler: handler},
	})
	require.Error(t, err)

	req := httptest.NewRequest(http.MethodGet, "/v1/books/alice", nil)
	resp := httptest.NewRecorder()
	server.mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestRegisterHTTPHandlersRejectsEquivalentPathConflicts(t *testing.T) {
	server := NewServer("", nil)
	handler := func(http.ResponseWriter, *http.Request, map[string]string) {}

	err := server.RegisterHTTPHandlers([]HTTPRoute{
		{Method: http.MethodGet, Path: "/v1/books/{name}", RPC: "/library.Library/GetBook", Handler: handler},
		{Method: http.MethodGet, Path: "/v1/books/{id=*}", RPC: "/library.Library/FindBook", Handler: handler},
	})
	require.Error(t, err)

	req := httptest.NewRequest(http.MethodGet, "/v1/books/alice", nil)
	resp := httptest.NewRecorder()
	server.mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestRegisterHTTPHandlersRejectsUnsupportedMethod(t *testing.T) {
	server := NewServer("", nil)
	err := server.RegisterHTTPHandlers([]HTTPRoute{{
		Method:  "REPORT",
		Path:    "/v1/books/{name}",
		RPC:     "/library.Library/GetBook",
		Handler: func(http.ResponseWriter, *http.Request, map[string]string) {},
	}})
	require.Error(t, err)
}

func TestRegisterHTTPHandlersPrefersSpecificOverlappingTemplate(t *testing.T) {
	server := NewServer("", nil)
	called := ""
	require.NoError(t, server.RegisterHTTPHandlers([]HTTPRoute{
		{
			Method: http.MethodGet,
			Path:   "/v1/{name}",
			RPC:    "/library.Library/GetByName",
			Handler: func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
				called = "wildcard"
				_, _ = w.Write([]byte(called))
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/v1/books/{id}",
			RPC:    "/library.Library/GetBook",
			Handler: func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
				called = "literal"
				_, _ = w.Write([]byte(called))
			},
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/books/42", nil)
	resp := httptest.NewRecorder()
	server.mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "literal", called)
}

func TestRegisterHTTPHandlersRejectsInvalidBatchBeforeRegistration(t *testing.T) {
	server := NewServer("", nil)
	handler := func(http.ResponseWriter, *http.Request, map[string]string) {}

	err := server.RegisterHTTPHandlers([]HTTPRoute{
		{Method: http.MethodGet, Path: "/v1/books/{name}", RPC: "/library.Library/GetBook", Handler: handler},
		{Method: http.MethodGet, Path: "/v1/books/{", RPC: "/library.Library/Invalid", Handler: handler},
	})
	require.Error(t, err)

	req := httptest.NewRequest(http.MethodGet, "/v1/books/alice", nil)
	resp := httptest.NewRecorder()
	server.mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestRegisterHTTPHandlersCORSPreflight(t *testing.T) {
	server := NewServer("", &global.TripleConfig{Cors: &global.CorsConfig{
		AllowOrigins: []string{"https://example.com"},
	}})
	require.NoError(t, server.RegisterHTTPHandlers([]HTTPRoute{{
		Method: http.MethodPatch,
		Path:   "/v1/books/{name}",
		RPC:    "/library.Library/UpdateBook",
		Handler: func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
			_, _ = w.Write([]byte("ok"))
		},
	}}))

	preflight := httptest.NewRequest(http.MethodOptions, "/v1/books/42", nil)
	preflight.Header.Set("Origin", "https://example.com")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPatch)
	preflightResponse := httptest.NewRecorder()
	server.mux.ServeHTTP(preflightResponse, preflight)
	assert.Equal(t, http.StatusNoContent, preflightResponse.Code)
	assert.Equal(t, "https://example.com", preflightResponse.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, preflightResponse.Header().Get("Access-Control-Allow-Methods"), http.MethodPatch)

	request := httptest.NewRequest(http.MethodPatch, "/v1/books/42", nil)
	request.Header.Set("Origin", "https://example.com")
	response := httptest.NewRecorder()
	server.mux.ServeHTTP(response, request)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "https://example.com", response.Header().Get("Access-Control-Allow-Origin"))
}
