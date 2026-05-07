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
	"io"
	"net/http"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"golang.org/x/net/http2"
)

// retryTransport is an http.RoundTripper decorator that transparently retries
// requests on transient connection-level errors. It only retries requests that
// are safe to replay: those with no body (Body == nil) or those that provide a
// GetBody function so the body can be reconstructed.
//
// Note: Triple protocol uses io.Pipe for request bodies, which cannot be
// replayed. As a result, retries only apply to body-less requests such as
// health-check probes.
type retryTransport struct {
	inner      http.RoundTripper
	maxRetries int
}

// newRetryTransport wraps inner with retry logic. maxRetries must be >= 0;
// a value of 0 disables retries and makes retryTransport a pass-through.
func newRetryTransport(inner http.RoundTripper, maxRetries int) http.RoundTripper {
	if maxRetries <= 0 {
		return inner
	}
	return &retryTransport{inner: inner, maxRetries: maxRetries}
}

// RoundTrip executes the request and retries on eligible connection-level errors.
func (rt *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Only retry requests whose body is either absent or rewindable.
	if !canRetry(req) {
		return rt.inner.RoundTrip(req)
	}

	var (
		resp *http.Response
		err  error
	)

	for attempt := 0; attempt <= rt.maxRetries; attempt++ {
		// Rebuild the request body for subsequent attempts when GetBody is available.
		if attempt > 0 {
			if req.GetBody != nil {
				newBody, bodyErr := req.GetBody()
				if bodyErr != nil {
					// Cannot reconstruct body; return the previous transport error.
					return nil, err
				}
				req.Body = newBody
			}
			logger.Debugf("retryTransport: retrying request (attempt %d/%d) for %s after error: %v",
				attempt, rt.maxRetries, req.URL.String(), err)
		}

		resp, err = rt.inner.RoundTrip(req)
		if err == nil {
			return resp, nil
		}

		if !isRetriableError(err) {
			return nil, err
		}
	}

	return nil, err
}

// canRetry reports whether the request body can survive a retry.
// A request is safe to retry when:
//   - Body is nil (no body at all, e.g. health-check HEAD/GET), or
//   - GetBody is set (body can be reconstructed from a factory function,
//     as http.NewRequest does for bytes.Buffer / bytes.Reader / strings.Reader).
func canRetry(req *http.Request) bool {
	return req.Body == nil || req.Body == http.NoBody || req.GetBody != nil
}

// isRetriableError reports whether err is a transient connection-level error
// that is safe to retry.
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	// Standard I/O sentinel errors.
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return true
	}

	// http2-specific retriable conditions detected via error message substrings.
	// golang.org/x/net/http2 surfaces these as opaque error strings.
	msg := err.Error()
	if strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "http2: server sent GOAWAY") ||
		strings.Contains(msg, "http2: Transport: cannot retry err") {
		return true
	}

	// http2.ErrNoCachedConn: the connection pool has no usable connection.
	if err == http2.ErrNoCachedConn {
		return true
	}

	return false
}
