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
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

// originPattern represents a pre-compiled origin matching pattern.
type originPattern struct {
	scheme      string // empty means any scheme
	host        string // hostname without port
	port        string // port (canonical for URL patterns, raw for hostname-only patterns)
	isWildcard  bool   // true for "*"
	isSubdomain bool   // true for "*.example.com"
	raw         string // original string for exact match
}

// corsConfig is an internal CORS configuration struct.
type corsConfig struct {
	allowOrigins     []string
	allowMethods     []string
	allowHeaders     []string
	exposeHeaders    []string
	allowCredentials bool
	maxAge           int
	hasWildcard      bool
	compiledPatterns []originPattern // pre-compiled patterns for fast matching
	// Pre-computed header values to avoid repeated strings.Join calls
	prebuiltAllowMethods  string
	prebuiltAllowHeaders  string
	prebuiltExposeHeaders string
}

const (
	defaultPreflightMaxAge = 86400 // 24 hours in seconds
	wildcardOrigin         = "*"
	defaultHTTPPort        = "80"
	defaultHTTPSPort       = "443"
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

	hasWildcard := c.checkHasWildcard()
	built := &corsConfig{
		allowOrigins:     append([]string(nil), c.allowOrigins...),
		allowMethods:     append([]string(nil), c.allowMethods...),
		allowHeaders:     append([]string(nil), c.allowHeaders...),
		exposeHeaders:    append([]string(nil), c.exposeHeaders...),
		allowCredentials: c.allowCredentials,
		maxAge:           c.maxAge,
		hasWildcard:      hasWildcard,
	}

	// Warn if wildcard "*" is used with other origins and credentials are disabled
	if hasWildcard && !c.allowCredentials && len(c.allowOrigins) > 1 {
		logger.Warnf("[TRIPLE] CORS: wildcard \"*\" in allowOrigins will override all other origins when allowCredentials=false. Other origins will be ignored.")
	}

	built.allowMethods = built.normalizeMethods(handlers)
	built.compiledPatterns = built.compilePatterns()

	built.prebuiltAllowMethods = strings.Join(built.allowMethods, ", ")
	built.prebuiltAllowHeaders = strings.Join(built.allowHeaders, ", ")
	built.prebuiltExposeHeaders = strings.Join(built.exposeHeaders, ", ")

	// maxAge < 0: invalid, use default value
	// maxAge == 0: disable caching (don't send header)
	// maxAge > 0: use configured value
	if built.maxAge < 0 {
		built.maxAge = defaultPreflightMaxAge
	}

	return built
}

// buildCorsConfig calls build on the corsConfig.
func buildCorsConfig(cfg *corsConfig, handlers []protocolHandler) *corsConfig {
	return cfg.build(handlers)
}

// checkHasWildcard checks if the global wildcard "*" is present in allowOrigins.
// Note: only checks for "*", not subdomain wildcards like "*.example.com".
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

// normalizeMethods normalizes and deduplicates CORS methods.
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

// collectMethods collects methods from config, handlers, or defaults (in that priority).
func (c *corsConfig) collectMethods(handlers []protocolHandler) map[string]struct{} {
	methodSet := make(map[string]struct{})

	// Priority: explicit configuration
	if len(c.allowMethods) > 0 {
		for _, m := range c.allowMethods {
			if m != "" {
				methodSet[strings.ToUpper(m)] = struct{}{}
			}
		}
		return methodSet
	}

	// Fallback: extract from handlers
	if len(handlers) > 0 {
		for _, hdl := range handlers {
			for m := range hdl.Methods() {
				methodSet[strings.ToUpper(m)] = struct{}{}
			}
		}
		if len(methodSet) > 0 {
			return methodSet
		}
	}

	// Default: use standard CORS methods
	for _, m := range defaultCorsMethods {
		methodSet[m] = struct{}{}
	}

	return methodSet
}

// compilePatterns pre-compiles origin patterns.
func (c *corsConfig) compilePatterns() []originPattern {
	patterns := make([]originPattern, 0, len(c.allowOrigins))
	for _, origin := range c.allowOrigins {
		if origin == "" {
			continue
		}
		pattern := newOriginPattern(origin)
		patterns = append(patterns, *pattern)
	}
	return patterns
}

// newOriginPattern parses an origin string into a compiled pattern.
func newOriginPattern(origin string) *originPattern {
	p := &originPattern{}
	if origin == wildcardOrigin {
		p.isWildcard = true
		p.raw = origin
		return p
	}

	// Try parsing as URL first
	if u, err := url.Parse(origin); err == nil && u.Host != "" {
		hostname := u.Hostname()
		port := p.canonicalPort(u.Port(), u.Scheme)
		isSubdomain := strings.HasPrefix(hostname, "*.")

		p.scheme = u.Scheme
		if isSubdomain {
			p.host = hostname[2:] // Remove "*." prefix
			p.isSubdomain = true
		} else {
			p.host = hostname
		}
		p.port = port
		p.raw = origin
		return p
	}

	// Try as hostname-only subdomain format
	if strings.HasPrefix(origin, "*.") {
		base := origin[2:] // Remove "*." prefix
		hostname, port := p.splitHostPort(base)
		p.host = hostname
		p.port = port
		p.isSubdomain = true
		p.raw = origin
		return p
	}

	// Plain hostname or hostname:port (no scheme)
	hostname, port := p.splitHostPort(origin)
	p.host = hostname
	p.port = port
	p.raw = origin
	return p
}

// splitHostPort splits host:port into hostname and port.
func (p *originPattern) splitHostPort(hostPort string) (hostname, port string) {
	if hostPort == "" {
		return "", ""
	}
	hostname, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort, ""
	}
	return hostname, port
}

// canonicalPort returns the canonical port for a given port and scheme.
func (p *originPattern) canonicalPort(port, scheme string) string {
	if port != "" {
		return port
	}
	switch scheme {
	case "http":
		return defaultHTTPPort
	case "https":
		return defaultHTTPSPort
	default:
		return ""
	}
}

// match checks if the pattern matches the request origin components.
func (p *originPattern) match(reqScheme, reqHostname, reqPort string) bool {
	if p.isWildcard {
		return true
	}
	return p.matchScheme(reqScheme) &&
		p.matchHost(reqHostname) &&
		p.matchPort(reqPort, reqScheme)
}

// matchScheme checks if the request scheme matches the pattern.
func (p *originPattern) matchScheme(reqScheme string) bool {
	if p.scheme == "" {
		return true
	}
	return p.scheme == reqScheme
}

// matchHost checks if the request hostname matches.
func (p *originPattern) matchHost(reqHostname string) bool {
	if p.isSubdomain {
		return p.isSubdomainOf(reqHostname, p.host)
	}
	return p.host == reqHostname
}

// matchPort checks if the request port matches the pattern.
func (p *originPattern) matchPort(reqPort, reqScheme string) bool {
	if p.port == "" {
		if p.scheme == "" {
			return true
		}
		switch reqScheme {
		case "http":
			return reqPort == defaultHTTPPort
		case "https":
			return reqPort == defaultHTTPSPort
		default:
			return false
		}
	}

	if p.scheme == "" {
		return p.canonicalPort(p.port, reqScheme) == reqPort
	}
	return p.port == reqPort
}

// isSubdomainOf checks if originHost is a subdomain of base.
func (p *originPattern) isSubdomainOf(originHost, base string) bool {
	if base == "" || originHost == "" {
		return false
	}
	return originHost != base && strings.HasSuffix(originHost, "."+base)
}

// matchOrigin checks if the request origin matches any allowed pattern.
func (c *corsConfig) matchOrigin(origin string) bool {
	if origin == "" || len(c.allowOrigins) == 0 {
		return false
	}

	patterns := c.compiledPatterns
	if len(patterns) == 0 {
		patterns = c.compilePatterns()
	}
	if len(patterns) == 0 {
		return false
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}

	reqScheme := originURL.Scheme
	reqHostname := originURL.Hostname()
	reqPort := (*originPattern)(nil).canonicalPort(originURL.Port(), reqScheme)

	for _, pattern := range patterns {
		if pattern.match(reqScheme, reqHostname, reqPort) {
			return true
		}
	}

	return false
}

func (c *corsConfig) allowOrigin(origin string) bool {
	if c == nil {
		return false
	}
	return c.matchOrigin(origin)
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
		logger.Debugf("[TRIPLE] CORS forbidden origin: %s", origin)
		w.Header().Add(corsVary, corsOrigin)
		w.WriteHeader(http.StatusForbidden)
		return true
	}

	if !c.isMethodAllowed(r) {
		requestedMethod := r.Header.Get(corsRequestMethod)
		logger.Debugf("[TRIPLE] CORS forbidden method: %s (origin: %s)", requestedMethod, origin)
		w.Header().Add(corsVary, corsOrigin)
		w.WriteHeader(http.StatusForbidden)
		return true
	}

	c.applyCORSOrigin(w, origin)
	c.setAllowMethods(w)
	c.setAllowHeaders(w, r)
	if c.maxAge > 0 {
		w.Header().Set(corsMaxAge, strconv.Itoa(c.maxAge))
	}
	w.WriteHeader(http.StatusNoContent)
	return true
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

func (c *corsConfig) isMethodAllowed(r *http.Request) bool {
	requestedMethod := r.Header.Get(corsRequestMethod)
	return requestedMethod == "" || c.containsMethod(requestedMethod)
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

func (c *corsConfig) setAllowMethods(w http.ResponseWriter) {
	w.Header().Set(corsAllowMethods, c.prebuiltAllowMethods)
}

func (c *corsConfig) setAllowHeaders(w http.ResponseWriter, r *http.Request) {
	if len(c.allowHeaders) > 0 {
		w.Header().Set(corsAllowHeaders, c.prebuiltAllowHeaders)
	} else if requestedHeaders := r.Header.Get(corsRequestHeaders); requestedHeaders != "" {
		w.Header().Set(corsAllowHeaders, requestedHeaders)
	}
}

func (c *corsConfig) setExposeHeaders(w http.ResponseWriter) {
	if len(c.exposeHeaders) > 0 {
		w.Header().Set(corsExposeHeaders, c.prebuiltExposeHeaders)
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
