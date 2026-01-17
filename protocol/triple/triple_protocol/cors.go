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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// CorsConfig is a CORS configuration struct for handler options.
type CorsConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// corsPolicy is an internal CORS policy.
type corsPolicy struct {
	CorsConfig
	hasWildcard bool
}

const (
	corsOrigin           = "Origin"
	corsVary             = "Vary"
	corsAllowOrigin      = "Access-Control-Allow-Origin"
	corsAllowMethods     = "Access-Control-Allow-Methods"
	corsAllowHeaders     = "Access-Control-Allow-Headers"
	corsExposeHeaders    = "Access-Control-Expose-Headers"
	corsAllowCredentials = "Access-Control-Allow-Credentials"
	corsMaxAge           = "Access-Control-Max-Age"
	corsRequestMethod    = "Access-Control-Request-Method"
	corsRequestHeaders   = "Access-Control-Request-Headers"
)

var defaultCorsMethods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

// buildCorsPolicy processes the corsPolicy with handlers and returns a configured corsPolicy.
func buildCorsPolicy(cfg *corsPolicy, handlers []protocolHandler) *corsPolicy {
	if cfg == nil || len(cfg.AllowOrigins) == 0 {
		return nil
	}

	built := &corsPolicy{
		CorsConfig: CorsConfig{
			AllowOrigins:     append([]string(nil), cfg.AllowOrigins...),
			AllowMethods:     normalizeMethods(cfg.AllowMethods, handlers),
			AllowHeaders:     append([]string(nil), cfg.AllowHeaders...),
			ExposeHeaders:    append([]string(nil), cfg.ExposeHeaders...),
			AllowCredentials: cfg.AllowCredentials,
			MaxAge:           cfg.MaxAge,
		},
		hasWildcard: hasWildcardOrigin(cfg.AllowOrigins),
	}

	if built.hasWildcard && !cfg.AllowCredentials && len(cfg.AllowOrigins) > 1 {
		logger.Warnf("[TRIPLE] CORS: wildcard \"*\" will override other origins when allowCredentials=false")
	}

	return built
}

// hasWildcardOrigin checks if "*" is present in allowOrigins.
func hasWildcardOrigin(origins []string) bool {
	for _, origin := range origins {
		if origin == constant.AnyValue {
			return true
		}
	}
	return false
}

// normalizeMethods normalizes and deduplicates CORS methods.
func normalizeMethods(configMethods []string, handlers []protocolHandler) []string {
	methodSet := make(map[string]struct{})

	// Priority 1: explicit configuration
	if len(configMethods) > 0 {
		for _, m := range configMethods {
			if m != "" {
				methodSet[strings.ToUpper(m)] = struct{}{}
			}
		}
	} else {
		// Priority 2: extract from handlers
		if len(handlers) > 0 {
			for _, hdl := range handlers {
				for m := range hdl.Methods() {
					methodSet[strings.ToUpper(m)] = struct{}{}
				}
			}
		}
		// Priority 3: use defaults
		if len(methodSet) == 0 {
			for _, m := range defaultCorsMethods {
				methodSet[m] = struct{}{}
			}
		}
	}

	// Always include OPTIONS for preflight
	methodSet[http.MethodOptions] = struct{}{}

	methods := make([]string, 0, len(methodSet))
	for m := range methodSet {
		methods = append(methods, m)
	}
	sort.Strings(methods)
	return methods
}

// matchOrigin checks if the request origin matches any allowed pattern.
func (c *corsPolicy) matchOrigin(origin string) bool {
	if origin == "" || c == nil || len(c.AllowOrigins) == 0 {
		return false
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}

	originScheme := strings.ToLower(originURL.Scheme)
	originHost := originURL.Hostname()
	originPort := originURL.Port()
	if originPort == "" {
		originPort = defaultPort(originScheme)
	}

	for _, pattern := range c.AllowOrigins {
		if pattern == constant.AnyValue {
			return true
		}

		// Try parsing pattern as URL
		patternURL, err := url.Parse(pattern)
		if err == nil && patternURL.Host != "" {
			if matchPattern(originScheme, originHost, originPort, patternURL) {
				return true
			}
			continue
		}

		// Try as hostname pattern (*.example.com or example.com)
		if matchHostnamePattern(originHost, pattern) {
			return true
		}
	}

	return false
}

// matchPattern matches origin against a URL pattern.
func matchPattern(originScheme, originHost, originPort string, patternURL *url.URL) bool {
	patternScheme := strings.ToLower(patternURL.Scheme)
	patternHost := patternURL.Hostname()
	patternPort := patternURL.Port()
	if patternPort == "" {
		patternPort = defaultPort(patternScheme)
	}

	// Scheme must match
	if patternScheme != "" && patternScheme != originScheme {
		return false
	}

	// Check host (supports *.example.com)
	if strings.HasPrefix(patternHost, "*.") {
		base := patternHost[2:]
		if !strings.HasSuffix(originHost, "."+base) || originHost == base {
			return false
		}
	} else if patternHost != originHost {
		return false
	}

	// Check port
	return patternPort == originPort
}

// matchHostnamePattern matches origin host against hostname pattern (*.example.com or example.com).
func matchHostnamePattern(originHost, pattern string) bool {
	if strings.HasPrefix(pattern, "*.") {
		base := pattern[2:]
		return strings.HasSuffix(originHost, "."+base) && originHost != base
	}
	return originHost == pattern
}

// defaultPort returns the default port for a scheme.
func defaultPort(scheme string) string {
	switch scheme {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}

// handlePreflight handles CORS preflight requests.
func (c *corsPolicy) handlePreflight(w http.ResponseWriter, r *http.Request) bool {
	if c == nil {
		return false
	}

	origin := r.Header.Get(corsOrigin)
	if origin == "" || !c.matchOrigin(origin) {
		if origin != "" {
			logger.Debugf("[TRIPLE] CORS forbidden origin: %s", origin)
		}
		w.Header().Add(corsVary, corsOrigin)
		w.WriteHeader(http.StatusForbidden)
		return true
	}

	requestedMethod := r.Header.Get(corsRequestMethod)
	if requestedMethod != "" && !c.containsMethod(requestedMethod) {
		logger.Debugf("[TRIPLE] CORS forbidden method: %s (origin: %s)", requestedMethod, origin)
		w.Header().Add(corsVary, corsOrigin)
		w.WriteHeader(http.StatusForbidden)
		return true
	}

	c.setCORSOrigin(w, origin)
	c.setAllowMethods(w)
	c.setAllowHeaders(w, r)
	if c.MaxAge > 0 {
		w.Header().Set(corsMaxAge, strconv.Itoa(c.MaxAge))
	}
	w.WriteHeader(http.StatusNoContent)
	return true
}

// addCORSHeaders adds CORS headers to the response.
func (c *corsPolicy) addCORSHeaders(w http.ResponseWriter, r *http.Request) {
	if c == nil {
		return
	}

	origin := r.Header.Get(corsOrigin)
	if origin == "" || !c.matchOrigin(origin) {
		return
	}

	c.setCORSOrigin(w, origin)
	if len(c.ExposeHeaders) > 0 {
		w.Header().Set(corsExposeHeaders, strings.Join(c.ExposeHeaders, ", "))
	}
}

// containsMethod checks if the method is allowed.
func (c *corsPolicy) containsMethod(target string) bool {
	if c == nil {
		return false
	}
	targetUpper := strings.ToUpper(target)
	for _, method := range c.AllowMethods {
		if method == targetUpper {
			return true
		}
	}
	return false
}

// setAllowMethods sets the Access-Control-Allow-Methods header.
func (c *corsPolicy) setAllowMethods(w http.ResponseWriter) {
	w.Header().Set(corsAllowMethods, strings.Join(c.AllowMethods, ", "))
}

// setAllowHeaders sets the Access-Control-Allow-Headers header.
func (c *corsPolicy) setAllowHeaders(w http.ResponseWriter, r *http.Request) {
	if len(c.AllowHeaders) > 0 {
		w.Header().Set(corsAllowHeaders, strings.Join(c.AllowHeaders, ", "))
	} else if requestedHeaders := r.Header.Get(corsRequestHeaders); requestedHeaders != "" {
		w.Header().Set(corsAllowHeaders, requestedHeaders)
	}
}

// setCORSOrigin sets the Access-Control-Allow-Origin header.
func (c *corsPolicy) setCORSOrigin(w http.ResponseWriter, origin string) {
	if c.AllowCredentials {
		w.Header().Set(corsAllowOrigin, origin)
		w.Header().Add(corsVary, corsOrigin)
		w.Header().Set(corsAllowCredentials, "true")
	} else if c.hasWildcard {
		w.Header().Set(corsAllowOrigin, constant.AnyValue)
	} else {
		w.Header().Set(corsAllowOrigin, origin)
		w.Header().Add(corsVary, corsOrigin)
	}
}
