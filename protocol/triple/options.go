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

package triple

import (
	"errors"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

// TODO: The triple options for the server and client are mixed together now.
// We need to find a way to separate them later.
type Options struct {
	Triple *global.TripleConfig
}

func defaultOptions() *Options {
	return &Options{Triple: global.DefaultTripleConfig()}
}

func NewOptions(opts ...Option) *Options {
	defSrvOpts := defaultOptions()
	for _, opt := range opts {
		opt(defSrvOpts)
	}
	return defSrvOpts
}

type Option func(*Options)

// WithKeepAlive sets the keep-alive interval and timeout for the Triple protocol.
// interval: The duration between keep-alive pings.
// timeout: The duration to wait for a keep-alive response before considering the connection dead.
// If not set, default interval is 10s, default timeout is 20s.
func WithKeepAlive(interval, timeout time.Duration) Option {
	return func(opts *Options) {
		opts.Triple.KeepAliveInterval = interval.String()
		opts.Triple.KeepAliveTimeout = timeout.String()
	}
}

// WithKeepAliveInterval sets the keep-alive interval for the Triple protocol.
// interval: The duration between keep-alive pings.
// If not set, default interval is 10s.
func WithKeepAliveInterval(interval time.Duration) Option {
	return func(opts *Options) {
		opts.Triple.KeepAliveInterval = interval.String()
	}
}

// WithKeepAliveTimeout sets the keep-alive timeout for the Triple protocol.
// timeout: The duration to wait for a keep-alive response before considering the connection dead.
// If not set, default timeout is 20s.
func WithKeepAliveTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.Triple.KeepAliveTimeout = timeout.String()
	}
}

// WithMaxServerSendMsgSize sets the maximum size of messages that the server can send.
// size: The maximum message size in bytes, specified as a string (e.g., "4MB").
// If not set, default value is 2147MB (math.MaxInt32).
func WithMaxServerSendMsgSize(size string) Option {
	return func(opts *Options) {
		opts.Triple.MaxServerSendMsgSize = size
	}
}

// WithMaxServerRecvMsgSize sets the maximum size of messages that the server can receive.
// size: The maximum message size in bytes, specified as a string (e.g., "4MB").
// If not set, default value is 4MB (4194304 bytes).
func WithMaxServerRecvMsgSize(size string) Option {
	return func(opts *Options) {
		opts.Triple.MaxServerRecvMsgSize = size
	}
}

// WithCORS applies CORS configuration to triple options.
// Invalid configs are logged as errors and ignored (no-op).
func WithCORS(opts ...CORSOption) Option {
	cors := global.DefaultCorsConfig()
	for _, opt := range opts {
		opt(cors)
	}
	if err := validateCorsConfig(cors); err != nil {
		logger.Errorf("[TRIPLE] invalid CORS config: %v", err)
		// Return a no-op function to ignore invalid CORS configuration
		return func(*Options) {}
	}
	return func(opts *Options) {
		opts.Triple.Cors = cors
	}
}

// Http3Enable enables HTTP/3 support for the Triple protocol.
// This option configures the server to start both HTTP/2 and HTTP/3 servers
// simultaneously, providing modern HTTP/3 capabilities alongside traditional HTTP/2.
//
// When enabled, the server will:
//   - Start an HTTP/3 server using QUIC protocol
//   - Continue running the existing HTTP/2 server
//   - Enable protocol negotiation between HTTP/2 and HTTP/3
//   - Provide improved performance and security benefits of HTTP/3
//
// Usage Examples:
//
//	// Basic HTTP/3 enablement
//	server := triple.NewServer(
//	    triple.Http3Enable(),
//	)
//
// Requirements:
//   - TLS configuration is required for HTTP/3
//   - Server must have valid TLS certificates
//   - Clients must support HTTP/3 for full benefits
//   - Fallback to HTTP/2 is automatic for unsupported clients
//
// Default Behavior:
//   - HTTP/3 is disabled by default for backward compatibility
//   - When enabled, negotiation defaults to true
//   - Both HTTP/2 and HTTP/3 servers run on the same port
//
// # Experimental
//
// NOTICE: This API is EXPERIMENTAL and may be changed or removed in
// a later release.
func Http3Enable() Option {
	return func(opts *Options) {
		opts.Triple.Http3.Enable = true
	}
}

// Http3Negotiation configures HTTP/3 negotiation behavior for the Triple protocol.
// This option controls whether HTTP/2 Alternative Services (Alt-Svc) negotiation
// is enabled when both HTTP/2 and HTTP/3 servers are running simultaneously.
//
// Usage Examples:
//
//	// Enable HTTP/3 negotiation (default behavior)
//	server := triple.NewServer(
//	    triple.Http3Enable(),
//	    triple.Http3Negotiation(true),
//	)
//
//	// Disable HTTP/3 negotiation for explicit protocol control
//	server := triple.NewServer(
//	    triple.Http3Enable(),
//	    triple.Http3Negotiation(false),
//	)
//
// Default Behavior:
//   - When HTTP/3 is enabled, negotiation defaults to true
//   - This ensures backward compatibility and optimal client experience
//
// # Experimental
//
// NOTICE: This API is EXPERIMENTAL and may be changed or removed in
// a later release.
func Http3Negotiation(negotiation bool) Option {
	return func(opts *Options) {
		opts.Triple.Http3.Negotiation = negotiation
	}
}

// CORSOption configures a single aspect of CORS.
type CORSOption func(*global.CorsConfig)

var (
	errWildcardOriginWithCookies = errors.New("allowCredentials cannot be true when allow-origins contains \"*\"")
)

// CORSAllowOrigins sets allowed origins for CORS requests.
func CORSAllowOrigins(origins ...string) CORSOption {
	return func(c *global.CorsConfig) {
		c.AllowOrigins = append([]string(nil), origins...)
	}
}

// CORSAllowMethods sets allowed HTTP methods for CORS requests.
func CORSAllowMethods(methods ...string) CORSOption {
	return func(c *global.CorsConfig) {
		c.AllowMethods = append([]string(nil), methods...)
	}
}

// CORSAllowHeaders sets allowed request headers for CORS requests.
func CORSAllowHeaders(headers ...string) CORSOption {
	return func(c *global.CorsConfig) {
		c.AllowHeaders = append([]string(nil), headers...)
	}
}

// CORSExposeHeaders sets headers exposed to the browser.
func CORSExposeHeaders(headers ...string) CORSOption {
	return func(c *global.CorsConfig) {
		c.ExposeHeaders = append([]string(nil), headers...)
	}
}

// CORSAllowCredentials toggles whether credentials are allowed.
func CORSAllowCredentials(allow bool) CORSOption {
	return func(c *global.CorsConfig) {
		c.AllowCredentials = allow
	}
}

// CORSMaxAge sets the max age for preflight cache.
func CORSMaxAge(maxAge int) CORSOption {
	return func(c *global.CorsConfig) {
		c.MaxAge = maxAge
	}
}

// validateCorsConfig validates CORS configuration for unsafe combinations.
// CORS spec requires that when allowCredentials is true, Access-Control-Allow-Origin
// cannot be "*" (must be a specific origin).
func validateCorsConfig(cors *global.CorsConfig) error {
	if cors != nil && cors.AllowCredentials {
		for _, origin := range cors.AllowOrigins {
			if origin == "*" {
				return errWildcardOriginWithCookies
			}
		}
	}
	return nil
}
