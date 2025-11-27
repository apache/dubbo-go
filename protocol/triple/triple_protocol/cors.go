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

const (
	defaultPreflightMaxAge = 86400 // 24 hours in seconds
	wildcardOrigin         = "*"
)

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

// build processes the corsConfig with handlers and returns a fully configured corsConfig.
func (c *corsConfig) build(handlers []protocolHandler) *corsConfig {
	if c == nil {
		return nil
	}
	if len(c.allowOrigins) == 0 {
		return nil
	}

	built := &corsConfig{
		allowOrigins:     append([]string(nil), c.allowOrigins...),
		allowHeaders:     append([]string(nil), c.allowHeaders...),
		exposeHeaders:    append([]string(nil), c.exposeHeaders...),
		allowCredentials: c.allowCredentials,
		maxAge:           c.maxAge,
		hasWildcard:      c.checkHasWildcard(),
	}

	built.allowMethods = c.normalizeMethods(handlers)

	if built.maxAge <= 0 {
		built.maxAge = defaultPreflightMaxAge
	}

	return built
}

// buildCorsConfig is a convenience function that calls build on the corsConfig.
func buildCorsConfig(cfg *corsConfig, handlers []protocolHandler) *corsConfig {
	return cfg.build(handlers)
}

// normalizeMethods normalizes and deduplicates CORS methods, adding defaults if needed.
func (c *corsConfig) normalizeMethods(handlers []protocolHandler) []string {
	methodSet := c.collectMethods(handlers)
	methodSet[http.MethodOptions] = struct{}{} // Always include OPTIONS for preflight

	methods := make([]string, 0, len(methodSet))
	for m := range methodSet {
		methods = append(methods, m)
	}
	sort.Strings(methods)
	return methods
}

// collectMethods collects methods from config, handlers, or defaults.
func (c *corsConfig) collectMethods(handlers []protocolHandler) map[string]struct{} {
	methodSet := make(map[string]struct{})

	for _, m := range c.allowMethods {
		if m != "" {
			methodSet[strings.ToUpper(m)] = struct{}{}
		}
	}

	if len(methodSet) == 0 {
		for _, hdl := range handlers {
			for m := range hdl.Methods() {
				methodSet[strings.ToUpper(m)] = struct{}{}
			}
		}
	}

	if len(methodSet) == 0 {
		for _, m := range defaultCorsMethods {
			methodSet[m] = struct{}{}
		}
	}

	return methodSet
}

// matchOrigin checks if the request origin matches any allowed pattern.
// Supports:
//   - exact match
//   - "*" wildcard
//   - subdomain wildcard: "*.example.com" or "https://*.example.com"
func (c *corsConfig) matchOrigin(origin string) bool {
	if origin == "" || len(c.allowOrigins) == 0 {
		return false
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	originHost := originURL.Hostname()
	originScheme := originURL.Scheme

	for _, allowed := range c.allowOrigins {
		if allowed == "" {
			continue
		}
		if c.matchesOrigin(allowed, origin, originHost, originScheme) {
			return true
		}
	}
	return false
}

// matchesOrigin checks if a single allowed pattern matches the origin.
func (c *corsConfig) matchesOrigin(allowed, origin, originHost, originScheme string) bool {
	if allowed == wildcardOrigin || allowed == origin {
		return true
	}

	// Try parsing as URL first (handles "https://*.example.com" format)
	if allowedURL, err := url.Parse(allowed); err == nil && allowedURL.Hostname() != "" {
		if allowedURL.Scheme != "" && allowedURL.Scheme != originScheme {
			return false
		}
		return c.matchesHost(allowedURL.Hostname(), originHost)
	}

	// Try as hostname-only format (handles "*.example.com" format)
	return c.matchesHost(allowed, originHost)
}

// matchesHost checks if an allowed host pattern matches the origin host.
func (c *corsConfig) matchesHost(allowedHost, originHost string) bool {
	if strings.HasPrefix(allowedHost, "*.") {
		base := allowedHost[2:] // Skip "*."
		return c.isSubdomainOf(originHost, base)
	}
	return allowedHost == originHost
}

func (c *corsConfig) isSubdomainOf(originHost, base string) bool {
	if base == "" || originHost == "" {
		return false
	}
	return originHost != base && strings.HasSuffix(originHost, "."+base)
}

func (c *corsConfig) handlePreflight(w http.ResponseWriter, r *http.Request) bool {
	if c == nil {
		return false
	}
	origin := r.Header.Get(corsOrigin)
	if origin == "" {
		return false
	}

	if !c.matchOrigin(origin) {
		logger.Warnf("[TRIPLE] CORS preflight forbidden origin: %s", origin)
		w.WriteHeader(http.StatusForbidden)
		return true
	}

	if !c.isMethodAllowed(r) {
		w.WriteHeader(http.StatusForbidden)
		return true
	}

	c.applyCORSOrigin(w, origin)
	c.setAllowMethods(w)
	c.setAllowHeaders(w, r)
	w.Header().Set(corsMaxAge, strconv.Itoa(c.maxAge))
	w.WriteHeader(http.StatusNoContent)
	return true
}

func (c *corsConfig) isMethodAllowed(r *http.Request) bool {
	requestedMethod := r.Header.Get(corsRequestMethod)
	return requestedMethod == "" || c.containsMethod(requestedMethod)
}

func (c *corsConfig) setAllowMethods(w http.ResponseWriter) {
	// allowMethods is always non-empty after normalizeMethods (at least contains OPTIONS)
	w.Header().Set(corsAllowMethods, strings.Join(c.allowMethods, ", "))
}

func (c *corsConfig) setAllowHeaders(w http.ResponseWriter, r *http.Request) {
	if len(c.allowHeaders) > 0 {
		w.Header().Set(corsAllowHeaders, strings.Join(c.allowHeaders, ", "))
	} else if requestedHeaders := r.Header.Get(corsRequestHeaders); requestedHeaders != "" {
		w.Header().Set(corsAllowHeaders, requestedHeaders)
	}
}

func (c *corsConfig) allowOrigin(origin string) bool {
	if c == nil {
		return false
	}
	if c.matchOrigin(origin) {
		return true
	}
	logger.Warnf("[TRIPLE] CORS forbidden origin: %s", origin)
	return false
}

func (c *corsConfig) addCORSHeaders(w http.ResponseWriter, r *http.Request) {
	if c == nil {
		return
	}
	origin := r.Header.Get(corsOrigin)
	if origin == "" || !c.matchOrigin(origin) {
		return
	}

	c.applyCORSOrigin(w, origin)
	c.setExposeHeaders(w)
}

func (c *corsConfig) setExposeHeaders(w http.ResponseWriter) {
	if len(c.exposeHeaders) > 0 {
		w.Header().Set(corsExposeHeaders, strings.Join(c.exposeHeaders, ", "))
	}
}

func (c *corsConfig) applyCORSOrigin(w http.ResponseWriter, origin string) {
	switch {
	case c.allowCredentials:
		w.Header().Set(corsAllowOrigin, origin)
		w.Header().Add(corsVary, corsOrigin)
		w.Header().Set(corsAllowCredentials, "true")
	case c.hasWildcard:
		w.Header().Set(corsAllowOrigin, wildcardOrigin)
	default:
		w.Header().Set(corsAllowOrigin, origin)
		w.Header().Add(corsVary, corsOrigin)
	}
}

func (c *corsConfig) containsMethod(target string) bool {
	if c == nil {
		return false
	}

	targetUpper := strings.ToUpper(target)
	for _, method := range c.allowMethods {
		if method == targetUpper {
			return true
		}
	}
	return false
}

func (c *corsConfig) checkHasWildcard() bool {
	if c == nil {
		return false
	}
	for _, origin := range c.allowOrigins {
		if origin == wildcardOrigin {
			return true
		}
	}
	return false
}
