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

package jsonrpc

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/opentracing/opentracing-go"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// Request is HTTP protocol request
type Request struct {
	ID       int64
	group    string
	protocol string
	version  string
	service  string
	method   string
	args     interface{}
}

// HTTPOptions is a HTTP option include HandshakeTimeout and HTTPTimeout.
type HTTPOptions struct {
	HandshakeTimeout time.Duration
	HTTPTimeout      time.Duration
}

var defaultHTTPOptions = HTTPOptions{
	HandshakeTimeout: 3 * time.Second,
	HTTPTimeout:      3 * time.Second,
}

// HTTPClient is a HTTP client ,include ID and options.
type HTTPClient struct {
	ID      int64
	options HTTPOptions
}

// NewHTTPClient creates a new HTTP client with HTTPOptions.
func NewHTTPClient(opt *HTTPOptions) *HTTPClient {
	if opt == nil {
		opt = &defaultHTTPOptions
	}

	switch {
	case opt.HandshakeTimeout == 0:
		opt.HandshakeTimeout = defaultHTTPOptions.HandshakeTimeout
	case opt.HTTPTimeout == 0:
		opt.HTTPTimeout = defaultHTTPOptions.HTTPTimeout
	}

	t := time.Now()
	return &HTTPClient{
		ID:      int64(uint32(os.Getpid() * t.Second() * t.Nanosecond())),
		options: *opt,
	}
}

// NewRequest creates a new HTTP request with @service ,@method and @arguments.
func (c *HTTPClient) NewRequest(service *common.URL, method string, args interface{}) *Request {
	return &Request{
		ID:       atomic.AddInt64(&c.ID, 1),
		group:    service.GetParam(constant.GroupKey, ""),
		protocol: service.Protocol,
		version:  service.GetParam(constant.VersionKey, ""),
		service:  service.Path,
		method:   method,
		args:     args,
	}
}

// Call makes a HTTP call with @ctx , @service ,@req and @rsp
func (c *HTTPClient) Call(ctx context.Context, service *common.URL, req *Request, rsp interface{}) error {
	// header
	httpHeader := http.Header{}
	httpHeader.Set("Content-Type", "application/json")
	httpHeader.Set("Accept", "application/json")

	reqTimeout := c.options.HTTPTimeout
	if reqTimeout <= 0 {
		reqTimeout = 100 * time.Millisecond
	}
	httpHeader.Set("Timeout", reqTimeout.String())
	if md, ok := ctx.Value(constant.DubboGoCtxKey).(map[string]string); ok {
		for k := range md {
			httpHeader.Set(k, md[k])
		}
	}

	if span := opentracing.SpanFromContext(ctx); span != nil {
		err := opentracing.GlobalTracer().Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(httpHeader))
		if err != nil {
			logger.Error("Could not inject the Context into http header.")
		}
	}

	// body
	codec := newJsonClientCodec()
	codecData := CodecData{
		ID:     req.ID,
		Method: req.method,
		Args:   req.args,
	}
	reqBody, err := codec.Write(&codecData)
	if err != nil {
		return perrors.WithStack(err)
	}

	rspBody, err := c.Do(service.Location, service.Path, httpHeader, reqBody)
	if err != nil {
		return perrors.WithStack(err)
	}

	return perrors.WithStack(codec.Read(rspBody, rsp))
}

// Do is the high level of complexity and the likelihood that the fasthttp client has not been extensively used
// in production means that you would need to expect a very large benefit to justify the adoption of fasthttp today.
func (c *HTTPClient) Do(addr, path string, httpHeader http.Header, body []byte) ([]byte, error) {
	u := url.URL{Host: strings.TrimSuffix(addr, ":"), Path: path}
	httpReq, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, perrors.WithStack(err)
	}
	httpReq.Header = httpHeader
	httpReq.Close = true

	reqBuf := bytes.NewBuffer(make([]byte, 0))
	if err = httpReq.Write(reqBuf); err != nil {
		return nil, perrors.WithStack(err)
	}

	tcpConn, err := net.DialTimeout("tcp", addr, c.options.HandshakeTimeout)
	if err != nil {
		return nil, perrors.WithStack(err)
	}
	defer tcpConn.Close()
	setNetConnTimeout := func(conn net.Conn, timeout time.Duration) error {
		t := time.Time{}
		if timeout > time.Duration(0) {
			t = time.Now().Add(timeout)
		}

		return conn.SetDeadline(t)
	}
	if err = setNetConnTimeout(tcpConn, c.options.HTTPTimeout); err != nil {
		return nil, err
	}

	if _, err = reqBuf.WriteTo(tcpConn); err != nil {
		return nil, perrors.WithStack(err)
	}

	httpRsp, err := http.ReadResponse(bufio.NewReader(tcpConn), httpReq)
	if err != nil {
		return nil, perrors.WithStack(err)
	}
	defer httpRsp.Body.Close()

	b, err := ioutil.ReadAll(httpRsp.Body)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	if httpRsp.StatusCode != http.StatusOK {
		return nil, perrors.New(fmt.Sprintf("http status:%q, error string:%q", httpRsp.Status, string(b)))
	}

	return b, nil
}
