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
)

type stubProtocolHandler struct {
	methods map[string]struct{}
}

func (s stubProtocolHandler) Methods() map[string]struct{} {
	return s.methods
}

// convertCorsConfigForTest converts a *global.CorsConfig to an internal *corsConfig for testing.
func convertCorsConfigForTest(cfg *global.CorsConfig) *corsConfig {
	if cfg == nil {
		return nil
	}
	return &corsConfig{
		allowOrigins:     append([]string(nil), cfg.AllowOrigins...),
		allowMethods:     append([]string(nil), cfg.AllowMethods...),
		allowHeaders:     append([]string(nil), cfg.AllowHeaders...),
		exposeHeaders:    append([]string(nil), cfg.ExposeHeaders...),
		allowCredentials: cfg.AllowCredentials,
		maxAge:           cfg.MaxAge,
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
	}

	for _, tt := range tests {
		c := &corsConfig{allowOrigins: tt.allowed}
		if got := c.matchOrigin(tt.origin); got != tt.want {
			t.Fatalf("%s: matchOrigin(%q, %v) = %v, want %v", tt.name, tt.origin, tt.allowed, got, tt.want)
		}
	}
}

func TestPreflightSuccess(t *testing.T) {
	handler := stubProtocolHandler{methods: map[string]struct{}{http.MethodPost: {}}}
	populated := buildCorsConfig(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), []protocolHandler{handler})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "X-Test")
	rr := httptest.NewRecorder()

	if handled := populated.handlePreflight(rr, req); !handled {
		t.Fatalf("expected preflight to be handled")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://a.com" {
		t.Fatalf("unexpected allow-origin: %q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Methods"); got == "" || !populated.containsMethod(http.MethodPost) {
		t.Fatalf("allow-methods missing POST, got %q", rr.Header().Get("Access-Control-Allow-Methods"))
	}
	if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "X-Test" {
		t.Fatalf("allow-headers echo expected, got %q", got)
	}
	if got := rr.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Fatalf("unexpected max-age: %q", got)
	}
}

func TestPreflightForbidden(t *testing.T) {
	policy := buildCorsConfig(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://b.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rr := httptest.NewRecorder()

	handled := policy.handlePreflight(rr, req)
	if !handled {
		t.Fatalf("preflight should be handled even when forbidden")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for forbidden origin, got %d", rr.Code)
	}
}

func TestAddCORSHeaders(t *testing.T) {
	policy := buildCorsConfig(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins:     []string{"https://a.com"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Expose"},
	}), nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	rr := httptest.NewRecorder()

	policy.addCORSHeaders(rr, req)
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://a.com" {
		t.Fatalf("unexpected allow-origin: %q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("credentials header missing")
	}
	if got := rr.Header().Get("Access-Control-Expose-Headers"); got != "X-Expose" {
		t.Fatalf("unexpected expose headers: %q", got)
	}
	// Check Vary header more strictly using Values to handle multiple values
	vary := rr.Header().Values("Vary")
	found := false
	for _, v := range vary {
		if v == "Origin" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Vary header to contain Origin, got %v", vary)
	}
}

func TestAddCORSHeadersNilPolicy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	rr := httptest.NewRecorder()

	var nilPolicy *corsConfig
	nilPolicy.addCORSHeaders(rr, req)
	// Should not add any CORS headers when policy is nil
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS headers for nil policy, got %q", got)
	}
}

func TestAddCORSHeadersEmptyOrigin(t *testing.T) {
	policy := buildCorsConfig(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Don't set Origin header (empty origin)
	rr := httptest.NewRecorder()

	policy.addCORSHeaders(rr, req)
	// Should not add CORS headers when origin is empty
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS headers for empty origin, got %q", got)
	}
}

func TestAddCORSHeadersMultipleOrigins(t *testing.T) {
	policy := buildCorsConfig(convertCorsConfigForTest(&global.CorsConfig{
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
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			rr := httptest.NewRecorder()

			policy.addCORSHeaders(rr, req)
			got := rr.Header().Get("Access-Control-Allow-Origin")
			if tt.shouldAdd {
				if got != tt.expectedOrigin {
					t.Fatalf("expected allow-origin %q, got %q", tt.expectedOrigin, got)
				}
			} else {
				if got != "" {
					t.Fatalf("expected no allow-origin, got %q", got)
				}
			}
		})
	}
}

func TestBuildCorsPolicyDisabled(t *testing.T) {
	if got := buildCorsConfig(convertCorsConfigForTest(&global.CorsConfig{}), nil); got != nil {
		t.Fatalf("expected nil policy for empty origins, got %+v", got)
	}
}

func TestBuildCorsPolicyNilConfig(t *testing.T) {
	if got := buildCorsConfig(nil, nil); got != nil {
		t.Fatalf("expected nil policy for nil config, got %+v", got)
	}
}

func TestServeHTTPCORSIntegration(t *testing.T) {
	// Build a minimal handler with CORS enabled.
	h := NewUnaryHandler("dummy", func() any { return nil }, func(ctx context.Context, req *Request) (*Response, error) {
		w := http.Header{}
		return &Response{Msg: struct{}{}, header: w, trailer: w}, nil
	}, WithCORS(&global.CorsConfig{
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
	if rr.Code == http.StatusForbidden {
		t.Fatalf("request should not be forbidden for allowed origin")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://a.com" {
		t.Fatalf("unexpected allow-origin: %q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("missing allow-credentials header")
	}
	// Check Vary header more strictly using Values to handle multiple values
	vary := rr.Header().Values("Vary")
	found := false
	for _, v := range vary {
		if v == "Origin" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Vary header to contain Origin, got %v", vary)
	}
}

func TestServeHTTPForbiddenOrigin(t *testing.T) {
	h := NewUnaryHandler("dummy", func() any { return nil }, func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{Msg: struct{}{}}, nil
	}, WithCORS(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://b.com")
	req.Header.Set(tripleServiceGroup, "")
	req.Header.Set(tripleServiceVersion, "")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for forbidden origin, got %d", rr.Code)
	}
}

func TestServeHTTPNoCORSConfig(t *testing.T) {
	h := NewUnaryHandler("dummy", func() any { return nil }, func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{Msg: struct{}{}}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://a.com")
	req.Header.Set(tripleServiceGroup, "")
	req.Header.Set(tripleServiceVersion, "")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusForbidden {
		t.Fatalf("should not forbid when CORS disabled")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("CORS headers should not be injected when disabled, got %q", got)
	}
}

func TestBuildCorsPolicyMaxAge(t *testing.T) {
	p := buildCorsConfig(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
		MaxAge:       123,
	}), nil)
	if p == nil {
		t.Fatalf("policy should not be nil")
	}
	if p.maxAge != 123 {
		t.Fatalf("expected maxAge=123, got %d", p.maxAge)
	}
	p2 := buildCorsConfig(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
		MaxAge:       0,
	}), nil)
	if p2.maxAge != defaultPreflightMaxAge {
		t.Fatalf("expected default maxAge %d, got %d", defaultPreflightMaxAge, p2.maxAge)
	}
}

func TestBuildCorsPolicyMethodsFallback(t *testing.T) {
	p := buildCorsConfig(convertCorsConfigForTest(&global.CorsConfig{
		AllowOrigins: []string{"https://a.com"},
	}), nil)
	if p == nil {
		t.Fatalf("policy should not be nil")
	}
	expect := map[string]struct{}{
		http.MethodGet:     {},
		http.MethodPost:    {},
		http.MethodPut:     {},
		http.MethodDelete:  {},
		http.MethodOptions: {},
	}
	for _, m := range p.allowMethods {
		delete(expect, m)
	}
	if len(expect) != 0 {
		t.Fatalf("fallback methods missing: %v", expect)
	}
}
