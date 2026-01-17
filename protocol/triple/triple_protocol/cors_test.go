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

type stubProtocolHandler struct {
	methods map[string]struct{}
}

func (s stubProtocolHandler) Methods() map[string]struct{} {
	return s.methods
}

// convertCorsConfigForTest converts a *global.CorsConfig to an internal *corsPolicy for testing.
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

func (s stubProtocolHandler) ContentTypes() map[string]struct{} { return nil }
func (s stubProtocolHandler) SetTimeout(r *http.Request) (context.Context, context.CancelFunc, error) {
	return r.Context(), nil, nil
}
func (s stubProtocolHandler) CanHandlePayload(*http.Request, string) bool { return true }
func (s stubProtocolHandler) NewConn(http.ResponseWriter, *http.Request) (handlerConnCloser, bool) {
	return nil, false
}

func TestMatchOriginVariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		origin  string
		allowed []string
		want    bool
	}{
		// Basic matching
		{"exact match", "https://api.example.com", []string{"https://api.example.com"}, true},
		{"wildcard any", "https://foo.bar", []string{"*"}, true},
		{"wildcard subdomain", "https://a.example.com", []string{"https://*.example.com"}, true},
		{"subdomain no scheme", "https://b.example.com", []string{"*.example.com"}, true},
		{"scheme mismatch", "http://a.example.com", []string{"https://*.example.com"}, false},
		{"not matched", "https://a.other.com", []string{"https://*.example.com"}, false},
		// Edge cases
		{"empty origin", "", []string{"https://api.example.com"}, false},
		{"multiple origins match", "https://b.com", []string{"https://a.com", "https://b.com"}, true},
		{"multiple origins no match", "https://c.com", []string{"https://a.com", "https://b.com"}, false},
		// Port matching
		{"port exact match", "https://api.example.com:8443", []string{"https://api.example.com:8443"}, true},
		{"port mismatch", "https://api.example.com:9999", []string{"https://api.example.com:8443"}, false},
		{"default port https", "https://api.example.com:443", []string{"https://api.example.com"}, true},
		{"default port http", "http://api.example.com:80", []string{"http://api.example.com"}, true},
		{"non-default port no match", "https://api.example.com:8443", []string{"https://api.example.com"}, false},
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
		name         string
		policy       *corsPolicy
		origin       string
		wantOrigin   string
		wantCredFlag bool
	}{
		{"normal", buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
			AllowOrigins:     []string{"https://a.com"},
			AllowCredentials: true,
		}), nil), "https://a.com", "https://a.com", true},
		{"nil policy", nil, "https://a.com", "", false},
		{"empty origin", buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
			AllowOrigins: []string{"https://a.com"},
		}), nil), "", "", false},
		{"not allowed", buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
			AllowOrigins: []string{"https://a.com"},
		}), nil), "https://c.com", "", false},
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
			if tt.wantCredFlag {
				assert.Equal(t, rr.Header().Get("Access-Control-Allow-Credentials"), "true")
			}
		})
	}
}

func TestBuildCorsPolicy(t *testing.T) {
	t.Parallel()
	// Nil/disabled config
	assert.Nil(t, buildCorsPolicy(nil, nil))
	assert.Nil(t, buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{}), nil))

	// With config
	p := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
		MaxAge:       123,
	}), nil)
	assert.NotNil(t, p)
	assert.Equal(t, p.MaxAge, 123)
}

func TestServeHTTPCORS(t *testing.T) {
	t.Parallel()
	// Allowed origin with credentials
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
		wantOrigin      string
	}{
		{"nil config", nil, http.MethodGet, "https://a.com", "", false, 0, ""},
		{"no origin", cors, http.MethodGet, "", "", false, 0, ""},
		{"preflight success", cors, http.MethodOptions, "https://a.com", http.MethodPost, true, http.StatusNoContent, "https://a.com"},
		{"preflight no origin", cors, http.MethodOptions, "", http.MethodPost, false, 0, ""},
		{"OPTIONS without preflight header", cors, http.MethodOptions, "https://a.com", "", false, 0, "https://a.com"},
		{"forbidden origin", cors, http.MethodGet, "https://b.com", "", true, http.StatusForbidden, ""},
		{"allowed origin", cors, http.MethodGet, "https://a.com", "", false, 0, "https://a.com"},
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
			assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), tt.wantOrigin)
		})
	}
}
