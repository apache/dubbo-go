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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
)

// unaryClientCall is the transport surface shared by grpcClientConn and
// tripleUnaryClientConn. unary calls use unaryFastPathCall; streaming calls
// keep using duplexHTTPCall.
type unaryClientCall interface {
	io.Writer
	io.Reader
	Header() http.Header
	CloseWrite() error
	CloseRead() error
	BlockUntilResponseReady()
	SetValidateResponse(func(*http.Response) *Error)
	SetError(error)
	ResponseTrailer() http.Header
}

// Both duplexHTTPCall and unaryFastPathCall satisfy the interface.
var (
	_ unaryClientCall = (*duplexHTTPCall)(nil)
	_ unaryClientCall = (*unaryFastPathCall)(nil)
)

// unaryFastPathCall is a synchronous replacement for duplexHTTPCall on the
// unary hot path. Write accumulates the marshaled payload into a pooled
// buffer; CloseWrite issues the request synchronously (no io.Pipe, no
// per-request goroutine).
type unaryFastPathCall struct {
	ctx              context.Context
	httpClient       HTTPClient
	request          *http.Request
	bufferPool       *bufferPool
	validateResponse func(*http.Response) *Error

	// writeMu serializes Write against CloseWrite. StreamingClientConn's
	// contract requires Send and CloseRequest to be safe to call concurrently;
	// CloseWrite runs makeRequest synchronously under this lock.
	writeMu sync.Mutex

	// body accumulates the marshaled payload between Write and CloseWrite.
	body *bytes.Buffer

	// bodySent is set once makeRequest hands the body to the transport; a
	// later Write then returns io.EOF instead of racing with the transport.
	bodySent bool

	errMu sync.Mutex
	err   error

	response      *http.Response
	responseReady chan struct{}

	sendOnce sync.Once
}

func newUnaryFastPathCall(
	ctx context.Context,
	httpClient HTTPClient,
	url *url.URL,
	spec Spec,
	header http.Header,
	bufferPool *bufferPool,
) *unaryFastPathCall {
	// Clone the URL so a transport we don't control can't mutate the caller's,
	// then bind the concrete RPC path.
	url = cloneURL(url)
	url.Path = spec.Procedure
	url.RawPath = ""
	request := (&http.Request{
		Method:     http.MethodPost,
		URL:        url,
		Header:     header,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Host:       url.Host,
	}).WithContext(ctx)
	return &unaryFastPathCall{
		ctx:           ctx,
		httpClient:    httpClient,
		request:       request,
		bufferPool:    bufferPool,
		responseReady: make(chan struct{}),
	}
}

// Write accumulates the request payload into the pooled body buffer. Unlike
// duplexHTTPCall.Write it never touches the network.
func (c *unaryFastPathCall) Write(data []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.bodySent {
		// A racing CloseWrite already handed the body to the transport; a
		// write now could race with the transport reading it. Mirror
		// duplexHTTPCall.Write, which returns io.EOF once the pipe is closed.
		return 0, io.EOF
	}
	if err := c.getError(); err != nil {
		return 0, err
	}
	if err := c.ctx.Err(); err != nil {
		c.SetError(err)
		return 0, wrapIfContextError(err)
	}
	if len(data) == 0 {
		// Empty writes must not pull a buffer from the pool: a zero-length
		// body is sent as http.NoBody, so the buffer would never be
		// returned via unaryRequestBody.Close.
		return 0, nil
	}
	if c.body == nil {
		c.body = c.bufferPool.Get()
	}
	return c.body.Write(data)
}

// CloseWrite issues the request once the body is complete. Failures are
// stored via SetError and surface from Read. Runs under writeMu so it never
// sees a body being appended concurrently.
func (c *unaryFastPathCall) CloseWrite() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.sendOnce.Do(func() {
		c.makeRequest()
	})
	return nil
}

// unaryRequestBody is an io.ReadCloser over the pooled request buffer with no
// extra allocation. The buffer is returned to the pool from Close, which
// x/net/http2 invokes exactly once after it stops reading the body. Read and
// Close are guarded by a mutex so an aborted-write Close can never race with
// an in-flight Read.
type unaryRequestBody struct {
	mu   sync.Mutex
	buf  *bytes.Buffer
	pool *bufferPool
}

func (b *unaryRequestBody) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.buf == nil {
		return 0, io.EOF
	}
	return b.buf.Read(p)
}

func (b *unaryRequestBody) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.buf != nil {
		b.pool.Put(b.buf)
		b.buf = nil
	}
	return nil
}

// makeRequest issues the HTTP request with the same error wrapping chain as
// duplexHTTPCall. The request body buffer is returned to the pool from
// unaryRequestBody.Close, which x/net/http2 calls after it stops reading.
func (c *unaryFastPathCall) makeRequest() {
	defer close(c.responseReady)
	// Advertise Content-Length so the server can pre-size its HTTP/2 data
	// buffer; empty payloads use http.NoBody.
	var bodyLen int
	if c.body != nil {
		bodyLen = c.body.Len()
		if bodyLen > 0 {
			c.request.Body = &unaryRequestBody{buf: c.body, pool: c.bufferPool}
		} else {
			// Zero-length body: no transport read will ever happen, so
			// unaryRequestBody.Close can't return the buffer to the pool.
			// Recycle it here instead.
			c.bufferPool.Put(c.body)
			c.body = nil
			c.request.Body = http.NoBody
		}
	} else {
		c.request.Body = http.NoBody
	}
	c.bodySent = true
	c.request.ContentLength = int64(bodyLen)
	response, err := c.httpClient.Do(c.request) //nolint:bodyclose
	if err != nil {
		err = wrapIfContextError(err)
		err = wrapIfLikelyH2CNotConfiguredError(c.request, err)
		err = wrapIfLikelyWithGRPCNotUsedError(err)
		err = wrapIfRSTError(err)
		if _, ok := asError(err); !ok {
			err = NewError(CodeUnavailable, err)
		}
		c.SetError(err)
		return
	}
	c.response = response
	if err := c.validateResponse(response); err != nil {
		// Leave the response body open for CloseRead: callers may still read
		// the error body, and CloseResponse is the single close point.
		c.SetError(err)
	}
}

// Read reads the response body. BlockUntilResponseReady is already resolved
// by the time Read is called.
func (c *unaryFastPathCall) Read(data []byte) (int, error) {
	c.BlockUntilResponseReady()
	if err := c.getError(); err != nil {
		return 0, err
	}
	if err := c.ctx.Err(); err != nil {
		c.SetError(err)
		return 0, wrapIfContextError(err)
	}
	if c.response == nil {
		return 0, fmt.Errorf("nil response from %v", c.request.URL)
	}
	n, err := c.response.Body.Read(data)
	return n, wrapIfRSTError(err)
}

// CloseRead closes the response body. The request body buffer is recycled by
// unaryRequestBody.Close, not here.
func (c *unaryFastPathCall) CloseRead() error {
	c.BlockUntilResponseReady()
	if c.response == nil {
		return nil
	}
	return wrapIfRSTError(c.response.Body.Close())
}

// Header returns the HTTP request headers.
func (c *unaryFastPathCall) Header() http.Header {
	return c.request.Header
}

// SetValidateResponse sets the response validation function.
func (c *unaryFastPathCall) SetValidateResponse(validate func(*http.Response) *Error) {
	c.validateResponse = validate
}

// BlockUntilResponseReady blocks until the response is available. The fast
// path resolves it synchronously inside CloseWrite.
func (c *unaryFastPathCall) BlockUntilResponseReady() {
	<-c.responseReady
}

// SetError stores the first error encountered; safe for concurrent use.
func (c *unaryFastPathCall) SetError(err error) {
	c.errMu.Lock()
	defer c.errMu.Unlock()
	if c.err == nil {
		c.err = wrapIfContextError(err)
	}
}

// ResponseTrailer returns the response HTTP trailers.
func (c *unaryFastPathCall) ResponseTrailer() http.Header {
	c.BlockUntilResponseReady()
	if c.response != nil {
		return c.response.Trailer
	}
	return make(http.Header)
}

func (c *unaryFastPathCall) getError() error {
	c.errMu.Lock()
	defer c.errMu.Unlock()
	return c.err
}
