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

// CorsConfig is a CORS configuration struct for handler options.
type CorsConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// corsPolicy is an internal CORS policy that contains compiled patterns and runtime state.
type corsPolicy struct {
	CorsConfig       // embed public config to avoid duplication
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

// build processes the corsPolicy with handlers and returns a fully configured corsPolicy.
func (c *corsPolicy) build(handlers []protocolHandler) *corsPolicy {
	if c == nil {
		return nil
	}
	if len(c.AllowOrigins) == 0 {
		return nil
	}

	hasWildcard := c.checkHasWildcard()
	built := &corsPolicy{
		CorsConfig: CorsConfig{
			AllowOrigins:     append([]string(nil), c.AllowOrigins...),
			AllowMethods:     append([]string(nil), c.AllowMethods...),
			AllowHeaders:     append([]string(nil), c.AllowHeaders...),
			ExposeHeaders:    append([]string(nil), c.ExposeHeaders...),
			AllowCredentials: c.AllowCredentials,
			MaxAge:           c.MaxAge,
		},
		hasWildcard: hasWildcard,
	}

	// Warn if wildcard "*" is used with other origins and credentials are disabled
	if hasWildcard && !c.AllowCredentials && len(c.AllowOrigins) > 1 {
		logger.Warnf("[TRIPLE] CORS: wildcard \"*\" in allowOrigins will override all other origins when allowCredentials=false. Other origins will be ignored.")
	}

	built.AllowMethods = built.normalizeMethods(handlers)
	built.compiledPatterns = built.compilePatterns()

	built.prebuiltAllowMethods = strings.Join(built.AllowMethods, ", ")
	built.prebuiltAllowHeaders = strings.Join(built.AllowHeaders, ", ")
	built.prebuiltExposeHeaders = strings.Join(built.ExposeHeaders, ", ")

	// maxAge < 0: invalid, use default value
	// maxAge == 0: disable caching (don't send header)
	// maxAge > 0: use configured value
	if built.MaxAge < 0 {
		built.MaxAge = defaultPreflightMaxAge
	}

	return built
}

// buildCorsPolicy calls build on the corsPolicy.
func buildCorsPolicy(cfg *corsPolicy, handlers []protocolHandler) *corsPolicy {
	return cfg.build(handlers)
}

// checkHasWildcard checks if the global wildcard "*" is present in allowOrigins.
// Note: only checks for "*", not subdomain wildcards like "*.example.com".
func (c *corsPolicy) checkHasWildcard() bool {
	if c == nil {
		return false
	}
	for _, origin := range c.AllowOrigins {
		if origin == wildcardOrigin {
			return true
		}
	}
	return false
}

// normalizeMethods normalizes and deduplicates CORS methods.
func (c *corsPolicy) normalizeMethods(handlers []protocolHandler) []string {
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
func (c *corsPolicy) collectMethods(handlers []protocolHandler) map[string]struct{} {
	methodSet := make(map[string]struct{})

	// Priority: explicit configuration
	if len(c.AllowMethods) > 0 {
		for _, m := range c.AllowMethods {
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
func (c *corsPolicy) compilePatterns() []originPattern {
	patterns := make([]originPattern, 0, len(c.AllowOrigins))
	for _, origin := range c.AllowOrigins {
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
	u, err := url.Parse(origin)
	if err == nil && u.Host != "" {
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
func (c *corsPolicy) matchOrigin(origin string) bool {
	if origin == "" || len(c.AllowOrigins) == 0 {
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
	var tmp originPattern
	reqPort := tmp.canonicalPort(originURL.Port(), reqScheme)

	for _, pattern := range patterns {
		if pattern.match(reqScheme, reqHostname, reqPort) {
			return true
		}
	}

	return false
}

func (c *corsPolicy) allowOrigin(origin string) bool {
	if c == nil {
		return false
	}
	return c.matchOrigin(origin)
}

func (c *corsPolicy) handlePreflight(w http.ResponseWriter, r *http.Request) bool {
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
	if c.MaxAge > 0 {
		w.Header().Set(corsMaxAge, strconv.Itoa(c.MaxAge))
	}
	w.WriteHeader(http.StatusNoContent)
	return true
}

func (c *corsPolicy) addCORSHeaders(w http.ResponseWriter, r *http.Request) {
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

func (c *corsPolicy) isMethodAllowed(r *http.Request) bool {
	requestedMethod := r.Header.Get(corsRequestMethod)
	return requestedMethod == "" || c.containsMethod(requestedMethod)
}

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

func (c *corsPolicy) setAllowMethods(w http.ResponseWriter) {
	w.Header().Set(corsAllowMethods, c.prebuiltAllowMethods)
}

func (c *corsPolicy) setAllowHeaders(w http.ResponseWriter, r *http.Request) {
	if len(c.AllowHeaders) > 0 {
		w.Header().Set(corsAllowHeaders, c.prebuiltAllowHeaders)
	} else if requestedHeaders := r.Header.Get(corsRequestHeaders); requestedHeaders != "" {
		w.Header().Set(corsAllowHeaders, requestedHeaders)
	}
}

func (c *corsPolicy) setExposeHeaders(w http.ResponseWriter) {
	if len(c.ExposeHeaders) > 0 {
		w.Header().Set(corsExposeHeaders, c.prebuiltExposeHeaders)
	}
}

func (c *corsPolicy) applyCORSOrigin(w http.ResponseWriter, origin string) {
	switch {
	case c.AllowCredentials:
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
