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
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"runtime/debug"
	"sync"
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
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// A value sent as a placeholder for the server's response value when the server
// receives an invalid request. It is never decoded by the client since the Response
// contains an error when it is used.
var invalidRequest = struct{}{}

const (
	// DefaultMaxSleepTime max sleep interval in accept
	DefaultMaxSleepTime = 1 * time.Second
	// DefaultHTTPRspBufferSize ...
	DefaultHTTPRspBufferSize = 1024
	// PathPrefix ...
	PathPrefix = byte('/')
)

// Server is JSON RPC server wrapper
type Server struct {
	done chan struct{}
	once sync.Once

	sync.RWMutex
	wg      sync.WaitGroup
	timeout time.Duration
}

// NewServer creates new JSON RPC server.
func NewServer() *Server {
	return &Server{
		done: make(chan struct{}),
	}
}

func (s *Server) handlePkg(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			logger.Warnf("connection{local:%v, remote:%v} panic error:%#v, debug stack:%s",
				conn.LocalAddr(), conn.RemoteAddr(), r, string(debug.Stack()))
		}

		conn.Close()
	}()

	setTimeout := func(conn net.Conn, timeout time.Duration) {
		t := time.Time{}
		if timeout > time.Duration(0) {
			t = time.Now().Add(timeout)
		}

		if err := conn.SetDeadline(t); err != nil {
			logger.Error("connection.SetDeadline(t:%v) = error:%v", t, err)
		}
	}

	sendErrorResp := func(header http.Header, body []byte) error {
		rsp := &http.Response{
			Header:        header,
			StatusCode:    500,
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: int64(len(body)),
			Body:          ioutil.NopCloser(bytes.NewReader(body)),
		}
		rsp.Header.Del("Content-Type")
		rsp.Header.Del("Content-Length")
		rsp.Header.Del("Timeout")

		rspBuf := bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize))
		rspBuf.Reset()
		err := rsp.Write(rspBuf)
		if err != nil {
			return perrors.WithStack(err)
		}
		_, err = rspBuf.WriteTo(conn)
		return perrors.WithStack(err)
	}

	for {
		bufReader := bufio.NewReader(conn)
		r, err := http.ReadRequest(bufReader)
		if err != nil {
			logger.Warnf("[ReadRequest] error: %v", err)
			return
		}

		reqBody, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			return
		}

		reqHeader := make(map[string]string)
		for k := range r.Header {
			reqHeader[k] = r.Header.Get(k)
		}
		reqHeader["Path"] = r.URL.Path[1:] // to get service name
		if r.URL.Path[0] != PathPrefix {
			reqHeader["Path"] = r.URL.Path
		}
		reqHeader["HttpMethod"] = r.Method

		httpTimeout := s.timeout
		contentType := reqHeader["Content-Type"]
		if contentType != "application/json" && contentType != "application/json-rpc" {
			setTimeout(conn, httpTimeout)
			r.Header.Set("Content-Type", "text/plain")
			if errRsp := sendErrorResp(r.Header, []byte(perrors.WithStack(err).Error())); errRsp != nil {
				logger.Warnf("sendErrorResp(header:%#v, error:%v) = error:%s",
					r.Header, perrors.WithStack(err), errRsp)
			}
			return
		}

		ctx := context.Background()

		spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(r.Header))
		if err == nil {
			ctx = context.WithValue(ctx, constant.TracingRemoteSpanCtx, spanCtx)
		}

		if len(reqHeader["Timeout"]) > 0 {
			timeout, err := time.ParseDuration(reqHeader["Timeout"])
			if err == nil {
				httpTimeout = timeout
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, httpTimeout)
				defer cancel()
			}
			delete(reqHeader, "Timeout")
		}
		setTimeout(conn, httpTimeout)

		if err := serveRequest(ctx, reqHeader, reqBody, conn); err != nil {
			if errRsp := sendErrorResp(r.Header, []byte(perrors.WithStack(err).Error())); errRsp != nil {
				logger.Warnf("sendErrorResp(header:%#v, error:%v) = error:%s",
					r.Header, perrors.WithStack(err), errRsp)
			}

			logger.Infof("Unexpected error serving request, closing socket: %v", err)
			return
		}
	}
}

func accept(listener net.Listener, fn func(net.Conn)) error {
	var (
		ok       bool
		ne       net.Error
		tmpDelay time.Duration
	)

	for {
		c, err := listener.Accept()
		if err != nil {
			if ne, ok = err.(net.Error); ok && ne.Temporary() {
				if tmpDelay != 0 {
					tmpDelay <<= 1
				} else {
					tmpDelay = 5 * time.Millisecond
				}
				if tmpDelay > DefaultMaxSleepTime {
					tmpDelay = DefaultMaxSleepTime
				}
				logger.Infof("http: Accept error: %v; retrying in %v\n", err, tmpDelay)
				time.Sleep(tmpDelay)
				continue
			}
			return perrors.WithStack(err)
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					logger.Errorf("http: panic serving %v: %v\n%s", c.RemoteAddr(), r, buf)
					c.Close()
				}
			}()

			fn(c)
		}()
	}
}

// Start JSON RPC server then ready for accept request.
func (s *Server) Start(url *common.URL) {
	listener, err := net.Listen("tcp", url.Location)
	if err != nil {
		logger.Errorf("jsonrpc server [%s] start failed: %v", url.Path, err)
		return
	}
	logger.Infof("rpc server start to listen on %s", listener.Addr())

	s.wg.Add(1)
	go func() {
		if err := accept(listener, func(conn net.Conn) { s.handlePkg(conn) }); err != nil {
			logger.Error("accept() = error:%v", err)
		}
		s.wg.Done()
	}()

	s.wg.Add(1)
	go func() { // Server done goroutine
		var err error
		<-s.done               // step1: block to wait for done channel(wait Server.Stop step2)
		err = listener.Close() // step2: and then close listener
		if err != nil {
			logger.Warnf("listener{addr:%s}.Close() = error{%#v}", listener.Addr(), err)
		}
		s.wg.Done()
	}()
}

// Stop JSON RPC server, just can be call once.
func (s *Server) Stop() {
	s.once.Do(func() {
		close(s.done)
		s.wg.Wait()
	})
}

func serveRequest(ctx context.Context, header map[string]string, body []byte, conn net.Conn) error {
	sendErrorResp := func(header map[string]string, body []byte) error {
		rsp := &http.Response{
			Header:        make(http.Header),
			StatusCode:    500,
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: int64(len(body)),
			Body:          ioutil.NopCloser(bytes.NewReader(body)),
		}
		rsp.Header.Del("Content-Type")
		rsp.Header.Del("Content-Length")
		rsp.Header.Del("Timeout")
		for k, v := range header {
			rsp.Header.Set(k, v)
		}

		rspBuf := bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize))
		rspBuf.Reset()
		err := rsp.Write(rspBuf)
		if err != nil {
			return perrors.WithStack(err)
		}
		_, err = rspBuf.WriteTo(conn)
		return perrors.WithStack(err)
	}

	sendResp := func(header map[string]string, body []byte) error {
		rsp := &http.Response{
			Header:        make(http.Header),
			StatusCode:    200,
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: int64(len(body)),
			Body:          ioutil.NopCloser(bytes.NewReader(body)),
		}
		rsp.Header.Del("Content-Type")
		rsp.Header.Del("Content-Length")
		rsp.Header.Del("Timeout")
		for k, v := range header {
			rsp.Header.Set(k, v)
		}

		rspBuf := bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize))
		rspBuf.Reset()
		err := rsp.Write(rspBuf)
		if err != nil {
			return perrors.WithStack(err)
		}
		_, err = rspBuf.WriteTo(conn)
		return perrors.WithStack(err)
	}

	// read request header
	codec := newServerCodec()
	err := codec.ReadHeader(header, body)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return perrors.WithStack(err)
		}
		return perrors.New("server cannot decode request: " + err.Error())
	}

	path := header["Path"]
	methodName := codec.req.Method
	if len(path) == 0 || len(methodName) == 0 {
		return perrors.New("service/method request ill-formed: " + path + "/" + methodName)
	}

	// read body
	var args []interface{}
	if err = codec.ReadBody(&args); err != nil {
		return perrors.WithStack(err)
	}
	logger.Debugf("args: %v", args)

	// exporter invoke
	exporter, _ := jsonrpcProtocol.ExporterMap().Load(path)
	invoker := exporter.(*JsonrpcExporter).GetInvoker()
	if invoker != nil {
		result := invoker.Invoke(ctx, invocation.NewRPCInvocation(methodName, args, map[string]interface{}{
			constant.PathKey:    path,
			constant.VersionKey: codec.req.Version,
		}))
		if err := result.Error(); err != nil {
			rspStream, codecErr := codec.Write(err.Error(), invalidRequest)
			if codecErr != nil {
				return perrors.WithStack(codecErr)
			}
			if errRsp := sendErrorResp(header, rspStream); errRsp != nil {
				logger.Warnf("Exporter: sendErrorResp(header:%#v, error:%v) = error:%s",
					header, err, errRsp)
			}
		} else {
			res := result.Result()
			rspStream, err := codec.Write("", res)
			if err != nil {
				return perrors.WithStack(err)
			}
			if errRsp := sendResp(header, rspStream); errRsp != nil {
				logger.Warnf("Exporter: sendResp(header:%#v, error:%v) = error:%s",
					header, err, errRsp)
			}
		}
	}

	return nil
}
