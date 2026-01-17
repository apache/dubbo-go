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
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

func convertCorsConfigForTest(cfg *global.CorsConfig) *corsPolicy {
	if cfg == nil {
		return nil
	}
	return &corsPolicy{
		CorsConfig: CorsConfig{
			AllowOrigins:     append([]string(nil), cfg.AllowOrigins...),
			AllowMethods:     append([]string(nil), cfg.AllowMethods...),
			AllowHeaders:     append([]string(nil), cfg.AllowHeaders...),
			ExposeHeaders:    append([]string(nil), cfg.ExposeHeaders...),
			AllowCredentials: cfg.AllowCredentials,
			MaxAge:           cfg.MaxAge,
		},
	}
}

func TestMatchOrigin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		origin  string
		allowed []string
		want    bool
	}{
		{"exact match", "https://api.example.com", []string{"https://api.example.com"}, true},
		{"wildcard any", "https://foo.bar", []string{"*"}, true},
		{"subdomain wildcard", "https://a.example.com", []string{"https://*.example.com"}, true},
		{"subdomain no scheme", "https://b.example.com", []string{"*.example.com"}, true},
		{"scheme mismatch", "http://a.example.com", []string{"https://*.example.com"}, false},
		{"not matched", "https://other.com", []string{"https://example.com"}, false},
		{"empty origin", "", []string{"https://api.example.com"}, false},
		{"default port https", "https://api.example.com:443", []string{"https://api.example.com"}, true},
		{"default port http", "http://api.example.com:80", []string{"http://api.example.com"}, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &corsPolicy{CorsConfig: CorsConfig{AllowOrigins: tt.allowed}}
			assert.Equal(t, c.matchOrigin(tt.origin), tt.want)
		})
	}
}

func TestAddCORSHeaders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		policy     *corsPolicy
		origin     string
		wantOrigin string
		wantCreds  bool
	}{
		{"allowed with creds", buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
			AllowOrigins:     []string{"https://a.com"},
			AllowCredentials: true,
		}), nil), "https://a.com", "https://a.com", true},
		{"nil policy", nil, "https://a.com", "", false},
		{"empty origin", buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
			AllowOrigins: []string{"https://a.com"},
		}), nil), "", "", false},
		{"not allowed", buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
			AllowOrigins: []string{"https://a.com"},
		}), nil), "https://b.com", "", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rr := httptest.NewRecorder()

			tt.policy.addCORSHeaders(rr, req)
			assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), tt.wantOrigin)
			if tt.wantCreds {
				assert.Equal(t, rr.Header().Get("Access-Control-Allow-Credentials"), "true")
			}
		})
	}
}

func TestBuildCorsPolicy(t *testing.T) {
	t.Parallel()
	assert.Nil(t, buildCorsPolicy(nil, nil))
	assert.Nil(t, buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{}), nil))

	p := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
		MaxAge:       123,
	}), nil)
	assert.NotNil(t, p)
	assert.Equal(t, p.MaxAge, 123)
}

func TestServeHTTPCORS(t *testing.T) {
	t.Parallel()
	// Allowed origin
	h1 := NewUnaryHandler("dummy", func() any { return nil }, func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{Msg: struct{}{}, header: http.Header{}, trailer: http.Header{}}, nil
	}, WithCORS(&CorsConfig{AllowOrigins: []string{"https://a.com"}, AllowCredentials: true}))

	req1 := httptest.NewRequest(http.MethodPost, "/", nil)
	req1.Header.Set("Origin", "https://a.com")
	req1.Header.Set(tripleServiceGroup, "")
	req1.Header.Set(tripleServiceVersion, "")
	rr1 := httptest.NewRecorder()
	h1.ServeHTTP(rr1, req1)
	assert.Equal(t, rr1.Header().Get("Access-Control-Allow-Origin"), "https://a.com")
	assert.Equal(t, rr1.Header().Get("Access-Control-Allow-Credentials"), "true")

	// Forbidden origin
	h2 := NewUnaryHandler("dummy", func() any { return nil }, func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{Msg: struct{}{}}, nil
	}, WithCORS(&CorsConfig{AllowOrigins: []string{"https://a.com"}}))

	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.Header.Set("Origin", "https://b.com")
	req2.Header.Set(tripleServiceGroup, "")
	req2.Header.Set(tripleServiceVersion, "")
	rr2 := httptest.NewRecorder()
	h2.ServeHTTP(rr2, req2)
	assert.Equal(t, rr2.Code, http.StatusForbidden)
}

func TestHandleCORS(t *testing.T) {
	t.Parallel()
	cors := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)

	tests := []struct {
		name            string
		cors            *corsPolicy
		method          string
		origin          string
		preflightHeader string
		wantHandled     bool
		wantCode        int
	}{
		{"nil config", nil, http.MethodGet, "https://a.com", "", false, 0},
		{"preflight success", cors, http.MethodOptions, "https://a.com", http.MethodPost, true, http.StatusNoContent},
		{"forbidden origin", cors, http.MethodGet, "https://b.com", "", true, http.StatusForbidden},
		{"allowed origin", cors, http.MethodGet, "https://a.com", "", false, 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := &Handler{cors: tt.cors}
			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.preflightHeader != "" {
				req.Header.Set("Access-Control-Request-Method", tt.preflightHeader)
			}
			rr := httptest.NewRecorder()

			handled := h.handleCORS(rr, req)
			assert.Equal(t, handled, tt.wantHandled)
			if tt.wantCode != 0 {
				assert.Equal(t, rr.Code, tt.wantCode)
			}
		})
	}
}
