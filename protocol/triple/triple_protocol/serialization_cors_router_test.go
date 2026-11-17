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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/emptypb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

func TestGroupedRouterPreservesCORSPreflightPolicy(t *testing.T) {
	server := NewServer("", nil)
	owner, second := &serialization.Resources{}, &serialization.Resources{}
	unary := func(context.Context, *Request) (*Response, error) { return NewResponse(&emptypb.Empty{}), nil }
	register := func(group, version, origin string, resources *serialization.Resources) {
		t.Helper()
		require.NoError(t, server.RegisterUnaryHandler("/codec.CORS/Call", func() any { return &emptypb.Empty{} }, unary, WithGroup(group), WithVersion(version), WithSerializationOwner(resources), WithCORS(&CorsConfig{AllowOrigins: []string{origin}, AllowMethods: []string{"POST"}, AllowHeaders: []string{"Content-Type", "tri-service-group", "tri-service-version"}, AllowCredentials: true, MaxAge: 123})))
	}
	register("group-a", "v1", "https://a.example", owner)
	transport := httptest.NewServer(server.mux)
	defer transport.Close()
	preflight := func(origin, method string, want int) {
		t.Helper()
		r, err := http.NewRequest(http.MethodOptions, transport.URL+"/codec.CORS/Call", nil)
		require.NoError(t, err)
		r.Header.Set(corsOrigin, origin)
		r.Header.Set(corsRequestMethod, method)
		r.Header.Set(corsRequestHeaders, "Content-Type, tri-service-group, tri-service-version")
		response, err := transport.Client().Do(r)
		require.NoError(t, err)
		defer response.Body.Close()
		require.Equal(t, want, response.StatusCode)
		if want == http.StatusNoContent {
			require.Equal(t, origin, response.Header.Get(corsAllowOrigin))
			require.Equal(t, "true", response.Header.Get(corsAllowCredentials))
			require.Contains(t, response.Header.Get(corsAllowHeaders), "tri-service-group")
			require.Equal(t, "123", response.Header.Get(corsMaxAge))
		}
	}
	preflight("https://a.example", "POST", http.StatusNoContent)
	preflight("https://evil.example", "POST", http.StatusForbidden)
	preflight("https://a.example", "DELETE", http.StatusForbidden)
	register("group-b", "v2", "https://b.example", second)
	// Preserve the historical procedure policy; do not union group permissions.
	preflight("https://a.example", "POST", http.StatusNoContent)
	preflight("https://b.example", "POST", http.StatusForbidden)
	// Actual requests still use their selected group's policy.
	r, err := http.NewRequest(http.MethodPost, transport.URL+"/codec.CORS/Call", strings.NewReader("{}"))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set(tripleServiceGroup, "group-b")
	r.Header.Set(tripleServiceVersion, "v2")
	r.Header.Set(corsOrigin, "https://b.example")
	response, err := transport.Client().Do(r)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "https://b.example", response.Header.Get(corsAllowOrigin))
	// Replacements and retirement cannot drop the independent route policy.
	register("group-a", "v1", "https://replacement.example", second)
	server.RemoveSerializationOwner(owner)
	require.NoError(t, owner.Close())
	preflight("https://a.example", "POST", http.StatusNoContent)
	server.RemoveSerializationOwner(second)
	require.NoError(t, second.Close())
	router := server.handlers["/codec.CORS/Call"]
	require.Empty(t, router.variants)
	require.Nil(t, router.serializationOwner)
	require.Nil(t, router.codecResources)
	require.Empty(t, router.implementations)
}

func TestDefaultVariantPreflightUsesProcedurePolicy(t *testing.T) {
	for _, defaultPolicy := range []bool{false, true} {
		t.Run(fmt.Sprint(defaultPolicy), func(t *testing.T) {
			server := NewServer("", nil)
			unary := func(context.Context, *Request) (*Response, error) { return NewResponse(&emptypb.Empty{}), nil }
			require.NoError(t, server.RegisterUnaryHandler("/codec.DefaultCORS/Call", func() any { return &emptypb.Empty{} }, unary, WithGroup("group-a"), WithVersion("v1"), WithCORS(&CorsConfig{AllowOrigins: []string{"https://a.example"}, AllowMethods: []string{"POST"}})))
			preflight := func(origin, method string, want int) {
				t.Helper()
				req := httptest.NewRequest(http.MethodOptions, "/codec.DefaultCORS/Call", nil)
				req.Header.Set(corsOrigin, origin)
				req.Header.Set(corsRequestMethod, method)
				rec := httptest.NewRecorder()
				server.mux.ServeHTTP(rec, req)
				require.Equal(t, want, rec.Code)
				if want == http.StatusNoContent {
					require.Equal(t, origin, rec.Header().Get(corsAllowOrigin))
				}
			}
			preflight("https://a.example", "POST", http.StatusNoContent)
			options := []HandlerOption{}
			if defaultPolicy {
				options = append(options, WithCORS(&CorsConfig{AllowOrigins: []string{"https://b.example"}, AllowMethods: []string{"POST", "DELETE"}}))
			}
			require.NoError(t, server.RegisterUnaryHandler("/codec.DefaultCORS/Call", func() any { return &emptypb.Empty{} }, unary, options...))
			preflight("https://a.example", "POST", http.StatusNoContent)
			preflight("https://b.example", "POST", http.StatusForbidden)
			preflight("https://a.example", "DELETE", http.StatusForbidden)
			// Ordinary requests retain the default variant's own policy, and OPTIONS
			// without Access-Control-Request-Method remains normal method negotiation.
			req := httptest.NewRequest(http.MethodPost, "/codec.DefaultCORS/Call", strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(corsOrigin, "https://b.example")
			rec := httptest.NewRecorder()
			server.mux.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)
			if defaultPolicy {
				require.Equal(t, "https://b.example", rec.Header().Get(corsAllowOrigin))
			} else {
				require.Empty(t, rec.Header().Get(corsAllowOrigin))
			}
			plain := httptest.NewRequest(http.MethodOptions, "/codec.DefaultCORS/Call", nil)
			plain.Header.Set(corsOrigin, "https://b.example")
			normal := httptest.NewRecorder()
			server.mux.ServeHTTP(normal, plain)
			require.Equal(t, http.StatusMethodNotAllowed, normal.Code)
		})
	}
}

func TestDefaultVariantCannotEnableProcedurePreflight(t *testing.T) {
	server := NewServer("", nil)
	unary := func(context.Context, *Request) (*Response, error) { return NewResponse(&emptypb.Empty{}), nil }
	require.NoError(t, server.RegisterUnaryHandler("/codec.NoProcedureCORS/Call", func() any { return &emptypb.Empty{} }, unary, WithGroup("group-a")))
	require.NoError(t, server.RegisterUnaryHandler("/codec.NoProcedureCORS/Call", func() any { return &emptypb.Empty{} }, unary, WithCORS(&CorsConfig{AllowOrigins: []string{"https://b.example"}})))
	req := httptest.NewRequest(http.MethodOptions, "/codec.NoProcedureCORS/Call", nil)
	req.Header.Set(corsOrigin, "https://b.example")
	req.Header.Set(corsRequestMethod, "POST")
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	require.Empty(t, rec.Header().Get(corsAllowOrigin), "a later variant must not enable a disabled procedure preflight policy")
}
