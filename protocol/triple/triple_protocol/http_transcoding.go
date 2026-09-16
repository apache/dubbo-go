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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

// HTTPRoute is a single google.api.http route registered on the Triple
// listener. The handler performs request decoding and RPC invocation.
type HTTPRoute struct {
	Method  string
	Path    string
	RPC     string
	Group   string
	Version string
	Handler runtime.HandlerFunc
}

type httpTranscodingRoute struct {
	mu              sync.RWMutex
	method          string
	path            string
	implementations map[string]httpTranscodingImplementation
	cors            *CorsConfig
}

type httpTranscodingImplementation struct {
	rpc     string
	handler runtime.HandlerFunc
}

// RegisterHTTPHandlers registers validated HTTP routes. Routes with the same
// method/path may coexist when they select different group/version
// implementations, while conflicting RPCs for one selector are rejected.
func (s *Server) RegisterHTTPHandlers(routes []HTTPRoute) error {
	if len(routes) == 0 {
		return nil
	}
	if s == nil || s.mux == nil {
		return fmt.Errorf("triple server route mux is nil")
	}
	return s.mux.registerHTTPHandlers(routes)
}

func (m *methodRouteMux) registerHTTPHandlers(routes []HTTPRoute) error {
	// Validate every pattern before mutating the live route table. The gateway
	// mux does not expose a pattern compiler, so a temporary mux is used for
	// validation and the same pattern is registered below only after all input
	// has passed validation.
	validator := runtime.NewServeMux()
	for index, route := range routes {
		method := strings.ToUpper(strings.TrimSpace(route.Method))
		if method == "" {
			return fmt.Errorf("HTTP route %d has an empty method", index)
		}
		if strings.TrimSpace(route.Path) == "" {
			return fmt.Errorf("HTTP route %d has an empty path", index)
		}
		if strings.TrimSpace(route.RPC) == "" {
			return fmt.Errorf("HTTP route %d has an empty RPC name", index)
		}
		if route.Handler == nil {
			return fmt.Errorf("HTTP route %d has a nil handler", index)
		}
		if err := validator.HandlePath(method, route.Path, func(http.ResponseWriter, *http.Request, map[string]string) {}); err != nil {
			return fmt.Errorf("validate HTTP route %d (%s %s): %w", index, method, route.Path, err)
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	pending := make(map[string]*httpTranscodingRoute)
	order := make([]string, 0, len(routes))
	for _, route := range routes {
		method := strings.ToUpper(strings.TrimSpace(route.Method))
		key := method + " " + route.Path
		transcodingRoute, exists := pending[key]
		if !exists {
			transcodingRoute = m.transcodingRoutes[key]
			if transcodingRoute != nil {
				transcodingRoute = transcodingRoute.clone()
			} else {
				transcodingRoute = &httpTranscodingRoute{
					method:          method,
					path:            route.Path,
					implementations: make(map[string]httpTranscodingImplementation),
					cors:            m.transcodingCORS,
				}
			}
			pending[key] = transcodingRoute
			order = append(order, key)
		}

		identifier := getIdentifier(route.Group, route.Version)
		if implementation, exists := transcodingRoute.implementations[identifier]; exists && implementation.rpc != route.RPC {
			return fmt.Errorf("HTTP route %q selects conflicting RPCs %q and %q for group %q and version %q", key, implementation.rpc, route.RPC, route.Group, route.Version)
		}
		transcodingRoute.implementations[identifier] = httpTranscodingImplementation{
			rpc:     route.RPC,
			handler: route.Handler,
		}
	}

	for _, key := range order {
		m.addTranscodingMethod(pending[key].method)
		if existing, exists := m.transcodingRoutes[key]; exists {
			existing.mu.Lock()
			existing.implementations = pending[key].implementations
			existing.mu.Unlock()
			continue
		}
		transcodingRoute := pending[key]
		if err := m.transcoding.HandlePath(transcodingRoute.method, transcodingRoute.path, transcodingRoute.serveHTTP); err != nil {
			return fmt.Errorf("register HTTP route %q: %w", key, err)
		}
		m.transcodingRoutes[key] = transcodingRoute
	}
	return nil
}

func (r *httpTranscodingRoute) serveHTTP(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	if r.cors != nil {
		r.cors.addCORSHeaders(w, req)
	}
	identifier := getIdentifier(req.Header.Get(tripleServiceGroup), req.Header.Get(tripleServiceVersion))
	r.mu.RLock()
	implementation, ok := r.implementations[identifier]
	r.mu.RUnlock()
	if !ok {
		writeHTTPTranscodingError(w, http.StatusNotImplemented, CodeUnimplemented,
			fmt.Sprintf("no implementation found for service group %s and service version %s", req.Header.Get(tripleServiceGroup), req.Header.Get(tripleServiceVersion)))
		return
	}
	implementation.handler(w, req, pathParams)
}

func (r *httpTranscodingRoute) clone() *httpTranscodingRoute {
	r.mu.RLock()
	defer r.mu.RUnlock()
	clone := &httpTranscodingRoute{
		method:          r.method,
		path:            r.path,
		implementations: make(map[string]httpTranscodingImplementation, len(r.implementations)),
		cors:            r.cors,
	}
	for identifier, implementation := range r.implementations {
		clone.implementations[identifier] = implementation
	}
	return clone
}

type httpTranscodingErrorBody struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
	Details []any  `json:"details"`
}

func writeHTTPTranscodingError(w http.ResponseWriter, status int, code Code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(httpTranscodingErrorBody{
		Code:    uint32(code),
		Message: message,
		Details: []any{},
	})
}
