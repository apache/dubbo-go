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
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"dubbo.apache.org/dubbo-go/v3/internal/httpbinding"
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
	validatedKeys := make(map[string]struct{}, len(routes))
	for index, route := range routes {
		method := strings.ToUpper(strings.TrimSpace(route.Method))
		if method == "" {
			return fmt.Errorf("HTTP route %d has an empty method", index)
		}
		if !httpbinding.IsSupportedMethod(method) {
			return fmt.Errorf("HTTP route %d uses unsupported method %q", index, method)
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
		key := httpbinding.CanonicalRouteKey(method, route.Path)
		if _, alreadyValidated := validatedKeys[key]; alreadyValidated {
			continue
		}
		if err := validator.HandlePath(method, route.Path, func(http.ResponseWriter, *http.Request, map[string]string) {}); err != nil {
			return fmt.Errorf("validate HTTP route %d (%s %s): %w", index, method, route.Path, err)
		}
		validatedKeys[key] = struct{}{}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	candidateCORS := cloneCorsConfig(m.transcodingCORS)
	if candidateCORS != nil && m.transcodingCORSAutoMethods {
		for _, route := range routes {
			method := strings.ToUpper(strings.TrimSpace(route.Method))
			if !slices.Contains(candidateCORS.AllowMethods, method) {
				candidateCORS.AllowMethods = append(candidateCORS.AllowMethods, method)
			}
		}
		slices.Sort(candidateCORS.AllowMethods)
	}

	pending := make(map[string]*httpTranscodingRoute)
	for _, route := range routes {
		method := strings.ToUpper(strings.TrimSpace(route.Method))
		key := httpbinding.CanonicalRouteKey(method, route.Path)
		transcodingRoute, exists := pending[key]
		if !exists {
			transcodingRoute = m.transcodingRoutes[key]
			if transcodingRoute != nil {
				transcodingRoute = transcodingRoute.clone()
				transcodingRoute.cors = candidateCORS
			} else {
				transcodingRoute = &httpTranscodingRoute{
					method:          method,
					path:            route.Path,
					implementations: make(map[string]httpTranscodingImplementation),
					cors:            candidateCORS,
				}
			}
			pending[key] = transcodingRoute
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

	candidateRoutes := make(map[string]*httpTranscodingRoute, len(m.transcodingRoutes)+len(pending))
	for key, route := range m.transcodingRoutes {
		clone := route.clone()
		clone.cors = candidateCORS
		candidateRoutes[key] = clone
	}
	for key, route := range pending {
		candidateRoutes[key] = route
	}

	candidateMux := m.newTranscodingMux()
	if err := m.registerTranscodingCORS(candidateMux, candidateCORS); err != nil {
		return fmt.Errorf("register transcoding CORS route: %w", err)
	}
	routeKeys := make([]string, 0, len(candidateRoutes))
	for key := range candidateRoutes {
		routeKeys = append(routeKeys, key)
	}
	// ServeMux prepends every newly registered pattern. Register broader
	// patterns first so a literal/more constrained route wins deterministically
	// when two valid templates overlap.
	sort.Slice(routeKeys, func(i, j int) bool {
		left := candidateRoutes[routeKeys[i]]
		right := candidateRoutes[routeKeys[j]]
		leftWildcards, leftLiterals := transcodingRouteSpecificity(left.path)
		rightWildcards, rightLiterals := transcodingRouteSpecificity(right.path)
		if leftWildcards != rightWildcards {
			return leftWildcards > rightWildcards
		}
		if leftLiterals != rightLiterals {
			return leftLiterals < rightLiterals
		}
		return routeKeys[i] < routeKeys[j]
	})
	for _, key := range routeKeys {
		route := candidateRoutes[key]
		if err := candidateMux.HandlePath(route.method, route.path, route.serveHTTP); err != nil {
			return fmt.Errorf("register HTTP route %q: %w", key, err)
		}
	}

	m.transcodingCORS = candidateCORS
	m.transcoding = candidateMux
	m.transcodingRoutes = candidateRoutes
	return nil
}

func transcodingRouteSpecificity(path string) (wildcards, literals int) {
	wildcards = strings.Count(path, "*")
	for index := 0; index < len(path); index++ {
		if path[index] == '{' || path[index] == '}' || path[index] == '*' {
			continue
		}
		literals++
	}
	return wildcards, literals
}

func cloneCorsConfig(config *CorsConfig) *CorsConfig {
	if config == nil {
		return nil
	}
	return &CorsConfig{
		AllowOrigins:     append([]string(nil), config.AllowOrigins...),
		AllowMethods:     append([]string(nil), config.AllowMethods...),
		AllowHeaders:     append([]string(nil), config.AllowHeaders...),
		ExposeHeaders:    append([]string(nil), config.ExposeHeaders...),
		AllowCredentials: config.AllowCredentials,
		MaxAge:           config.MaxAge,
	}
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
