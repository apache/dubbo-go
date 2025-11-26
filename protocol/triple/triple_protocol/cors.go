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
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

// corsConfig is an internal CORS configuration struct that doesn't depend on global package.
type corsConfig struct {
	allowOrigins     []string
	allowMethods     []string
	allowHeaders     []string
	exposeHeaders    []string
	allowCredentials bool
	maxAge           int
	hasWildcard      bool
}

const defaultPreflightMaxAge = 86400

// convertCorsConfig converts a *global.CorsConfig to an internal *corsConfig.
func convertCorsConfig(cfg *global.CorsConfig) *corsConfig {
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

func buildCorsConfig(cfg *corsConfig, handlers []protocolHandler) *corsConfig {
	if cfg == nil {
		return nil
	}
	if len(cfg.allowOrigins) == 0 {
		// Empty origins means CORS is disabled.
		return nil
	}

	built := &corsConfig{
		allowOrigins:     append([]string(nil), cfg.allowOrigins...),
		allowHeaders:     append([]string(nil), cfg.allowHeaders...),
		exposeHeaders:    append([]string(nil), cfg.exposeHeaders...),
		allowCredentials: cfg.allowCredentials,
		maxAge:           cfg.maxAge,
		hasWildcard:      hasWildcardOrigin(cfg.allowOrigins),
	}

	built.allowMethods = normalizeCorsMethods(cfg.allowMethods, handlers)

	if built.maxAge <= 0 {
		built.maxAge = defaultPreflightMaxAge
	}

	return built
}

// normalizeCorsMethods normalizes and deduplicates CORS methods, adding defaults if needed.
func normalizeCorsMethods(configuredMethods []string, handlers []protocolHandler) []string {
	methodSet := make(map[string]struct{})

	// Add configured methods (normalized to uppercase)
	for _, m := range configuredMethods {
		if m == "" {
			continue
		}
		methodSet[strings.ToUpper(m)] = struct{}{}
	}

	// If no methods configured, try to infer from handlers
	if len(methodSet) == 0 {
		for _, hdl := range handlers {
			for m := range hdl.Methods() {
				methodSet[strings.ToUpper(m)] = struct{}{}
			}
		}
	}

	// If still no methods, use defaults
	if len(methodSet) == 0 {
		for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
			methodSet[m] = struct{}{}
		}
	}

	// OPTIONS is always included for preflight requests
	methodSet[http.MethodOptions] = struct{}{}

	// Convert set to sorted slice
	methods := make([]string, 0, len(methodSet))
	for m := range methodSet {
		methods = append(methods, m)
	}
	sort.Strings(methods)
	return methods
}

// matchOrigin checks if the request origin matches any allowed pattern.
// Supports:
//   - exact match
//   - "*" wildcard
//   - subdomain wildcard: "*.example.com" or "https://*.example.com"
func matchOrigin(origin string, allowedOrigins []string) bool {
	if origin == "" || len(allowedOrigins) == 0 {
		return false
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	originHost := originURL.Hostname()
	originScheme := originURL.Scheme

	for _, allowed := range allowedOrigins {
		switch allowed {
		case "":
			continue
		case "*":
			return true
		case origin:
			return true
		}

		allowedURL, err := url.Parse(allowed)
		if err == nil && allowedURL.Hostname() != "" {
			if allowedURL.Scheme != "" && allowedURL.Scheme != originScheme {
				continue
			}
			allowedHost := allowedURL.Hostname()
			if strings.HasPrefix(allowedHost, "*.") {
				if subdomainOf(originHost, strings.TrimPrefix(allowedHost, "*.")) {
					return true
				}
				continue
			}
			if allowedHost == originHost {
				return true
			}
			continue
		}

		if strings.HasPrefix(allowed, "*.") {
			if subdomainOf(originHost, strings.TrimPrefix(allowed, "*.")) {
				return true
			}
		} else if allowed == originHost {
			return true
		}
	}

	return false
}

func subdomainOf(originHost, base string) bool {
	if base == "" || originHost == "" {
		return false
	}
	return originHost != base && strings.HasSuffix(originHost, "."+base)
}

func handlePreflight(w http.ResponseWriter, r *http.Request, cors *corsConfig) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || cors == nil {
		return false
	}

	if !matchOrigin(origin, cors.allowOrigins) {
		logger.Warnf("[TRIPLE] CORS preflight forbidden origin: %s", origin)
		w.WriteHeader(http.StatusForbidden)
		return true
	}

	requestedMethod := r.Header.Get("Access-Control-Request-Method")
	allowedMethods := cors.allowMethods
	if requestedMethod != "" && !containsMethod(allowedMethods, requestedMethod) {
		w.WriteHeader(http.StatusForbidden)
		return true
	}

	applyCORSOrigin(w, cors, origin)
	if len(allowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
	}

	if len(cors.allowHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(cors.allowHeaders, ", "))
	} else if requestedHeaders := r.Header.Get("Access-Control-Request-Headers"); requestedHeaders != "" {
		w.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
	}

	w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cors.maxAge))

	w.WriteHeader(http.StatusNoContent)
	return true
}

func allowOrigin(origin string, cors *corsConfig) bool {
	if matchOrigin(origin, cors.allowOrigins) {
		return true
	}
	logger.Warnf("[TRIPLE] CORS forbidden origin: %s", origin)
	return false
}

func addCORSHeaders(w http.ResponseWriter, r *http.Request, cors *corsConfig) {
	if cors == nil {
		return
	}
	origin := r.Header.Get("Origin")
	if origin == "" || !matchOrigin(origin, cors.allowOrigins) {
		return
	}

	applyCORSOrigin(w, cors, origin)
	if len(cors.exposeHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(cors.exposeHeaders, ", "))
	}
}

func applyCORSOrigin(w http.ResponseWriter, cors *corsConfig, origin string) {
	if cors.allowCredentials {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Add("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		return
	}

	if cors.hasWildcard {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Add("Vary", "Origin")
}

func containsMethod(methods []string, target string) bool {
	target = strings.ToUpper(target)
	for _, method := range methods {
		if strings.ToUpper(method) == target {
			return true
		}
	}
	return false
}

func hasWildcardOrigin(origins []string) bool {
	for _, origin := range origins {
		if origin == "*" {
			return true
		}
	}
	return false
}
