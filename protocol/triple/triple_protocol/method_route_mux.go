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
	"slices"
	"sort"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

// methodRouteMux wraps http.ServeMux to provide case-insensitive procedure
// routing without duplicating method metadata in higher layers.
//
// Lookup order:
//  1. exact match via http.ServeMux
//  2. lowercase-first-method fallback via an internal index
type methodRouteMux struct {
	exact *http.ServeMux

	mu       sync.RWMutex
	lower    map[string]methodRouteEntry
	fallback http.Handler

	transcoding                *runtime.ServeMux
	transcodingRoutes          map[string]*httpTranscodingRoute
	transcodingCORS            *CorsConfig
	transcodingCORSAutoMethods bool
}

type methodRouteEntry struct {
	pattern string
	handler http.Handler
}

func newMethodRouteMux() *methodRouteMux {
	mux := &methodRouteMux{
		exact:             http.NewServeMux(),
		lower:             make(map[string]methodRouteEntry),
		transcodingRoutes: make(map[string]*httpTranscodingRoute),
	}
	mux.transcoding = runtime.NewServeMux(runtime.WithRoutingErrorHandler(mux.handleTranscodingRoutingError))
	return mux
}

// Handle registers the handler in the exact mux and the lowercase-first-method
// fallback index. If two patterns collide after fallback normalization, the
// first registration wins while exact matching keeps the original behavior.
func (m *methodRouteMux) Handle(pattern string, handler http.Handler) {
	m.exact.Handle(pattern, handler)

	lowerKey := normalizeMethodRouteKey(pattern)
	m.mu.Lock()
	defer m.mu.Unlock()
	// Keep the first registration for a collided fallback key so mixed-case
	// registrations from different generators (for example Go/Java stubs) keep
	// deterministic behavior.
	if _, exists := m.lower[lowerKey]; !exists {
		m.lower[lowerKey] = methodRouteEntry{
			pattern: pattern,
			handler: handler,
		}
	}
}

func (m *methodRouteMux) Handler(r *http.Request) (http.Handler, string) {
	if handler, pattern := m.exact.Handler(r); pattern != "" {
		return handler, pattern
	}

	lowerKey := normalizeMethodRouteKey(r.URL.Path)
	m.mu.RLock()
	entry, ok := m.lower[lowerKey]
	m.mu.RUnlock()
	if ok {
		return entry.handler, entry.pattern
	}

	return http.NotFoundHandler(), ""
}

func (m *methodRouteMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, pattern := m.Handler(r)
	if pattern != "" {
		handler.ServeHTTP(w, r)
		return
	}

	m.mu.RLock()
	transcoding := m.transcoding
	hasTranscodingRoutes := len(m.transcodingRoutes) > 0
	m.mu.RUnlock()
	if hasTranscodingRoutes {
		transcoding.ServeHTTP(w, r)
		return
	}

	m.mu.RLock()
	fallback := m.fallback
	m.mu.RUnlock()
	if fallback != nil {
		fallback.ServeHTTP(w, r)
		return
	}

	http.NotFound(w, r)
}

func (m *methodRouteMux) SetFallbackHandler(handler http.Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fallback = handler
}

func (m *methodRouteMux) setTranscodingCORS(config *CorsConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if config == nil || len(config.AllowOrigins) == 0 {
		m.transcodingCORS = nil
		m.transcodingCORSAutoMethods = false
		return
	}
	m.transcodingCORS = buildCorsPolicy(config, nil)
	m.transcodingCORSAutoMethods = len(config.AllowMethods) == 0
	if m.transcodingCORS == nil {
		return
	}
	if err := m.transcoding.HandlePath(http.MethodOptions, "/{path=**}", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		m.mu.RLock()
		cors := m.transcodingCORS
		m.mu.RUnlock()
		if cors != nil {
			cors.handlePreflight(w, r)
			return
		}
		http.NotFound(w, r)
	}); err != nil {
		m.transcodingCORS = nil
		m.transcodingCORSAutoMethods = false
	}
}

func (m *methodRouteMux) addTranscodingMethod(method string) {
	if !m.transcodingCORSAutoMethods || m.transcodingCORS == nil || slices.Contains(m.transcodingCORS.AllowMethods, method) {
		return
	}
	m.transcodingCORS.AllowMethods = append(m.transcodingCORS.AllowMethods, method)
	sort.Strings(m.transcodingCORS.AllowMethods)
}

func (m *methodRouteMux) handleTranscodingRoutingError(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, status int) {
	if status == http.StatusNotFound {
		m.mu.RLock()
		fallback := m.fallback
		m.mu.RUnlock()
		if fallback != nil {
			fallback.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
		return
	}
	if status == http.StatusMethodNotAllowed {
		if r.Method == http.MethodOptions {
			m.mu.RLock()
			cors := m.transcodingCORS
			m.mu.RUnlock()
			if cors != nil && cors.handlePreflight(w, r) {
				return
			}
		}
		writeHTTPTranscodingError(w, http.StatusMethodNotAllowed, CodeUnimplemented, http.StatusText(status))
		return
	}
	runtime.DefaultRoutingErrorHandler(ctx, mux, marshaler, w, r, status)
}

func normalizeMethodRouteKey(path string) string {
	lastSlash := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			lastSlash = i
			break
		}
	}
	if lastSlash < 0 || lastSlash == len(path)-1 {
		return path
	}
	return path[:lastSlash+1] + lowerFirstRune(path[lastSlash+1:])
}

func lowerFirstRune(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 0 {
		return s
	}
	return string(unicode.ToLower(r)) + s[size:]
}
