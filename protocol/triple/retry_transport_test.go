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
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransport is a controllable RoundTripper for testing.
type mockTransport struct {
	// responses[i] is returned on the (i+1)-th call. If exhausted, the last
	// entry is reused.
	responses []mockResponse
	calls     int
}

type mockResponse struct {
	resp *http.Response
	err  error
}

func (m *mockTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	idx := m.calls
	if idx >= len(m.responses) {
		idx = len(m.responses) - 1
	}
	m.calls++
	r := m.responses[idx]
	return r.resp, r.err
}

func successResp() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}
}

// -----------------------------------------------------------------------
// AC-2: EOF on first attempt, success on second (body-less request)
// -----------------------------------------------------------------------
func TestRetryTransport_RetryOnEOF_NoBody(t *testing.T) {
	mock := &mockTransport{
		responses: []mockResponse{
			{err: io.EOF},
			{resp: successResp()},
		},
	}
	rt := newRetryTransport(mock, 2)

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/health", nil)
	resp, err := rt.RoundTrip(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, mock.calls, "should have called inner transport twice")
}

// AC-2 variant: GetBody-enabled body is also retriable
func TestRetryTransport_RetryOnEOF_GetBody(t *testing.T) {
	bodyBytes := []byte("hello")
	mock := &mockTransport{
		responses: []mockResponse{
			{err: io.EOF},
			{resp: successResp()},
		},
	}
	rt := newRetryTransport(mock, 2)

	req, _ := http.NewRequest(http.MethodPost, "http://example.com/rpc",
		bytes.NewReader(bodyBytes))
	// http.NewRequest sets GetBody automatically for bytes.Reader
	require.NotNil(t, req.GetBody, "http.NewRequest should set GetBody for bytes.Reader")

	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, mock.calls)
}

// AC-3: stream body (no GetBody) should NOT be retried
func TestRetryTransport_NoRetry_StreamBody(t *testing.T) {
	mock := &mockTransport{
		responses: []mockResponse{
			{err: io.EOF},
			{resp: successResp()},
		},
	}
	rt := newRetryTransport(mock, 2)

	// io.NopCloser wraps a plain io.Reader — http.NewRequest won't set GetBody
	req, _ := http.NewRequest(http.MethodPost, "http://example.com/stream",
		io.NopCloser(bytes.NewReader([]byte("payload"))))
	req.GetBody = nil // Explicitly unset to simulate pipe body

	_, err := rt.RoundTrip(req)

	assert.Error(t, err, "should propagate error without retrying")
	assert.Equal(t, 1, mock.calls, "should have called inner transport only once")
}

// AC-4: max retries exhausted — last error is returned
func TestRetryTransport_MaxRetriesExhausted(t *testing.T) {
	persistentErr := io.EOF
	mock := &mockTransport{
		responses: []mockResponse{
			{err: persistentErr},
		},
	}
	rt := newRetryTransport(mock, 2)

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/health", nil)
	_, err := rt.RoundTrip(req)

	assert.Error(t, err)
	assert.Equal(t, 3, mock.calls, "should have tried 1 initial + 2 retries = 3 total")
}

// AC-4 variant: non-retriable error is never retried regardless of maxRetries
func TestRetryTransport_NonRetriableError_NotRetried(t *testing.T) {
	appErr := errors.New("application error: invalid argument")
	mock := &mockTransport{
		responses: []mockResponse{
			{err: appErr},
		},
	}
	rt := newRetryTransport(mock, 2)

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/health", nil)
	_, err := rt.RoundTrip(req)

	assert.ErrorIs(t, err, appErr)
	assert.Equal(t, 1, mock.calls, "non-retriable error should not be retried")
}

// maxRetries=0 disables retries and returns inner transport directly
func TestNewRetryTransport_ZeroRetries_Passthrough(t *testing.T) {
	mock := &mockTransport{
		responses: []mockResponse{{err: io.EOF}},
	}
	rt := newRetryTransport(mock, 0)

	// Should be the mock itself, not a retryTransport wrapper
	assert.Equal(t, mock, rt)
}

// -----------------------------------------------------------------------
// isRetriableError unit tests
// -----------------------------------------------------------------------
func TestIsRetriableError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"io.EOF", io.EOF, true},
		{"io.ErrUnexpectedEOF", io.ErrUnexpectedEOF, true},
		{"connection reset", errors.New("read tcp: connection reset by peer"), true},
		{"broken pipe", errors.New("write tcp: broken pipe"), true},
		{"GOAWAY", errors.New("http2: server sent GOAWAY and closed the connection"), true},
		{"cannot retry", errors.New("http2: Transport: cannot retry err [context canceled] after Request.Body was written"), true},
		{"arbitrary error", errors.New("some unrelated error"), false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, isRetriableError(c.err))
		})
	}
}

// -----------------------------------------------------------------------
// canRetry unit tests
// -----------------------------------------------------------------------
func TestCanRetry(t *testing.T) {
	// nil body
	reqNilBody, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	assert.True(t, canRetry(reqNilBody))

	// http.NoBody
	reqNoBody, _ := http.NewRequest(http.MethodGet, "http://example.com", http.NoBody)
	assert.True(t, canRetry(reqNoBody))

	// bytes.Reader body — http.NewRequest sets GetBody
	reqWithGetBody, _ := http.NewRequest(http.MethodPost, "http://example.com",
		bytes.NewReader([]byte("data")))
	assert.True(t, canRetry(reqWithGetBody))

	// stream body, no GetBody
	reqStream, _ := http.NewRequest(http.MethodPost, "http://example.com",
		io.NopCloser(bytes.NewReader([]byte("stream"))))
	reqStream.GetBody = nil
	assert.False(t, canRetry(reqStream))
}
