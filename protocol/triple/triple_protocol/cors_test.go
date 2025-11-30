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
		{"exact match", "https://api.example.com", []string{"https://api.example.com"}, true},
		{"wildcard any", "https://foo.bar", []string{"*"}, true},
		{"wildcard subdomain", "https://a.example.com", []string{"https://*.example.com"}, true},
		{"subdomain no scheme", "https://b.example.com", []string{"*.example.com"}, true},
		{"scheme mismatch", "http://a.example.com", []string{"https://*.example.com"}, false},
		{"not matched", "https://a.other.com", []string{"https://*.example.com"}, false},
		{"empty origin", "", []string{"https://api.example.com"}, false},
		{"empty origin with wildcard", "", []string{"*"}, false},
		{"multiple origins - first match", "https://a.com", []string{"https://a.com", "https://b.com"}, true},
		{"multiple origins - second match", "https://b.com", []string{"https://a.com", "https://b.com"}, true},
		{"multiple origins - no match", "https://c.com", []string{"https://a.com", "https://b.com"}, false},
		{"multiple origins with wildcard", "https://any.com", []string{"https://a.com", "*", "https://b.com"}, true},
		// Port matching tests
		{"origin with port matches exact port", "https://api.example.com:8443", []string{"https://api.example.com:8443"}, true},
		{"origin with port does not match different port", "https://api.example.com:9999", []string{"https://api.example.com:8443"}, false},
		{"origin without port matches default port config", "https://api.example.com", []string{"https://api.example.com"}, true},
		{"origin with default port matches config without port", "https://api.example.com:443", []string{"https://api.example.com"}, true},
		{"origin with non-default port does not match config without port", "https://api.example.com:8443", []string{"https://api.example.com"}, false},
		{"http default port matching", "http://api.example.com:80", []string{"http://api.example.com"}, true},
		{"http non-default port does not match", "http://api.example.com:8080", []string{"http://api.example.com"}, false},
		// Test explicit default port in config matches origin without port
		{"config with explicit 80 matches origin with explicit 80", "http://api.example.com:80", []string{"http://api.example.com:80"}, true},
		{"config with non-default port does not match origin without port", "https://api.example.com", []string{"https://api.example.com:8443"}, false},
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

func TestPreflightSuccess(t *testing.T) {
	t.Parallel()
	handler := stubProtocolHandler{methods: map[string]struct{}{http.MethodPost: {}}}
	populated := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
		MaxAge:       86400,
	}), []protocolHandler{handler})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "X-Test")
	rr := httptest.NewRecorder()

	handled := populated.handlePreflight(rr, req)
	assert.True(t, handled)
	assert.Equal(t, rr.Code, http.StatusNoContent)
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), "https://a.com")
	gotMethods := rr.Header().Get("Access-Control-Allow-Methods")
	assert.True(t, gotMethods != "" && populated.containsMethod(http.MethodPost), assert.Sprintf("allow-methods missing POST, got %q", gotMethods))
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Headers"), "X-Test")
	assert.Equal(t, rr.Header().Get("Access-Control-Max-Age"), "86400")
}

func TestPreflightForbidden(t *testing.T) {
	t.Parallel()
	policy := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://b.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rr := httptest.NewRecorder()

	handled := policy.handlePreflight(rr, req)
	assert.True(t, handled, assert.Sprintf("preflight should be handled even when forbidden"))
	assert.Equal(t, rr.Code, http.StatusForbidden)
}

func TestAddCORSHeaders(t *testing.T) {
	t.Parallel()
	policy := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins:     []string{"https://a.com"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Expose"},
	}), nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	rr := httptest.NewRecorder()

	policy.addCORSHeaders(rr, req)
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), "https://a.com")
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Credentials"), "true")
	assert.Equal(t, rr.Header().Get("Access-Control-Expose-Headers"), "X-Expose")
	// Check Vary header more strictly using Values to handle multiple values
	vary := rr.Header().Values("Vary")
	found := false
	for _, v := range vary {
		if v == "Origin" {
			found = true
			break
		}
	}
	assert.True(t, found, assert.Sprintf("expected Vary header to contain Origin, got %v", vary))
}

func TestAddCORSHeadersNilPolicy(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	rr := httptest.NewRecorder()

	var nilPolicy *corsPolicy
	nilPolicy.addCORSHeaders(rr, req)
	// Should not add any CORS headers when policy is nil
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), "")
}

func TestAddCORSHeadersEmptyOrigin(t *testing.T) {
	t.Parallel()
	policy := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Don't set Origin header (empty origin)
	rr := httptest.NewRecorder()

	policy.addCORSHeaders(rr, req)
	// Should not add CORS headers when origin is empty
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), "")
}

func TestAddCORSHeadersMultipleOrigins(t *testing.T) {
	t.Parallel()
	policy := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins:  []string{"https://a.com", "https://b.com", "https://*.example.com"},
		ExposeHeaders: []string{"X-Expose"},
	}), nil)

	tests := []struct {
		name           string
		origin         string
		shouldAdd      bool
		expectedOrigin string
	}{
		{"first origin", "https://a.com", true, "https://a.com"},
		{"second origin", "https://b.com", true, "https://b.com"},
		{"wildcard subdomain", "https://sub.example.com", true, "https://sub.example.com"},
		{"not allowed origin", "https://c.com", false, ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			rr := httptest.NewRecorder()

			policy.addCORSHeaders(rr, req)
			got := rr.Header().Get("Access-Control-Allow-Origin")
			if tt.shouldAdd {
				assert.Equal(t, got, tt.expectedOrigin)
			} else {
				assert.Equal(t, got, "")
			}
		})
	}
}

func TestBuildCorsPolicyDisabled(t *testing.T) {
	t.Parallel()
	got := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{}), nil)
	assert.Nil(t, got)
}

func TestBuildCorsPolicyNilConfig(t *testing.T) {
	t.Parallel()
	got := buildCorsPolicy(nil, nil)
	assert.Nil(t, got)
}

func TestServeHTTPCORSIntegration(t *testing.T) {
	t.Parallel()
	// Build a minimal handler with CORS enabled.
	h := NewUnaryHandler("dummy", func() any { return nil }, func(ctx context.Context, req *Request) (*Response, error) {
		w := http.Header{}
		return &Response{Msg: struct{}{}, header: w, trailer: w}, nil
	}, WithCORS(&CorsConfig{
		AllowOrigins:     []string{"https://a.com"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Expose"},
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	req.Header.Set(tripleServiceGroup, "")
	req.Header.Set(tripleServiceVersion, "")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	assert.NotEqual(t, rr.Code, http.StatusForbidden, assert.Sprintf("request should not be forbidden for allowed origin"))
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), "https://a.com")
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Credentials"), "true")
	// Check Vary header more strictly using Values to handle multiple values
	vary := rr.Header().Values("Vary")
	found := false
	for _, v := range vary {
		if v == "Origin" {
			found = true
			break
		}
	}
	assert.True(t, found, assert.Sprintf("expected Vary header to contain Origin, got %v", vary))
}

func TestServeHTTPForbiddenOrigin(t *testing.T) {
	t.Parallel()
	h := NewUnaryHandler("dummy", func() any { return nil }, func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{Msg: struct{}{}}, nil
	}, WithCORS(&CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://b.com")
	req.Header.Set(tripleServiceGroup, "")
	req.Header.Set(tripleServiceVersion, "")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, http.StatusForbidden)
}

func TestServeHTTPNoCORSConfig(t *testing.T) {
	t.Parallel()
	h := NewUnaryHandler("dummy", func() any { return nil }, func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{Msg: struct{}{}}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	req.Header.Set(tripleServiceGroup, "")
	req.Header.Set(tripleServiceVersion, "")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	assert.NotEqual(t, rr.Code, http.StatusForbidden, assert.Sprintf("should not forbid when CORS disabled"))
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), "")
}

func TestBuildCorsPolicyMaxAge(t *testing.T) {
	t.Parallel()
	p := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
		MaxAge:       123,
	}), nil)
	assert.NotNil(t, p)
	assert.Equal(t, p.MaxAge, 123)
	p2 := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
		MaxAge:       0,
	}), nil)
	// maxAge == 0 means disable caching, should remain 0
	assert.Equal(t, p2.MaxAge, 0)
	p3 := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
		MaxAge:       -1,
	}), nil)
	// maxAge < 0 should use default value
	assert.Equal(t, p3.MaxAge, defaultPreflightMaxAge)
}

func TestBuildCorsPolicyMethodsFallback(t *testing.T) {
	t.Parallel()
	p := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)
	assert.NotNil(t, p)
	expect := map[string]struct{}{
		http.MethodGet:     {},
		http.MethodPost:    {},
		http.MethodPut:     {},
		http.MethodDelete:  {},
		http.MethodOptions: {},
	}
	for _, m := range p.AllowMethods {
		delete(expect, m)
	}
	assert.Equal(t, len(expect), 0, assert.Sprintf("fallback methods missing: %v", expect))
}

func TestHandleCORSNilConfig(t *testing.T) {
	t.Parallel()
	h := &Handler{cors: nil}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	rr := httptest.NewRecorder()

	handled := h.handleCORS(rr, req)
	assert.False(t, handled, assert.Sprintf("should return false when cors is nil"))
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), "")
}

func TestHandleCORSNoOrigin(t *testing.T) {
	t.Parallel()
	cors := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)
	h := &Handler{cors: cors}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Don't set Origin header
	rr := httptest.NewRecorder()

	handled := h.handleCORS(rr, req)
	assert.False(t, handled, assert.Sprintf("should return false when no Origin header"))
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), "")
}

func TestHandleCORSPreflight(t *testing.T) {
	t.Parallel()
	cors := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)
	h := &Handler{cors: cors}
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rr := httptest.NewRecorder()

	handled := h.handleCORS(rr, req)
	assert.True(t, handled, assert.Sprintf("should handle preflight request"))
	assert.Equal(t, rr.Code, http.StatusNoContent)
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), "https://a.com")
}

func TestHandleCORSPreflightNoOrigin(t *testing.T) {
	t.Parallel()
	cors := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)
	h := &Handler{cors: cors}
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	// Don't set Origin header
	rr := httptest.NewRecorder()

	handled := h.handleCORS(rr, req)
	assert.False(t, handled, assert.Sprintf("preflight without Origin should return false"))
}

func TestHandleCORSOptionsWithoutPreflightHeader(t *testing.T) {
	t.Parallel()
	cors := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)
	h := &Handler{cors: cors}
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	// Don't set Access-Control-Request-Method header
	rr := httptest.NewRecorder()

	handled := h.handleCORS(rr, req)
	assert.False(t, handled, assert.Sprintf("OPTIONS without preflight header should be treated as normal request"))
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), "https://a.com")
}

func TestHandleCORSForbiddenOrigin(t *testing.T) {
	t.Parallel()
	cors := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)
	h := &Handler{cors: cors}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://b.com")
	rr := httptest.NewRecorder()

	handled := h.handleCORS(rr, req)
	assert.True(t, handled, assert.Sprintf("should handle and stop processing for forbidden origin"))
	assert.Equal(t, rr.Code, http.StatusForbidden)
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), "")
}

func TestHandleCORSAllowedOrigin(t *testing.T) {
	t.Parallel()
	cors := buildCorsPolicy(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)
	h := &Handler{cors: cors}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	rr := httptest.NewRecorder()

	handled := h.handleCORS(rr, req)
	assert.False(t, handled, assert.Sprintf("should return false to continue processing"))
	assert.Equal(t, rr.Header().Get("Access-Control-Allow-Origin"), "https://a.com")
}
