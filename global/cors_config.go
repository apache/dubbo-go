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

package global

// CorsConfig represents the CORS configuration for Triple protocol handlers.
type CorsConfig struct {
	// AllowOrigins specifies the allowed origins for CORS requests.
	// Supports exact match, wildcard "*", and subdomain wildcard like "https://*.example.com".
	// Empty slice means CORS is disabled.
	AllowOrigins []string `yaml:"allow-origins" json:"allow-origins,omitempty"`

	// AllowMethods specifies the allowed HTTP methods for CORS requests.
	// If empty, methods will be automatically determined from handler registrations.
	// OPTIONS method is always included automatically.
	AllowMethods []string `yaml:"allow-methods" json:"allow-methods,omitempty"`

	// AllowHeaders specifies the allowed request headers for CORS requests.
	// If empty, the requested headers from Access-Control-Request-Headers will be echoed back.
	AllowHeaders []string `yaml:"allow-headers" json:"allow-headers,omitempty"`

	// ExposeHeaders specifies the headers that browsers are allowed to access.
	ExposeHeaders []string `yaml:"expose-headers" json:"expose-headers,omitempty"`

	// AllowCredentials indicates whether credentials are allowed in CORS requests.
	// When true, AllowOrigins cannot contain "*".
	AllowCredentials bool `yaml:"allow-credentials" json:"allow-credentials,omitempty"`

	// MaxAge specifies the maximum age (in seconds) for preflight cache.
	// Must be non-negative. If zero, disables caching (no Access-Control-Max-Age header will be sent).
	// If positive, specifies the cache duration in seconds.
	MaxAge int `yaml:"max-age" json:"max-age,omitempty"`
}

// DefaultCorsConfig returns a default CorsConfig instance.
func DefaultCorsConfig() *CorsConfig {
	return &CorsConfig{
		AllowOrigins:     []string{},
		AllowMethods:     []string{},
		AllowHeaders:     []string{},
		ExposeHeaders:    []string{},
		AllowCredentials: false,
		MaxAge:           0,
	}
}

// Clone a new CorsConfig
func (c *CorsConfig) Clone() *CorsConfig {
	if c == nil {
		return nil
	}

	return &CorsConfig{
		AllowOrigins:     append([]string(nil), c.AllowOrigins...),
		AllowMethods:     append([]string(nil), c.AllowMethods...),
		AllowHeaders:     append([]string(nil), c.AllowHeaders...),
		ExposeHeaders:    append([]string(nil), c.ExposeHeaders...),
		AllowCredentials: c.AllowCredentials,
		MaxAge:           c.MaxAge,
	}
}
