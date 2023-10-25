//go:build go1.19
// +build go1.19

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
	"errors"
	"fmt"
	"net/http"
)

// MaxBytesHandler is the same as http.MaxBytesHandler for compatibility since
// there is another MaxBytesHandler in maxbytes_low_version.go.
func MaxBytesHandler(h http.Handler, n int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2 := *r
		r2.Body = http.MaxBytesReader(w, r.Body, n)
		h.ServeHTTP(w, &r2)
	})
}

func asMaxBytesError(err error, tmpl string, args ...interface{}) *Error {
	var maxBytesErr *http.MaxBytesError
	if ok := errors.As(err, &maxBytesErr); !ok {
		return nil
	}
	prefix := fmt.Sprintf(tmpl, args...)
	return errorf(CodeResourceExhausted, "%s: exceeded %d byte http.MaxBytesReader limit", prefix, maxBytesErr.Limit)
}
