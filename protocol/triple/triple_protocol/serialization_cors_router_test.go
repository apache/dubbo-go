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
