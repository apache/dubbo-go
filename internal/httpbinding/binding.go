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

package httpbinding

import (
	"net/http"
	"strings"
)

// HTTPBinding is a validated google.api.http route for one protobuf method.
type HTTPBinding struct {
	RPC          string
	Method       string
	PathTemplate string
	Body         string
	ResponseBody string
	PathFields   []string
	Index        int
}

func (b HTTPBinding) Key() string {
	return CanonicalRouteKey(b.Method, b.PathTemplate)
}

// CanonicalRouteKey normalizes HTTP path templates to their matching shape so
// equivalent templates cannot be registered as distinct routes. A variable
// with a literal or wildcard pattern has the same shape as that pattern written
// directly (for example, /v1/{id=books} and /v1/books).
func CanonicalRouteKey(method, path string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = pathVariablePattern.ReplaceAllStringFunc(path, func(variable string) string {
		matches := pathVariablePattern.FindStringSubmatch(variable)
		pattern := "*"
		if len(matches) > 2 && matches[2] != "" {
			pattern = matches[2]
		}
		return pattern
	})
	return method + " " + path
}

// IsSupportedMethod reports whether the route can be represented by the
// runtime and OpenAPI implementations in this package.
func IsSupportedMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete, http.MethodPatch:
		return true
	default:
		return false
	}
}
