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
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/quic-go/quic-go/http3"
)

// AltSvcInfo represents cached alternative service information
// This struct stores the parsed information from Alt-Svc HTTP headers
// according to RFC 7838 (HTTP Alternative Services)
type AltSvcInfo struct {
	// protocol specifies the alternative protocol to use
	// Examples: "h3" for HTTP/3, "h2" for HTTP/2, "h3-29" for HTTP/3 draft 29
	Protocol string

	// host specifies the alternative hostname (optional)
	// If empty, use the same host as the original request
	// Example: "example.com" or "" (same host)
	Host string

	// port specifies the alternative port number (optional)
	// If empty, use the same port as the original request
	// Example: "443" or "" (same port)
	Port string

	// expires specifies when this alternative service information becomes invalid
	// After this time, the cached information should be discarded and
	// a new Alt-Svc header should be requested from the server
	//
	// This field is parsed from the "ma" (max-age) parameter in Alt-Svc headers:
	// - Alt-Svc: h3=":443"; ma=86400  -> expires = now + 86400 seconds
	// - Alt-Svc: h3=":443"; ma=3600   -> expires = now + 3600 seconds
	//
	// If no "ma" parameter is provided, a default expiration time is used
	// to prevent stale cache entries from being used indefinitely
	Expires time.Time
}

// AltSvcCache manages cached alternative service information
type AltSvcCache struct {
	cache map[string]*AltSvcInfo
	mu    sync.RWMutex
}

// NewAltSvcCache creates a new Alt-Svc cache
func NewAltSvcCache() *AltSvcCache {
	return &AltSvcCache{
		cache: make(map[string]*AltSvcInfo),
	}
}

// Get retrieves cached alternative service information for a host
func (c *AltSvcCache) Get(host string) *AltSvcInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	altSvc, exists := c.cache[host]
	// Return nil if the cached information doesn't exist or has expired
	if !exists || altSvc.Expires.Before(time.Now()) {
		return nil
	}

	return altSvc
}

// Set stores alternative service information for a host
func (c *AltSvcCache) Set(host string, altSvc *AltSvcInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[host] = altSvc
}

// UpdateFromHeaders updates the cache based on Alt-Svc headers from an HTTP response
func (c *AltSvcCache) UpdateFromHeaders(host string, headers http.Header) {
	altSvcHeader := headers.Get("Alt-Svc")
	if altSvcHeader == "" {
		return
	}

	// Parse Alt-Svc header according to RFC 7838
	// Example: "h3=\":443\"; ma=86400, h3-29=\":443\"; ma=86400"
	altSvcs := ParseAltSvcHeader(altSvcHeader)
	if len(altSvcs) == 0 {
		return
	}

	// Prefer HTTP/3 over HTTP/2
	var preferredAltSvc *AltSvcInfo
	for _, altSvc := range altSvcs {
		if altSvc.Protocol == "h3" {
			preferredAltSvc = altSvc
			break
		} else if altSvc.Protocol == "h2" && preferredAltSvc == nil {
			preferredAltSvc = altSvc
		}
	}

	if preferredAltSvc != nil {
		c.Set(host, preferredAltSvc)
		logger.Debugf("Updated alt-svc cache for %s: %s", host, preferredAltSvc.Protocol)
	}
}

// ParseAltSvcHeader parses the Alt-Svc header according to RFC 7838
func ParseAltSvcHeader(altSvcHeader string) []*AltSvcInfo {
	var altSvcs []*AltSvcInfo

	// Split by comma to get individual alternative services
	parts := strings.Split(altSvcHeader, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse protocol and authority
		// Format: protocol="authority"; ma=max_age
		// Example: h3=":443"; ma=86400
		altSvc := parseAltSvcPart(part)
		if altSvc != nil {
			altSvcs = append(altSvcs, altSvc)
		}
	}

	return altSvcs
}

// parseAltSvcPart parses a single alternative service part
func parseAltSvcPart(part string) *AltSvcInfo {
	// Find the protocol (before the first '=')
	eqIndex := strings.Index(part, "=")
	if eqIndex == -1 {
		return nil
	}

	protocol := strings.TrimSpace(part[:eqIndex])
	if protocol != "h3" && protocol != "h2" {
		return nil
	}

	// Find the authority (between quotes)
	quoteStart := strings.Index(part, "\"")
	if quoteStart == -1 {
		return nil
	}

	quoteEnd := strings.LastIndex(part, "\"")
	if quoteEnd == -1 || quoteEnd <= quoteStart {
		return nil
	}

	authority := part[quoteStart+1 : quoteEnd]

	// Parse authority to get host and port
	host, port := parseAuthority(authority)

	// Parse max_age if present
	// The "ma" parameter specifies how long this alternative service is valid
	// Example: ma=86400 means valid for 86400 seconds (24 hours)
	maxAge := 24 * time.Hour // Default to 24 hours if no ma parameter is provided
	if maIndex := strings.Index(part, "ma="); maIndex != -1 {
		maPart := part[maIndex+3:]
		if semicolonIndex := strings.Index(maPart, ";"); semicolonIndex != -1 {
			maPart = maPart[:semicolonIndex]
		}
		if age, err := strconv.Atoi(strings.TrimSpace(maPart)); err == nil && age > 0 {
			maxAge = time.Duration(age) * time.Second
		}
	}

	return &AltSvcInfo{
		Protocol: protocol,
		Host:     host,
		Port:     port,
		Expires:  time.Now().Add(maxAge), // Calculate expiration time based on max-age
	}
}

// parseAuthority parses the authority part of an alternative service
func parseAuthority(authority string) (host, port string) {
	// Authority format can be:
	// - ":port" (host is the same as original)
	// - "host:port"
	// - "host" (port is the same as original)

	if strings.HasPrefix(authority, ":") {
		// Just port specified
		port = authority[1:]
		return "", port
	}

	if colonIndex := strings.Index(authority, ":"); colonIndex != -1 {
		// Both host and port specified
		host = authority[:colonIndex]
		port = authority[colonIndex+1:]
		return host, port
	}

	// Just host specified
	host = authority
	return host, ""
}

// AltSvcHandler wraps an http.Handler to automatically add Alt-Svc headers
// for non-HTTP/3 requests, enabling HTTP Alternative Services discovery
type AltSvcHandler struct {
	handler     http.Handler
	http3Server *http3.Server
	negotiation bool // Whether to enable HTTP/3 negotiation via Alt-Svc headers
}

// ServeHTTP implements http.Handler interface
func (h *AltSvcHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Add Alt-Svc header for non-HTTP/3 requests to advertise HTTP/3 availability
	// Only add Alt-Svc headers if negotiation is enabled
	if h.negotiation && r.ProtoMajor < 3 {
		if err := h.http3Server.SetQUICHeaders(w.Header()); err != nil {
			logger.Warnf("Failed to set QUIC headers for %s: %v", r.URL.String(), err)
		}
	}

	// Call the wrapped handler
	h.handler.ServeHTTP(w, r)
}

// NewAltSvcHandler creates a new AltSvcHandler that wraps the given handler
func NewAltSvcHandler(handler http.Handler, http3Server *http3.Server, negotiation bool) *AltSvcHandler {
	return &AltSvcHandler{
		handler:     handler,
		http3Server: http3Server,
		negotiation: negotiation,
	}
}
