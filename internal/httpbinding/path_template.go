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
	"regexp"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

var pathVariablePattern = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_.]*)(?:=[^{}]+)?\}`)

func resolvePathFields(method, pathTemplate string) ([]string, error) {
	mux := runtime.NewServeMux()
	if err := mux.HandlePath(method, pathTemplate, func(http.ResponseWriter, *http.Request, map[string]string) {}); err != nil {
		return nil, err
	}

	matches := pathVariablePattern.FindAllStringSubmatch(pathTemplate, -1)
	fields := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if _, ok := seen[match[1]]; ok {
			continue
		}
		seen[match[1]] = struct{}{}
		fields = append(fields, match[1])
	}
	return fields, nil
}
