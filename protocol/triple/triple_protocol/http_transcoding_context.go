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
	"context"
	"fmt"
	"net/http"
)

// HTTPTranscodingResponse collects response metadata emitted through the
// standard Triple SetHeader and SetTrailer APIs during an HTTP-transcoded call.
type HTTPTranscodingResponse struct {
	conn *httpTranscodingHandlerConn
}

// Header returns headers set by the service handler.
func (r *HTTPTranscodingResponse) Header() http.Header {
	if r == nil || r.conn == nil {
		return nil
	}
	return r.conn.ResponseHeader()
}

// Trailer returns trailers set by the service handler.
func (r *HTTPTranscodingResponse) Trailer() http.Header {
	if r == nil || r.conn == nil {
		return nil
	}
	return r.conn.ResponseTrailer()
}

// NewHTTPTranscodingContext bridges an HTTP request to the same incoming and
// outgoing metadata context used by the canonical Triple handler. The returned
// response handle exposes headers and trailers emitted by SetHeader/SetTrailer.
func NewHTTPTranscodingContext(ctx context.Context, requestHeader http.Header) (context.Context, *HTTPTranscodingResponse) {
	conn := &httpTranscodingHandlerConn{
		requestHeader:   requestHeader.Clone(),
		responseHeader:  make(http.Header),
		responseTrailer: make(http.Header),
	}
	ctx = newIncomingContext(ctx, requestHeader)
	ctx = context.WithValue(ctx, handlerOutgoingKey{}, conn)
	return ctx, &HTTPTranscodingResponse{conn: conn}
}

// httpTranscodingHandlerConn only supplies the metadata surface needed by the
// server-side Triple header helpers. Message transport methods are deliberately
// unsupported because HTTP transcoding decodes and encodes messages itself.
type httpTranscodingHandlerConn struct {
	requestHeader   http.Header
	responseHeader  http.Header
	responseTrailer http.Header
}

func (c *httpTranscodingHandlerConn) Spec() Spec { return Spec{} }

func (c *httpTranscodingHandlerConn) Peer() Peer { return Peer{} }

func (c *httpTranscodingHandlerConn) Receive(any) error {
	return fmt.Errorf("triple: HTTP transcoding connection does not receive messages")
}

func (c *httpTranscodingHandlerConn) RequestHeader() http.Header {
	return c.requestHeader
}

func (c *httpTranscodingHandlerConn) ExportableHeader() http.Header {
	return c.requestHeader
}

func (c *httpTranscodingHandlerConn) Send(any) error {
	return fmt.Errorf("triple: HTTP transcoding connection does not send messages")
}

func (c *httpTranscodingHandlerConn) ResponseHeader() http.Header {
	return c.responseHeader
}

func (c *httpTranscodingHandlerConn) ResponseTrailer() http.Header {
	return c.responseTrailer
}
