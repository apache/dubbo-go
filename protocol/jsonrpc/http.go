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
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

//////////////////////////////////////////////
// Request
//////////////////////////////////////////////

type Request struct {
	ID          int64
	group       string
	protocol    string
	version     string
	service     string
	method      string
	args        interface{}
	contentType string
}

//////////////////////////////////////////////
// HTTP Client
//////////////////////////////////////////////

type HTTPOptions struct {
	HandshakeTimeout time.Duration
	HTTPTimeout      time.Duration
}

var defaultHTTPOptions = HTTPOptions{
	HandshakeTimeout: 3e9,
	HTTPTimeout:      3e9,
}

type HTTPClient struct {
	ID      int64
	options HTTPOptions
}

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

func (c *HTTPClient) NewRequest(service common.URL, method string, args interface{}) *Request {

	return &Request{
		ID:       atomic.AddInt64(&c.ID, 1),
		group:    service.GetParam(constant.GROUP_KEY, ""),
		protocol: service.Protocol,
		version:  service.GetParam(constant.VERSION_KEY, ""),
		service:  service.Path,
		method:   method,
		args:     args,
	}
}

func (c *HTTPClient) Call(ctx context.Context, service common.URL, req *Request, rsp interface{}) error {
	// header
	httpHeader := http.Header{}
	httpHeader.Set("Content-Type", "application/json")
	httpHeader.Set("Accept", "application/json")

	reqTimeout := c.options.HTTPTimeout
	if reqTimeout <= 0 {
		reqTimeout = 1e8
	}
	httpHeader.Set("Timeout", reqTimeout.String())
	if md, ok := ctx.Value(constant.DUBBOGO_CTX_KEY).(map[string]string); ok {
		for k := range md {
			httpHeader.Set(k, md[k])
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

// !!The high level of complexity and the likelihood that the fasthttp client has not been extensively used
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
	if err := httpReq.Write(reqBuf); err != nil {
		return nil, perrors.WithStack(err)
	}

	tcpConn, err := net.DialTimeout("tcp", addr, c.options.HandshakeTimeout)
	if err != nil {
		return nil, perrors.WithStack(err)
	}
	defer tcpConn.Close()
	setNetConnTimeout := func(conn net.Conn, timeout time.Duration) {
		t := time.Time{}
		if timeout > time.Duration(0) {
			t = time.Now().Add(timeout)
		}

		conn.SetDeadline(t)
	}
	setNetConnTimeout(tcpConn, c.options.HTTPTimeout)

	if _, err := reqBuf.WriteTo(tcpConn); err != nil {
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
