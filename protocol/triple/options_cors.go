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
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

// CORSOption configures a single aspect of CORS.
type CORSOption func(*global.CorsConfig)

var (
	errNilCORSConfig             = errors.New("cors config is nil")
	errWildcardOriginWithCookies = errors.New("allowCredentials cannot be true when allow-origins contains \"*\"")
	errNegativeMaxAge            = errors.New("max-age must be non-negative")
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
func validateCorsConfig(cors *global.CorsConfig) error {
	if cors == nil {
		return errNilCORSConfig
	}
	if cors.AllowCredentials {
		for _, origin := range cors.AllowOrigins {
			if origin == "*" {
				return errWildcardOriginWithCookies
			}
		}
	}
	if cors.MaxAge < 0 {
		return errNegativeMaxAge
	}
	return nil
}
