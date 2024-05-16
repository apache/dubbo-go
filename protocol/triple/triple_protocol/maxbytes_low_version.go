//go:build !go1.19
// +build !go1.19

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
	"io"
	"net/http"
)

// maxBytesReaderFunc is the same as http.MaxBytesReader for golang version smaller than 1.19
func maxBytesReaderFunc(w http.ResponseWriter, r io.ReadCloser, n int64) io.ReadCloser {
	if n < 0 { // Treat negative limits as equivalent to 0.
		n = 0
	}
	return &maxBytesReader{w: w, r: r, i: n, n: n}
}

// maxBytesReader is the same as http.maxBytesReader for golang version smaller than 1.19.
// We need to copy the whole type from http library because we need to return maxBytesError in maxBytesReader.Read
type maxBytesReader struct {
	w   http.ResponseWriter
	r   io.ReadCloser // underlying reader
	i   int64         // max bytes initially, for MaxBytesError
	n   int64         // max bytes remaining
	err error         // sticky error
}

func (l *maxBytesReader) Read(p []byte) (n int, err error) {
	if l.err != nil {
		return 0, l.err
	}
	if len(p) == 0 {
		return 0, nil
	}
	// If they asked for a 32KB byte read but only 5 bytes are
	// remaining, no need to read 32KB. 6 bytes will answer the
	// question of the whether we hit the limit or go past it.
	// 0 < len(p) < 2^63
	if int64(len(p))-1 > l.n {
		p = p[:l.n+1]
	}
	n, err = l.r.Read(p)

	if int64(n) <= l.n {
		l.n -= int64(n)
		l.err = err
		return n, err
	}

	n = int(l.n)
	l.n = 0

	// The server code and client code both use
	// maxBytesReader. This "requestTooLarge" check is
	// only used by the server code. To prevent binaries
	// which only using the HTTP Client code (such as
	// cmd/go) from also linking in the HTTP server, don't
	// use a static type assertion to the server
	// "*response" type. Check this interface instead:
	type requestTooLarger interface {
		requestTooLarge()
	}
	if res, ok := l.w.(requestTooLarger); ok {
		res.requestTooLarge()
	}
	l.err = &maxBytesError{l.i}
	return n, l.err
}

func (l *maxBytesReader) Close() error {
	return l.r.Close()
}

// maxBytesError is the same as http.MaxBytesError for golang version smaller than 1.19
type maxBytesError struct {
	Limit int64
}

func (e *maxBytesError) Error() string {
	// Due to Hyrum's law, this text cannot be changed.
	return "http: request body too large"
}

// MaxBytesHandler is the same as http.MaxBytesHandler for golang version smaller than 1.19
func MaxBytesHandler(h http.Handler, n int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2 := *r
		r2.Body = maxBytesReaderFunc(w, r.Body, n)
		h.ServeHTTP(w, &r2)
	})
}

func asMaxBytesError(err error, tmpl string, args ...interface{}) *Error {
	var maxBytesErr *maxBytesError
	if ok := errors.As(err, &maxBytesErr); !ok {
		return nil
	}
	prefix := fmt.Sprintf(tmpl, args...)
	return errorf(CodeResourceExhausted, "%s: exceeded %d byte http.MaxBytesReader limit", prefix, maxBytesErr.Limit)
}
